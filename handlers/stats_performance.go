package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/racing"
)

// @Summary Get track performance
// @Description Get per-track statistics for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/track-performance [get]
func (h *Handler) GetTrackPerformance(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	if racerIDStr != "" {
		racerID, err := strconv.Atoi(racerIDStr)
		if err != nil || racerID <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid racer_id"})
			return
		}
		results, err := racing.TrackPerformance(h.S.DB, racerID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if results == nil {
			results = []racing.TrackPerformanceData{}
		}
		c.JSON(http.StatusOK, results)
		return
	}

	// All tracks summary mode
	type TrackSummary struct {
		TrackID       string `json:"track_id"`
		TrackName     string `json:"track_name"`
		Country       string `json:"country"`
		UniqueDrivers int    `json:"unique_drivers"`
		TotalEntries  int    `json:"total_entries"`
	}
	rows, err := h.S.DB.Query(`
		SELECT rh.track_id, rh.track, rh.country,
			COUNT(DISTINCT rr.racer_id) as unique_drivers,
			COUNT(*) as total_entries
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'
		GROUP BY rh.track_id
		ORDER BY rh.track
	`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var summary []TrackSummary
	for rows.Next() {
		var s TrackSummary
		var winner string
		if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.UniqueDrivers, &s.TotalEntries, &winner); err != nil {
			continue
		}
		summary = append(summary, s)
	}
	c.JSON(http.StatusOK, summary)
}

// @Summary Qualifying vs race delta
// @Description Get qualifying position vs race position delta for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/qualifying-delta [get]
func (h *Handler) GetQualifyingRaceDelta(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	if racerIDStr == "" {
		c.JSON(http.StatusOK, []racing.QualifyingRaceDeltaData{})
		return
	}
	racerID, err := strconv.Atoi(racerIDStr)
	if err != nil || racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid racer_id"})
		return
	}

	deltas, err := racing.QualifyingRaceDelta(h.S.DB, racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if deltas == nil {
		deltas = []racing.QualifyingRaceDeltaData{}
	}
	c.JSON(http.StatusOK, deltas)
}

// @Summary Get consistency ratings
// @Description Get consistency ratings for all racers
// @Tags Stats
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/consistency [get]
func (h *Handler) GetConsistencyRatings(c *gin.Context) {
	ratings, err := racing.ConsistencyRatings(h.S.DB)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ratings)
}

// @Summary Get pace heatmap
// @Description Get lap-by-lap pace data for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/pace-heatmap [get]
func (h *Handler) GetPaceHeatmap(c *gin.Context) {
	racerIDStr := c.Query("racer_id")

	query := `SELECT lr.racer_id, COALESCE(r.name, ''), lr.lap_number, lr.position, lr.gear_used, lr.heat_generated, lr.turbo_used
		FROM lap_records lr
		LEFT JOIN racers r ON r.id = lr.racer_id
		WHERE 1=1`

	var args []interface{}
	if racerIDStr != "" {
		query += " AND lr.racer_id = ?"
		args = append(args, racerIDStr)
	}
	query += " ORDER BY lr.racer_id, lr.lap_number"

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type PacePoint struct {
		RacerID       int    `json:"racer_id"`
		RacerName     string `json:"racer_name"`
		Lap           int    `json:"lap"`
		Position      int    `json:"position"`
		GearUsed      int    `json:"gear_used"`
		HeatGenerated int    `json:"heat_generated"`
		TurboUsed     bool   `json:"turbo_used"`
	}

	points := make([]PacePoint, 0)
	for rows.Next() {
		var pp PacePoint
		var turbo int
		if err := rows.Scan(&pp.RacerID, &pp.RacerName, &pp.Lap, &pp.Position, &pp.GearUsed, &pp.HeatGenerated, &turbo); err != nil {
			continue
		}
		pp.TurboUsed = turbo == 1
		points = append(points, pp)
	}
	c.JSON(http.StatusOK, points)
}
