package middleware

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
)

func AuthMiddleware(s *app.Server) gin.HandlerFunc {
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

		s.SessionStoreMu.RLock()
		info, ok := s.SessionStore[sessionCookie]
		s.SessionStoreMu.RUnlock()
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}
		if time.Now().Unix() > info.Expiry {
			s.SessionStoreMu.Lock()
			delete(s.SessionStore, sessionCookie)
			s.SessionStoreMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		if info.IP != "" && c.ClientIP() != info.IP {
			s.SessionStoreMu.Lock()
			delete(s.SessionStore, sessionCookie)
			s.SessionStoreMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session invalid"})
			return
		}

		c.Next()
	}
}
