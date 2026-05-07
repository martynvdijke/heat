package handlers

import (
	"encoding/csv"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// @Summary Get racer stats
// @Description Get statistics for all racers or a specific racer by ID
// @Tags Stats
// @Produce json
// @Param id query int false "Racer ID"
// @Param season_id query int false "Season ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/racer-stats [get]
func GetRacerStats(c *gin.Context) {
	id := c.Query("id")
	seasonID := c.Query("season_id")

	if seasonID != "" {
		var startDate, endDate string
		err := app.DB.QueryRow("SELECT start_date, COALESCE(end_date, '9999-12-31') FROM seasons WHERE id = ?", seasonID).Scan(&startDate, &endDate)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "season not found"})
			return
		}

		if id == "" {
			rows, err := app.DB.Query(`
				SELECT rr.racer_id,
					COUNT(*) as races,
					SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
					SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as gold,
					SUM(CASE WHEN rr.position = 2 THEN 1 ELSE 0 END) as silver,
					SUM(CASE WHEN rr.position = 3 THEN 1 ELSE 0 END) as bronze,
					SUM(CASE WHEN rr.fastest_lap = 1 THEN 1 ELSE 0 END) as fastest_laps,
					SUM(rr.points) as points,
					SUM(CASE WHEN rr.position >= 900 THEN 1 ELSE 0 END) as dnf,
					0 as dns
				FROM race_results rr
				JOIN race_history rh ON rh.id = rr.race_id
				WHERE rh.race_date >= ? AND rh.race_date <= ? AND rh.race_type = 'season'
				GROUP BY rr.racer_id
				ORDER BY points DESC
			`, startDate, endDate)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			defer rows.Close()

			stats := make([]models.RacerStats, 0)
			for rows.Next() {
				var s models.RacerStats
				rows.Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
				stats = append(stats, s)
			}
			c.JSON(http.StatusOK, stats)
			return
		}

		var s models.RacerStats
		err = app.DB.QueryRow(`
			SELECT rr.racer_id,
				COUNT(*) as races,
				SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
				SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as gold,
				SUM(CASE WHEN rr.position = 2 THEN 1 ELSE 0 END) as silver,
				SUM(CASE WHEN rr.position = 3 THEN 1 ELSE 0 END) as bronze,
				SUM(CASE WHEN rr.fastest_lap = 1 THEN 1 ELSE 0 END) as fastest_laps,
				SUM(rr.points) as points,
				SUM(CASE WHEN rr.position >= 900 THEN 1 ELSE 0 END) as dnf,
				0 as dns
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rr.racer_id = ? AND rh.race_date >= ? AND rh.race_date <= ? AND rh.race_type = 'season'
			GROUP BY rr.racer_id
		`, id, startDate, endDate).Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		if err != nil {
			s = models.RacerStats{RacerID: 0}
		}

		var rInfo models.Racer
		app.DB.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", id).Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)

		c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
		return
	}

	if id == "" {
		stats := make([]models.RacerStats, 0)
		rows, _ := app.DB.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf, dns FROM racer_stats")
		if rows != nil {
			for rows.Next() {
				var s models.RacerStats
				rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
				stats = append(stats, s)
			}
			rows.Close()
		}
		c.JSON(http.StatusOK, stats)
		return
	}

	var s models.RacerStats
	err := app.DB.QueryRow("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf, dns FROM racer_stats WHERE racer_id = ?", id).Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
	if err != nil {
		s = models.RacerStats{RacerID: 0}
	}

	var rInfo models.Racer
	app.DB.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", id).Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)

	c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
}

