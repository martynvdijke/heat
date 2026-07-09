package handlers

import (
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

		if id == "" {
			cacheKey := "stats:racer-stats:season:" + seasonID
			if cached, ok := h.S.StatsCache.Get(cacheKey); ok {
				c.JSON(http.StatusOK, cached)
				return
			}
			stats := racing.RacerStatsBySeason(h.S.DB, sid)
			if len(stats) == 0 {
				stats = racing.AllRacerStats(h.S.DB)
			}
			h.S.StatsCache.Set(cacheKey, stats)
			c.JSON(http.StatusOK, stats)
			return
		}

		racerID, _ := strconv.Atoi(id)
		s, found := racing.SingleRacerStatsBySeason(h.S.DB, racerID, sid)
		if !found {
			s, _ = racing.SingleRacerStatsFallback(h.S.DB, racerID)
		}
		rInfo := racing.RacerInfo(h.S.DB, racerID)
		c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
		return
	}

	if id == "" {
		if cached, ok := h.S.StatsCache.Get("stats:racer-stats:all"); ok {
			c.JSON(http.StatusOK, cached)
			return
		}
		stats := racing.AllRacerStats(h.S.DB)
		h.S.StatsCache.Set("stats:racer-stats:all", stats)
		c.JSON(http.StatusOK, stats)
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
	h.S.StatsCache.InvalidatePrefix("stats:")
	c.Status(http.StatusOK)
}

// @Summary Get track stats
// @Description Get performance statistics grouped by track
// @Tags Stats
// @Produce json
// @Success 200 {array} models.TrackStats
// @Router /api/track-stats [get]
func (h *Handler) GetTrackStats(c *gin.Context) {
	if cached, ok := h.S.StatsCache.Get("stats:track-stats"); ok {
		c.JSON(http.StatusOK, cached)
		return
	}
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
	h.S.StatsCache.Set("stats:track-stats", result)
	c.JSON(http.StatusOK, result)
}
