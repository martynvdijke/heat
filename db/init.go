package db

import (
	"context"
	"log"

	"heat/app"
)

var srv *app.Server

func SetServer(s *app.Server) {
	srv = s
}

func Init(s *app.Server) {
	srv = s
	ctx := context.Background()

	if err := srv.Ent.Schema.Create(ctx); err != nil {
		log.Fatalf("[DB] Failed to run Ent auto-migration: %v", err)
	}

	// Backfill round_snapshots: set season_id to the active season or 1 for NULL/0 values
	srv.DB.Exec(`UPDATE round_snapshots SET season_id = (SELECT COALESCE((SELECT id FROM seasons WHERE status = 'active' LIMIT 1), 1)) WHERE season_id IS NULL OR season_id = 0`)

	// Deduplicate (season_id, round) pairs: re-number duplicate rounds to the next available number
	srv.DB.Exec(`UPDATE round_snapshots SET round = (SELECT COALESCE(MAX(r2.round), 0) + 1 FROM round_snapshots r2 WHERE r2.season_id = round_snapshots.season_id AND r2.id != round_snapshots.id) WHERE id IN (SELECT id FROM round_snapshots r1 WHERE (SELECT COUNT(*) FROM round_snapshots r2 WHERE r2.season_id = r1.season_id AND r2.round = r1.round AND r2.id < r1.id) > 0)`)

	// Add UNIQUE constraint on (season_id, round) for round-season enforcement
	srv.DB.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_round_snapshots_season_round ON round_snapshots(season_id, round)")

	srv.DB.Exec("PRAGMA foreign_keys=OFF")
	defer func() {
		srv.DB.Exec("PRAGMA foreign_keys=ON")
	}()

	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS app_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL DEFAULT (datetime('now')),
		level TEXT NOT NULL,
		module TEXT NOT NULL,
		message TEXT NOT NULL,
		data TEXT,
		trace_id TEXT
	)`)
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS log_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		module TEXT UNIQUE NOT NULL,
		level TEXT NOT NULL DEFAULT 'WARN'
	)`)

	// Migrate round_snapshots table (add status column if missing)
	srv.DB.Exec("ALTER TABLE round_snapshots ADD COLUMN status TEXT NOT NULL DEFAULT 'draft'")
	srv.DB.Exec("ALTER TABLE round_snapshot_scores ADD COLUMN dnf INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE round_snapshot_scores ADD COLUMN dns INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE round_snapshot_scores ADD COLUMN spins INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE racer_stats ADD COLUMN spins INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE round_snapshot_scores ADD COLUMN overheated INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE racer_stats ADD COLUMN overheated INTEGER NOT NULL DEFAULT 0")

	// Migrate tracks table: ensure columns from the Ent schema exist
	srv.DB.Exec("ALTER TABLE tracks ADD COLUMN use_map_image INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE tracks ADD COLUMN map_image_url TEXT NOT NULL DEFAULT ''")
	srv.DB.Exec("ALTER TABLE tracks ADD COLUMN refresh_geojson INTEGER NOT NULL DEFAULT 1")

	// Migrate race_info table: next race event date ('' = unset)
	srv.DB.Exec("ALTER TABLE race_info ADD COLUMN next_race_date TEXT NOT NULL DEFAULT ''")

	// Extension/module catalog tables (module-extension-tracker)
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS extensions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		is_base INTEGER NOT NULL DEFAULT 0,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS modules (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		extension_id INTEGER NOT NULL DEFAULT 0,
		sort_order INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`)
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS season_modules (
		season_id INTEGER NOT NULL,
		module_id INTEGER NOT NULL,
		PRIMARY KEY (season_id, module_id)
	)`)

	// Migrate content tables: attribute tracks/upgrades/legends to an extension
	srv.DB.Exec("ALTER TABLE tracks ADD COLUMN extension_id INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE upgrade_cards ADD COLUMN extension_id INTEGER NOT NULL DEFAULT 0")
	srv.DB.Exec("ALTER TABLE legend_abilities ADD COLUMN extension_id INTEGER NOT NULL DEFAULT 0")

	// Migrate tracks table: module attribution (0 = not module-specific)
	srv.DB.Exec("ALTER TABLE tracks ADD COLUMN module_id INTEGER NOT NULL DEFAULT 0")

	// Board game track list: the curated set of tracks the group plays with
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS board_game_tracks (
		track_id TEXT PRIMARY KEY
	)`)

	// Owned extensions: the set of extension packs the group owns. The Base
	// Game is always owned (seeded below and enforced by the APIs).
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS owned_extensions (
		extension_id INTEGER PRIMARY KEY
	)`)

	// Performance indexes for common query patterns
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_results_racer_id ON race_results(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_results_race_id ON race_results(race_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_history_race_type ON race_history(race_type)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_history_race_date ON race_history(race_date)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_history_type_date ON race_history(race_type, race_date)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_lap_records_racer_id ON lap_records(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_lap_records_race_id ON lap_records(race_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_events_race_id ON race_events(race_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racer_sectors_race_id ON racer_sectors(race_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racers_rank ON racers(rank)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racers_name ON racers(name)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racers_points ON racers(points)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racers_team_id ON racers(team_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_teams_name ON teams(name)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_heat_cards_racer_id ON heat_cards(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_gear_shifts_racer_id ON gear_shifts(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_player_sessions_racer_id ON player_sessions(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_round_snapshots_season_id ON round_snapshots(season_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_racerstats_racer_id ON racer_stats(racer_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_race_results_racer_race ON race_results(racer_id, race_id)")
	srv.DB.Exec("CREATE INDEX IF NOT EXISTS idx_lap_records_racer_lap ON lap_records(racer_id, lap_number)")

	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM racers").Scan(&count)
	if count == 0 {
		SeedData()
	}
	SeedTracks()
	SeedQuotes()
	SeedTeams()
	SeedSeason()
	SeedNotificationSettings()
	SeedAISettings()
	SeedEmailSettings()
	SeedUmamiSettings()
	SeedBackupSettings()
	SeedUpgrades()
	SeedLegendAbilities()
	SeedExtensions()
	// Owned extensions: the Base Game is always owned (idempotent)
	srv.DB.Exec("INSERT OR IGNORE INTO owned_extensions (extension_id) SELECT id FROM extensions WHERE is_base = 1")
	backfillBaseGameContent()
	SeedModules()
	SeedBoardGameTracks()
	SeedSectors()
	SeedLogSettings()
	SeedOTelSettings()
}
