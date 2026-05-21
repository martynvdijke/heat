package handlers

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/ent/racer"
	"heat/ent/team"
	"heat/models"
)

func GetTeams(c *gin.Context) {
	teams, err := app.Ent.Team.Query().Order(team.ByName()).All(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]models.Team, 0, len(teams))
	for _, t := range teams {
		result = append(result, models.Team{
			ID:        t.ID,
			Name:      t.Name,
			Color:     t.Color,
			CreatedAt: t.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, result)
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

	var err error
	if team.ID == 0 {
		_, err = app.Ent.Team.Create().SetName(team.Name).SetColor(team.Color).Save(c.Request.Context())
	} else {
		err = app.Ent.Team.UpdateOneID(team.ID).SetName(team.Name).SetColor(team.Color).Exec(c.Request.Context())
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
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
	app.Ent.Racer.Update().Where(racer.TeamID(id)).SetTeamID(0).Exec(c.Request.Context())
	app.Ent.Team.DeleteOneID(id).Exec(c.Request.Context())
	c.Status(http.StatusOK)
}

func GetConstructorStandings(c *gin.Context) {
	ctx := c.Request.Context()

	teams, err := app.Ent.Team.Query().All(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	racers, err := app.Ent.Racer.Query().All(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	raceHistories, err := app.Ent.RaceHistory.Query().All(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	raceResults, err := app.Ent.RaceResult.Query().All(ctx)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	raceTypeMap := make(map[int]string, len(raceHistories))
	for _, rh := range raceHistories {
		raceTypeMap[rh.ID] = rh.RaceType
	}

	racerTeamMap := make(map[int]int, len(racers))
	for _, r := range racers {
		racerTeamMap[r.ID] = r.TeamID
	}

	type ConstructorStanding struct {
		models.Team
		TotalPoints int `json:"total_points"`
		Races       int `json:"races"`
		Drivers     int `json:"drivers"`
	}

	results := make(map[int]*ConstructorStanding, len(teams))
	for _, t := range teams {
		results[t.ID] = &ConstructorStanding{
			Team: models.Team{
				ID:    t.ID,
				Name:  t.Name,
				Color: t.Color,
			},
		}
	}

	teamRaces := make(map[int]map[int]struct{})
	teamDrivers := make(map[int]map[int]struct{})

	for _, rr := range raceResults {
		if raceTypeMap[rr.RaceID] != "season" {
			continue
		}
		tid := racerTeamMap[rr.RacerID]
		if tid == 0 {
			continue
		}
		cs, ok := results[tid]
		if !ok {
			continue
		}
		cs.TotalPoints += rr.Points

		if teamRaces[tid] == nil {
			teamRaces[tid] = make(map[int]struct{})
			teamDrivers[tid] = make(map[int]struct{})
		}
		teamRaces[tid][rr.RaceID] = struct{}{}
		teamDrivers[tid][rr.RacerID] = struct{}{}
	}

	standings := make([]ConstructorStanding, 0, len(results))
	for _, cs := range results {
		cs.Races = len(teamRaces[cs.ID])
		cs.Drivers = len(teamDrivers[cs.ID])
		standings = append(standings, *cs)
	}
	sort.Slice(standings, func(i, j int) bool {
		return standings[i].TotalPoints > standings[j].TotalPoints
	})

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
	err := app.Ent.Racer.UpdateOneID(req.RacerID).SetTeamID(req.TeamID).Exec(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
