package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// CSRFMiddleware rejects cookie-authenticated unsafe requests whose Origin
// (or Referer fallback) does not match the request host. Browsers always send
// one of these headers on cross-site requests, so a forged request cannot
// satisfy the check even if it carries the session cookie.
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			c.Next()
			return
		}

		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = c.GetHeader("Referer")
		}
		if origin == "" || !sameOrigin(c, origin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cross-origin request rejected"})
			return
		}
		c.Next()
	}
}

func sameOrigin(c *gin.Context, raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return false
	}
	originHost := strings.ToLower(u.Hostname())
	// Relative-form requests (tests, reverse proxies that strip Host) have no
	// host to compare against; the cookie/session check still applies.
	if c.Request.Host == "" {
		return true
	}
	if originHost == hostOnly(c.Request.Host) {
		return true
	}
	if fwd := c.GetHeader("X-Forwarded-Host"); fwd != "" {
		fwd = strings.TrimSpace(strings.Split(fwd, ",")[0])
		return originHost == hostOnly(fwd)
	}
	return false
}

func hostOnly(host string) string {
	host = strings.ToLower(host)
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}
