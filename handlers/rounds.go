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

	// Reject if season is archived
	var seasonStatus string
	h.S.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", input.SeasonID).Scan(&seasonStatus)
	if seasonStatus == "archived" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot create rounds in an archived season"})
		return
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

	res, err := tx.Exec("INSERT INTO round_snapshots (season_id, race_name, race_date, round, status) VALUES (?, ?, ?, ?, 'draft')",
		input.SeasonID, input.RaceName, raceDate, roundNum)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	snapshotID, _ := res.LastInsertId()

	if len(scores) > 0 {
		query := "INSERT INTO round_snapshot_scores (snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES "
		args := make([]interface{}, 0, len(scores)*9)
		for i, s := range scores {
			if i > 0 {
				query += ", "
			}
			query += "(?, ?, ?, ?, ?, 0, 0, 0, 0)"
			args = append(args, snapshotID, s.ID, s.Name, s.Points, i+1)
		}
		if _, err := tx.Exec(query, args...); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
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
		err := h.S.DB.QueryRow("SELECT id, season_id, race_name, race_date, round, created_at, status FROM round_snapshots WHERE id = ?", id).
			Scan(&s.ID, &s.SeasonID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt, &s.Status)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}

		scoreRows, err := h.S.DB.Query("SELECT id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated FROM round_snapshot_scores WHERE snapshot_id = ? ORDER BY position", id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer scoreRows.Close()

		for scoreRows.Next() {
			var sc models.RoundSnapshotScore
			if err := scoreRows.Scan(&sc.ID, &sc.SnapshotID, &sc.RacerID, &sc.RacerName, &sc.Points, &sc.Position, &sc.DNF, &sc.DNS, &sc.Spins, &sc.Overheated); err != nil {
				continue
			}
			s.Scores = append(s.Scores, sc)
		}

		c.JSON(http.StatusOK, s)
		return
	}

	query := "SELECT id, season_id, race_name, race_date, round, created_at, status FROM round_snapshots ORDER BY round ASC"
	args := []any{}
	if seasonIDStr != "" {
		query = "SELECT id, season_id, race_name, race_date, round, created_at, status FROM round_snapshots WHERE season_id = ? ORDER BY round ASC"
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
		if err := rows.Scan(&s.ID, &s.SeasonID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt, &s.Status); err != nil {
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

	// Check if parent season is archived
	var seasonID int
	h.S.DB.QueryRow("SELECT season_id FROM round_snapshots WHERE id = ?", id).Scan(&seasonID)
	if seasonID > 0 {
		var seasonStatus string
		h.S.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", seasonID).Scan(&seasonStatus)
		if seasonStatus == "archived" {
			c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot delete rounds in an archived season"})
			return
		}
	}

	h.S.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id = ?", id)
	h.S.DB.Exec("DELETE FROM round_snapshots WHERE id = ?", id)
	c.Status(http.StatusOK)
}

// @Summary Update round scores (draft only)
// @Description Update the scores for a draft round. Only positions < 900 are counted as finished.
// @Tags Seasons
// @Accept json
// @Produce json
// @Param id query string true "Round snapshot ID"
// @Param scores body []RoundSnapshotScore true "Array of updated scores"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security cookieAuth
// @Router /api/rounds [patch]
func (h *Handler) UpdateRoundScores(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	var status string
	var seasonID int
	err := h.S.DB.QueryRow("SELECT status, season_id FROM round_snapshots WHERE id = ?", idStr).Scan(&status, &seasonID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "round not found"})
		return
	}
	if status != "draft" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot edit a finalized round"})
		return
	}

	// Check if parent season is archived
	var seasonStatus string
	h.S.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", seasonID).Scan(&seasonStatus)
	if seasonStatus == "archived" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot edit rounds in an archived season"})
		return
	}

	var scores []models.RoundSnapshotScore
	if err := c.ShouldBindJSON(&scores); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.S.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	for _, sc := range scores {
		_, err := tx.Exec("UPDATE round_snapshot_scores SET points = ?, position = ?, dnf = ?, dns = ?, spins = ?, overheated = ? WHERE id = ? AND snapshot_id = ?",
			sc.Points, sc.Position, sc.DNF, sc.DNS, sc.Spins, sc.Overheated, sc.ID, idStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// @Summary Finalize a round
// @Description Lock a draft round and update driver totals (points, stats, spins).
// @Tags Seasons
// @Accept json
// @Produce json
// @Param id query string true "Round snapshot ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Security cookieAuth
// @Router /api/rounds/finalize [patch]
func (h *Handler) FinalizeRound(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	var status string
	var seasonID int
	err := h.S.DB.QueryRow("SELECT status, season_id FROM round_snapshots WHERE id = ?", idStr).Scan(&status, &seasonID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "round not found"})
		return
	}
	if status != "draft" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "round is already finalized"})
		return
	}
	// Check if parent season is archived
	var seasonStatus string
	h.S.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", seasonID).Scan(&seasonStatus)
	if seasonStatus == "archived" {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "cannot finalize rounds in an archived season"})
		return
	}

	rows, err := h.S.DB.Query("SELECT racer_id, points, dnf, dns, spins, overheated FROM round_snapshot_scores WHERE snapshot_id = ?", idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type score struct {
		RacerID    int
		Points     int
		DNF        bool
		DNS        bool
		Spins      int
		Overheated int
	}
	var scores []score
	for rows.Next() {
		var s score
		if err := rows.Scan(&s.RacerID, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated); err != nil {
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

	for _, s := range scores {
		// Add points to racer total
		tx.Exec("UPDATE racers SET points = points + ? WHERE id = ?", s.Points, s.RacerID)

		// Update racer_stats — increment races, add DNF/DNS/spins
		var existingID int
		err := tx.QueryRow("SELECT id FROM racer_stats WHERE racer_id = ?", s.RacerID).Scan(&existingID)
		if err != nil {
			// Create new stats row
			var name string
			tx.QueryRow("SELECT name FROM racers WHERE id = ?", s.RacerID).Scan(&name)
			dnfVal := 0
			if s.DNF {
				dnfVal = 1
			}
			dnsVal := 0
			if s.DNS {
				dnsVal = 1
			}
			tx.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns, spins, overheated) VALUES (?, 1, 0, 0, 0, 0, ?, ?, ?, ?)",
				s.RacerID, dnfVal, dnsVal, s.Spins, s.Overheated)
		} else {
			dnfAdd := 0
			if s.DNF {
				dnfAdd = 1
			}
			dnsAdd := 0
			if s.DNS {
				dnsAdd = 1
			}
			tx.Exec("UPDATE racer_stats SET races = races + 1, dnf = dnf + ?, dns = dns + ?, spins = spins + ?, overheated = overheated + ? WHERE id = ?",
				dnfAdd, dnsAdd, s.Spins, s.Overheated, existingID)
		}
	}

	// Mark round as final
	_, err = tx.Exec("UPDATE round_snapshots SET status = 'final' WHERE id = ?", idStr)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.S.BroadcastRacers()
	c.JSON(http.StatusOK, gin.H{"status": "finalized"})
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
