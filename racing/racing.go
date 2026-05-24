package racing

import (
	"database/sql"
	"math"
	"sort"

	"heat/models"
)

// RacerStatsFallback retrieves racer statistics from the racer_stats table (legacy).
func RacerStatsFallback(db *sql.DB) []models.RacerStats {
	stats := make([]models.RacerStats, 0)
	rows, err := db.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_stats.racer_id) as pts, dnf, dns FROM racer_stats")
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		stats = append(stats, s)
	}
	return stats
}

// RacerStatsBySeason computes statistics for all racers within a season date range.
func RacerStatsBySeason(db *sql.DB, startDate, endDate string) []models.RacerStats {
	rows, err := db.Query(`
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
		return nil
	}
	defer rows.Close()

	stats := make([]models.RacerStats, 0)
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		stats = append(stats, s)
	}
	return stats
}

// SingleRacerStatsBySeason computes season statistics for a single racer.
// Returns the stats and whether the racer was found.
func SingleRacerStatsBySeason(db *sql.DB, racerID int, startDate, endDate string) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow(`
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
	`, racerID, startDate, endDate).Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
	if err != nil {
		return s, false
	}
	return s, true
}

// SingleRacerStatsFallback retrieves legacy stats for a single racer from racer_stats table.
func SingleRacerStatsFallback(db *sql.DB, racerID int) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf, dns FROM racer_stats WHERE racer_id = ?", racerID).
		Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
	if err != nil {
		return s, false
	}
	return s, true
}

// RacerInfo retrieves basic racer information by ID.
func RacerInfo(db *sql.DB, racerID int) models.Racer {
	var rInfo models.Racer
	db.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", racerID).
		Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)
	return rInfo
}

// SeasonDates retrieves the start and end dates for a season.
func SeasonDates(db *sql.DB, seasonID int) (startDate, endDate string, err error) {
	err = db.QueryRow("SELECT start_date, COALESCE(end_date, '9999-12-31') FROM seasons WHERE id = ?", seasonID).
		Scan(&startDate, &endDate)
	return
}

// UpsertRacerStats creates or updates racer statistics in the racer_stats table.
func UpsertRacerStats(db *sql.DB, stats models.RacerStats) error {
	if stats.ID == 0 {
		_, err := db.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			stats.RacerID, stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS)
		return err
	}
	_, err := db.Exec("UPDATE racer_stats SET races = ?, wins = ?, gold = ?, silver = ?, bronze = ?, fastest_laps = ?, dnf = ?, dns = ? WHERE id = ?",
		stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS, stats.ID)
	return err
}

// AllRacerStats retrieves all racer stats from the racer_stats table.
func AllRacerStats(db *sql.DB) []models.RacerStats {
	stats := make([]models.RacerStats, 0)
	rows, err := db.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf, dns FROM racer_stats")
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		stats = append(stats, s)
	}
	return stats
}

// TrackStatsResult holds per-track performance statistics.
type TrackStatsResult struct {
	TrackID    string
	TrackName  string
	Country    string
	RacesCount int
	Winner     string
	FastestLap string
}

// TrackStats computes performance statistics grouped by track.
func TrackStats(db *sql.DB) ([]TrackStatsResult, error) {
	rows, err := db.Query(`
		SELECT rh.track_id, rh.track, rh.country, COUNT(*) as races_count,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh.id AND rr.position = 1 LIMIT 1), '') as winner,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh.id AND rr.fastest_lap = 1 LIMIT 1), '') as fastest_lap
		FROM race_history rh
		WHERE rh.race_type = 'season'
		GROUP BY rh.track_id
		ORDER BY races_count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TrackStatsResult
	for rows.Next() {
		var s TrackStatsResult
		if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.RacesCount, &s.Winner, &s.FastestLap); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, nil
}

// HeadToHeadData holds the result of a head-to-head comparison.
type HeadToHeadData struct {
	Racer1     string
	Racer2     string
	Races      int
	Racer1Wins int
	Racer2Wins int
	Racer1Avg  float64
	Racer2Avg  float64
}

