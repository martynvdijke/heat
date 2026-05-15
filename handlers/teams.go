package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

func GetTeams(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, color, created_at FROM teams ORDER BY name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	teams := make([]models.Team, 0)
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			continue
		}
		teams = append(teams, t)
	}
	c.JSON(http.StatusOK, teams)
}

func SaveTeam(c *gin.Context) {
	var team models.Team
	if err := c.ShouldBindJSON(&team); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if team.Name == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Team name is required"})
		return
	}
	if team.Color == "" {
		team.Color = "#d40000"
	}

	if team.ID == 0 {
		_, err := app.DB.Exec("INSERT INTO teams (name, color) VALUES (?, ?)", team.Name, team.Color)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE teams SET name=?, color=? WHERE id=?", team.Name, team.Color, team.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)
}

func DeleteTeam(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}
	app.DB.Exec("UPDATE racers SET team_id = 0 WHERE team_id = ?", id)
	app.DB.Exec("DELETE FROM teams WHERE id = ?", id)
	c.Status(http.StatusOK)
}

func GetConstructorStandings(c *gin.Context) {
	rows, err := app.DB.Query(`
		SELECT t.id, t.name, t.color,
			COALESCE(SUM(CASE WHEN rh.race_type = 'season' THEN rr.points ELSE 0 END), 0) as total_points,
			COUNT(DISTINCT rr.race_id) as races,
			COUNT(DISTINCT r.id) as drivers
		FROM teams t
		LEFT JOIN racers r ON r.team_id = t.id
		LEFT JOIN race_results rr ON rr.racer_id = r.id
		LEFT JOIN race_history rh ON rh.id = rr.race_id
		GROUP BY t.id
		ORDER BY total_points DESC
	`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ConstructorStanding struct {
		models.Team
		TotalPoints int `json:"total_points"`
		Races       int `json:"races"`
		Drivers     int `json:"drivers"`
	}

	standings := make([]ConstructorStanding, 0)
	for rows.Next() {
		var cs ConstructorStanding
		if err := rows.Scan(&cs.ID, &cs.Name, &cs.Color, &cs.TotalPoints, &cs.Races, &cs.Drivers); err != nil {
			continue
		}
		standings = append(standings, cs)
	}
	c.JSON(http.StatusOK, standings)
}

func AssignTeam(c *gin.Context) {
	var req struct {
		RacerID int `json:"racer_id"`
		TeamID  int `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := app.DB.Exec("UPDATE racers SET team_id = ? WHERE id = ?", req.TeamID, req.RacerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
