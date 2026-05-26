package db

import (
	"heat/models"
)

func SeedTracks() {
	tracks := []models.Track{
		{ID: "monza", Name: "Monza", Country: "Italy", GeoJSON: "monza", Length: 5, LapRecord: "1:18.887"},
		{ID: "spa", Name: "Spa-Francorchamps", Country: "Belgium", GeoJSON: "spa", Length: 7, LapRecord: "1:42.513"},
		{ID: "silverstone", Name: "Silverstone", Country: "UK", GeoJSON: "silverstone", Length: 5, LapRecord: "1:24.303"},
		{ID: "monaco", Name: "Monaco", Country: "Monaco", GeoJSON: "monaco", Length: 3, LapRecord: "1:10.166"},
		{ID: "interlagos", Name: "Interlagos", Country: "Brazil", GeoJSON: "interlagos", Length: 4, LapRecord: "1:07.369"},
	}
	for _, t := range tracks {
		srv.DB.Exec("INSERT OR IGNORE INTO tracks (id, name, country, geojson, length_km, lap_record) VALUES (?, ?, ?, ?, ?, ?)",
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
		srv.DB.Exec("INSERT OR IGNORE INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
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
		srv.DB.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank) VALUES (?, ?, ?, ?, ?, ?)",
			r.Name, r.ProfilePicture, r.CarColor, r.CarName, r.Points, r.Rank)
	}

	srv.DB.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		"Italy", "Monza", "monza", 53)
}

func SeedTeams() {
	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM teams").Scan(&count)
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
		srv.DB.Exec("INSERT INTO teams (name, color) VALUES (?, ?)", t.Name, t.Color)
	}
}

func SeedSeason() {
	var count int
	srv.DB.QueryRow("SELECT COUNT(*) FROM seasons").Scan(&count)
	if count == 0 {
		srv.DB.Exec("INSERT INTO seasons (id, name, start_date, status) VALUES (1, 'Season 1', date('now'), 'active')")
	}
}
