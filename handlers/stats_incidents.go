package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// @Summary Get race incidents report
// @Description Get incidents report for a specific race
// @Tags Stats
// @Produce json
// @Param race_id query int true "Race ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/race-incidents [get]
func (h *Handler) GetRaceIncidentsReport(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	racerIDStr := c.Query("racer_id")

	query := `SELECT re.id, re.race_id, re.lap, re.event_type, re.racer_id, re.racer_id2, re.note, re.timestamp,
		COALESCE(r1.name, ''), COALESCE(r2.name, '')
		FROM race_events re
		LEFT JOIN racers r1 ON r1.id = re.racer_id
		LEFT JOIN racers r2 ON r2.id = re.racer_id2
		WHERE 1=1`

	var args []any
	if raceIDStr != "" {
		query += " AND re.race_id = ?"
		args = append(args, raceIDStr)
	}
	if racerIDStr != "" {
		query += " AND (re.racer_id = ? OR re.racer_id2 = ?)"
		args = append(args, racerIDStr, racerIDStr)
	}
	query += " ORDER BY re.lap, re.id"

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type IncidentView struct {
		models.RaceEvent
		Racer1Name string `json:"racer1_name"`
		Racer2Name string `json:"racer2_name,omitempty"`
	}
	incidents := make([]IncidentView, 0)
	for rows.Next() {
		var iv IncidentView
		if err := rows.Scan(&iv.ID, &iv.RaceID, &iv.Lap, &iv.EventType, &iv.RacerID, &iv.RacerID2, &iv.Note, &iv.Timestamp,
			&iv.Racer1Name, &iv.Racer2Name); err != nil {
			continue
		}
		incidents = append(incidents, iv)
	}
	c.JSON(http.StatusOK, incidents)
}
