package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type logSettingInput struct {
	Module string `json:"module" binding:"required"`
	Level  string `json:"level" binding:"required"`
}

var validLevels = map[string]bool{
	"DEBUG": true,
	"INFO":  true,
	"WARN":  true,
	"ERROR": true,
}

// @Summary Get log settings
// @Description Get the current log verbosity settings per module
// @Tags Logs
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/admin/log-settings [get]
func (h *Handler) GetLogSettings(c *gin.Context) {
	type Setting struct {
		Module string `json:"module"`
		Level  string `json:"level"`
	}

	rows, err := h.S.DB.Query("SELECT module, level FROM log_settings ORDER BY module")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"settings": []Setting{}, "default": "WARN"})
		return
	}
	defer rows.Close()

	settings := make([]Setting, 0)
	hasDefault := false
	for rows.Next() {
		var s Setting
		if err := rows.Scan(&s.Module, &s.Level); err != nil {
			continue
		}
		if s.Module == "default" {
			hasDefault = true
		}
		settings = append(settings, s)
	}

	defaultLevel := "WARN"
	for _, s := range settings {
		if s.Module == "default" {
			defaultLevel = s.Level
			break
		}
	}

	if !hasDefault {
		settings = append(settings, Setting{Module: "default", Level: defaultLevel})
	}

	c.JSON(http.StatusOK, gin.H{
		"settings": settings,
		"default":  defaultLevel,
	})
}

// @Summary Save log settings
// @Description Update log verbosity settings for one or more modules
// @Tags Logs
// @Accept json
// @Produce json
// @Param settings body []logSettingInput true "Log settings array"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/admin/log-settings [post]
func (h *Handler) SaveLogSettings(c *gin.Context) {
	var inputs []logSettingInput
	if err := c.ShouldBindJSON(&inputs); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	for _, input := range inputs {
		if !validLevels[input.Level] {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid level: " + input.Level + ". Must be one of: DEBUG, INFO, WARN, ERROR"})
			return
		}
		_, err := h.S.DB.Exec(
			`INSERT INTO log_settings (module, level) VALUES (?, ?) ON CONFLICT(module) DO UPDATE SET level = ?`,
			input.Module, input.Level, input.Level,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save settings: " + err.Error()})
			return
		}
	}

	// Refresh logger settings
	if h.S.Log != nil {
		h.S.Log.RefreshSettings()
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
