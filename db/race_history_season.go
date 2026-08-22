package db

import (
	"database/sql"
)

// EnsureRaceHistorySeasonColumn adds the nullable season_id column to
// race_history when it is missing (pragma-guarded, idempotent) and creates
// the lookup index used by season-scoped stats queries.
func EnsureRaceHistorySeasonColumn(database *sql.DB) error {
	var count int
	if err := database.QueryRow(
		`SELECT COUNT(*) FROM pragma_table_info('race_history') WHERE name = 'season_id'`,
	).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if _, err := database.Exec("ALTER TABLE race_history ADD COLUMN season_id INTEGER"); err != nil {
			return err
		}
	}
	_, err := database.Exec("CREATE INDEX IF NOT EXISTS idx_race_history_season ON race_history(season_id)")
	return err
}

// BackfillRaceHistorySeasons links historical races to seasons by matching
// round_snapshots on (race_name, race_date). Idempotent: only rows without a
// season are touched.
func BackfillRaceHistorySeasons(database *sql.DB) error {
	_, err := database.Exec(`UPDATE race_history
		SET season_id = (
			SELECT rs.season_id FROM round_snapshots rs
			WHERE rs.race_name = race_history.name
			  AND rs.race_date = race_history.race_date
			  AND rs.season_id IS NOT NULL
			ORDER BY rs.id LIMIT 1
		)
		WHERE season_id IS NULL`)
	return err
}
