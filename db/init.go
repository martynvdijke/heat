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
	srv.DB.Exec(`CREATE TABLE IF NOT EXISTS log_settings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		module TEXT UNIQUE NOT NULL,
		level TEXT NOT NULL DEFAULT 'WARN'
	)`)

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
	SeedLogSettings()
	SeedOTelSettings()
}
