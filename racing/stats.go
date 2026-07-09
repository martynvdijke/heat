package racing

import (
	"database/sql"

	"heat/models"
)

// RacerStatsBySeason computes statistics for all racers within a season,
// aggregated from finalized round snapshot scores.
func RacerStatsBySeason(db *sql.DB, seasonID int) []models.RacerStats {
	rows, err := db.Query(`
		SELECT rss.racer_id,
			COUNT(*) as races,
			SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as gold,
			SUM(CASE WHEN rss.position = 2 THEN 1 ELSE 0 END) as silver,
			SUM(CASE WHEN rss.position = 3 THEN 1 ELSE 0 END) as bronze,
			0 as fastest_laps,
			SUM(rss.points) as points,
			SUM(CASE WHEN rss.dnf = 1 THEN 1 ELSE 0 END) as dnf,
			SUM(CASE WHEN rss.dns = 1 THEN 1 ELSE 0 END) as dns,
			SUM(rss.spins) as spins,
			SUM(rss.overheated) as overheated
		FROM round_snapshot_scores rss
		JOIN round_snapshots rs ON rs.id = rss.snapshot_id
		WHERE rs.season_id = ? AND rs.status = 'final'
		GROUP BY rss.racer_id
		ORDER BY points DESC
	`, seasonID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	stats := make([]models.RacerStats, 0)
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
		stats = append(stats, s)
	}
	return stats
}

// SingleRacerStatsBySeason computes season statistics for a single racer,
// aggregated from finalized round snapshot scores.
// Returns the stats and whether the racer was found.
func SingleRacerStatsBySeason(db *sql.DB, racerID int, seasonID int) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow(`
		SELECT rss.racer_id,
			COUNT(*) as races,
			SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as wins,
			SUM(CASE WHEN rss.position = 1 THEN 1 ELSE 0 END) as gold,
			SUM(CASE WHEN rss.position = 2 THEN 1 ELSE 0 END) as silver,
			SUM(CASE WHEN rss.position = 3 THEN 1 ELSE 0 END) as bronze,
			0 as fastest_laps,
			SUM(rss.points) as points,
			SUM(CASE WHEN rss.dnf = 1 THEN 1 ELSE 0 END) as dnf,
			SUM(CASE WHEN rss.dns = 1 THEN 1 ELSE 0 END) as dns,
			SUM(rss.spins) as spins,
			SUM(rss.overheated) as overheated
		FROM round_snapshot_scores rss
		JOIN round_snapshots rs ON rs.id = rss.snapshot_id
		WHERE rss.racer_id = ? AND rs.season_id = ? AND rs.status = 'final'
		GROUP BY rss.racer_id
	`, racerID, seasonID).Scan(&s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
	if err != nil {
		return s, false
	}
	return s, true
}

// SingleRacerStatsFallback retrieves legacy stats for a single racer from racer_stats table.
func SingleRacerStatsFallback(db *sql.DB, racerID int) (models.RacerStats, bool) {
	var s models.RacerStats
	err := db.QueryRow("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf, dns, spins, overheated FROM racer_stats WHERE racer_id = ?", racerID).
		Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
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
		_, err := db.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns, spins, overheated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
			stats.RacerID, stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS, stats.Spins, stats.Overheated)
		return err
	}
	_, err := db.Exec("UPDATE racer_stats SET races = ?, wins = ?, gold = ?, silver = ?, bronze = ?, fastest_laps = ?, dnf = ?, dns = ?, spins = ?, overheated = ? WHERE id = ?",
		stats.Races, stats.Wins, stats.Gold, stats.Silver, stats.Bronze, stats.FastestLaps, stats.DNF, stats.DNS, stats.Spins, stats.Overheated, stats.ID)
	return err
}

// AllRacerStats retrieves all racer stats from the racer_stats table.
func AllRacerStats(db *sql.DB) []models.RacerStats {
	stats := make([]models.RacerStats, 0)
	rows, err := db.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf, dns, spins, overheated FROM racer_stats")
	if err != nil {
		return stats
	}
	defer rows.Close()
	for rows.Next() {
		var s models.RacerStats
		rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
		stats = append(stats, s)
	}
	return stats
}
