package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// @Summary Get e-ink settings
// @Description Get the current e-ink display mode setting
// @Tags Settings
// @Produce json
// @Success 200 {object} models.EInkSettings
// @Security cookieAuth
// @Router /api/eink-settings [get]
func (h *Handler) GetEInkSettings(c *gin.Context) {
	var s models.EInkSettings
	var enabled int
	err := h.S.DB.QueryRow("SELECT id, COALESCE(enabled, 0) FROM eink_settings WHERE id = 1").
		Scan(&s.ID, &enabled)
	if err != nil {
		s = models.EInkSettings{ID: 1, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1
	c.JSON(http.StatusOK, s)
}

// @Summary Save e-ink settings
// @Description Enable or disable e-ink display mode site-wide
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.EInkSettings true "E-ink settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/eink-settings [post]
func (h *Handler) SaveEInkSettings(c *gin.Context) {
	var s models.EInkSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := h.S.DB.Exec(`INSERT INTO eink_settings (id, enabled) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET enabled = ?`, enabled, enabled)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save e-ink settings: " + err.Error()})
		return
	}
	h.S.Log.Infof("eink", "E-ink settings saved: enabled=%v", s.Enabled)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