// HeadToHead compares two racers across all season races.
func HeadToHead(db *sql.DB, racer1, racer2 int) (*HeadToHeadData, error) {
	var name1, name2 string
	db.QueryRow("SELECT name FROM racers WHERE id = ?", racer1).Scan(&name1)
	db.QueryRow("SELECT name FROM racers WHERE id = ?", racer2).Scan(&name2)

	rows, err := db.Query(`
		SELECT rr1.race_id, rr1.position as pos1, rr2.position as pos2
		FROM race_results rr1
		JOIN race_results rr2 ON rr1.race_id = rr2.race_id
		JOIN race_history rh ON rh.id = rr1.race_id AND rh.race_type = 'season'
		WHERE rr1.racer_id = ? AND rr2.racer_id = ?
		ORDER BY rr1.race_id
	`, racer1, racer2)
	if err != nil {
		return nil, err
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

	result := &HeadToHeadData{
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
	return result, nil
}

// PointsProgressionData holds a single point in the points progression timeline.
type PointsProgressionData struct {
	RaceID     int     `json:"race_id"`
	RaceDate   string  `json:"race_date"`
	RaceName   string  `json:"race_name"`
	Position   int     `json:"position"`
	Points     float64 `json:"points"`
	Cumulative float64 `json:"cumulative"`
}

// PointsProgression computes the cumulative points progression for a racer.
func PointsProgression(db *sql.DB, racerID int) ([]PointsProgressionData, error) {
	rows, err := db.Query(`
		SELECT rh.id, rh.race_date, rh.name, rr.position, CAST(rr.points AS REAL) as pts
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
		ORDER BY rh.race_date ASC
	`, racerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var progression []PointsProgressionData
	var cumulative float64
	for rows.Next() {
		var p PointsProgressionData
		if err := rows.Scan(&p.RaceID, &p.RaceDate, &p.RaceName, &p.Position, &p.Points); err != nil {
			continue
		}
		cumulative += p.Points
		p.Cumulative = cumulative
		progression = append(progression, p)
	}
	return progression, nil
}

// StreakData holds streak information for a racer.
type StreakData struct {
	CurrentStreak int     `json:"current_streak"`
	BestStreak    int     `json:"best_streak"`
	StreakType    string  `json:"streak_type"`
	TotalRaces    int     `json:"total_races"`
	AvgPosition   float64 `json:"avg_position"`
	Wins          int     `json:"wins"`
	Podiums       int     `json:"podiums"`
	RacerID       int     `json:"racer_id"`
	RacerName     string  `json:"racer_name,omitempty"`
}

type positionEntry struct {
	RaceID   int
	Position int
}

func calcStreak(positions []positionEntry) (current int, best int) {
	if len(positions) == 0 {
		return 0, 0
	}

	// Sort by race_id ascending
	sorted := make([]positionEntry, len(positions))
	copy(sorted, positions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RaceID < sorted[j].RaceID
	})

	current = 0
	best = 0
	streak := 0

	for _, p := range sorted {
		if p.Position <= 3 {
			streak++
			if streak > best {
				best = streak
			}
		} else {
			streak = 0
		}
	}
	current = streak
	return current, best
}

// Streaks computes streak data for a racer based on their race history.
func Streaks(db *sql.DB, racerID int) (*StreakData, error) {
	var name string
	db.QueryRow("SELECT name FROM racers WHERE id = ?", racerID).Scan(&name)

	rows, err := db.Query(`
		SELECT rh.id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
		ORDER BY rh.id
	`, racerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var positions []positionEntry
	for rows.Next() {
		var p positionEntry
		if err := rows.Scan(&p.RaceID, &p.Position); err != nil {
			continue
		}
		positions = append(positions, p)
	}

	current, best := calcStreak(positions)

	var wins, podiums int
	db.QueryRow(`
		SELECT COUNT(*) FILTER (WHERE position = 1) as wins,
			COUNT(*) FILTER (WHERE position <= 3) as podiums,
			COUNT(*) as total,
			AVG(CAST(position AS REAL))
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
	`, racerID).Scan(&wins, &podiums, new(int), new(float64))

	totalRows := db.QueryRow(`
		SELECT COUNT(*), AVG(CAST(rr.position AS REAL))
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
	`, racerID)
	var totalRaces int
	var avgPosition float64
	totalRows.Scan(&totalRaces, &avgPosition)

	result := &StreakData{
		CurrentStreak: current,
		BestStreak:    best,
		StreakType:    "podium",
		TotalRaces:    totalRaces,
		AvgPosition:   math.Round(avgPosition*100) / 100,
		Wins:          wins,
		Podiums:       podiums,
		RacerID:       racerID,
		RacerName:     name,
	}
	return result, nil
}

// AllStreaks computes streak data for all racers.
func AllStreaks(db *sql.DB) []StreakData {
	rows, err := db.Query("SELECT id, name FROM racers ORDER BY rank")
	if err != nil {
		return []StreakData{}
	}

	// Collect all racers first, then close rows to free the connection
	type racerInfo struct {
		ID   int
		Name string
	}
	var racers []racerInfo
	for rows.Next() {
		var r racerInfo
		if err := rows.Scan(&r.ID, &r.Name); err != nil {
			continue
		}
		racers = append(racers, r)
	}
	rows.Close()

	var allStreaks []StreakData
	for _, racer := range racers {
		raceRows, err := db.Query(`
			SELECT rh.id, rr.position
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rr.racer_id = ? AND rh.race_type = 'season'
			ORDER BY rh.id
		`, racer.ID)
		if err != nil {
			continue
		}

		var positions []positionEntry
		for raceRows.Next() {
			var p positionEntry
			if err := raceRows.Scan(&p.RaceID, &p.Position); err != nil {
				continue
			}
			positions = append(positions, p)
		}
		raceRows.Close()

		if len(positions) == 0 {
			continue
		}

		current, best := calcStreak(positions)
		allStreaks = append(allStreaks, StreakData{
			CurrentStreak: current,
			BestStreak:    best,
			StreakType:    "podium",
			RacerID:       racer.ID,
			RacerName:     racer.Name,
		})
	}
	return allStreaks
}

// ELORatingData holds ELO rating information for a racer.
type ELORatingData struct {
	RacerID   int     `json:"racer_id"`
	RacerName string  `json:"racer_name"`
	Rating    float64 `json:"rating"`
	Races     int     `json:"races"`
}

// ELORatings computes ELO-style ratings for all racers based on race results.
func ELORatings(db *sql.DB) ([]ELORatingData, error) {
	ratings := make(map[int]float64)
	raceCount := make(map[int]int)

	rows, err := db.Query(`
		SELECT rh.id as race_id, rr.racer_id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'
		ORDER BY rh.id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type raceRow struct {
		RaceID   int
		RacerID  int
		Position int
	}
	var allRows []raceRow
	for rows.Next() {
		var r raceRow
		if err := rows.Scan(&r.RaceID, &r.RacerID, &r.Position); err != nil {
			continue
		}
		allRows = append(allRows, r)
	}

	type raceGroup struct {
		racers []raceRow
	}
	races := make(map[int]*raceGroup)
	for _, r := range allRows {
		if races[r.RaceID] == nil {
			races[r.RaceID] = &raceGroup{}
		}
		races[r.RaceID].racers = append(races[r.RaceID].racers, r)
	}

	for _, racers := range allRows {
		if _, ok := ratings[racers.RacerID]; !ok {
			ratings[racers.RacerID] = 1500
		}
	}

	for _, group := range races {
		if len(group.racers) < 2 {
			continue
		}
		for _, r := range group.racers {
			raceCount[r.RacerID]++
			for _, opponent := range group.racers {
				if r.RacerID == opponent.RacerID {
					continue
				}
				expected := 1.0 / (1 + math.Pow(10, (ratings[opponent.RacerID]-ratings[r.RacerID])/400))
				var actual float64
				if r.Position < opponent.Position {
					actual = 1.0
				} else if r.Position == opponent.Position {
					actual = 0.5
				}
				k := 32.0
				if raceCount[r.RacerID] > 10 {
					k = 16.0
				}
				ratings[r.RacerID] += k * (actual - expected)
			}
		}
	}

	var results []ELORatingData
	for id, rating := range ratings {
		var name string
		db.QueryRow("SELECT name FROM racers WHERE id = ?", id).Scan(&name)
		results = append(results, ELORatingData{
			RacerID:   id,
			RacerName: name,
			Rating:    math.Round(rating*100) / 100,
			Races:     raceCount[id],
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rating > results[j].Rating
	})
	return results, nil
}

// TrackPerformanceData holds per-track performance for a racer.
type TrackPerformanceData struct {
	TrackID      string  `json:"track_id"`
	TrackName    string  `json:"track_name"`
	Country      string  `json:"country"`
	Races        int     `json:"races"`
	Wins         int     `json:"wins"`
	Podiums      int     `json:"podiums"`
	AvgPosition  float64 `json:"avg_position"`
	BestPosition int     `json:"best_position"`
	TotalPoints  float64 `json:"total_points"`
	DNFRate      float64 `json:"dnf_rate"`
}

// TrackPerformance computes per-track statistics for a racer.
func TrackPerformance(db *sql.DB, racerID int) ([]TrackPerformanceData, error) {
	rows, err := db.Query(`
		SELECT rh.track_id, rh.track, rh.country,
			COUNT(*) as races,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rr.position <= 3 THEN 1 ELSE 0 END) as podiums,
			AVG(CAST(rr.position AS REAL)) as avg_position,
			MIN(rr.position) as best_position,
			SUM(rr.points) as total_points,
			AVG(CASE WHEN rr.position >= 900 THEN 1.0 ELSE 0.0 END) as dnf_rate
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
		GROUP BY rh.track_id
		ORDER BY races DESC
	`, racerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []TrackPerformanceData
	for rows.Next() {
		var t TrackPerformanceData
		if err := rows.Scan(&t.TrackID, &t.TrackName, &t.Country, &t.Races, &t.Wins, &t.Podiums, &t.AvgPosition, &t.BestPosition, &t.TotalPoints, &t.DNFRate); err != nil {
			continue
		}
		t.AvgPosition = math.Round(t.AvgPosition*100) / 100
		t.DNFRate = math.Round(t.DNFRate*100) / 100
		results = append(results, t)
	}
	return results, nil
}

// QualifyingRaceDeltaData holds the delta between qualifying and race positions.
type QualifyingRaceDeltaData struct {
	RaceID        int    `json:"race_id"`
	RaceDate      string `json:"race_date"`
	TrackName     string `json:"track_name"`
	QualifyingPos int    `json:"qualifying_position"`
	RacePosition  int    `json:"race_position"`
	Delta         int    `json:"delta"`
}

// QualifyingRaceDelta computes the delta between qualifying and race positions.
func QualifyingRaceDelta(db *sql.DB, racerID int) ([]QualifyingRaceDeltaData, error) {
	rows, err := db.Query(`
		SELECT rh.id, rh.race_date, rh.track,
			rr.position as race_position,
			COALESCE((SELECT position FROM race_results rr2 WHERE rr2.race_id = rr.race_id AND rr2.racer_id = rr.racer_id AND rr2.lap = 1), rr.position) as qualifying_position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
	`, racerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []QualifyingRaceDeltaData
	for rows.Next() {
		var q QualifyingRaceDeltaData
		if err := rows.Scan(&q.RaceID, &q.RaceDate, &q.TrackName, &q.RacePosition, &q.QualifyingPos); err != nil {
			continue
		}
		q.Delta = q.QualifyingPos - q.RacePosition
		results = append(results, q)
	}
	return results, nil
}

// ConsistencyRatingData holds consistency statistics for a racer.
type ConsistencyRatingData struct {
	RacerID      int     `json:"racer_id"`
	RacerName    string  `json:"racer_name"`
	Races        int     `json:"races"`
	AvgPosition  float64 `json:"avg_position"`
	StdDeviation float64 `json:"std_deviation"`
	Consistency  float64 `json:"consistency"`
}

// ConsistencyRatings computes consistency ratings for all racers.
func ConsistencyRatings(db *sql.DB) ([]ConsistencyRatingData, error) {
	rows, err := db.Query(`
		SELECT rr.racer_id, COUNT(*) as races, AVG(CAST(rr.position AS REAL)) as avg_pos
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'
		GROUP BY rr.racer_id
		HAVING COUNT(*) >= 3
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ConsistencyRatingData
	for rows.Next() {
		var c ConsistencyRatingData
		if err := rows.Scan(&c.RacerID, &c.Races, &c.AvgPosition); err != nil {
			continue
		}

		var name string
		db.QueryRow("SELECT name FROM racers WHERE id = ?", c.RacerID).Scan(&name)
		c.RacerName = name

		var stdDev float64
		db.QueryRow(`
			SELECT AVG(CAST((position - ?) * (position - ?) AS REAL))
			FROM race_results rr
			JOIN race_history rh ON rh.id = rr.race_id
			WHERE rr.racer_id = ? AND rh.race_type = 'season'
		`, c.AvgPosition, c.AvgPosition, c.RacerID).Scan(&stdDev)

		c.StdDeviation = math.Sqrt(stdDev)
		c.Consistency = math.Round((1.0/(1.0+c.StdDeviation))*100*100) / 100
		c.AvgPosition = math.Round(c.AvgPosition*100) / 100
		results = append(results, c)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Consistency > results[j].Consistency
	})
	return results, nil
}

// RaceReportData holds a comprehensive race report.
type RaceReportData struct {
	RaceID     int               `json:"race_id"`
	RaceName   string            `json:"race_name"`
	RaceDate   string            `json:"race_date"`
	Track      string            `json:"track"`
	Results    []RaceResultEntry `json:"results"`
	LapRecords []LapRecordEntry  `json:"lap_records,omitempty"`
	RaceRadio  []RaceRadioEntry  `json:"race_radio,omitempty"`
}

// RaceResultEntry holds a single race result entry.
type RaceResultEntry struct {
	Position   int     `json:"position"`
	RacerName  string  `json:"racer_name"`
	Points     float64 `json:"points"`
	FastestLap bool    `json:"fastest_lap"`
	Finished   bool    `json:"finished"`
	DNF        bool    `json:"dnf,omitempty"`
	DNS        bool    `json:"dns,omitempty"`
}

// LapRecordEntry holds lap record information for a race.
type LapRecordEntry struct {
	RacerName string  `json:"racer_name"`
	LapNumber int     `json:"lap_number"`
	LapTime   float64 `json:"lap_time"`
}

// RaceRadioEntry holds a race radio message.
type RaceRadioEntry struct {
	RacerName string `json:"racer_name"`
	Message   string `json:"message"`
}

// RaceReport generates a comprehensive race report.
func RaceReport(db *sql.DB, raceID int) (*RaceReportData, error) {
	report := &RaceReportData{RaceID: raceID}
	err := db.QueryRow("SELECT name, race_date, track FROM race_history WHERE id = ?", raceID).
		Scan(&report.RaceName, &report.RaceDate, &report.Track)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT rr.position, rr.racer_name, rr.points, rr.fastest_lap, rr.finished
		FROM race_results rr
		WHERE rr.race_id = ?
		ORDER BY rr.position ASC
	`, raceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r RaceResultEntry
		if err := rows.Scan(&r.Position, &r.RacerName, &r.Points, &r.FastestLap, &r.Finished); err != nil {
			continue
		}
		if r.Position >= 900 {
			r.DNF = true
		}
		report.Results = append(report.Results, r)
	}
	return report, nil
}

// ExportStatsCSVData holds the racer stats data for CSV export.
type ExportStatsCSVData struct {
	RacerName   string
	Races       int
	Wins        int
	Podiums     int
	Points      float64
	AvgPosition float64
	WinRate     float64
}

// ExportStatsCSV computes racer statistics formatted for CSV export.
func ExportStatsCSV(db *sql.DB) ([]ExportStatsCSVData, error) {
	rows, err := db.Query(`
		SELECT rr.racer_id,
			rr.racer_name,
			COUNT(*) as races,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rr.position <= 3 THEN 1 ELSE 0 END) as podiums,
			SUM(rr.points) as points,
			AVG(CAST(rr.position AS REAL)) as avg_position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'
		GROUP BY rr.racer_id
		ORDER BY points DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var data []ExportStatsCSVData
	for rows.Next() {
		var d ExportStatsCSVData
		var racerName string
		if err := rows.Scan(new(int), &racerName, &d.Races, &d.Wins, &d.Podiums, &d.Points, &d.AvgPosition); err != nil {
			continue
		}
		d.RacerName = racerName
		d.AvgPosition = math.Round(d.AvgPosition*100) / 100
		if d.Races > 0 {
			d.WinRate = math.Round(float64(d.Wins)/float64(d.Races)*100*100) / 100
		}
		data = append(data, d)
	}
	return data, nil
}