// @Summary Update racer stats
// @Description Manually update a racer's statistics
// @Tags Stats
// @Accept json
// @Produce json
// @Param stats body models.RacerStats true "Racer stats"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/racer-stats [post]
func UpdateRacerStats(c *gin.Context) {
	var stats models.RacerStats
	if err := c.ShouldBindJSON(&stats); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if stats.ID == 0 {
		_, err := app.DB.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			stats.RacerID, stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := app.DB.Exec("UPDATE racer_stats SET races = ?, wins = ?, gold = ?, silver = ?, bronze = ?, fastest_laps = ?, dnf = ?, dns = ? WHERE id = ?",
			stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS, stats.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)
}

// @Summary Get track stats
// @Description Get performance statistics grouped by track
// @Tags Stats
// @Produce json
// @Success 200 {array} models.TrackStats
// @Router /api/track-stats [get]
func GetTrackStats(c *gin.Context) {
	rows, err := app.DB.Query(`
		SELECT rh.track_id, rh.track, rh.country, COUNT(*) as races_count,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh.id AND rr.position = 1 LIMIT 1), '') as winner,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh.id AND rr.fastest_lap = 1 LIMIT 1), '') as fastest_lap
		FROM race_history rh
		WHERE rh.race_type = 'season'
		GROUP BY rh.track_id
		ORDER BY races_count DESC
	`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var stats []models.TrackStats
	for rows.Next() {
		var s models.TrackStats
		if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.RacesCount, &s.Winner, &s.FastestLap); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	c.JSON(http.StatusOK, stats)
}

// @Summary Head to head comparison
// @Description Compare two racers head to head
// @Tags Stats
// @Produce json
// @Param racer1 query int true "First racer ID"
// @Param racer2 query int true "Second racer ID"
// @Success 200 {object} models.HeadToHead
// @Failure 400 {object} map[string]string
// @Router /api/stats/head-to-head [get]
func GetHeadToHead(c *gin.Context) {
	racer1Str := c.Query("racer1")
	racer2Str := c.Query("racer2")

	racer1, err1 := strconv.Atoi(racer1Str)
	racer2, err2 := strconv.Atoi(racer2Str)
	if err1 != nil || err2 != nil || racer1 <= 0 || racer2 <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "racer1 and racer2 query params required"})
		return
	}

	var name1, name2 string
	app.DB.QueryRow("SELECT name FROM racers WHERE id = ?", racer1).Scan(&name1)
	app.DB.QueryRow("SELECT name FROM racers WHERE id = ?", racer2).Scan(&name2)

	rows, err := app.DB.Query(`
		SELECT rr1.race_id, rr1.position as pos1, rr2.position as pos2
		FROM race_results rr1
		JOIN race_results rr2 ON rr1.race_id = rr2.race_id
		JOIN race_history rh ON rh.id = rr1.race_id AND rh.race_type = 'season'
		WHERE rr1.racer_id = ? AND rr2.racer_id = ?
		ORDER BY rr1.race_id
	`, racer1, racer2)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var races int
	var racer1Wins, racer2Wins int
	var racer1PosSum, racer2PosSum float64

	for rows.Next() {
		var raceID, pos1, pos2 int
		if err := rows.Scan(&raceID, &pos1, &pos2); err != nil {
			continue
		}
		races++
		racer1PosSum += float64(pos1)
		racer2PosSum += float64(pos2)
		if pos1 < pos2 {
			racer1Wins++
		} else if pos2 < pos1 {
			racer2Wins++
		}
	}

	result := models.HeadToHead{
		Racer1:     name1,
		Racer2:     name2,
		Races:      races,
		Racer1Wins: racer1Wins,
		Racer2Wins: racer2Wins,
	}
	if races > 0 {
		result.Racer1Avg = math.Round(racer1PosSum/float64(races)*100) / 100
		result.Racer2Avg = math.Round(racer2PosSum/float64(races)*100) / 100
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Points progression
// @Description Get cumulative points progression for a racer
// @Tags Stats
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Success 200 {array} models.PointsProgression
// @Failure 400 {object} map[string]string
// @Router /api/stats/points-progression [get]
func GetPointsProgression(c *gin.Context) {
	racerIDStr := c.Query("racer_id")
	racerID, err := strconv.Atoi(racerIDStr)
	if err != nil || racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "racer_id query param required"})
		return
	}

	rows, err := app.DB.Query(`
		SELECT rh.id, COALESCE(rh.name, ''), rh.race_date, rr.points
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
		ORDER BY rh.race_date ASC
	`, racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	progression := make([]models.PointsProgression, 0)
	var cumulative int
	for rows.Next() {
		var p models.PointsProgression
		if err := rows.Scan(&p.RaceID, &p.RaceName, &p.RaceDate, &p.Points); err != nil {
			continue
		}
		cumulative += p.Points
		p.Points = cumulative
		progression = append(progression, p)
	}
	c.JSON(http.StatusOK, progression)
}

