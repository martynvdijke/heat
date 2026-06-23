package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
	"heat/racing"
)

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
	cacheKey := "stats:streaks:all"
	if cached, ok := h.S.StatsCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}
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
	h.S.StatsCache.Set(cacheKey, streaks)
	c.JSON(http.StatusOK, streaks)
}

// @Summary Get ELO ratings
// @Description Get ELO-style ratings for all racers
// @Tags Stats
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Router /api/stats/elo [get]
func (h *Handler) GetELORatings(c *gin.Context) {
	if cached, ok := h.S.StatsCache.Get("stats:elo"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}
	ratings, err := racing.ELORatings(h.S.DB)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.StatsCache.Set("stats:elo", ratings)
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
