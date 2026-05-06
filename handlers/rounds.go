package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
	"heat/ws"
)

func TakeRoundSnapshot(c *gin.Context) {
	var input struct {
		RaceName string `json:"race_name"`
		Round    int    `json:"round"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if input.RaceName == "" {
		input.RaceName = time.Now().Format("Round 2006-01-02")
	}

	raceDate := time.Now().Format("2006-01-02")

	tx, err := app.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO round_snapshots (race_name, race_date, round) VALUES (?, ?, ?)",
		input.RaceName, raceDate, input.Round)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	snapshotID, _ := res.LastInsertId()

	rows, err := app.DB.Query("SELECT id, name, points FROM racers ORDER BY points DESC, name ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	position := 1
	for rows.Next() {
		var id, points int
		var name string
		if err := rows.Scan(&id, &name, &points); err != nil {
			continue
		}
		tx.Exec("INSERT INTO round_snapshot_scores (snapshot_id, racer_id, racer_name, points, position) VALUES (?, ?, ?, ?, ?)",
			snapshotID, id, name, points, position)
		position++
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ws.BroadcastRacers()

	c.JSON(http.StatusOK, gin.H{"id": snapshotID})
}

func GetRoundSnapshots(c *gin.Context) {
	idStr := c.Query("id")

	if idStr != "" {
		id, _ := strconv.Atoi(idStr)
		var s models.RoundSnapshot
		err := app.DB.QueryRow("SELECT id, race_name, race_date, round, created_at FROM round_snapshots WHERE id = ?", id).
			Scan(&s.ID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "snapshot not found"})
			return
		}

		scoreRows, err := app.DB.Query("SELECT id, snapshot_id, racer_id, racer_name, points, position FROM round_snapshot_scores WHERE snapshot_id = ? ORDER BY position", id)
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

	rows, err := app.DB.Query("SELECT id, race_name, race_date, round, created_at FROM round_snapshots ORDER BY round ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	snapshots := make([]models.RoundSnapshot, 0)
	for rows.Next() {
		var s models.RoundSnapshot
		if err := rows.Scan(&s.ID, &s.RaceName, &s.RaceDate, &s.Round, &s.CreatedAt); err != nil {
			continue
		}
		snapshots = append(snapshots, s)
	}
	c.JSON(http.StatusOK, snapshots)
}

func DeleteRoundSnapshot(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}
	app.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id = ?", id)
	app.DB.Exec("DELETE FROM round_snapshots WHERE id = ?", id)
	c.Status(http.StatusOK)
}
