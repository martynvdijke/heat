package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
)

// RacerAuthMiddleware authenticates a verified racer's website session. It is
// deliberately separate from AuthMiddleware (admin sessions) so that a racer
// session can never satisfy an admin-only route.
//
// Unlike admin sessions, racer sessions are not bound to a client IP because
// racers frequently switch networks on mobile devices.
func RacerAuthMiddleware(s *app.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		for _, cookie := range c.Request.Cookies() {
			if cookie.Name == app.RacerSessionCookie {
				token = cookie.Value
				break
			}
		}
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		s.RacerSessionsMu.RLock()
		info, ok := s.RacerSessions[token]
		s.RacerSessionsMu.RUnlock()
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}
		if time.Now().Unix() > info.Expiry {
			s.RacerSessionsMu.Lock()
			delete(s.RacerSessions, token)
			s.RacerSessionsMu.Unlock()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		c.Set("racer_id", info.RacerID)
		c.Next()
	}
}

// RacerID returns the authenticated racer id set by RacerAuthMiddleware.
func RacerID(c *gin.Context) int {
	if v, ok := c.Get("racer_id"); ok {
		if id, ok := v.(int); ok {
			return id
		}
	}
	return 0
}
