package racing

import (
	"database/sql"
	"math"
	"sort"
)

// ELORatingData holds ELO rating information for a racer.
type ELORatingData struct {
	RacerID   int     `json:"racer_id"`
	RacerName string  `json:"racer_name"`
	Rating    float64 `json:"rating"`
	Races     int     `json:"races"`
}

// ELORatings computes ELO-style ratings for all racers based on race results.
// An empty season list means "all seasons"; a scoped rating is computed over
// only those seasons' races (order-dependent by race ID).
func ELORatings(db *sql.DB, seasonIDs []int) ([]ELORatingData, error) {
	ratings := make(map[int]float64)
	raceCount := make(map[int]int)

	filter, args := SeasonFilter("rh", seasonIDs)
	rows, err := db.Query(`
		SELECT rh.id as race_id, rr.racer_id, rr.position
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_type = 'season'`+filter+`
		ORDER BY rh.id
	`, args...)
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

	// Fetch all racer names in one batch query
	nameMap := make(map[int]string)
	nameRows, err := db.Query("SELECT id, name FROM racers")
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

	results := make([]ELORatingData, 0, len(ratings))
	for id, rating := range ratings {
		results = append(results, ELORatingData{
			RacerID:   id,
			RacerName: nameMap[id],
			Rating:    math.Round(rating*100) / 100,
			Races:     raceCount[id],
		})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Rating > results[j].Rating
	})
	return results, nil
}
