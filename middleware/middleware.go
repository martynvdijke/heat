package middleware

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/time/rate"
	"heat/app"
	"heat/models"
)

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Next()
	}
}

func RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		app.LoginLimitersMu.Lock()
		limiter, exists := app.LoginLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(5), 10)
			app.LoginLimiters[ip] = limiter
		}
		app.LoginLimitersMu.Unlock()
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		c.Next()
	}
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[AUTH] Checking session for: %s", c.Request.URL.Path)

		var sessionCookie string
		for _, cookie := range c.Request.Cookies() {
			if cookie.Name == "session" {
				if _, ok := app.SessionStore[cookie.Value]; ok {
					sessionCookie = cookie.Value
					break
				}
			}
		}

		if sessionCookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		expiry, ok := app.SessionStore[sessionCookie]
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}
		if time.Now().Unix() > expiry {
			delete(app.SessionStore, sessionCookie)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		c.Next()
	}
}

type umamiResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (w *umamiResponseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func UmamiMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		w := &umamiResponseWriter{ResponseWriter: c.Writer}
		c.Writer = w
		c.Next()

		ct := w.Header().Get("Content-Type")
		if strings.Contains(ct, "text/html") && w.body.Len() > 0 {
			html := w.body.String()
			var s models.UmamiSettings
			err := app.DB.QueryRow("SELECT COALESCE(url, ''), COALESCE(website_id, ''), COALESCE(enabled, 0) FROM umami_settings WHERE id = 1").
				Scan(&s.URL, &s.WebsiteID, &s.Enabled)
			if err == nil && s.Enabled && s.URL != "" && s.WebsiteID != "" {
				script := fmt.Sprintf(`<script defer src="%s/script.js" data-website-id="%s"></script>`, s.URL, s.WebsiteID)
				html = strings.Replace(html, "</head>", script+"\n</head>", 1)
			}
			w.ResponseWriter.WriteHeader(w.Status())
			w.ResponseWriter.Write([]byte(html))
		} else {
			w.ResponseWriter.WriteHeader(w.Status())
			w.ResponseWriter.Write(w.body.Bytes())
		}
	}
}
