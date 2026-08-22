package racing

import (
	"database/sql"
	"math"
	"sort"
)

// ConsistencyRatingData holds consistency statistics for a racer.
type ConsistencyRatingData struct {
	RacerID      int     `json:"racer_id"`
	RacerName    string  `json:"racer_name"`
	Races        int     `json:"races"`
	AvgPosition  float64 `json:"avg_position"`
	StdDeviation float64 `json:"std_deviation"`
	Consistency  float64 `json:"consistency"`
}

// ConsistencyRatings computes consistency ratings for all racers. An empty
// season list means "all seasons".
func ConsistencyRatings(db *sql.DB, seasonIDs []int) ([]ConsistencyRatingData, error) {
	// Single batched query: compute avg and variance per racer
	filter, args := SeasonFilter("rh", seasonIDs)
	rows, err := db.Query(`
		SELECT
			rr.racer_id,
			COALESCE(r.name, ''),
			COUNT(*) as races,
			AVG(CAST(rr.position AS REAL)) as avg_pos,
			AVG(CAST(rr.position AS REAL) * CAST(rr.position AS REAL)) - AVG(CAST(rr.position AS REAL)) * AVG(CAST(rr.position AS REAL)) as variance
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		LEFT JOIN racers r ON r.id = rr.racer_id
		WHERE rh.race_type = 'season'`+filter+`
		GROUP BY rr.racer_id
		HAVING COUNT(*) >= 3
	`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := make([]ConsistencyRatingData, 0)
	for rows.Next() {
		var c ConsistencyRatingData
		var variance float64
		if err := rows.Scan(&c.RacerID, &c.RacerName, &c.Races, &c.AvgPosition, &variance); err != nil {
			continue
		}
		if variance < 0 {
			variance = 0
		}
		c.StdDeviation = math.Sqrt(variance)
		c.Consistency = math.Round((1.0/(1.0+c.StdDeviation))*100*100) / 100
		c.AvgPosition = math.Round(c.AvgPosition*100) / 100
		results = append(results, c)
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Consistency > results[j].Consistency
	})
	return results, nil
}
