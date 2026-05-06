package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
	"heat/ws"
)

// @Summary Get all racers
// @Description Get the list of all racers sorted by rank
// @Tags Racers
// @Produce json
// @Success 200 {array} models.Racer
// @Router /api/racers [get]
func GetRacers(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers ORDER BY rank ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	racers := make([]models.Racer, 0)
	for rows.Next() {
		var r models.Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
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
func UpdateRacer(c *gin.Context) {
	var racer models.Racer
	if err := c.ShouldBindJSON(&racer); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if racer.ID == 0 {
		_, err := app.DB.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank, position) VALUES (?, ?, ?, ?, ?, ?, ?)",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, rank=?, position=? WHERE id=?",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position, racer.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	ws.BroadcastRacers()
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
func DeleteRacer(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid racer ID"})
		return
	}
	_, err = app.DB.Exec("DELETE FROM racers WHERE id=?", id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	ws.BroadcastRacers()
	c.Status(http.StatusOK)
}
