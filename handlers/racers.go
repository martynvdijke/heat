package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// @Summary Get all racers
// @Description Get the list of all racers sorted by rank
// @Tags Racers
// @Produce json
// @Success 200 {array} models.Racer
// @Router /api/racers [get]
func (h *Handler) GetRacers(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT r.id, r.name, r.profile_picture, r.car_color, r.car_name, r.points, r.rank, r.position, COALESCE(r.team_id, 0), COALESCE(t.name, '') FROM racers r LEFT JOIN teams t ON r.team_id = t.id ORDER BY r.rank ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	racers := make([]models.Racer, 0)
	for rows.Next() {
		var r models.Racer
		if err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position, &r.TeamID, &r.TeamName); err != nil {
			continue
		}
		racers = append(racers, r)
	}

	c.JSON(http.StatusOK, racers)
}

// @Summary Create or update a racer
// @Description Creates a new racer if ID is 0, otherwise updates the existing racer
// @Tags Racers
// @Accept json
// @Produce json
// @Param racer body models.Racer true "Racer data"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/racers [post]
func (h *Handler) UpdateRacer(c *gin.Context) {
	var r models.Racer
	if err := c.ShouldBindJSON(&r); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if r.ID == 0 {
		err := h.S.Ent.Racer.Create().
			SetName(r.Name).
			SetProfilePicture(r.ProfilePicture).
			SetCarColor(r.CarColor).
			SetCarName(r.CarName).
			SetPoints(r.Points).
			SetRank(r.Rank).
			SetPosition(r.Position).
			SetTeamID(r.TeamID).
			Exec(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		err := h.S.Ent.Racer.UpdateOneID(r.ID).
			SetName(r.Name).
			SetProfilePicture(r.ProfilePicture).
			SetCarColor(r.CarColor).
			SetCarName(r.CarName).
			SetPoints(r.Points).
			SetRank(r.Rank).
			SetPosition(r.Position).
			SetTeamID(r.TeamID).
			Exec(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}

// @Summary Batch update racer ranks
// @Description Update rank positions for multiple racers in one call
// @Tags Racers
// @Accept json
// @Produce json
// @Param ranks body []models.RacerRankUpdate true "Array of {id, rank} pairs"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/racers/ranks [put]
func (h *Handler) UpdateRacerRanks(c *gin.Context) {
	var updates []struct {
		ID   int `json:"id"`
		Rank int `json:"rank"`
	}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := h.S.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()

	for _, u := range updates {
		tx.Exec("UPDATE racers SET rank = ? WHERE id = ?", u.Rank, u.ID)
	}

	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}

// @Summary Delete a racer
// @Description Delete a racer by ID
// @Tags Racers
// @Produce json
// @Param id query int true "Racer ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/racers [delete]
func (h *Handler) DeleteRacer(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid racer ID"})
		return
	}

	err = h.S.Ent.Racer.DeleteOneID(id).Exec(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}
