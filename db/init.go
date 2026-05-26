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
}
