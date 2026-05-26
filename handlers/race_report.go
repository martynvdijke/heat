package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/racing"
)

// @Summary Get race report
// @Description Get a comprehensive race report
// @Tags Stats
// @Produce json
// @Param race_id query int true "Race ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/race-report [get]
func (h *Handler) GetRaceReport(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	var raceID int
	if raceIDStr == "" {
		h.S.DB.QueryRow("SELECT COALESCE(MAX(id), 0) FROM race_history").Scan(&raceID)
	} else {
		raceID, _ = strconv.Atoi(raceIDStr)
	}
	if raceID <= 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "race not found"})
		return
	}

	report, err := racing.RaceReport(h.S.DB, raceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "race not found"})
		return
	}
	c.JSON(http.StatusOK, report)
}
