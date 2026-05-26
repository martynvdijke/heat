package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"heat/app"
)

func RateLimitMiddleware(s *app.Server) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		s.LoginLimitersMu.Lock()
		limiter, exists := s.LoginLimiters[ip]
		if !exists {
			limiter = rate.NewLimiter(rate.Limit(5), 10)
			s.LoginLimiters[ip] = limiter
		}
		s.LoginLimitersMu.Unlock()
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}
		c.Next()
	}
}
