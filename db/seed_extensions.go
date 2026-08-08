package db

import "log"

// SeedExtensions inserts the known expansion packs. Idempotent: skips when
// the extensions table already has rows. Uses fixed ids so content seeders
// can link to the Base Game extension deterministically.
func SeedExtensions() {
	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM extensions").Scan(&count)
	if count > 0 {
		return
	}
	srv.DB.Exec("INSERT INTO extensions (id, name, description, is_base, sort_order) VALUES (1, 'Base Game', 'The core Heat: Pedal to the Metal game.', 1, 1)")
	srv.DB.Exec("INSERT INTO extensions (id, name, description, is_base, sort_order) VALUES (2, 'Heavy Rain', 'The Heavy Rain expansion for Heat: Pedal to the Metal.', 0, 2)")
	log.Printf("[DB] Seeded extensions")
}

// backfillBaseGameContent normalizes any content with extension_id = 0 to the
// Base Game extension (id 1). Runs on every startup: on pre-existing databases
// the ALTER TABLE backfill leaves existing rows at 0, and on fresh databases
// seeders write 1 explicitly, so this is idempotent either way.
func backfillBaseGameContent() {
	srv.DB.Exec("UPDATE tracks SET extension_id = 1 WHERE extension_id = 0")
	srv.DB.Exec("UPDATE upgrade_cards SET extension_id = 1 WHERE extension_id = 0")
	srv.DB.Exec("UPDATE legend_abilities SET extension_id = 1 WHERE extension_id = 0")
}

// SeedModules inserts the starter gameplay modules. Idempotent: skips when
// the modules table already has rows. Base modules are owned by the Base
// Game extension (id 1); expansion modules by their extension (id 2).
func SeedModules() {
	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM modules").Scan(&count)
	if count > 0 {
		return
	}
	modules := []struct {
		Name, Description string
		ExtensionID       int
		SortOrder         int
	}{
		{"Championship", "Full championship with multiple races and points", 1, 1},
		{"Legend Drivers", "Legend drivers with special abilities", 1, 2},
		{"Weather", "Dynamic weather conditions affecting grip", 1, 3},
		{"Upgrades", "Upgrade cards bought between races", 1, 4},
		{"Turbo", "Turbo boost tokens for extra speed", 1, 5},
		{"Extreme Weather", "Severe weather conditions from the Heavy Rain expansion", 2, 6},
	}
	for _, m := range modules {
		srv.DB.Exec("INSERT INTO modules (name, description, extension_id, sort_order) VALUES (?, ?, ?, ?)",
			m.Name, m.Description, m.ExtensionID, m.SortOrder)
	}
	log.Printf("[DB] Seeded %d modules", len(modules))
}
