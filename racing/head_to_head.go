package racing

import (
	"database/sql"
	"math"
)

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

	progression := make([]PointsProgressionData, 0)
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