// @Summary Win/podium streaks
// @Description Get win and podium streak information for all racers
// @Tags Stats
// @Produce json
// @Success 200 {array} models.StreakInfo
// @Router /api/stats/streaks [get]
func GetStreaks(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name FROM racers ORDER BY rank")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type racerInfo struct {
		ID   int
		Name string
	}
	var racers []racerInfo
	for rows.Next() {
		var r racerInfo
		rows.Scan(&r.ID, &r.Name)
		racers = append(racers, r)
	}
	rows.Close()

	allStreaks := make([]models.StreakInfo, 0)
	for _, racer := range racers {
		raceRows, err := app.DB.Query(`
			SELECT rr.position, rh.race_date
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rr.racer_id = ? AND rh.race_type = 'season'
			ORDER BY rh.race_date ASC
		`, racer.ID)
		if err != nil {
			continue
		}

		var positions []struct {
			pos  int
			date string
		}
		for raceRows.Next() {
			var pos int
			var date string
			raceRows.Scan(&pos, &date)
			positions = append(positions, struct {
				pos  int
				date string
			}{pos, date})
		}
		raceRows.Close()

		if len(positions) == 0 {
			continue
		}

		calcStreak(positions, 1, "wins", racer.Name, &allStreaks)
		calcStreak(positions, 3, "podiums", racer.Name, &allStreaks)
		calcStreak(positions, 999, "dnf", racer.Name, &allStreaks)
	}

	c.JSON(http.StatusOK, allStreaks)
}

func calcStreak(positions []struct {
	pos  int
	date string
}, threshold int, streakType, racerName string, results *[]models.StreakInfo) {
	var current, best int
	var bestStart, bestEnd string
	var currentStart string

	for i, p := range positions {
		isMatch := false
		if streakType == "dnf" {
			isMatch = p.pos >= 900 // DNF positions are high numbers
		} else if streakType == "wins" {
			isMatch = p.pos == threshold
		} else {
			isMatch = p.pos <= threshold
		}

		if isMatch {
			if current == 0 {
				currentStart = p.date
			}
			current++
			if current > best {
				best = current
				bestStart = currentStart
				bestEnd = p.date
			}
		} else {
			current = 0
		}
		_ = i
	}

	var currentValue int
	for i := len(positions) - 1; i >= 0; i-- {
		isMatch := false
		if streakType == "dnf" {
			isMatch = positions[i].pos >= 900
		} else if streakType == "wins" {
			isMatch = positions[i].pos == threshold
		} else {
			isMatch = positions[i].pos <= threshold
		}
		if isMatch {
			currentValue++
		} else {
			break
		}
	}

	if best > 1 || currentValue > 0 {
		*results = append(*results, models.StreakInfo{
			RacerName:    racerName,
			StreakType:   streakType,
			CurrentValue: currentValue,
			BestValue:    best,
			BestStart:    bestStart,
			BestEnd:      bestEnd,
		})
	}
}

