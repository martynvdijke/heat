package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

func HandleFlag(c *gin.Context) {
	var cmd models.FlagCommand
	if err := c.ShouldBindJSON(&cmd); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if cmd.Type != "flag" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid flag type"})
		return
	}
	select {
	case app.FlagBroadcast <- cmd:
	default:
	}
	c.Status(http.StatusOK)
}
