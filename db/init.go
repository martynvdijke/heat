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

	srv.DB.Exec("PRAGMA foreign_keys=OFF")

	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS app_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL DEFAULT (datetime('now')),
		level TEXT NOT NULL,
		module TEXT NOT NULL,
		message TEXT NOT NULL,
		data TEXT,
		trace_id TEXT
	)`)
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS eink_settings (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		enabled INTEGER NOT NULL DEFAULT 0
	)`)

	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS log_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		module TEXT UNIQUE NOT NULL,
		level TEXT NOT NULL DEFAULT 'WARN'
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
	SeedSectors()
	SeedEInkSettings()
	SeedLogSettings()
	SeedOTelSettings()
}
