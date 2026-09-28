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

func SeedOTelSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO otel_settings (id, endpoint, traces_enabled, metrics_enabled, logs_enabled) VALUES (1, '', 0, 0, 0)")
}

func SeedWLEDSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO wled_settings (id, enabled) VALUES (1, 0)")
}

func SeedTelegramSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO telegram_settings (id, enabled, notify_results, notify_next_race, reminder_days, reminder_hour) VALUES (1, 0, 1, 1, '7,1', 18)")
}

func SeedLogSettings() {
	srv.DB.Exec("INSERT OR IGNORE INTO log_settings (id, module, level) VALUES (1, 'default', 'WARN')")
}
