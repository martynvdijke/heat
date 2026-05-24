package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
	"heat/racing"
)

// @Summary Get racer stats
// @Description Get statistics for all racers or a specific racer by ID
// @Tags Stats
// @Produce json
// @Param id query int false "Racer ID"
// @Param season_id query int false "Season ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/racer-stats [get]
func (h *Handler) GetRacerStats(c *gin.Context) {
	id := c.Query("id")
	seasonID := c.Query("season_id")

	if seasonID != "" {
		sid, err := strconv.Atoi(seasonID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid season_id"})
			return
		}
		startDate, endDate, err := racing.SeasonDates(h.S.DB, sid)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}

		if id == "" {
			stats := racing.RacerStatsBySeason(h.S.DB, startDate, endDate)
			if len(stats) == 0 {
				stats = racing.RacerStatsFallback(h.S.DB)
			}
			c.JSON(http.StatusOK, stats)
			return
		}

		racerID, _ := strconv.Atoi(id)
		s, found := racing.SingleRacerStatsBySeason(h.S.DB, racerID, startDate, endDate)
		if !found {
			s, _ = racing.SingleRacerStatsFallback(h.S.DB, racerID)
		}
		rInfo := racing.RacerInfo(h.S.DB, racerID)
		c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
		return
	}

	if id == "" {
		c.JSON(http.StatusOK, racing.AllRacerStats(h.S.DB))
		return
	}

	racerID, _ := strconv.Atoi(id)
	s, found := racing.SingleRacerStatsFallback(h.S.DB, racerID)
	if !found {
		s = models.RacerStats{RacerID: 0}
	}
	rInfo := racing.RacerInfo(h.S.DB, racerID)
	c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
}

// @Summary Update racer stats
// @Description Manually update a racer's statistics
// @Tags Stats
// @Accept json
// @Produce json
// @Param stats body models.RacerStats true "Racer stats"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/racer-stats [post]
func (h *Handler) UpdateRacerStats(c *gin.Context) {
	var stats models.RacerStats
	if err := c.ShouldBindJSON(&stats); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := racing.UpsertRacerStats(h.S.DB, stats); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// @Summary Get track stats
// @Description Get performance statistics grouped by track
// @Tags Stats
// @Produce json
// @Success 200 {array} models.TrackStats
// @Router /api/track-stats [get]
func (h *Handler) GetTrackStats(c *gin.Context) {
	stats, err := racing.TrackStats(h.S.DB)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	result := make([]models.TrackStats, len(stats))
	for i, s := range stats {
		result[i] = models.TrackStats{
			TrackID:    s.TrackID,
			TrackName:  s.TrackName,
			Country:    s.Country,
			RacesCount: s.RacesCount,
			Winner:     s.Winner,
			FastestLap: s.FastestLap,
		}
	}
	c.JSON(http.StatusOK, result)
}

// @Summary Head to head comparison
// @Description Compare two racers head to head
// @Tags Stats
// @Produce json
// @Param racer1 query int true "First racer ID"
// @Param racer2 query int true "Second racer ID"
// @Success 200 {object} models.HeadToHead
// @Failure 400 {object} map[string]string
// @Router /api/stats/head-to-head [get]
func (h *Handler) GetHeadToHead(c *gin.Context) {
	racer1Str := c.Query("racer1")
	racer2Str := c.Query("racer2")

	racer1, err1 := strconv.Atoi(racer1Str)
	racer2, err2 := strconv.Atoi(racer2Str)
	if err1 != nil || err2 != nil || racer1 <= 0 || racer2 <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "racer1 and racer2 query params required"})
		return
	}

	result, err := racing.HeadToHead(h.S.DB, racer1, racer2)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.HeadToHead{
		Racer1:     result.Racer1,
		Racer2:     result.Racer2,
		Races:      result.Races,
		Racer1Wins: result.Racer1Wins,
		Racer2Wins: result.Racer2Wins,
		Racer1Avg:  result.Racer1Avg,
		Racer2Avg:  result.Racer2Avg,
	})
}

// @Summary Points progression
// @Description Get cumulative points progression for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/points-progression [get]
func (h *Handler) GetPointsProgression(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, err := strconv.Atoi(racerIDStr)
	if err != nil || racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "racer_id required"})
		return
	}

	progression, err := racing.PointsProgression(h.S.DB, racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if progression == nil {
		progression = []racing.PointsProgressionData{}
	}
	c.JSON(http.StatusOK, progression)
}

// @Summary Get racer streaks
// @Description Get current streak information for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/stats/streaks [get]
func (h *Handler) GetStreaks(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	if racerIDStr != "" {
		racerID, err := strconv.Atoi(racerIDStr)
		if err != nil || racerID <= 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid racer_id"})
			return
		}
		result, err := racing.Streaks(h.S.DB, racerID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		streaks := []models.StreakInfo{}
		if result != nil {
			streaks = append(streaks, models.StreakInfo{
				RacerName:    result.RacerName,
				StreakType:   result.StreakType,
				CurrentValue: result.CurrentStreak,
				BestValue:    result.BestStreak,
			})
		}
		c.JSON(http.StatusOK, streaks)
		return
	}

	// All racers mode
	allData := racing.AllStreaks(h.S.DB)
	streaks := make([]models.StreakInfo, 0, len(allData))
	for _, s := range allData {
		streaks = append(streaks, models.StreakInfo{
			RacerName:    s.RacerName,
			StreakType:   s.StreakType,
			CurrentValue: s.CurrentStreak,
			BestValue:    s.BestStreak,
		})
	}
	c.JSON(http.StatusOK, streaks)
}

// @Summary Get ELO ratings
// @Description Get ELO-style ratings for all racers
// @Tags Stats
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/elo [get]
func (h *Handler) GetELORatings(c *gin.Context) {
	ratings, err := racing.ELORatings(h.S.DB)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ratings)
}

// @Summary Export stats CSV
// @Description Export racer statistics as a CSV file
// @Tags Stats
// @Produce text/csv
// @Success 200
// @Router /api/stats/export [get]
func (h *Handler) ExportStatsCSV(c *gin.Context) {
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=heat_racer_stats.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"ID", "Name", "Car", "Points", "Rank", "Races", "Wins", "Gold", "Silver", "Bronze", "Fastest Laps", "DNF", "DNS"})

	rows, err := h.S.DB.Query("SELECT r.id, r.name, r.car_name, r.points, r.rank, COALESCE(rs.races, 0), COALESCE(rs.wins, 0), COALESCE(rs.gold, 0), COALESCE(rs.silver, 0), COALESCE(rs.bronze, 0), COALESCE(rs.fastest_laps, 0), COALESCE(rs.dnf, 0), COALESCE(rs.dns, 0) FROM racers r LEFT JOIN racer_stats rs ON rs.racer_id = r.id ORDER BY r.rank")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, points, rank, races, wins, gold, silver, bronze, fl, dnf, dns int
		var name, carName string
		rows.Scan(&id, &name, &carName, &points, &rank, &races, &wins, &gold, &silver, &bronze, &fl, &dnf, &dns)
		writer.Write([]string{
			strconv.Itoa(id), name, carName,
			strconv.Itoa(points), strconv.Itoa(rank),
			strconv.Itoa(races), strconv.Itoa(wins),
			strconv.Itoa(gold), strconv.Itoa(silver), strconv.Itoa(bronze),
			strconv.Itoa(fl), strconv.Itoa(dnf), strconv.Itoa(dns),
		})
	}
	writer.Flush()
}

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

	var args []interface{}
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
