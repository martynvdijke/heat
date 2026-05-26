package racing

import "database/sql"

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

	results := make([]QualifyingRaceDeltaData, 0)
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
