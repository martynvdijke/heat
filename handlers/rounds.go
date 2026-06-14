package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// @Summary Take round snapshot
// @Description Take a snapshot of current racer standings for a season round
// @Tags Seasons
// @Accept json
// @Produce json
// @Param snapshot body object true "Round snapshot data (race_name, season_id, round)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/rounds [post]
func (h *Handler) TakeRoundSnapshot(c *gin.Context) {
	var input struct {
		RaceName string `json:"race_name"`
		Round    int    `json:"round"`
		SeasonID int    `json:"season_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.RaceName == "" {
		input.RaceName = time.Now().Format("Round 2006-01-02")
	}
	if input.SeasonID == 0 {
		input.SeasonID = 1
	}

	raceDate := time.Now().Format("2006-01-02")

	var roundNum int
	h.S.DB.QueryRow("SELECT COALESCE(MAX(round), 0) + 1 FROM round_snapshots WHERE season_id = ?", input.SeasonID).Scan(&roundNum)
	if input.Round > 0 {
		roundNum = input.Round
	}

	rows, err := h.S.DB.Query("SELECT id, name, points FROM racers ORDER BY points DESC, name ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type racerScore struct {
		ID     int
		Name   string
		Points int
	}
	var scores []racerScore
	for rows.Next() {
		var s racerScore
		if err := rows.Scan(&s.ID, &s.Name, &s.Points); err != nil {
			continue
		}
		scores = append(scores, s)
	}
	rows.Close()

	tx, err := h.S.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO round_snapshots (season_id, race_name, race_date, round) VALUES (?, ?, ?, ?)",
		input.SeasonID, input.RaceName, raceDate, roundNum)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	snapshotID, _ := res.LastInsertId()

	for i, s := range scores {
		tx.Exec("INSERT INTO round_snapshot_scores (snapshot_id, racer_id, racer_name, points, position) VALUES (?, ?, ?, ?, ?)",
			snapshotID, s.ID, s.Name, s.Points, i+1)
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.S.BroadcastRacers()
	c.JSON(http.StatusOK, gin.H{"id": snapshotID, "round": roundNum})
}

// @Summary Get round snapshots
// @Description Get round snapshots, optionally filtered by ID or season_id
// @Tags Seasons
// @Produce json
// @Param id query int false "Snapshot ID"
// @Param season_id query int false "Season ID"
// @Success 200 {array} models.RoundSnapshot
// @Router /api/rounds [get]
func (h *Handler) GetRoundSnapshots(c *gin.Context) {
	idStr := c.Query("id")
	seasonIDStr := c.Query("season_id")

	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		var s models.RoundSnapshot
		err := h.S.DB.QueryRow("SELECT id, season_id, race_name, race_date, round, created_at FROM round_snapshots WHERE id = ?", id).
			Scan(&s.ID, &s.SeasonID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}

		scoreRows, err := h.S.DB.Query("SELECT id, snapshot_id, racer_id, racer_name, points, position FROM round_snapshot_scores WHERE snapshot_id = ? ORDER BY position", id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer scoreRows.Close()

		for scoreRows.Next() {
			var sc models.RoundSnapshotScore
			if err := scoreRows.Scan(&sc.ID, &sc.SnapshotID, &sc.RacerID, &sc.RacerName, &sc.Points, &sc.Position); err != nil {
				continue
			}
			s.Scores = append(s.Scores, sc)
		}

		c.JSON(http.StatusOK, s)
		return
	}

	query := "SELECT id, season_id, race_name, race_date, round, created_at FROM round_snapshots ORDER BY round ASC"
	args := []any{}
	if seasonIDStr != "" {
		query = "SELECT id, season_id, race_name, race_date, round, created_at FROM round_snapshots WHERE season_id = ? ORDER BY round ASC"
		args = append(args, seasonIDStr)
	}

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	snapshots := make([]models.RoundSnapshot, 0)
	for rows.Next() {
		var s models.RoundSnapshot
		if err := rows.Scan(&s.ID, &s.SeasonID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt); err != nil {
			continue
		}
		snapshots = append(snapshots, s)
	}
	c.JSON(http.StatusOK, snapshots)
}

// @Summary Delete round snapshot
// @Description Delete a round snapshot by ID
// @Tags Seasons
// @Produce json
// @Param id query string true "Snapshot ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /api/rounds [delete]
func (h *Handler) DeleteRoundSnapshot(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	h.S.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id = ?", id)
	h.S.DB.Exec("DELETE FROM round_snapshots WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// @Summary List seasons
// @Description Get the list of all seasons
// @Tags Seasons
// @Produce json
// @Success 200 {array} models.Season
// @Router /api/seasons [get]
func (h *Handler) GetSeasons(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT id, name, start_date, COALESCE(end_date, ''), status, created_at FROM seasons ORDER BY id DESC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	seasons := make([]models.Season, 0)
	for rows.Next() {
		var s models.Season
		if err := rows.Scan(&s.ID, &s.Name, &s.StartDate, &s.EndDate, &s.Status, &s.CreatedAt); err != nil {
			continue
		}
		seasons = append(seasons, s)
	}
	c.JSON(http.StatusOK, seasons)
}

// @Summary Create season
// @Description Create a new season
// @Tags Seasons
// @Accept json
// @Produce json
// @Param season body object true "Season data (name)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/seasons [post]
func (h *Handler) CreateSeason(c *gin.Context) {
	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "name required"})
		return
	}

	res, err := h.S.DB.Exec("INSERT INTO seasons (name, start_date, status) VALUES (?, date('now'), 'active')", input.Name)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	c.JSON(http.StatusOK, gin.H{"id": id})
}

// @Summary Delete season
// @Description Delete a season by ID
// @Tags Seasons
// @Produce json
// @Param id query string true "Season ID"
// @Success 200
// @Security cookieAuth
// @Router /api/seasons [delete]
func (h *Handler) DeleteSeason(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	h.S.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id IN (SELECT id FROM round_snapshots WHERE season_id = ?)", id)
	h.S.DB.Exec("DELETE FROM round_snapshots WHERE season_id = ?", id)
	h.S.DB.Exec("DELETE FROM seasons WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// @Summary Archive season
// @Description Archive a season by ID
// @Tags Seasons
// @Produce json
// @Param id query string true "Season ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/seasons/archive [post]
func (h *Handler) ArchiveSeason(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	_, err := h.S.DB.Exec("UPDATE seasons SET status = 'archived', end_date = date('now') WHERE id = ?", id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
