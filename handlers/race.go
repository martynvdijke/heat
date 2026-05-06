package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// @Summary Get current race info
// @Description Get the current race information (country, track, laps)
// @Tags Race
// @Produce json
// @Success 200 {object} models.RaceInfo
// @Router /api/race-info [get]
func GetRaceInfo(c *gin.Context) {
	var ri models.RaceInfo
	err := app.DB.QueryRow("SELECT country, track, COALESCE(track_id, 'monza'), laps FROM race_info ORDER BY id DESC LIMIT 1").
		Scan(&ri.Country, &ri.Track, &ri.TrackID, &ri.Laps)
	if err != nil {
		ri = models.RaceInfo{Country: "Italy", Track: "Monza", TrackID: "monza", Laps: 53}
	}
	c.JSON(http.StatusOK, ri)
}

// @Summary Update race info
// @Description Update the current race information
// @Tags Race
// @Accept json
// @Produce json
// @Param raceInfo body models.RaceInfo true "Race info"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/race-info [post]
func UpdateRaceInfo(c *gin.Context) {
	var ri models.RaceInfo
	if err := c.ShouldBindJSON(&ri); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ri.TrackID == "" {
		ri.TrackID = "monza"
	}

	_, err := app.DB.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		ri.Country, ri.Track, ri.TrackID, ri.Laps)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// @Summary Save race to history
// @Description Archive a completed race to history with results
// @Tags Race
// @Accept json
// @Produce json
// @Param race body object true "Race history data with results"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/race-history [post]
func SaveRaceToHistory(c *gin.Context) {
	var input struct {
		Name      string `json:"name"`
		RaceDate  string `json:"race_date"`
		Country   string `json:"country"`
		Track     string `json:"track"`
		TrackID   string `json:"track_id"`
		TotalLaps int    `json:"total_laps"`
		RaceType  string `json:"race_type"`
		Results   []struct {
			RacerID     int    `json:"racer_id"`
			RacerName   string `json:"racer_name"`
			Position    int    `json:"position"`
			Points      int    `json:"points"`
			FastestLap  bool   `json:"fastest_lap"`
			Finished    bool   `json:"finished"`
			DidNotStart bool   `json:"did_not_start"`
		} `json:"results"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.RaceDate == "" {
		input.RaceDate = time.Now().Format("2006-01-02")
	}
	if input.Name == "" {
		input.Name = input.RaceDate
	}
	if input.RaceType == "" {
		input.RaceType = "season"
	}

	isOneOff := input.RaceType == "oneoff"

	result, err := app.DB.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES (?, ?, ?, ?, ?, ?, ?)",
		input.Name, input.RaceDate, input.Country, input.Track, input.TrackID, input.TotalLaps, input.RaceType)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	raceID, _ := result.LastInsertId()

	for _, res := range input.Results {
		app.DB.Exec("INSERT INTO race_results (race_id, racer_id, racer_name, position, points, fastest_lap) VALUES (?, ?, ?, ?, ?, ?)",
			raceID, res.RacerID, res.RacerName, res.Position, res.Points, boolToInt(res.FastestLap))

		if !isOneOff {
			app.DB.Exec(`INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (?, 1, ?, ?, ?, ?, ?, ?, ?)
					 ON CONFLICT(racer_id) DO UPDATE SET
					 races = races + 1,
					 wins = wins + excluded.wins,
					 gold = gold + excluded.gold,
					 silver = silver + excluded.silver,
					 bronze = bronze + excluded.bronze,
					 fastest_laps = fastest_laps + excluded.fastest_laps,
					 dnf = dnf + excluded.dnf,
					 dns = dns + excluded.dns`,
				res.RacerID,
				boolToInt(res.Position == 1),
				boolToInt(res.Position == 1),
				boolToInt(res.Position == 2),
				boolToInt(res.Position == 3),
				boolToInt(res.FastestLap),
				boolToInt(!res.Finished && !res.DidNotStart),
				boolToInt(res.DidNotStart))
		}
	}

	if !isOneOff && len(input.Results) > 0 {
		winner := ""
		second := ""
		third := ""
		for _, res := range input.Results {
			if res.Position == 1 {
				winner = res.RacerName
			} else if res.Position == 2 {
				second = res.RacerName
			} else if res.Position == 3 {
				third = res.RacerName
			}
		}
		NotifyRaceWinner(winner, input.Track)
		if second != "" && third != "" {
			NotifyRacePodium(winner, second, third, input.Track)
		}
	}

	results := make([]models.RaceResult, len(input.Results))
	for i, res := range input.Results {
		results[i] = models.RaceResult{
			RacerID:    res.RacerID,
			RacerName:  res.RacerName,
			Position:   res.Position,
			Points:     res.Points,
			FastestLap: res.FastestLap,
		}
	}
	go SendRaceEmail(input.Name, input.Country, input.Track, input.TotalLaps, results)

	c.JSON(http.StatusOK, gin.H{"id": raceID})
}

// @Summary Get race history
// @Description Get race history entries, optionally filtered by ID or type
// @Tags Race
// @Produce json
// @Param id query int false "Race ID"
// @Param type query string false "Race type (season, oneoff)"
// @Success 200 {array} models.RaceHistory
// @Router /api/race-history [get]
func GetRaceHistory(c *gin.Context) {
	raceID := c.Query("id")
	raceType := c.Query("type")

	var query string
	var args []interface{}

	if raceID != "" {
		query = `SELECT rh.id, COALESCE(rh.name, ''), rh.race_date, rh.country, rh.track, rh.track_id, rh.total_laps, COALESCE(rh.race_type, 'season'),
				 COALESCE(GROUP_CONCAT(rr.racer_id || ':' || rr.racer_name || ':' || rr.position || ':' || rr.points || ':' || rr.fastest_lap, '|'), '') as results
				 FROM race_history rh
				 LEFT JOIN race_results rr ON rh.id = rr.race_id
				 WHERE rh.id = ?
				 GROUP BY rh.id`
		args = []interface{}{raceID}
	} else {
		if raceType != "" {
			query = `SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'season')
					 FROM race_history WHERE race_type = ? ORDER BY race_date DESC LIMIT 20`
			args = []interface{}{raceType}
		} else {
			query = `SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'season') FROM race_history ORDER BY race_date DESC LIMIT 20`
		}
	}

	rows, err := app.DB.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	history := make([]models.RaceHistory, 0)
	for rows.Next() {
		var h models.RaceHistory
		var resultsStr string
		var raceType string
		if raceID != "" {
			rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &raceType, &resultsStr)
			h.RaceType = raceType
			if resultsStr != "" {
				for _, r := range strings.Split(resultsStr, "|") {
					parts := strings.Split(r, ":")
					if len(parts) >= 5 {
						rid, _ := strconv.Atoi(parts[0])
						pos, _ := strconv.Atoi(parts[2])
						pts, _ := strconv.Atoi(parts[3])
						fl, _ := strconv.Atoi(parts[4])
						h.Results = append(h.Results, models.RaceResult{
							RacerID:    rid,
							RacerName:  parts[1],
							Position:   pos,
							Points:     pts,
							FastestLap: fl == 1,
						})
					}
				}
			}
		} else {
			rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &h.RaceType)
		}
		history = append(history, h)
	}
	c.JSON(http.StatusOK, history)
}

// @Summary Delete race from history
// @Description Delete a race history entry by ID
// @Tags Race
// @Produce json
// @Param id query string true "Race ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/race-history [delete]
func DeleteRaceHistory(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}
	app.DB.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	app.DB.Exec("DELETE FROM race_history WHERE id = ?", id)
	c.Status(http.StatusOK)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
