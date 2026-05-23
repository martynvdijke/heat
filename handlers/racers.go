package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/ent/racer"
	"heat/models"
)

// @Summary Get all racers
// @Description Get the list of all racers sorted by rank
// @Tags Racers
// @Produce json
// @Success 200 {array} models.Racer
// @Router /api/racers [get]
func (h *Handler) GetRacers(c *gin.Context) {
	entRacers, err := h.S.Ent.Racer.Query().Order(racer.ByRank()).All(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	racers := make([]models.Racer, 0, len(entRacers))
	for _, r := range entRacers {
		racers = append(racers, models.Racer{
			ID:             r.ID,
			Name:           r.Name,
			ProfilePicture: r.ProfilePicture,
			CarColor:       r.CarColor,
			CarName:        r.CarName,
			Points:         r.Points,
			Rank:           r.Rank,
			Position:       r.Position,
			TeamID:         r.TeamID,
		})
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
