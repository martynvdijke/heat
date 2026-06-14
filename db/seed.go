package db

import (
	"heat/models"
)

func trackGeoJSON() map[string]string {
	return map[string]string{
		"monza":       `{"type":"Feature","properties":{"id":"it-1922","Location":"Monza","Name":"Autodromo Nazionale Monza"},"geometry":{"type":"LineString","coordinates":[[9.281223,45.618975],[9.281692,45.622832],[9.281905,45.624449],[9.281928,45.624515],[9.281994,45.624553],[9.282076,45.624562],[9.282177,45.624553],[9.282272,45.624548],[9.28236,45.624553],[9.28242,45.624586],[9.282467,45.624633],[9.282479,45.624675],[9.282479,45.624722],[9.282147,45.625604],[9.2821,45.625844],[9.282082,45.626061],[9.282106,45.626297],[9.28223,45.627447],[9.282278,45.627635],[9.282337,45.62781],[9.282426,45.627998],[9.282544,45.628201],[9.28268,45.628399],[9.282881,45.628621],[9.283088,45.628804],[9.283367,45.629012],[9.283669,45.629196],[9.284006,45.629365],[9.284367,45.629511],[9.284775,45.629653],[9.285184,45.629752],[9.285634,45.629841],[9.286143,45.629903],[9.286664,45.629936],[9.288694,45.630058],[9.29118,45.630162],[9.291328,45.630171],[9.291405,45.6302],[9.29147,45.630247],[9.291552,45.63045],[9.291588,45.630506],[9.291647,45.630544],[9.291748,45.630567],[9.292085,45.63061],[9.292452,45.630666],[9.292973,45.630784],[9.295299,45.631331],[9.2955,45.631364],[9.29569,45.631364],[9.295873,45.63134],[9.296003,45.631303],[9.296128,45.631251],[9.296246,45.631194],[9.296394,45.631081],[9.296477,45.630992],[9.296536,45.630883],[9.296566,45.630761],[9.296666,45.629988],[9.296856,45.628668],[9.29685,45.62855],[9.29682,45.628474],[9.296773,45.628408],[9.296996,45.628361],[9.296554,45.628295],[9.295033,45.62772],[9.293796,45.627235],[9.293026,45.626938],[9.292653,45.626787],[9.292233,45.62658],[9.291931,45.626419],[9.291671,45.626259],[9.290321,45.625448],[9.289363,45.62484],[9.287338,45.623596],[9.286119,45.622846],[9.285953,45.622728],[9.285888,45.622653],[9.285876,45.622578],[9.285876,45.622488],[9.285918,45.62229],[9.28593,45.622182],[9.28593,45.622026],[9.285912,45.621941],[9.285864,45.621833],[9.285793,45.621706],[9.285675,45.621578],[9.285515,45.621446],[9.285332,45.621338],[9.285184,45.621258],[9.285107,45.621196],[9.285048,45.621112],[9.284994,45.620956],[9.28487,45.620244],[9.28474,45.619485],[9.283734,45.612679],[9.283692,45.612434],[9.283651,45.61232],[9.283568,45.612203],[9.283455,45.612108],[9.283325,45.612023],[9.283142,45.611939],[9.282941,45.611887],[9.28271,45.611858],[9.282514,45.611868],[9.282331,45.611896],[9.282118,45.611953],[9.281911,45.612019],[9.281757,45.612094],[9.281573,45.612193],[9.281366,45.612349],[9.281224,45.61249],[9.2811,45.612641],[9.280993,45.61282],[9.280893,45.613018],[9.280816,45.613221],[9.28078,45.613395],[9.280739,45.613659],[9.280733,45.613923],[9.280727,45.614173],[9.280709,45.614668],[9.280697,45.615102],[9.280709,45.615573],[9.280786,45.61619],[9.281076,45.618142],[9.281223,45.618975]]}}`,
		"spa":         `{"type":"Feature","properties":{"id":"be-spa","Location":"Spa-Francorchamps","Name":"Circuit de Spa-Francorchamps"},"geometry":{"type":"LineString","coordinates":[[5.971389,50.345833],[5.963056,50.433333],[5.956944,50.441667],[5.943056,50.436111],[5.931944,50.420833],[5.918333,50.408333],[5.909444,50.395],[5.900278,50.370833],[5.903611,50.351944],[5.918889,50.343611],[5.934722,50.343056],[5.948333,50.346667],[5.958611,50.351389],[5.969444,50.345833]]}}`,
		"silverstone": `{"type":"Feature","properties":{"id":"uk-silverstone","Location":"Silverstone","Name":"Silverstone Circuit"},"geometry":{"type":"LineString","coordinates":[[-1.016389,52.073056],[-1.016667,52.078333],[-1.015,52.081667],[-1.008333,52.081389],[-1.001944,52.078333],[-0.996111,52.073611],[-0.988611,52.070278],[-0.980278,52.066667],[-0.9725,52.066667],[-0.968611,52.069722],[-0.971389,52.073611],[-0.984722,52.076389],[-0.996944,52.074722],[-1.007222,52.072778],[-1.016389,52.073056]]}}`,
		"monaco":      `{"type":"Feature","properties":{"id":"mc-monaco","Location":"Monaco","Name":"Circuit de Monaco"},"geometry":{"type":"LineString","coordinates":[[7.421667,43.734722],[7.424722,43.736667],[7.427778,43.738056],[7.431111,43.738611],[7.433889,43.737222],[7.435278,43.735278],[7.434444,43.733333],[7.428333,43.729722],[7.421389,43.7275],[7.418611,43.729167],[7.417778,43.731667],[7.419722,43.733889],[7.421667,43.734722]]}}`,
		"interlagos":  `{"type":"Feature","properties":{"id":"br-interlagos","Location":"São Paulo","Name":"Autódromo José Carlos Pace"},"geometry":{"type":"LineString","coordinates":[[-46.699722,-23.701667],[-46.701389,-23.703333],[-46.703889,-23.705278],[-46.708333,-23.706389],[-46.711944,-23.705278],[-46.713611,-23.702778],[-46.712778,-23.700556],[-46.709444,-23.698333],[-46.703889,-23.697778],[-46.700833,-23.699167],[-46.699444,-23.701111],[-46.699722,-23.701667]]}}`,
	}
}

func SeedTracks() {
	geojson := trackGeoJSON()
	tracks := []models.Track{
		{ID: "monza", Name: "Monza", Country: "Italy", GeoJSON: geojson["monza"], Length: 5, LapRecord: "1:18.887"},
		{ID: "spa", Name: "Spa-Francorchamps", Country: "Belgium", GeoJSON: geojson["spa"], Length: 7, LapRecord: "1:42.513"},
		{ID: "silverstone", Name: "Silverstone", Country: "UK", GeoJSON: geojson["silverstone"], Length: 5, LapRecord: "1:24.303"},
		{ID: "monaco", Name: "Monaco", Country: "Monaco", GeoJSON: geojson["monaco"], Length: 3, LapRecord: "1:10.166"},
		{ID: "interlagos", Name: "Interlagos", Country: "Brazil", GeoJSON: geojson["interlagos"], Length: 4, LapRecord: "1:07.369"},
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
