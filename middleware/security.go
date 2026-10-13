package middleware

import (
	"os"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

// baseCSP is the Content-Security-Policy shared by every response. The
// frame-ancestors directive is appended per server policy (see
// frameAncestorsPolicy).
const baseCSP = "default-src 'self'; " +
	"style-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com https://fonts.googleapis.com https://unpkg.com 'unsafe-inline'; " +
	"script-src 'self' https://cdn.jsdelivr.net https://cdnjs.cloudflare.com https://unpkg.com 'unsafe-inline'; " +
	"font-src 'self' https://cdnjs.cloudflare.com https://fonts.gstatic.com; " +
	"img-src 'self' data: https://*.basemaps.cartocdn.com https://www.transparenttextures.com; " +
	"connect-src 'self' https://cdn.jsdelivr.net https://unpkg.com ws: wss:; " +
	"worker-src 'self'"

// frameAncestorsPolicy resolves the HEAT_FRAME_ANCESTORS environment variable
// into a CSP frame-ancestors value plus the matching legacy X-Frame-Options
// value (empty means "do not send X-Frame-Options").
//
// Unset/empty → deny all framing (the historical behaviour). "*" → allow any
// origin, so pages like the live race view can be embedded in external decks
// (e.g. a Slidev presentation). Any other value is a comma/space separated
// list of ancestor origins; the instance's own origin ('self') is always
// included so same-origin embeds keep working.
func frameAncestorsPolicy() (ancestors string, xFrameOptions string) {
	raw := strings.TrimSpace(os.Getenv("HEAT_FRAME_ANCESTORS"))
	switch raw {
	case "":
		return "'none'", "DENY"
	case "*":
		return "*", ""
	}

	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
	if len(fields) == 0 {
		return "'none'", "DENY"
	}

	seen := map[string]bool{}
	ordered := make([]string, 0, len(fields)+1)
	add := func(value string) {
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		ordered = append(ordered, value)
	}
	add("'self'") // same-origin embedding is always permitted when opting in
	for _, field := range fields {
		if field == "*" {
			// A wildcard anywhere in the list means "allow everyone".
			return "*", ""
		}
		if field == "self" {
			field = "'self'"
		}
		add(field)
	}

	ancestors = strings.Join(ordered, " ")
	if ancestors == "'self'" {
		// A single-origin policy still maps cleanly onto the legacy header.
		return ancestors, "SAMEORIGIN"
	}
	// X-Frame-Options cannot express a list of origins; CSP frame-ancestors
	// is the authoritative directive for modern browsers.
	return ancestors, ""
}

func SecurityHeaders() gin.HandlerFunc {
	ancestors, xFrameOptions := frameAncestorsPolicy()
	csp := baseCSP + "; frame-ancestors " + ancestors

	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		if xFrameOptions != "" {
			c.Header("X-Frame-Options", xFrameOptions)
		}
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Content-Security-Policy", csp)
		c.Next()
	}
}
