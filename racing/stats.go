package racing

import (
	"database/sql"

	"heat/models"
)

// RacerStatsBySeason computes statistics for all racers within a season date range.
func RacerStatsBySeason(db *sql.DB, startDate, endDate string) []models.RacerStats {
	rows, err := db.Query(`
		SELECT rr.racer_id,
			COUNT(*) as races,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as gold,
			SUM(CASE WHEN rr.position = 2 THEN 1 ELSE 0 END) as silver,
			SUM(CASE WHEN rr.position = 3 THEN 1 ELSE 0 END) as bronze,
			SUM(CASE WHEN rr.fastest_lap = 1 THEN 1 ELSE 0 END) as fastest_laps,
			SUM(rr.points) as points,
			SUM(CASE WHEN rr.position >= 900 THEN 1 ELSE 0 END) as dnf,
			0 as dns
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rh.race_date >= ? AND rh.race_date <= ? AND rh.race_type = 'season'
		GROUP BY rr.racer_id
		ORDER BY points DESC
	`, startDate, endDate)
	if err != nil {
		return nil
	}
	defer rows.Close()

	stats := make([]models.RacerStats, 0)
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		stats = append(stats, s)
	}
	return stats
}

// SingleRacerStatsBySeason computes season statistics for a single racer.
// Returns the stats and whether the racer was found.
func SingleRacerStatsBySeason(db *sql.DB, racerID int, startDate, endDate string) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow(`
		SELECT rr.racer_id,
			COUNT(*) as races,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rr.position = 1 THEN 1 ELSE 0 END) as gold,
			SUM(CASE WHEN rr.position = 2 THEN 1 ELSE 0 END) as silver,
			SUM(CASE WHEN rr.position = 3 THEN 1 ELSE 0 END) as bronze,
			SUM(CASE WHEN rr.fastest_lap = 1 THEN 1 ELSE 0 END) as fastest_laps,
			SUM(rr.points) as points,
			SUM(CASE WHEN rr.position >= 900 THEN 1 ELSE 0 END) as dnf,
			0 as dns
		FROM race_results rr
		JOIN race_history rh ON rh.id = rr.race_id
		WHERE rr.racer_id = ? AND rh.race_date >= ? AND rh.race_date <= ? AND rh.race_type = 'season'
		GROUP BY rr.racer_id
	`, racerID, startDate, endDate).Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
	if err != nil {
		return s, false
	}
	return s, true
}

// SingleRacerStatsFallback retrieves legacy stats for a single racer from racer_stats table.
func SingleRacerStatsFallback(db *sql.DB, racerID int) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf, dns FROM racer_stats WHERE racer_id = ?", racerID).
		Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
	if err != nil {
		return s, false
	}
	return s, true
}

// RacerInfo retrieves basic racer information by ID.
func RacerInfo(db *sql.DB, racerID int) models.Racer {
	var rInfo models.Racer
	db.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", racerID).
		Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)
	return rInfo
}

// UpsertRacerStats creates or updates racer statistics in the racer_stats table.
func UpsertRacerStats(db *sql.DB, stats models.RacerStats) error {
	if stats.ID == 0 {
		_, err := db.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
			stats.RacerID, stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS)
		return err
	}
	_, err := db.Exec("UPDATE racer_stats SET races = ?, wins = ?, gold = ?, silver = ?, bronze = ?, fastest_laps = ?, dnf = ?, dns = ? WHERE id = ?",
		stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS, stats.ID)
	return err
}

// AllRacerStats retrieves all racer stats from the racer_stats table.
func AllRacerStats(db *sql.DB) []models.RacerStats {
	stats := make([]models.RacerStats, 0)
	rows, err := db.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf, dns FROM racer_stats")
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS)
		stats = append(stats, s)
	}
	return stats
}
