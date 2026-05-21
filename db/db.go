package db

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"heat/app"
	"heat/models"
)

func Init() {
	ctx := context.Background()

	if err := app.Ent.Schema.Create(ctx); err != nil {
		log.Fatalf("[DB] Failed to run Ent auto-migration: %v", err)
	}

	app.DB.Exec("PRAGMA foreign_keys=OFF")

	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM racers").Scan(&count)
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

func CreateBackup() error {
	backupDir := filepath.Join(filepath.Dir(app.DBPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, "heat_backup_"+time.Now().Format("20060102_150405")+".db")
	_, err := app.DB.Exec("VACUUM INTO ?", backupPath)
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

func SeedTeams() {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count)
	if count > 0 {
		return
	}
	teams := []struct{ Name, Color string }{
		{"Scuderia Ferrari", "#d40000"},
		{"Scuderia Alfa Romeo", "#900000"},
		{"Team Lotus", "#005500"},
		{"McLaren Racing", "#ff8700"},
		{"Williams Racing", "#005aff"},
	}
	for _, t := range teams {
		app.DB.Exec("INSERT INTO teams (name, color) VALUES (?, ?)", t.Name, t.Color)
	}
	log.Printf("[DB] Seeded %d teams", len(teams))
}

func SeedSeason() {
	var count int
	app.DB.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&count)
	if count == 0 {
		app.DB.Exec("INSERT INTO seasons (id, name, start_date, status) VALUES (1, 'Season 1', date('now'), 'active')")
	}
}

func SeedNotificationSettings() {
	app.DB.Exec("INSERT OR IGNORE INTO notification_settings (id, notify_winner, notify_race_start, notify_podium) VALUES (1, 1, 0, 0)")
}

func SeedAISettings() {
	app.DB.Exec("INSERT OR IGNORE INTO ai_settings (id, enabled) VALUES (1, 0)")
}

func SeedEmailSettings() {
	app.DB.Exec("INSERT OR IGNORE INTO email_settings (id, enabled) VALUES (1, 0)")
}

func SeedUmamiSettings() {
	app.DB.Exec("INSERT OR IGNORE INTO umami_settings (id, enabled) VALUES (1, 0)")
}

func SeedBackupSettings() {
	app.DB.Exec("INSERT OR IGNORE INTO backup_settings (id, enabled, interval_hrs) VALUES (1, 1, 24)")
}

func BoolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func EscapeHTML(s string) string {
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
