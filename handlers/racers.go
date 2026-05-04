package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
	"heat/ws"
)

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
