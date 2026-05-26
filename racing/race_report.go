package racing

import "database/sql"

// RaceReportData holds a comprehensive race report.
type RaceReportData struct {
	RaceID    int               `json:"race_id"`
	RaceName  string            `json:"name"`
	RaceDate  string            `json:"race_date"`
	Track     string            `json:"track"`
	Country   string            `json:"country"`
	TotalLaps int               `json:"total_laps"`
	Results   []RaceResultEntry `json:"results"`
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

// RaceReport generates a comprehensive race report.
func RaceReport(db *sql.DB, raceID int) (*RaceReportData, error) {
	report := &RaceReportData{RaceID: raceID}
	err := db.QueryRow("SELECT name, race_date, track, country, total_laps FROM race_history WHERE id = ?", raceID).
		Scan(&report.RaceName, &report.RaceDate, &report.Track, &report.Country, &report.TotalLaps)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(`
		SELECT rr.position, rr.racer_name, rr.points, rr.fastest_lap, rr.finished, COALESCE(rr.did_not_start, 0)
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
		var dns int
		if err := rows.Scan(&r.Position, &r.RacerName, &r.Points, &r.FastestLap, &r.Finished, &dns); err != nil {
			continue
		}
		if r.Position >= 900 {
			r.DNF = true
		}
		if dns == 1 {
			r.DNS = true
		}
		report.Results = append(report.Results, r)
	}
	return report, nil
}
