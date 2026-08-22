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
// @Param season_ids query string false "Comma-separated season IDs; absent = all seasons"
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/race-incidents [get]
func (h *Handler) GetRaceIncidentsReport(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	racerIDStr := c.Query("racer_id")

	seasonIDs, err := parseSeasonScope(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
	// Season scoping goes through race_history; only applied when a scope is
	// requested so live sessions (race_id = 0) stay visible otherwise.
	if len(seasonIDs) > 0 {
		query += " AND re.race_id IN (SELECT rh.id FROM race_history rh WHERE rh.season_id IN ("
		for i := range seasonIDs {
			if i > 0 {
				query += ","
			}
			query += "?"
			args = append(args, seasonIDs[i])
		}
		query += "))"
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
