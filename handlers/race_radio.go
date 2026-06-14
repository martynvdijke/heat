package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func (h *Handler) GetRaceRadio(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	racerIDStr := c.Query("racer_id")
	limitStr := c.DefaultQuery("limit", "50")

	query := `SELECT rr.id, rr.race_id, rr.racer_id, COALESCE(r.name, ''), rr.message, rr.timestamp
		FROM race_radio rr
		LEFT JOIN racers r ON r.id = rr.racer_id
		WHERE 1=1`
	var args []any
	if raceIDStr != "" {
		query += " AND rr.race_id = ?"
		args = append(args, raceIDStr)
	}
	if racerIDStr != "" {
		query += " AND rr.racer_id = ?"
		args = append(args, racerIDStr)
	}
	query += " ORDER BY rr.id DESC LIMIT ?"
	args = append(args, limitStr)

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	messages := make([]models.RaceRadioMessage, 0)
	for rows.Next() {
		var m models.RaceRadioMessage
		if err := rows.Scan(&m.ID, &m.RaceID, &m.RacerID, &m.RacerName, &m.Message, &m.Timestamp); err != nil {
			continue
		}
		messages = append(messages, m)
	}

	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	c.JSON(http.StatusOK, messages)
}

func (h *Handler) AddRaceRadio(c *gin.Context) {
	var msg models.RaceRadioMessage
	if err := c.ShouldBindJSON(&msg); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if msg.Message == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}
	if msg.RacerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Racer ID is required"})
		return
	}

	result, err := h.S.DB.Exec("INSERT INTO race_radio (race_id, racer_id, message) VALUES (?, ?, ?)",
		msg.RaceID, msg.RacerID, msg.Message)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	msg.ID = int(id)

	var name string
	h.S.DB.QueryRow("SELECT name FROM racers WHERE id = ?", msg.RacerID).Scan(&name)
	msg.RacerName = name

	select {
	case h.S.RaceRadioBroadcast <- msg:
	default:
	}

	c.JSON(http.StatusOK, msg)
}
