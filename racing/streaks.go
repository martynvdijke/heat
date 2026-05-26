package racing

import (
	"database/sql"
	"math"
	"sort"
)

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

	var wins, podiums, totalRaces int
	var avgPosition float64
	db.QueryRow(`
		SELECT COUNT(*) FILTER (WHERE position = 1) as wins,
			COUNT(*) FILTER (WHERE position <= 3) as podiums,
			COUNT(*) as total,
			AVG(CAST(position AS REAL))
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'
	`, racerID).Scan(&wins, &podiums, &totalRaces, &avgPosition)

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

	allStreaks := make([]StreakData, 0)
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
