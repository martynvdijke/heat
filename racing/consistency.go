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

	results := make([]ConsistencyRatingData, 0)
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
