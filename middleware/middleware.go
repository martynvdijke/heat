package middleware

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"net/url"
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
		c.Header("Content-Security-Policy", "default-src 'self'; style-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com https://fonts.googleapis.com https://unpkg.com 'unsafe-inline'; script-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com https://unpkg.com 'unsafe-inline'; font-src 'self' https://cdnjs.cloudflare.com https://fonts.gstatic.com; img-src 'self' data: https://*.basemaps.cartocdn.com https://www.transparenttextures.com; connect-src 'self' https://cdn.jsdelivr.net ws: wss:")
		c.Next()
	}
}

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" || c.Request.Method == "HEAD" || c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = c.Request.Header.Get("Referer")
		}
		if origin == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF check failed"})
			return
		}
		u, err := url.Parse(origin)
		if err != nil || u.Host != c.Request.Host {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "CSRF check failed"})
			return
		}
		c.Next()
	}
}

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Request.Header.Get("X-Request-ID")
		if id == "" {
			b := make([]byte, 16)
			rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Header("X-Request-ID", id)
		c.Set("request_id", id)
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
				sessionCookie = cookie.Value
				break
			}
		}

		if sessionCookie == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		app.SessionStoreMu.RLock()
		info, ok := app.SessionStore[sessionCookie]
		app.SessionStoreMu.RUnlock()
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}
		if time.Now().Unix() > info.Expiry {
			app.SessionStoreMu.Lock()
			delete(app.SessionStore, sessionCookie)
			app.SessionStoreMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		if info.IP != "" && c.ClientIP() != info.IP {
			app.SessionStoreMu.Lock()
			delete(app.SessionStore, sessionCookie)
			app.SessionStoreMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session invalid"})
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
