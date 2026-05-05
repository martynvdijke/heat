package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

func GetAISettings(c *gin.Context) {
	var s models.AISettings
	var enabled int
	err := app.DB.QueryRow("SELECT id, COALESCE(track_extract_url, ''), COALESCE(api_key, ''), COALESCE(enabled, 0) FROM ai_settings WHERE id = 1").
		Scan(&s.ID, &s.TrackExtractURL, &s.APIKey, &enabled)
	if err != nil {
		s = models.AISettings{ID: 1, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1
	hasKey := s.APIKey != ""
	s.APIKey = ""
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "track_extract_url": s.TrackExtractURL, "has_api_key": hasKey, "enabled": s.Enabled})
}

func SaveAISettings(c *gin.Context) {
	var s models.AISettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.APIKey == "" {
		var existingKey string
		app.DB.QueryRow("SELECT COALESCE(api_key, '') FROM ai_settings WHERE id = 1").Scan(&existingKey)
		s.APIKey = existingKey
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO ai_settings (id, track_extract_url, api_key, enabled) VALUES (1, ?, ?, ?)`,
		s.TrackExtractURL, s.APIKey, boolToInt(s.Enabled))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func GetNotificationSettings(c *gin.Context) {
	var s models.NotificationSettings
	err := app.DB.QueryRow("SELECT id, COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), COALESCE(notify_winner, 0), COALESCE(notify_race_start, 0), COALESCE(notify_podium, 0) FROM notification_settings WHERE id = 1").
		Scan(&s.ID, &s.GotiFyURL, &s.GotiFyToken, &s.NotifyWinner, &s.NotifyRaceStart, &s.NotifyPodium)
	if err != nil {
		s = models.NotificationSettings{ID: 1, NotifyWinner: true, NotifyRaceStart: false, NotifyPodium: false}
	}
	hasToken := s.GotiFyToken != ""
	s.GotiFyToken = ""
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "gotify_url": s.GotiFyURL, "has_gotify_token": hasToken, "notify_winner": s.NotifyWinner, "notify_race_start": s.NotifyRaceStart, "notify_podium": s.NotifyPodium})
}

func SaveNotificationSettings(c *gin.Context) {
	var s models.NotificationSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.GotiFyToken == "" {
		var existingToken string
		app.DB.QueryRow("SELECT COALESCE(gotify_token, '') FROM notification_settings WHERE id = 1").Scan(&existingToken)
		s.GotiFyToken = existingToken
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO notification_settings (id, gotify_url, gotify_token, notify_winner, notify_race_start, notify_podium) VALUES (1, ?, ?, ?, ?, ?)`,
		s.GotiFyURL, s.GotiFyToken, boolToInt(s.NotifyWinner), boolToInt(s.NotifyRaceStart), boolToInt(s.NotifyPodium))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func GetEmailSettings(c *gin.Context) {
	var s models.EmailSettings
	var enabled int
	var password string
	err := app.DB.QueryRow("SELECT id, COALESCE(smtp_host, ''), COALESCE(smtp_port, 587), COALESCE(username, ''), COALESCE(password, ''), COALESCE(from_addr, ''), COALESCE(enabled, 0) FROM email_settings WHERE id = 1").
		Scan(&s.ID, &s.SMTPHost, &s.SMTPPort, &s.Username, &password, &s.FromAddr, &enabled)
	if err != nil {
		s = models.EmailSettings{ID: 1, SMTPPort: 587, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1
	c.JSON(http.StatusOK, gin.H{"id": s.ID, "smtp_host": s.SMTPHost, "smtp_port": s.SMTPPort, "username": s.Username, "has_password": password != "", "from_addr": s.FromAddr, "enabled": s.Enabled})
}

func SaveEmailSettings(c *gin.Context) {
	var s models.EmailSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.Password == "" {
		var existingPw string
		app.DB.QueryRow("SELECT COALESCE(password, '') FROM email_settings WHERE id = 1").Scan(&existingPw)
		s.Password = existingPw
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO email_settings (id, smtp_host, smtp_port, username, password, from_addr, enabled) VALUES (1, ?, ?, ?, ?, ?, ?)`,
		s.SMTPHost, s.SMTPPort, s.Username, s.Password, s.FromAddr, boolToInt(s.Enabled))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func GetUmamiSettings(c *gin.Context) {
	var s models.UmamiSettings
	err := app.DB.QueryRow("SELECT id, COALESCE(url, ''), COALESCE(website_id, ''), COALESCE(enabled, 0) FROM umami_settings WHERE id = 1").
		Scan(&s.ID, &s.URL, &s.WebsiteID, &s.Enabled)
	if err != nil {
		s = models.UmamiSettings{ID: 1, Enabled: false}
	}
	c.JSON(http.StatusOK, s)
}

func SaveUmamiSettings(c *gin.Context) {
	var s models.UmamiSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO umami_settings (id, url, website_id, enabled) VALUES (1, ?, ?, ?)`,
		s.URL, s.WebsiteID, boolToInt(s.Enabled))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func GetOneOffRaces(c *gin.Context) {
	rows, err := app.DB.Query(`SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'oneoff')
					   FROM race_history WHERE race_type = 'oneoff' ORDER BY race_date DESC LIMIT 20`)
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

func DeleteOneOffRace(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}
	app.DB.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	app.DB.Exec("DELETE FROM race_history WHERE id = ? AND race_type = 'oneoff'", id)
	c.Status(http.StatusOK)
}
