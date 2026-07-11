package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/ent"
	"heat/ent/racer"
	"heat/ent/team"
	"heat/middleware"
	"heat/models"
)

func (h *Handler) GetTeams(c *gin.Context) {
	teams, err := h.S.Ent.Team.Query().Order(team.ByName()).All(c.Request.Context())
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

func (h *Handler) SaveTeam(c *gin.Context) {
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
		_, err = h.S.Ent.Team.Create().SetName(team.Name).SetColor(team.Color).Save(c.Request.Context())
	} else {
		err = h.S.Ent.Team.UpdateOneID(team.ID).SetName(team.Name).SetColor(team.Color).Exec(c.Request.Context())
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func (h *Handler) DeleteTeam(c *gin.Context) {
	idStr := c.Query("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid team ID"})
		return
	}
	h.S.Ent.Racer.Update().Where(racer.TeamID(id)).SetTeamID(0).Exec(c.Request.Context())
	h.S.Ent.Team.DeleteOneID(id).Exec(c.Request.Context())
	c.Status(http.StatusOK)
}

func (h *Handler) GetConstructorStandings(c *gin.Context) {
	ctx := c.Request.Context()

	var teams []*ent.Team
	var racers []*ent.Racer
	var raceHistories []*ent.RaceHistory
	var raceResults []*ent.RaceResult
	var err error

	err = middleware.TraceDBQuery(ctx, "GetConstructorStandings/teams", func(ctx context.Context) error {
		teams, err = h.S.Ent.Team.Query().All(ctx)
		return err
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = middleware.TraceDBQuery(ctx, "GetConstructorStandings/racers", func(ctx context.Context) error {
		racers, err = h.S.Ent.Racer.Query().All(ctx)
		return err
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = middleware.TraceDBQuery(ctx, "GetConstructorStandings/raceHistories", func(ctx context.Context) error {
		raceHistories, err = h.S.Ent.RaceHistory.Query().All(ctx)
		return err
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = middleware.TraceDBQuery(ctx, "GetConstructorStandings/raceResults", func(ctx context.Context) error {
		raceResults, err = h.S.Ent.RaceResult.Query().All(ctx)
		return err
	})
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

func (h *Handler) AssignTeam(c *gin.Context) {
	var req struct {
		RacerID int `json:"racer_id"`
		TeamID  int `json:"team_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.S.Ent.Racer.UpdateOneID(req.RacerID).SetTeamID(req.TeamID).Exec(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	h.S.BroadcastRacers()
	c.Status(http.StatusOK)
}
