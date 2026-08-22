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
// @Param source query string false "legacy = raw racer_stats rows (admin view)"
// @Param season_id query int false "Season ID (alias for season_ids)"
// @Param season_ids query string false "Comma-separated season IDs; absent = all seasons"
// @Success 200 {object} map[string]interface{}
// @Router /api/racer-stats [get]
func (h *Handler) GetRacerStats(c *gin.Context) {
	id := c.Query("id")

	// Admin management view: return the raw legacy racer_stats rows (the
	// table the admin UI edits), independent of snapshot-derived scopes.
	if c.Query("source") == "legacy" && id == "" {
		c.JSON(http.StatusOK, racing.AllRacerStats(h.S.DB))
		return
	}

	seasonIDs, err := parseSeasonScope(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if id == "" {
		cacheKey := seasonScopeCacheKey("stats:racer-stats", seasonIDs)
		if cached, ok := h.S.StatsCache.Get(cacheKey); ok {
			c.JSON(http.StatusOK, cached)
			return
		}
		// Snapshot-derived stats only: an explicit scope with no data returns
		// an empty list (no silent fallback to legacy racer_stats).
		stats := racing.RacerStatsBySeasons(h.S.DB, seasonIDs)
		h.S.StatsCache.Set(cacheKey, stats)
		c.JSON(http.StatusOK, stats)
		return
	}

	racerID, _ := strconv.Atoi(id)
	s, found := racing.SingleRacerStatsBySeasons(h.S.DB, racerID, seasonIDs)
	if !found {
		s, _ = racing.SingleRacerStatsFallback(h.S.DB, racerID)
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
// @Param season_ids query string false "Comma-separated season IDs; absent = all seasons"
// @Success 200 {array} models.TrackStats
// @Router /api/track-stats [get]
func (h *Handler) GetTrackStats(c *gin.Context) {
	seasonIDs, err := parseSeasonScope(c)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cacheKey := seasonScopeCacheKey("stats:track-stats", seasonIDs)
	if cached, ok := h.S.StatsCache.Get(cacheKey); ok {
		c.JSON(http.StatusOK, cached)
		return
	}
	stats, err := racing.TrackStats(h.S.DB, seasonIDs)
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
	h.S.StatsCache.Set(cacheKey, result)
	c.JSON(http.StatusOK, result)
}
