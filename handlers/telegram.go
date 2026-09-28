package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/models"
	"heat/telegram"
)

// @Summary Get Telegram settings
// @Description Get the Telegram bot settings. The bot token is never returned; has_bot_token reports whether one is stored.
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Security cookieAuth
// @Router /api/telegram-settings [get]
func (h *Handler) GetTelegramSettings(c *gin.Context) {
	s, err := telegram.LoadSettings(h.S)
	if err != nil {
		s = models.TelegramSettings{ID: 1, NotifyResults: true, NotifyNextRace: true, ReminderDays: "7,1", ReminderHour: 18}
	}
	hasToken := s.BotToken != ""
	s.BotToken = ""

	var subscribers int
	h.S.DB.QueryRow("SELECT COUNT(*) FROM telegram_subscribers WHERE subscribed = 1").Scan(&subscribers)

	c.JSON(http.StatusOK, gin.H{
		"id":               s.ID,
		"has_bot_token":    hasToken,
		"enabled":          s.Enabled,
		"default_chat_id":  s.DefaultChatID,
		"notify_results":   s.NotifyResults,
		"notify_next_race": s.NotifyNextRace,
		"reminder_days":    s.ReminderDays,
		"reminder_hour":    s.ReminderHour,
		"subscribers":      subscribers,
	})
}

// @Summary Save Telegram settings
// @Description Save the Telegram bot settings. An empty bot_token keeps the stored one.
// @Tags Settings
// @Accept json
// @Produce json
// @Param settings body models.TelegramSettings true "Telegram settings"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/telegram-settings [post]
func (h *Handler) SaveTelegramSettings(c *gin.Context) {
	var s models.TelegramSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		h.S.Log.Errorf("telegram", "SaveTelegramSettings: invalid JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if s.BotToken == "" {
		var existing string
		h.S.DB.QueryRow("SELECT COALESCE(bot_token, '') FROM telegram_settings WHERE id = 1").Scan(&existing)
		s.BotToken = existing
	}

	_, err := h.S.DB.Exec(`INSERT OR REPLACE INTO telegram_settings
		(id, bot_token, enabled, default_chat_id, notify_results, notify_next_race, reminder_days, reminder_hour)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?)`,
		s.BotToken, db.BoolToInt(s.Enabled), s.DefaultChatID,
		db.BoolToInt(s.NotifyResults), db.BoolToInt(s.NotifyNextRace),
		s.ReminderDays, s.ReminderHour)
	if err != nil {
		h.S.Log.Errorf("telegram", "SaveTelegramSettings: DB error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.Log.Infof("telegram", "Telegram settings saved: enabled=%v chat=%q results=%v reminders=%v", s.Enabled, s.DefaultChatID, s.NotifyResults, s.NotifyNextRace)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Test Telegram bot
// @Description Send a test message to the configured default chat
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/telegram-settings/test [post]
func (h *Handler) TestTelegram(c *gin.Context) {
	s, err := telegram.LoadSettings(h.S)
	if err != nil || s.BotToken == "" || s.DefaultChatID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Bot token and default chat ID must be configured"})
		return
	}
	if err := telegram.New(h.S).SendTest(s.BotToken, s.DefaultChatID, "🏁 <b>HEAT bot online</b> — all systems go!"); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// GetTelegramSummary is the public read-only payload the bot consumes: the
// latest finalized round with its full finishing order, the complete season
// championship standings, the upcoming race, and the active season. It is
// safe to expose because it contains only results already visible to the
// public stats pages.
func (h *Handler) GetTelegramSummary(c *gin.Context) {
	latestRace := h.trmnlLatestRace(c, 50)
	season, seasonID := h.trmnlSeason()

	standings := make([]models.SeasonStanding, 0)
	if season != nil {
		standings = h.trmnlStandings(c, seasonID, 100)
	}

	nextRace, err := h.trmnlNextRace()
	if err != nil {
		h.S.Log.Errorf("telegram", "GetTelegramSummary: next race lookup failed: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"latest_race": latestRace,
		"standings":   standings,
		"next_race":   nextRace,
		"season":      season,
	})
}

// enqueueTelegram hands an event to the bot without ever blocking the caller.
func (h *Handler) enqueueTelegram(evt models.TelegramEvent) {
	if h.S.TelegramBroadcast == nil {
		return
	}
	select {
	case h.S.TelegramBroadcast <- evt:
	default:
		h.S.Log.Debugf("telegram", "enqueueTelegram: channel full, dropping %q event", evt.Kind)
	}
}
