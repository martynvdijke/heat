package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"heat/app"
	"heat/models"
)

var currentSchemaVersion = 20

func Init() {
	_, _ = app.DB.Exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)")

	var version int
	err := app.DB.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err != nil {
		version = 0
		app.DB.Exec("INSERT INTO schema_version (version) VALUES (0)")
	}

	log.Printf("[DB] Current schema version: %d, target: %d", version, currentSchemaVersion)

	for version < currentSchemaVersion {
		runMigration(version)
		version++
		app.DB.Exec("DELETE FROM schema_version")
		app.DB.Exec("INSERT INTO schema_version (version) VALUES (?)", version)
		log.Printf("[DB] Migrated to schema version %d", version)
	}

	createRacersTable := `
	CREATE TABLE IF NOT EXISTS racers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		profile_picture TEXT,
		car_color TEXT,
		car_name TEXT,
		points INTEGER,
		rank INTEGER,
		position INTEGER DEFAULT 0
	);`

	createRaceInfoTable := `
	CREATE TABLE IF NOT EXISTS race_info (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		country TEXT,
		track TEXT,
		track_id TEXT,
		laps INTEGER
	);`

	createAdminTable := `
	CREATE TABLE IF NOT EXISTS admin_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT
	);`

	_, err = app.DB.Exec(createRacersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = app.DB.Exec(createRaceInfoTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = app.DB.Exec(createAdminTable)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM racers").Scan(&count)
	if count == 0 {
		SeedData()
	}
	SeedUpgrades()
	SeedLegendAbilities()
	SeedSectors()
}

func runMigration(fromVersion int) {
	switch fromVersion {
	case 0:
		_, _ = app.DB.Exec("ALTER TABLE racers ADD COLUMN position INTEGER DEFAULT 0")
		_, _ = app.DB.Exec("ALTER TABLE race_info ADD COLUMN track_id TEXT DEFAULT 'monza'")
	case 1:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS race_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			race_date TEXT,
			country TEXT,
			track TEXT,
			track_id TEXT,
			total_laps INTEGER
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS race_results (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id INTEGER,
			racer_id INTEGER,
			racer_name TEXT,
			position INTEGER,
			points INTEGER,
			fastest_lap INTEGER DEFAULT 0,
			finished INTEGER DEFAULT 1,
			FOREIGN KEY (race_id) REFERENCES race_history(id),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS racer_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER UNIQUE,
			races INTEGER DEFAULT 0,
			wins INTEGER DEFAULT 0,
			podiums INTEGER DEFAULT 0,
			gold INTEGER DEFAULT 0,
			silver INTEGER DEFAULT 0,
			bronze INTEGER DEFAULT 0,
			fastest_laps INTEGER DEFAULT 0,
			dnf INTEGER DEFAULT 0,
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS tracks (
			id TEXT PRIMARY KEY,
			name TEXT,
			country TEXT,
			geojson TEXT,
			length_km INTEGER,
			lap_record TEXT
		)`)
		SeedTracks()
	case 2:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			author TEXT DEFAULT 'Commentator',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`)
		SeedQuotes()
	case 3:
		_, err := app.DB.Exec("SELECT name FROM race_history LIMIT 0")
		if err != nil {
			_, _ = app.DB.Exec("ALTER TABLE race_history ADD COLUMN name TEXT")
		}
	case 4:
		_, _ = app.DB.Exec("ALTER TABLE tracks ADD COLUMN use_map_image INTEGER DEFAULT 0")
		_, _ = app.DB.Exec("ALTER TABLE tracks ADD COLUMN map_image_url TEXT")
		_, _ = app.DB.Exec("ALTER TABLE tracks ADD COLUMN refresh_geojson INTEGER DEFAULT 1")
	case 5:
		_, _ = app.DB.Exec(`ALTER TABLE race_history ADD COLUMN race_type TEXT DEFAULT 'season'`)
	case 6:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS notification_settings (
			id INTEGER PRIMARY KEY,
			gotify_url TEXT,
			gotify_token TEXT,
			notify_winner INTEGER DEFAULT 1,
			notify_race_start INTEGER DEFAULT 0,
			notify_podium INTEGER DEFAULT 0
		)`)
		_, _ = app.DB.Exec("INSERT INTO notification_settings (id, notify_winner, notify_race_start, notify_podium) VALUES (1, 1, 0, 0)")
	case 7:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT UNIQUE,
			ext TEXT,
			url TEXT,
			resized_url TEXT,
			thumbnail_url TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`)
	case 8:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS ai_settings (
			id INTEGER PRIMARY KEY,
			track_extract_url TEXT,
			api_key TEXT,
			enabled INTEGER DEFAULT 0
		)`)
		_, _ = app.DB.Exec("INSERT INTO ai_settings (id, enabled) VALUES (1, 0)")
	case 9:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS email_settings (
			id INTEGER PRIMARY KEY,
			smtp_host TEXT,
			smtp_port INTEGER DEFAULT 587,
			username TEXT,
			password TEXT,
			from_addr TEXT,
			enabled INTEGER DEFAULT 0
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS racer_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER UNIQUE,
			email TEXT,
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec("INSERT OR IGNORE INTO email_settings (id, enabled) VALUES (1, 0)")
	case 10:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS umami_settings (
			id INTEGER PRIMARY KEY,
			url TEXT,
			website_id TEXT,
			enabled INTEGER DEFAULT 0
		)`)
		_, _ = app.DB.Exec("INSERT OR IGNORE INTO umami_settings (id, enabled) VALUES (1, 0)")
	case 11:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS backup_settings (
			id INTEGER PRIMARY KEY,
			enabled INTEGER DEFAULT 1,
			interval_hrs INTEGER DEFAULT 24
		)`)
		_, _ = app.DB.Exec("INSERT OR IGNORE INTO backup_settings (id, enabled, interval_hrs) VALUES (1, 1, 24)")
	case 12:
		_, _ = app.DB.Exec("ALTER TABLE racer_stats ADD COLUMN dns INTEGER DEFAULT 0")
	case 13:
		_, _ = app.DB.Exec("ALTER TABLE racer_stats ADD COLUMN gold INTEGER DEFAULT 0")
		_, _ = app.DB.Exec("ALTER TABLE racer_stats ADD COLUMN silver INTEGER DEFAULT 0")
		_, _ = app.DB.Exec("ALTER TABLE racer_stats ADD COLUMN bronze INTEGER DEFAULT 0")
	case 14:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS round_snapshots (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_name TEXT NOT NULL,
			race_date TEXT NOT NULL,
			round INTEGER DEFAULT 1,
			created_at TEXT DEFAULT (datetime('now'))
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS round_snapshot_scores (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			snapshot_id INTEGER NOT NULL,
			racer_id INTEGER NOT NULL,
			racer_name TEXT NOT NULL,
			points INTEGER DEFAULT 0,
			position INTEGER DEFAULT 0,
			FOREIGN KEY (snapshot_id) REFERENCES round_snapshots(id),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
	case 15:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT,
			status TEXT DEFAULT 'active',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
		_, _ = app.DB.Exec("INSERT OR IGNORE INTO seasons (id, name, start_date, status) VALUES (1, 'Season 1', date('now'), 'active')")
		_, _ = app.DB.Exec("ALTER TABLE round_snapshots ADD COLUMN season_id INTEGER DEFAULT 1 REFERENCES seasons(id)")
	case 16:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS seasons (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			start_date TEXT NOT NULL,
			end_date TEXT,
			status TEXT DEFAULT 'active',
			created_at TEXT DEFAULT (datetime('now'))
		)`)
		_, _ = app.DB.Exec("INSERT OR IGNORE INTO seasons (id, name, start_date, status) VALUES (1, 'Season 1', date('now'), 'active')")
		_, _ = app.DB.Exec("ALTER TABLE round_snapshots ADD COLUMN season_id INTEGER DEFAULT 1 REFERENCES seasons(id)")
	case 17:
		_, _ = app.DB.Exec("ALTER TABLE backup_settings ADD COLUMN retention_count INTEGER DEFAULT 7")
	case 18:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS heat_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			location TEXT NOT NULL DEFAULT 'hand',
			card_type TEXT NOT NULL DEFAULT 'heat',
			lap_added INTEGER DEFAULT 0,
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS gear_shifts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			race_id INTEGER NOT NULL,
			lap INTEGER NOT NULL DEFAULT 1,
			gear INTEGER NOT NULL DEFAULT 1,
			stress INTEGER DEFAULT 0,
			FOREIGN KEY (racer_id) REFERENCES racers(id),
			FOREIGN KEY (race_id) REFERENCES race_history(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS upgrade_cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			card_type TEXT NOT NULL DEFAULT 'upgrade',
			cost INTEGER DEFAULT 0,
			effects TEXT DEFAULT '{}'
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS player_upgrades (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			upgrade_id INTEGER NOT NULL,
			season_id INTEGER DEFAULT 0,
			equipped INTEGER DEFAULT 1,
			round_bought INTEGER DEFAULT 0,
			FOREIGN KEY (racer_id) REFERENCES racers(id),
			FOREIGN KEY (upgrade_id) REFERENCES upgrade_cards(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS legend_abilities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			ability_type TEXT NOT NULL,
			racer_name TEXT NOT NULL
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS racer_legend_abilities (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			ability_id INTEGER NOT NULL,
			active INTEGER DEFAULT 1,
			FOREIGN KEY (racer_id) REFERENCES racers(id),
			FOREIGN KEY (ability_id) REFERENCES legend_abilities(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS player_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL UNIQUE,
			token TEXT NOT NULL UNIQUE,
			device_name TEXT DEFAULT '',
			last_seen TEXT DEFAULT (datetime('now')),
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS weather_conditions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id INTEGER DEFAULT 0,
			condition TEXT NOT NULL DEFAULT 'dry',
			lap_start INTEGER DEFAULT 1,
			lap_end INTEGER DEFAULT 999,
			grip_modifier REAL DEFAULT 1.0
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS turbo_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			race_id INTEGER DEFAULT 0,
			lap INTEGER DEFAULT 1,
			times_used INTEGER DEFAULT 1,
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS lap_records (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id INTEGER DEFAULT 0,
			racer_id INTEGER NOT NULL,
			lap_number INTEGER NOT NULL,
			position INTEGER NOT NULL,
			gear_used INTEGER DEFAULT 1,
			heat_generated INTEGER DEFAULT 0,
			turbo_used INTEGER DEFAULT 0,
			timestamp TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS sectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			track_id TEXT NOT NULL,
			"order" INTEGER DEFAULT 0,
			FOREIGN KEY (track_id) REFERENCES tracks(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS racer_sectors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id INTEGER DEFAULT 0,
			racer_id INTEGER NOT NULL,
			sector_id INTEGER NOT NULL,
			lap INTEGER DEFAULT 1,
			entry_time TEXT DEFAULT (datetime('now')),
			exit_time TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (racer_id) REFERENCES racers(id),
			FOREIGN KEY (sector_id) REFERENCES sectors(id)
		)`)
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS race_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_id INTEGER DEFAULT 0,
			lap INTEGER DEFAULT 1,
			event_type TEXT NOT NULL,
			racer_id INTEGER NOT NULL,
			racer_id2 INTEGER DEFAULT 0,
			note TEXT DEFAULT '',
			timestamp TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
	case 19:
		_, _ = app.DB.Exec(`CREATE TABLE IF NOT EXISTS driver_shares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL UNIQUE,
			token TEXT NOT NULL UNIQUE,
			created_at TEXT DEFAULT (datetime('now')),
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
	}
}

func SeedTracks() {
	tracks := []models.Track{
		{ID: "monza", Name: "Monza", Country: "Italy", GeoJSON: "monza", Length: 5, LapRecord: "1:18.887"},
		{ID: "spa", Name: "Spa-Francorchamps", Country: "Belgium", GeoJSON: "spa", Length: 7, LapRecord: "1:42.513"},
		{ID: "silverstone", Name: "Silverstone", Country: "UK", GeoJSON: "silverstone", Length: 5, LapRecord: "1:24.303"},
		{ID: "monaco", Name: "Monaco", Country: "Monaco", GeoJSON: "monaco", Length: 3, LapRecord: "1:10.166"},
		{ID: "interlagos", Name: "Interlagos", Country: "Brazil", GeoJSON: "interlagos", Length: 4, LapRecord: "1:07.369"},
	}
	for _, t := range tracks {
		app.DB.Exec("INSERT OR IGNORE INTO tracks (id, name, country, geojson, length_km, lap_record) VALUES (?, ?, ?, ?, ?, ?)",
			t.ID, t.Name, t.Country, t.GeoJSON, t.Length, t.LapRecord)
	}
}

func SeedQuotes() {
	quotes := []struct{ Text, Author string }{
		{"AND THERE'S THE CHEQUERED FLAG! What a race this has been!", "Murray Walker"},
		{"The drama, the tension, the sheer exhilaration of Formula 1!", "James Allen"},
		{"They're on the final lap! This is what racing is all about!", "Martin Brundle"},
		{"PURE ADRENALINE! These drivers are pushing to the absolute limit!", "David Coulthard"},
		{"Unbelievable! This is why we love motorsport!", "Steve Rider"},
		{"The speed on that corner is just OUT OF THIS WORLD!", "Murray Walker"},
		{"Heart-stopping stuff from start to finish!", "James Allen"},
		{"The roar of those engines... music to any racing fan's ears!", "Martin Brundle"},
		{"This is edge-of-your-seat racing at its finest!", "David Coulthard"},
		{"The championship battle intensifies with every single lap!", "Steve Rider"},
		{"And he's DONE IT! What an incredible overtake!", "Murray Walker"},
		{"The pit lane strategy has been absolutely flawless today!", "Martin Brundle"},
		{"You cannot write scripts like this in Formula 1!", "James Allen"},
		{"The telemetry shows just how close these margins are!", "David Coulthard"},
		{"A masterclass in defensive driving!", "Steve Rider"},
		{"The crowd is on their feet! Can he hold on?", "Murray Walker"},
		{"That last lap was simply MIND-BLOWING!", "Martin Brundle"},
		{"Racing at its rawest, most emotional best!", "James Allen"},
		{"These machines are incredible feats of engineering!", "David Coulthard"},
		{"And the fans... the fans have been absolutely MAGNIFICENT!", "Steve Rider"},
	}
	for _, q := range quotes {
		app.DB.Exec("INSERT OR IGNORE INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
	}
}

func SeedData() {
	racers := []models.Racer{
		{Name: "A. PROST", ProfilePicture: "/static/images/helmet.svg", CarColor: "red", CarName: "Red Beast", Points: 78, Rank: 1},
		{Name: "M. SCHUMACHER", ProfilePicture: "/static/images/helmet.svg", CarColor: "blue", CarName: "Blue Bolt", Points: 62, Rank: 2},
		{Name: "A. SENNA", ProfilePicture: "/static/images/helmet.svg", CarColor: "green", CarName: "Green Machine", Points: 85, Rank: 3},
		{Name: "N. LAUDA", ProfilePicture: "/static/images/helmet.svg", CarColor: "yellow", CarName: "Yellow Flash", Points: 45, Rank: 4},
		{Name: "J. STEWART", ProfilePicture: "/static/images/helmet.svg", CarColor: "grey", CarName: "Grey Ghost", Points: 38, Rank: 5},
	}

	for _, r := range racers {
		app.DB.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank) VALUES (?, ?, ?, ?, ?, ?)",
			r.Name, r.ProfilePicture, r.CarColor, r.CarName, r.Points, r.Rank)
	}

	app.DB.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		"Italy", "Monza", "monza", 53)
}

func CreateBackup() error {
	backupDir := filepath.Join(filepath.Dir(app.DBPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, "heat_backup_"+time.Now().Format("20060102_150405")+".db")
	_, err := app.DB.Exec("VACUUM INTO ?", backupPath)
	if err != nil {
		_, err = app.DB.Exec(fmt.Sprintf("VACUUM INTO %q", backupPath))
	}
	return err
}

func PruneBackups() error {
	var retentionCount int
	err := app.DB.QueryRow("SELECT retention_count FROM backup_settings WHERE id = 1").Scan(&retentionCount)
	if err != nil || retentionCount <= 0 {
		retentionCount = 7
	}

	backupDir := filepath.Join(filepath.Dir(app.DBPath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "heat_backup_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	if len(backups) <= retentionCount {
		return nil
	}

	for _, name := range backups[retentionCount:] {
		os.Remove(filepath.Join(backupDir, name))
	}
	return nil
}

func SeedUpgrades() {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM upgrade_cards").Scan(&count)
	if count > 0 {
		return
	}
	upgrades := []models.UpgradeCard{
		{Name: "Racing Gearbox", Description: "Shift one gear higher without adding heat", CardType: "upgrade", Cost: 3, Effects: "{\"gear_bonus\":1}"},
		{Name: "Lightweight Chassis", Description: "Reduce heat generated by 1 per corner", CardType: "upgrade", Cost: 4, Effects: "{\"heat_reduction\":1}"},
		{Name: "High-RPM Engine", Description: "+1 speed on straights", CardType: "upgrade", Cost: 5, Effects: "{\"straight_speed\":1}"},
		{Name: "Cooling System", Description: "Discard 1 extra heat card per cooldown", CardType: "upgrade", Cost: 3, Effects: "{\"cooldown_bonus\":1}"},
		{Name: "Reinforced Brakes", Description: "Reduce stress from braking by 1", CardType: "upgrade", Cost: 2, Effects: "{\"brake_stress_reduction\":1}"},
		{Name: "Aerodynamic Kit", Description: "Better cornering, add +1 to corner speed", CardType: "upgrade", Cost: 4, Effects: "{\"corner_speed\":1}"},
		{Name: "Quick Shifter", Description: "Once per race, shift without adding stress", CardType: "upgrade", Cost: 3, Effects: "{\"free_shift\":1}"},
		{Name: "Turbo Charger", Description: "Extra turbo boost per race", CardType: "upgrade", Cost: 5, Effects: "{\"extra_turbo\":1}"},
	}
	for _, u := range upgrades {
		app.DB.Exec("INSERT INTO upgrade_cards (name, description, card_type, cost, effects) VALUES (?, ?, ?, ?, ?)",
			u.Name, u.Description, u.CardType, u.Cost, u.Effects)
	}
	log.Printf("[DB] Seeded %d upgrade cards", len(upgrades))
}

func SeedLegendAbilities() {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM legend_abilities").Scan(&count)
	if count > 0 {
		return
	}
	abilities := []models.LegendAbility{
		{Name: "Perfect Start", Description: "Gain +1 position on the first lap", AbilityType: "start", RacerName: "A. PROST"},
		{Name: "Rain Master", Description: "No speed penalty in wet conditions", AbilityType: "weather", RacerName: "M. SCHUMACHER"},
		{Name: "Aggressive Overtake", Description: "+1 overtake attempt per race", AbilityType: "overtake", RacerName: "A. SENNA"},
		{Name: "Consistent Performer", Description: "Reduce stress accumulation by 1 each lap", AbilityType: "consistency", RacerName: "N. LAUDA"},
		{Name: "Smooth Operator", Description: "Generate 1 less heat per gear shift", AbilityType: "smoothness", RacerName: "J. STEWART"},
	}
	for _, a := range abilities {
		app.DB.Exec("INSERT INTO legend_abilities (name, description, ability_type, racer_name) VALUES (?, ?, ?, ?)",
			a.Name, a.Description, a.AbilityType, a.RacerName)
	}
	log.Printf("[DB] Seeded %d legend abilities", len(abilities))
}

func SeedSectors() {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM sectors").Scan(&count)
	if count > 0 {
		return
	}
	trackSectors := map[string][]string{
		"monza":       {"Start/Finish", "Lesmo 1", "Lesmo 2", "Ascari", "Parabolica"},
		"spa":         {"Start/Finish", "Eau Rouge", "Raidillon", "Les Combes", "Pouhon", "Stavelot", "Blanchimont"},
		"silverstone": {"Start/Finish", "Copse", "Maggots-Becketts", "Stowe", "Village", "Club"},
		"monaco":      {"Start/Finish", "Casino", "Mirabeau", "Loews", "Portier", "Tunnel", "Nouvelle", "Rascasse"},
		"interlagos":  {"Start/Finish", "Senna S", "Descida", "Curva do Lago", "Ferradura", "Juncao"},
	}
	for trackID, sectors := range trackSectors {
		for i, name := range sectors {
			app.DB.Exec("INSERT INTO sectors (name, track_id, \"order\") VALUES (?, ?, ?)",
				name, trackID, i+1)
		}
	}
	log.Printf("[DB] Seeded sectors")
}

func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func EscapeHTML(s string) string {
	s = fmt.Sprint(s)
	// basic HTML escaping
	var result string
	for _, c := range s {
		switch c {
		case '&':
			result += "&amp;"
		case '<':
			result += "&lt;"
		case '>':
			result += "&gt;"
		case '"':
			result += "&quot;"
		default:
			result += string(c)
		}
	}
	return result
}
