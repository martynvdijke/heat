package racing

import (
	"database/sql"
	"math"
)

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

	results := make([]TrackPerformanceData, 0)
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

	stats := make([]TrackStatsResult, 0)
	for rows.Next() {
		var s TrackStatsResult
		if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.RacesCount, &s.Winner, &s.FastestLap); err != nil {
			continue
		}
		stats = append(stats, s)
	}
	return stats, nil
}
