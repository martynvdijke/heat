package handlers

import (
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// Commentary templates: hardcoded Go map (no DB store, no i18n in this change).
// Placeholders: {{driver}}, {{target}} (from racer_id2 when present, else a
// generic rival), {{lap}}.
var commentaryTemplates = map[string][]string{
	"overtake": {
		"{{driver}} overtakes {{target}} on lap {{lap}}!",
		"Brilliant move by {{driver}} — past {{target}} on lap {{lap}}!",
		"{{driver}} sweeps around {{target}} on lap {{lap}}!",
	},
	"crash": {
		"{{driver}} crashes out on lap {{lap}}!",
		"Big shunt for {{driver}} on lap {{lap}}!",
		"{{driver}} is in trouble — crash on lap {{lap}}!",
	},
	"spin": {
		"{{driver}} spins on lap {{lap}}!",
		"{{driver}} loses it and spins on lap {{lap}}!",
		"Off track! {{driver}} spins on lap {{lap}}!",
	},
	"safety_car": {
		"Safety car deployed on lap {{lap}}!",
		"The safety car is out on lap {{lap}}!",
	},
	"pit_stop": {
		"{{driver}} pits on lap {{lap}}!",
		"{{driver}} comes in for a pit stop on lap {{lap}}!",
	},
}

// Weather templates keyed by condition.
var weatherTemplates = map[string][]string{
	"dry":        {"Conditions change: Dry on lap {{lap}}."},
	"damp":       {"Conditions change: Damp on lap {{lap}}."},
	"wet":        {"Conditions change: Wet on lap {{lap}}."},
	"torrential": {"Conditions change: Torrential on lap {{lap}}."},
}

// GetCommentary returns commentary entries, newest-first. Filters: race_id,
// since (id > since), limit (default 50).
func (h *Handler) GetCommentary(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	sinceStr := c.Query("since")
	limitStr := c.DefaultQuery("limit", "50")

	query := `SELECT c.id, c.race_id, c.lap, COALESCE(c.racer_id, 0), COALESCE(r.name, ''), c.message, COALESCE(c.template_key, ''), c.created_at
		FROM commentary c
		LEFT JOIN racers r ON r.id = c.racer_id
		WHERE 1=1`
	var args []any
	if raceIDStr != "" {
		query += " AND c.race_id = ?"
		args = append(args, raceIDStr)
	}
	if sinceStr != "" {
		query += " AND c.id > ?"
		args = append(args, sinceStr)
	}
	query += " ORDER BY c.id DESC LIMIT ?"
	args = append(args, limitStr)

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	entries := make([]models.Commentary, 0)
	for rows.Next() {
		var e models.Commentary
		if err := rows.Scan(&e.ID, &e.RaceID, &e.Lap, &e.RacerID, &e.RacerName, &e.Message, &e.TemplateKey, &e.CreatedAt); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	c.JSON(http.StatusOK, entries)
}

// AddCommentary stores a manual commentary entry (verbatim, template_key NULL)
// and broadcasts it.
func (h *Handler) AddCommentary(c *gin.Context) {
	var req struct {
		RaceID  int    `json:"race_id"`
		Lap     int    `json:"lap"`
		RacerID int    `json:"racer_id"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Message == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	entry := models.Commentary{
		RaceID:  req.RaceID,
		Lap:     req.Lap,
		RacerID: req.RacerID,
		Message: req.Message,
	}
	if err := h.insertCommentary(&entry); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, entry)
}

// generateCommentary builds a commentary entry from a race event template and
// persists + broadcasts it. Unknown event types are ignored.
func (h *Handler) generateCommentary(raceID, lap int, eventType string, racerID, racerID2 int) {
	variants, ok := commentaryTemplates[eventType]
	if !ok {
		return
	}
	tmpl := variants[rand.Intn(len(variants))]
	driver := h.racerName(racerID)
	target := ""
	if racerID2 > 0 {
		target = h.racerName(racerID2)
	}
	if target == "" {
		target = "a rival"
	}
	message := strings.ReplaceAll(tmpl, "{{driver}}", driver)
	message = strings.ReplaceAll(message, "{{target}}", target)
	message = strings.ReplaceAll(message, "{{lap}}", strconv.Itoa(lap))

	entry := models.Commentary{
		RaceID:      raceID,
		Lap:         lap,
		RacerID:     racerID,
		Message:     message,
		TemplateKey: eventType,
	}
	if err := h.insertCommentary(&entry); err != nil {
		return
	}
}

// generateWeatherCommentary builds a condition-change entry keyed by condition.
func (h *Handler) generateWeatherCommentary(raceID, lap int, condition string) {
	variants, ok := weatherTemplates[condition]
	if !ok {
		return
	}
	tmpl := variants[rand.Intn(len(variants))]
	message := strings.ReplaceAll(tmpl, "{{lap}}", strconv.Itoa(lap))

	entry := models.Commentary{
		RaceID:      raceID,
		Lap:         lap,
		Message:     message,
		TemplateKey: "weather_" + condition,
	}
	if err := h.insertCommentary(&entry); err != nil {
		return
	}
}

// insertCommentary persists an entry, fills in racer name + created_at, and
// broadcasts it over the CommentaryBroadcast channel.
func (h *Handler) insertCommentary(entry *models.Commentary) error {
	result, err := h.S.DB.Exec("INSERT INTO commentary (race_id, lap, racer_id, message, template_key) VALUES (?, ?, ?, ?, ?)",
		entry.RaceID, entry.Lap, entry.RacerID, entry.Message, entry.TemplateKey)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	entry.ID = int(id)
	if entry.RacerID > 0 {
		h.S.DB.QueryRow("SELECT name FROM racers WHERE id = ?", entry.RacerID).Scan(&entry.RacerName)
	}
	h.S.DB.QueryRow("SELECT created_at FROM commentary WHERE id = ?", id).Scan(&entry.CreatedAt)

	select {
	case h.S.CommentaryBroadcast <- *entry:
	default:
	}
	return nil
}

// racerName resolves a racer id to a name, falling back to a generic label.
func (h *Handler) racerName(id int) string {
	var name string
	h.S.DB.QueryRow("SELECT name FROM racers WHERE id = ?", id).Scan(&name)
	if name == "" {
		return "a driver"
	}
	return name
}
