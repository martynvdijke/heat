package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// @Summary Send race flag command
// @Description Send a flag command (safety car, red, blue, black/white)
// @Tags Flags
// @Accept json
// @Produce json
// @Param flag body models.FlagCommand true "Flag command"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /api/flags [post]
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
