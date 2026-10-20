package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// internalError logs the underlying failure and returns a generic 500 body so
// database, filesystem, and upstream details never reach clients.
func internalError(c *gin.Context, err error) {
	if err != nil {
		log.Printf("%s %s: %v", c.Request.Method, c.Request.URL.Path, err)
	}
	c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
}
