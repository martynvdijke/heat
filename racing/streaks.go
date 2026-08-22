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
// An empty season list means "all seasons".
func Streaks(db *sql.DB, racerID int, seasonIDs []int) (*StreakData, error) {
	var name string
	db.QueryRow("SELECT name FROM racers WHERE id = ?", racerID).Scan(&name)

	filter, args := SeasonFilter("rh", seasonIDs)
	rows, err := db.Query(`
		SELECT rh.id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_type = 'season'`+filter+`
		ORDER BY rh.id
	`, append([]any{racerID}, args...)...)
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
		WHERE rr.racer_id = ? AND rh.race_type = 'season'`+filter+`
	`, append([]any{racerID}, args...)...).Scan(&wins, &podiums, &totalRaces, &avgPosition)

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

// AllStreaks computes streak data for all racers. An empty season list means
// "all seasons".
func AllStreaks(db *sql.DB, seasonIDs []int) []StreakData {
	// Single batched query: get all race results ordered by race_id
	filter, args := SeasonFilter("rh", seasonIDs)
	rows, err := db.Query(`
		SELECT rr.racer_id, rh.id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'`+filter+`
		ORDER BY rr.racer_id, rh.id
	`, args...)
	if err != nil {
		return []StreakData{}
	}
	defer rows.Close()

	racerPositions := make(map[int][]positionEntry)
	racerOrder := make([]int, 0)
	for rows.Next() {
		var racerID, raceID, position int
		if err := rows.Scan(&racerID, &raceID, &position); err != nil {
			continue
		}
		if _, ok := racerPositions[racerID]; !ok {
			racerOrder = append(racerOrder, racerID)
		}
		racerPositions[racerID] = append(racerPositions[racerID], positionEntry{RaceID: raceID, Position: position})
	}
	rows.Close()

	// Fetch all racer names in one batch query
	nameMap := make(map[int]string)
	nameRows, err := db.Query("SELECT id, name FROM racers ORDER BY rank")
	if err == nil {
		for nameRows.Next() {
			var id int
			var name string
			if nameRows.Scan(&id, &name) == nil {
				nameMap[id] = name
			}
		}
		nameRows.Close()
	}

	allStreaks := make([]StreakData, 0, len(racerOrder))
	for _, racerID := range racerOrder {
		positions := racerPositions[racerID]
		if len(positions) == 0 {
			continue
		}
		current, best := calcStreak(positions)
		allStreaks = append(allStreaks, StreakData{
			CurrentStreak: current,
			BestStreak:    best,
			StreakType:    "podium",
			RacerID:       racerID,
			RacerName:     nameMap[racerID],
		})
	}
	return allStreaks
}
