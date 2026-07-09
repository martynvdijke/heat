package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func (h *Handler) generateShareToken() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (h *Handler) GenerateDriverShareToken(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, err := strconv.Atoi(racerIDStr)
	if err != nil || racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid racer_id"})
		return
	}

	var exists int
	h.S.DB.QueryRow("SELECT COUNT(*) FROM racers WHERE id = ?", racerID).Scan(&exists)
	if exists == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Racer not found"})
		return
	}

	token := h.generateShareToken()
	_, err = h.S.DB.Exec("INSERT OR REPLACE INTO driver_shares (id, racer_id, token, created_at) VALUES ((SELECT id FROM driver_shares WHERE racer_id = ?), ?, ?, datetime('now'))",
		racerID, racerID, token)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.DriverShare{RacerID: racerID, Token: token})
}

func (h *Handler) GetDriverShareTokens(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT ds.id, ds.racer_id, ds.token, ds.created_at, r.name FROM driver_shares ds JOIN racers r ON r.id = ds.racer_id ORDER BY r.name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type DriverShareWithName struct {
		models.DriverShare
		RacerName string `json:"racer_name"`
	}

	shares := make([]DriverShareWithName, 0)
	for rows.Next() {
		var s DriverShareWithName
		if err := rows.Scan(&s.ID, &s.RacerID, &s.Token, &s.CreatedAt, &s.RacerName); err != nil {
			continue
		}
		shares = append(shares, s)
	}
	c.JSON(http.StatusOK, shares)
}

func (h *Handler) GetDriverStatsByToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "token required"})
		return
	}

	var racerID int
	err := h.S.DB.QueryRow("SELECT racer_id FROM driver_shares WHERE token = ?", token).Scan(&racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Invalid or expired share link"})
		return
	}

	var r models.Racer
	err = h.S.DB.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", racerID).
		Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Racer not found"})
		return
	}

	var s models.RacerStats
	err = h.S.DB.QueryRow("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf, dns, spins, overheated FROM racer_stats WHERE racer_id = ?", racerID).
		Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
	if err != nil {
		s = models.RacerStats{RacerID: racerID}
	}

	c.JSON(http.StatusOK, gin.H{"racer": r, "stats": s})
}

func (h *Handler) DeleteDriverShareToken(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, err := strconv.Atoi(racerIDStr)
	if err != nil || racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid racer_id"})
		return
	}

	_, err = h.S.DB.Exec("DELETE FROM driver_shares WHERE racer_id = ?", racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