// @Summary ELO ratings
// @Description Get ELO ratings for all racers
// @Tags Stats
// @Produce json
// @Success 200 {array} models.ELORating
// @Router /api/stats/elo [get]
func GetELORatings(c *gin.Context) {
	type raceEntry struct {
		RaceID   int
		RacerID  int
		Position int
	}

	rows, err := app.DB.Query(`
		SELECT rr.race_id, rr.racer_id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'
		ORDER BY rr.race_id, rr.position
	`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var entries []raceEntry
	for rows.Next() {
		var e raceEntry
		if err := rows.Scan(&e.RaceID, &e.RacerID, &e.Position); err != nil {
			continue
		}
		if e.Position < 900 {
			entries = append(entries, e)
		}
	}

	ratings := make(map[int]float64)
	raceCount := make(map[int]int)

	raceMap := make(map[int][]raceEntry)
	for _, e := range entries {
		raceMap[e.RaceID] = append(raceMap[e.RaceID], e)
	}

	const K = 32
	initialRating := 1500.0

	for _, race := range raceMap {
		if len(race) < 2 {
			continue
		}

		for _, e := range race {
			if _, exists := ratings[e.RacerID]; !exists {
				ratings[e.RacerID] = initialRating
				raceCount[e.RacerID] = 0
			}
		}

		for i := 0; i < len(race); i++ {
			for j := i + 1; j < len(race); j++ {
				e1, e2 := race[i], race[j]
				r1, r2 := ratings[e1.RacerID], ratings[e2.RacerID]

				expected1 := 1.0 / (1.0 + math.Pow(10, (r2-r1)/400.0))
				expected2 := 1.0 - expected1

				score1 := 0.0
				score2 := 0.0
				if e1.Position < e2.Position {
					score1 = 1.0
				} else {
					score2 = 1.0
				}

				ratings[e1.RacerID] = r1 + K*(score1-expected1)
				ratings[e2.RacerID] = r2 + K*(score2-expected2)
			}
		}
		for _, e := range race {
			raceCount[e.RacerID]++
		}
	}

	result := make([]models.ELORating, 0)
	for rid, r := range ratings {
		var name string
		app.DB.QueryRow("SELECT name FROM racers WHERE id = ?", rid).Scan(&name)
		result = append(result, models.ELORating{
			RacerID:   rid,
			RacerName: name,
			Rating:    math.Round(r*100) / 100,
			Races:     raceCount[rid],
		})
	}

	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Rating > result[i].Rating {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	c.JSON(http.StatusOK, result)
}

// @Summary Export stats as CSV
// @Description Export racer statistics as a CSV file
// @Tags Stats
// @Produce text/csv
// @Success 200 {file} text/csv
// @Router /api/stats/export [get]
func ExportStatsCSV(c *gin.Context) {
	format := c.Query("format")
	if format == "" {
		format = "csv"
	}

	rows, err := app.DB.Query("SELECT r.id, r.name, r.car_name, r.points, r.rank, COALESCE(rs.races, 0), COALESCE(rs.wins, 0), COALESCE(rs.gold, 0), COALESCE(rs.silver, 0), COALESCE(rs.bronze, 0), COALESCE(rs.fastest_laps, 0), COALESCE(rs.dnf, 0), COALESCE(rs.dns, 0) FROM racers r LEFT JOIN racer_stats rs ON rs.racer_id = r.id ORDER BY r.rank")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename=heat_racer_stats.csv")

	writer := csv.NewWriter(c.Writer)
	writer.Write([]string{"ID", "Name", "Car", "Points", "Rank", "Races", "Wins", "Gold", "Silver", "Bronze", "Fastest Laps", "DNF", "DNS"})

	for rows.Next() {
		var id, points, rank, races, wins, gold, silver, bronze, fl, dnf, dns int
		var name, carName string
		rows.Scan(&id, &name, &carName, &points, &rank, &races, &wins, &gold, &silver, &bronze, &fl, &dnf, &dns)
		writer.Write([]string{
			strconv.Itoa(id), name, carName,
			strconv.Itoa(points), strconv.Itoa(rank),
			strconv.Itoa(races), strconv.Itoa(wins),
			strconv.Itoa(gold), strconv.Itoa(silver), strconv.Itoa(bronze),
			strconv.Itoa(fl), strconv.Itoa(dnf), strconv.Itoa(dns),
		})
	}
	writer.Flush()
}

// @Summary Track performance
// @Description Get racer performance by track, optionally filtered by racer_id
// @Tags Stats
// @Produce json
// @Param racer_id query int false "Racer ID"
// @Success 200 {array} object
// @Router /api/stats/track-performance [get]
func GetTrackPerformance(c *gin.Context) {
	racerIDStr := c.Query("racer_id")

	if racerIDStr != "" {
		racerID, _ := strconv.Atoi(racerIDStr)
		rows, err := app.DB.Query(`
			SELECT rh.track_id, rh.track, rh.country,
				COUNT(*) as races,
				SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
				SUM(CASE WHEN rr.position <= 3 THEN 1 ELSE 0 END) as podiums,
				AVG(rr.position) as avg_position
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rr.racer_id = ? AND rh.race_type = 'season'
			GROUP BY rh.track_id
			ORDER BY races DESC
		`, racerID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type TrackPerf struct {
			TrackID     string  `json:"track_id"`
			TrackName   string  `json:"track_name"`
			Country     string  `json:"country"`
			Races       int     `json:"races"`
			Wins        int     `json:"wins"`
			Podiums     int     `json:"podiums"`
			AvgPosition float64 `json:"avg_position"`
		}
		var perf []TrackPerf
		for rows.Next() {
			var p TrackPerf
			if err := rows.Scan(&p.TrackID, &p.TrackName, &p.Country, &p.Races, &p.Wins, &p.Podiums, &p.AvgPosition); err != nil {
				continue
			}
			perf = append(perf, p)
		}
		c.JSON(http.StatusOK, perf)
		return
	} else {
		rows, err := app.DB.Query(`
			SELECT rh.track_id, rh.track, rh.country,
				COUNT(DISTINCT rr.racer_id) as unique_drivers,
				COUNT(*) as total_entries,
				rr.racer_name as most_winner
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rh.race_type = 'season'
			GROUP BY rh.track_id
			ORDER BY rh.track
		`)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		type TrackSummary struct {
			TrackID       string `json:"track_id"`
			TrackName     string `json:"track_name"`
			Country       string `json:"country"`
			UniqueDrivers int    `json:"unique_drivers"`
			TotalEntries  int    `json:"total_entries"`
		}
		var summary []TrackSummary
		for rows.Next() {
			var s TrackSummary
			var winner string
			if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.UniqueDrivers, &s.TotalEntries, &winner); err != nil {
				continue
			}
			summary = append(summary, s)
		}
		c.JSON(http.StatusOK, summary)
	}
}
