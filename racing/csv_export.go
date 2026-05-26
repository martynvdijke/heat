package racing

import (
	"database/sql"
	"math"
)

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

	data := make([]ExportStatsCSVData, 0)
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
