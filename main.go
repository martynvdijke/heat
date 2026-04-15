package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Racer struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
	CarColor       string `json:"car_color"`
	CarName        string `json:"car_name"`
	Points         int    `json:"points"`
	Gap            string `json:"gap"`
	Rank           int    `json:"rank"`
	FastestLap     string `json:"fastest_lap"`
}

type RaceInfo struct {
	Country     string `json:"country"`
	Track       string `json:"track"`
	Date        string `json:"date"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Temperature int    `json:"temperature"`
	Length      string `json:"length"`
	Laps        int    `json:"laps"`
}

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./heat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	// API Routes
	http.HandleFunc("/api/racers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRacers(w, r)
		case "POST":
			updateRacer(w, r)
		case "DELETE":
			deleteRacer(w, r)
		}
	})

	http.HandleFunc("/api/race-info", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRaceInfo(w, r)
		case "POST":
			updateRaceInfo(w, r)
		}
	})

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	createRacersTable := `
	CREATE TABLE IF NOT EXISTS racers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		profile_picture TEXT,
		car_color TEXT,
		car_name TEXT,
		points INTEGER,
		gap TEXT,
		rank INTEGER,
		fastest_lap TEXT
	);`

	createRaceInfoTable := `
	CREATE TABLE IF NOT EXISTS race_info (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		country TEXT,
		track TEXT,
		date TEXT,
		days INTEGER,
		hours INTEGER,
		temperature INTEGER,
		length TEXT,
		laps INTEGER
	);`

	_, err := db.Exec(createRacersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createRaceInfoTable)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM racers").Scan(&count)
	if count == 0 {
		seedData()
	}
}

func seedData() {
	racers := []Racer{
		{Name: "A. PROST", ProfilePicture: "https://i.pravatar.cc/100?u=prost", CarColor: "red", CarName: "Red Beast", Points: 78, Gap: "INTERVAL", Rank: 1, FastestLap: "1:21.042"},
		{Name: "M. SCHUMACHER", ProfilePicture: "https://i.pravatar.cc/100?u=schu", CarColor: "blue", CarName: "Blue Bolt", Points: 62, Gap: "+1.242s", Rank: 2},
		{Name: "A. SENNA", ProfilePicture: "https://i.pravatar.cc/100?u=senna", CarColor: "green", CarName: "Green Machine", Points: 85, Gap: "+4.501s", Rank: 3},
		{Name: "N. LAUDA", ProfilePicture: "https://i.pravatar.cc/100?u=lauda", CarColor: "yellow", CarName: "Yellow Flash", Points: 45, Gap: "+12.110s", Rank: 4},
		{Name: "J. STEWART", ProfilePicture: "https://i.pravatar.cc/100?u=stew", CarColor: "grey", CarName: "Grey Ghost", Points: 38, Gap: "+1 LAP", Rank: 5},
	}

	for _, r := range racers {
		db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, gap, rank, fastest_lap) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			r.Name, r.ProfilePicture, r.CarColor, r.CarName, r.Points, r.Gap, r.Rank, r.FastestLap)
	}

	db.Exec("INSERT INTO race_info (country, track, date, days, hours, temperature, length, laps) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"ITALY", "Autodromo Nazionale Monza", "SEPTEMBER 01-03, 2026", 4, 12, 24, "5.793 KM", 53)
}

func getRacers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, profile_picture, car_color, car_name, points, gap, rank, fastest_lap FROM racers ORDER BY rank ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var racers []Racer
	for rows.Next() {
		var r Racer
		var fastestLap sql.NullString
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Gap, &r.Rank, &fastestLap)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if fastestLap.Valid {
			r.FastestLap = fastestLap.String
		}
		racers = append(racers, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(racers)
}

func updateRacer(w http.ResponseWriter, r *http.Request) {
	var racer Racer
	if err := json.NewDecoder(r.Body).Decode(&racer); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if racer.ID == 0 {
		_, err := db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, gap, rank, fastest_lap) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Gap, racer.Rank, racer.FastestLap)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err := db.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, gap=?, rank=?, fastest_lap=? WHERE id=?",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Gap, racer.Rank, racer.FastestLap, racer.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func deleteRacer(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)
	_, err := db.Exec("DELETE FROM racers WHERE id=?", id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getRaceInfo(w http.ResponseWriter, r *http.Request) {
	var ri RaceInfo
	err := db.QueryRow("SELECT country, track, date, days, hours, temperature, length, laps FROM race_info ORDER BY id DESC LIMIT 1").
		Scan(&ri.Country, &ri.Track, &ri.Date, &ri.Days, &ri.Hours, &ri.Temperature, &ri.Length, &ri.Laps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ri)
}

func updateRaceInfo(w http.ResponseWriter, r *http.Request) {
	var ri RaceInfo
	if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := db.Exec("INSERT INTO race_info (country, track, date, days, hours, temperature, length, laps) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		ri.Country, ri.Track, ri.Date, ri.Days, ri.Hours, ri.Temperature, ri.Length, ri.Laps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
