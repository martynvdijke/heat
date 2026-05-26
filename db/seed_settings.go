package db

func SeedNotificationSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO notification_settings (id, notify_winner, notify_race_start, notify_podium) VALUES (1, 1, 0, 0)")
}

func SeedAISettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO ai_settings (id, enabled) VALUES (1, 0)")
}

func SeedEmailSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO email_settings (id, enabled) VALUES (1, 0)")
}

func SeedUmamiSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO umami_settings (id, enabled) VALUES (1, 0)")
}

func SeedBackupSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO backup_settings (id, enabled, interval_hrs) VALUES (1, 1, 24)")
}
