package handlers

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/middleware"
	"heat/models"
)

// @Summary Get AI settings
// @Description Get the AI track extraction settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/ai-settings [get]
func (h *Handler) GetAISettings(c *gin.Context) {
	var s models.AISettings
	var enabled int
	err := h.S.DB.QueryRow("SELECT id, COALESCE(track_extract_url, ''), COALESCE(api_key, ''), COALESCE(enabled, 0) FROM ai_settings WHERE id = 1").
		Scan(&s.ID, &s.TrackExtractURL, &s.APIKey, &enabled)
	if err != nil {
		s = models.AISettings{ID: 1, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1
	hasKey := s.APIKey != ""
	s.APIKey = ""
	h.S.Log.Debugf("ai", "GetAISettings: enabled=%v has_key=%v", s.Enabled, hasKey)
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "track_extract_url": s.TrackExtractURL, "has_api_key": hasKey, "enabled": s.Enabled})
}

// @Summary Save AI settings
// @Description Save the AI track extraction settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.AISettings true "AI settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/ai-settings [post]
func (h *Handler) SaveAISettings(c *gin.Context) {
	var s models.AISettings
	if err := c.ShouldBindJSON(&s); err != nil {
		h.S.Log.Errorf("ai", "SaveAISettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.APIKey == "" {
		var existingKey string
		h.S.DB.QueryRow("SELECT COALESCE(api_key, '') FROM ai_settings WHERE id = 1").Scan(&existingKey)
		s.APIKey = existingKey
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO ai_settings (id, track_extract_url, api_key, enabled) VALUES (1, ?, ?, ?)`,
		s.TrackExtractURL, s.APIKey, db.BoolToInt(s.Enabled))
	if err != nil {
		h.S.Log.Errorf("ai", "SaveAISettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.Log.Infof("ai", "AI settings saved: track_extract_url=%q enabled=%v", s.TrackExtractURL, s.Enabled)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Get notification settings
// @Description Get Gotify notification settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/notification-settings [get]
func (h *Handler) GetNotificationSettings(c *gin.Context) {
	var s models.NotificationSettings
	err := h.S.DB.QueryRow("SELECT id, COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), COALESCE(notify_winner, 0), COALESCE(notify_race_start, 0), COALESCE(notify_podium, 0) FROM notification_settings WHERE id = 1").
		Scan(&s.ID, &s.GotiFyURL, &s.GotiFyToken, &s.NotifyWinner, &s.NotifyRaceStart, &s.NotifyPodium)
	if err != nil {
		s = models.NotificationSettings{ID: 1, NotifyWinner: true, NotifyRaceStart: false, NotifyPodium: false}
	}
	hasToken := s.GotiFyToken != ""
	s.GotiFyToken = ""
	h.S.Log.Debugf("notification", "GetNotificationSettings: url=%q notify_winner=%v", s.GotiFyURL, s.NotifyWinner)
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "gotify_url": s.GotiFyURL, "has_gotify_token": hasToken, "notify_winner": s.NotifyWinner, "notify_race_start": s.NotifyRaceStart, "notify_podium": s.NotifyPodium})
}

// @Summary Save notification settings
// @Description Save Gotify notification settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.NotificationSettings true "Notification settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/notification-settings [post]
func (h *Handler) SaveNotificationSettings(c *gin.Context) {
	var s models.NotificationSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		h.S.Log.Errorf("notification", "SaveNotificationSettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.GotiFyURL == "" {
		var existingURL string
		h.S.DB.QueryRow("SELECT COALESCE(gotify_url, '') FROM notification_settings WHERE id = 1").Scan(&existingURL)
		s.GotiFyURL = existingURL
	}
	if s.GotiFyToken == "" {
		var existingToken string
		h.S.DB.QueryRow("SELECT COALESCE(gotify_token, '') FROM notification_settings WHERE id = 1").Scan(&existingToken)
		s.GotiFyToken = existingToken
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO notification_settings (id, gotify_url, gotify_token, notify_winner, notify_race_start, notify_podium) VALUES (1, ?, ?, ?, ?, ?)`,
		s.GotiFyURL, s.GotiFyToken, db.BoolToInt(s.NotifyWinner), db.BoolToInt(s.NotifyRaceStart), db.BoolToInt(s.NotifyPodium))
	if err != nil {
		h.S.Log.Errorf("notification", "SaveNotificationSettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.Log.Infof("notification", "Notification settings saved: url=%q winner=%v podium=%v race_start=%v", s.GotiFyURL, s.NotifyWinner, s.NotifyPodium, s.NotifyRaceStart)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Get email settings
// @Description Get SMTP email settings
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/email-settings [get]
func (h *Handler) GetEmailSettings(c *gin.Context) {
	var s models.EmailSettings
	var enabled int
	var password string
	err := h.S.DB.QueryRow("SELECT id, COALESCE(smtp_host, ''), COALESCE(smtp_port, 587), COALESCE(username, ''), COALESCE(password, ''), COALESCE(from_addr, ''), COALESCE(enabled, 0) FROM email_settings WHERE id = 1").
		Scan(&s.ID, &s.SMTPHost, &s.SMTPPort, &s.Username, &password, &s.FromAddr, &enabled)
	if err != nil {
		s = models.EmailSettings{ID: 1, SMTPPort: 587, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1

	var adminEmail string
	h.S.DB.QueryRow("SELECT COALESCE(email, '') FROM admin_users ORDER BY id LIMIT 1").Scan(&adminEmail)

	h.S.Log.Debugf("email", "GetEmailSettings: host=%q from=%q enabled=%v", s.SMTPHost, s.FromAddr, s.Enabled)
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "smtp_host": s.SMTPHost, "smtp_port": s.SMTPPort, "username": s.Username, "has_password": password != "", "from_addr": s.FromAddr, "enabled": s.Enabled, "admin_email": adminEmail})
}

// @Summary Save email settings
// @Description Save SMTP email settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.EmailSettings true "Email settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/email-settings [post]
func (h *Handler) SaveEmailSettings(c *gin.Context) {
	var input struct {
		models.EmailSettings
		AdminEmail string `json:"admin_email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		h.S.Log.Errorf("email", "SaveEmailSettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s := input.EmailSettings

	if s.Password == "" {
		var existingPw string
		h.S.DB.QueryRow("SELECT COALESCE(password, '') FROM email_settings WHERE id = 1").Scan(&existingPw)
		s.Password = existingPw
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO email_settings (id, smtp_host, smtp_port, username, password, from_addr, enabled) VALUES (1, ?, ?, ?, ?, ?, ?)`,
		s.SMTPHost, s.SMTPPort, s.Username, s.Password, s.FromAddr, db.BoolToInt(s.Enabled))
	if err != nil {
		h.S.Log.Errorf("email", "SaveEmailSettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Persist the admin recovery email on the first admin user (used for password resets).
	if input.AdminEmail != "" {
		h.S.DB.Exec("UPDATE admin_users SET email = ? WHERE id = (SELECT MIN(id) FROM admin_users)", input.AdminEmail)
	}

	h.S.Log.Infof("email", "Email settings saved: host=%q from=%q enabled=%v", s.SMTPHost, s.FromAddr, s.Enabled)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Get analytics settings
// @Description Get Umami analytics settings
// @Tags Settings
// @Produce json
// @Success 200 {object} models.UmamiSettings
// @Security cookieAuth
// @Router /api/umami-settings [get]
func (h *Handler) GetUmamiSettings(c *gin.Context) {
	var s models.UmamiSettings
	err := h.S.DB.QueryRow("SELECT id, COALESCE(url, ''), COALESCE(website_id, ''), COALESCE(enabled, 0) FROM umami_settings WHERE id = 1").
		Scan(&s.ID, &s.URL, &s.WebsiteID, &s.Enabled)
	if err != nil {
		s = models.UmamiSettings{ID: 1, Enabled: false}
	}
	h.S.Log.Debugf("umami", "GetUmamiSettings: url=%q enabled=%v", s.URL, s.Enabled)
	c.JSON(http.StatusOK, s)
}

// @Summary Save analytics settings
// @Description Save Umami analytics settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.UmamiSettings true "Umami settings"
// @Success 200 {object} models.UmamiSettings
// @Security cookieAuth
// @Router /api/umami-settings [post]
func (h *Handler) SaveUmamiSettings(c *gin.Context) {
	var s models.UmamiSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		h.S.Log.Errorf("umami", "SaveUmamiSettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO umami_settings (id, url, website_id, enabled) VALUES (1, ?, ?, ?)`,
		s.URL, s.WebsiteID, db.BoolToInt(s.Enabled))
	if err != nil {
		h.S.Log.Errorf("umami", "SaveUmamiSettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.Log.Infof("umami", "Umami settings saved: url=%q enabled=%v", s.URL, s.Enabled)
	c.JSON(http.StatusOK, s)
}

// @Summary Get OTel settings
// @Description Get the OpenTelemetry endpoint settings
// @Tags Settings
// @Produce json
// @Success 200 {object} models.OTelSettings
// @Security cookieAuth
// @Router /api/otel-settings [get]
func (h *Handler) GetOTelSettings(c *gin.Context) {
	var s models.OTelSettings
	var tracesEnabled, metricsEnabled, logsEnabled int
	err := middleware.TraceDBQuery(c.Request.Context(), "GetOTelSettings", func(ctx context.Context) error {
		return h.S.DB.QueryRowContext(ctx, "SELECT id, COALESCE(endpoint, ''), COALESCE(traces_enabled, 0), COALESCE(metrics_enabled, 0), COALESCE(logs_enabled, 0) FROM otel_settings WHERE id = 1").
			Scan(&s.ID, &s.Endpoint, &tracesEnabled, &metricsEnabled, &logsEnabled)
	})
	if err != nil {
		s = models.OTelSettings{ID: 1}
		c.JSON(http.StatusOK, s)
		return
	}
	s.TracesEnabled = tracesEnabled == 1
	s.MetricsEnabled = metricsEnabled == 1
	s.LogsEnabled = logsEnabled == 1
	h.S.Log.Debugf("otel", "GetOTelSettings: endpoint=%q", s.Endpoint)
	c.JSON(http.StatusOK, s)
}

// @Summary Save OTel settings
// @Description Save the OpenTelemetry endpoint settings
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.OTelSettings true "OTel settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/otel-settings [post]
func (h *Handler) SaveOTelSettings(c *gin.Context) {
	var s models.OTelSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		h.S.Log.Errorf("otel", "SaveOTelSettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO otel_settings (id, endpoint, traces_enabled, metrics_enabled, logs_enabled) VALUES (1, ?, ?, ?, ?)`,
		s.Endpoint, db.BoolToInt(s.TracesEnabled), db.BoolToInt(s.MetricsEnabled), db.BoolToInt(s.LogsEnabled))
	if err != nil {
		h.S.Log.Errorf("otel", "SaveOTelSettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.Log.Infof("otel", "OTel settings saved: endpoint=%q traces=%v metrics=%v logs=%v", s.Endpoint, s.TracesEnabled, s.MetricsEnabled, s.LogsEnabled)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Get one-off races
// @Description Get the list of one-off race history entries
// @Tags Race
// @Produce json
// @Success 200 {array} models.RaceHistory
// @Router /api/oneoff-races [get]
func (h *Handler) GetOneOffRaces(c *gin.Context) {
	var rows *sql.Rows
	err := middleware.TraceDBQuery(c.Request.Context(), "GetOneOffRaces", func(ctx context.Context) error {
		var innerErr error
		rows, innerErr = h.S.DB.QueryContext(ctx, `SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'oneoff')
					   FROM race_history WHERE race_type = 'oneoff' ORDER BY race_date DESC LIMIT 20`)
		return innerErr
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	history := make([]models.RaceHistory, 0)
	for rows.Next() {
		var h models.RaceHistory
		rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &h.RaceType)
		history = append(history, h)
	}
	c.JSON(http.StatusOK, history)
}

// @Summary Delete a one-off race
// @Description Delete a one-off race history entry by ID
// @Tags Race
// @Produce json
// @Param id query string true "Race ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/oneoff-races [delete]
func (h *Handler) DeleteOneOffRace(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}
	h.S.DB.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	h.S.DB.Exec("DELETE FROM race_history WHERE id = ? AND race_type = 'oneoff'", id)
	c.Status(http.StatusOK)
}
