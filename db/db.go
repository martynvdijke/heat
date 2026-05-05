package db

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"heat/app"
	"heat/models"
)

var currentSchemaVersion = 12

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
