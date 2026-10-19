package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// BlockStaticSubdir aborts requests under /media/<sub>/ with a 404. Asset
// binaries live there but are only served through the /api/assets endpoints,
// which validate the asset row and set controlled response headers.
func BlockStaticSubdir(sub string) gin.HandlerFunc {
	prefix := "/media/" + strings.Trim(sub, "/") + "/"
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, prefix) {
			c.AbortWithStatus(http.StatusNotFound)
			return
		}
		c.Next()
	}
}
