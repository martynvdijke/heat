package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

type Racer struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	ProfilePicture string `json:"profile_picture"`
	CarColor       string `json:"car_color"`
	CarName        string `json:"car_name"`
	Points         int    `json:"points"`
	Rank           int    `json:"rank"`
	Position       int    `json:"position"`
}

type RaceInfo struct {
	ID      int    `json:"id"`
	Country string `json:"country"`
	Track   string `json:"track"`
	Laps    int    `json:"laps"`
	TrackID string `json:"track_id"`
}

type Track struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Country   string `json:"country"`
	GeoJSON   string `json:"geojson"`
	Length    int    `json:"length_km"`
	LapRecord string `json:"lap_record"`
}

type RaceResult struct {
	ID         int    `json:"id"`
	RaceID     int    `json:"race_id"`
	RacerID    int    `json:"racer_id"`
	RacerName  string `json:"racer_name"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
	FastestLap bool   `json:"fastest_lap"`
	Finished   bool   `json:"finished"`
}

type RacerStats struct {
	ID          int `json:"id"`
	RacerID     int `json:"racer_id"`
	Races       int `json:"races"`
	Wins        int `json:"wins"`
	Podiums     int `json:"podiums"`
	FastestLaps int `json:"fastest_laps"`
	Points      int `json:"points"`
	DNF         int `json:"dnf"`
}

type Quote struct {
	ID        int    `json:"id"`
	Text      string `json:"text"`
	Author    string `json:"author"`
	CreatedAt string `json:"created_at"`
}

const currentSchemaVersion = 3

type AdminUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

var (
	db           *sql.DB
	sessionStore = make(map[string]int64)
	staticCache  = make(map[string][]byte)
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clients   = make(map[*websocket.Conn]bool)
	broadcast = make(chan []Racer)
	basePath  = "/app"
)

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("error upgrading: %v", err)
		return
	}
	defer ws.Close()

	clients[ws] = true
	log.Printf("[WS] New client connected. Total clients: %d", len(clients))

	for {
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client disconnected: %v", err)
			delete(clients, ws)
			break
		}
	}
}

func broadcastManager() {
	for {
		racers := <-broadcast
		for client := range clients {
			err := client.WriteJSON(racers)
			if err != nil {
				log.Printf("[WS] error broadcasting to client: %v", err)
				client.Close()
				delete(clients, client)
			}
		}
	}
}

func broadcastRacers() {
	rows, err := db.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers ORDER BY rank ASC")
	if err != nil {
		log.Printf("error fetching racers for broadcast: %v", err)
		return
	}
	defer rows.Close()

	var racers []Racer
	for rows.Next() {
		var r Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
		if err != nil {
			log.Printf("error scanning racer for broadcast: %v", err)
			return
		}
		racers = append(racers, r)
	}
	broadcast <- racers
}

func shorten(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}
	return s
}

func main() {
	if os.Getenv("DOCKER") != "true" {
		basePath = "."
	}
	if err := os.MkdirAll(filepath.Join(basePath, "static/images"), 0755); err != nil {
		log.Printf("Warning: could not create images directory: %v", err)
	}

	var err error
	db, err = sql.Open("sqlite3", "./heat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()
	go broadcastManager()

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/logout", handleLogout)
	http.HandleFunc("/api/check-setup", handleCheckSetup)

	http.HandleFunc("/admin.html", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[ADMIN] Access attempt to admin.html")

		// Find the session cookie that exists in our store
		var validSession string
		for _, c := range r.Cookies() {
			if c.Name == "session" {
				if _, ok := sessionStore[c.Value]; ok {
					validSession = c.Value
					break
				}
			}
		}

		if validSession == "" {
			log.Printf("[ADMIN] No valid session, redirecting to login")
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}

		log.Printf("[ADMIN] Session valid: %s, serving admin.html", shorten(validSession))
		http.ServeFile(w, r, filepath.Join(basePath, "static/admin.html"))
	})

	http.HandleFunc("/api/racers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRacers(w, r)
		case "POST":
			authMiddleware(updateRacer)(w, r)
		case "DELETE":
			authMiddleware(deleteRacer)(w, r)
		}
	})

	http.HandleFunc("/api/race-info", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRaceInfo(w, r)
		case "POST":
			authMiddleware(updateRaceInfo)(w, r)
		}
	})

	http.HandleFunc("/api/upload", authMiddleware(handleUpload))

	http.HandleFunc("/api/tracks", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			getTracks(w, r)
		}
	})

	http.HandleFunc("/api/race-history", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRaceHistory(w, r)
		case "POST":
			authMiddleware(saveRaceToHistory)(w, r)
		case "DELETE":
			authMiddleware(deleteRaceHistory)(w, r)
		}
	})

	http.HandleFunc("/api/racer-stats", func(w http.ResponseWriter, r *http.Request) {
		getRacerStats(w, r)
	})

	http.HandleFunc("/api/quotes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getQuotes(w, r)
		case "POST", "PUT", "DELETE":
			authMiddleware(handleQuotes)(w, r)
		}
	})

	http.HandleFunc("/api/quote/random", func(w http.ResponseWriter, r *http.Request) {
		getRandomQuote(w, r)
	})

	http.HandleFunc("/login.html", func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err == nil {
			expiry, ok := sessionStore[cookie.Value]
			if ok && time.Now().Unix() <= expiry {
				http.Redirect(w, r, "/admin.html", http.StatusFound)
				return
			}
		}
		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		if count == 0 {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}
		http.ServeFile(w, r, filepath.Join(basePath, "static/login.html"))
	})

	http.HandleFunc("/setup", func(w http.ResponseWriter, r *http.Request) {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		if count > 0 {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		http.ServeFile(w, r, filepath.Join(basePath, "static/setup.html"))
	})

	fs := http.FileServer(http.Dir(filepath.Join(basePath, "static")))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filepath.Join(basePath, "static/index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initDB() {
	_, _ = db.Exec("CREATE TABLE IF NOT EXISTS schema_version (version INTEGER PRIMARY KEY)")

	var version int
	err := db.QueryRow("SELECT version FROM schema_version LIMIT 1").Scan(&version)
	if err != nil {
		version = 0
		db.Exec("INSERT INTO schema_version (version) VALUES (0)")
	}

	log.Printf("[DB] Current schema version: %d, target: %d", version, currentSchemaVersion)

	for version < currentSchemaVersion {
		runMigration(version)
		version++
		db.Exec("UPDATE schema_version SET version = ?", version)
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

	_, err = db.Exec(createRacersTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createRaceInfoTable)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(createAdminTable)
	if err != nil {
		log.Fatal(err)
	}

	var count int
	db.QueryRow("SELECT COUNT(*) FROM racers").Scan(&count)
	if count == 0 {
		seedData()
	}
}

func runMigration(fromVersion int) {
	switch fromVersion {
	case 0:
		_, _ = db.Exec("ALTER TABLE racers ADD COLUMN position INTEGER DEFAULT 0")
		_, _ = db.Exec("ALTER TABLE race_info ADD COLUMN track_id TEXT DEFAULT 'monza'")
	case 1:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS race_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			race_date TEXT,
			country TEXT,
			track TEXT,
			track_id TEXT,
			total_laps INTEGER
		)`)
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS race_results (
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
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS racer_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER UNIQUE,
			races INTEGER DEFAULT 0,
			wins INTEGER DEFAULT 0,
			podiums INTEGER DEFAULT 0,
			fastest_laps INTEGER DEFAULT 0,
			dnf INTEGER DEFAULT 0,
			FOREIGN KEY (racer_id) REFERENCES racers(id)
		)`)
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS tracks (
			id TEXT PRIMARY KEY,
			name TEXT,
			country TEXT,
			geojson TEXT,
			length_km INTEGER,
			lap_record TEXT
		)		`)
		seedTracks()
	case 2:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			author TEXT DEFAULT 'Commentator',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`)
		seedQuotes()
	}
}

func seedTracks() {
	tracks := []Track{
		{ID: "monza", Name: "Monza", Country: "Italy", GeoJSON: "monza", Length: 5, LapRecord: "1:18.887"},
		{ID: "spa", Name: "Spa-Francorchamps", Country: "Belgium", GeoJSON: "spa", Length: 7, LapRecord: "1:42.513"},
		{ID: "silverstone", Name: "Silverstone", Country: "UK", GeoJSON: "silverstone", Length: 5, LapRecord: "1:24.303"},
		{ID: "monaco", Name: "Monaco", Country: "Monaco", GeoJSON: "monaco", Length: 3, LapRecord: "1:10.166"},
		{ID: "interlagos", Name: "Interlagos", Country: "Brazil", GeoJSON: "interlagos", Length: 4, LapRecord: "1:07.369"},
	}
	for _, t := range tracks {
		db.Exec("INSERT OR IGNORE INTO tracks (id, name, country, geojson, length_km, lap_record) VALUES (?, ?, ?, ?, ?, ?)",
			t.ID, t.Name, t.Country, t.GeoJSON, t.Length, t.LapRecord)
	}
}

func seedQuotes() {
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
		db.Exec("INSERT OR IGNORE INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
	}
}

func seedData() {
	racers := []Racer{
		{Name: "A. PROST", ProfilePicture: "/static/images/prost.png", CarColor: "red", CarName: "Red Beast", Points: 78, Rank: 1},
		{Name: "M. SCHUMACHER", ProfilePicture: "/static/images/schumacher.png", CarColor: "blue", CarName: "Blue Bolt", Points: 62, Rank: 2},
		{Name: "A. SENNA", ProfilePicture: "/static/images/senna.png", CarColor: "green", CarName: "Green Machine", Points: 85, Rank: 3},
		{Name: "N. LAUDA", ProfilePicture: "/static/images/lauda.png", CarColor: "yellow", CarName: "Yellow Flash", Points: 45, Rank: 4},
		{Name: "J. STEWART", ProfilePicture: "/static/images/stewart.png", CarColor: "grey", CarName: "Grey Ghost", Points: 38, Rank: 5},
	}

	for _, r := range racers {
		db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank) VALUES (?, ?, ?, ?, ?, ?)",
			r.Name, r.ProfilePicture, r.CarColor, r.CarName, r.Points, r.Rank)
	}

	db.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		"Italy", "Monza", "monza", 53)
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[AUTH] Checking session for: %s", r.URL.Path)

		// Get all cookies and find the valid session one
		cookies := r.Cookies()
		log.Printf("[AUTH] All cookies: %v", cookies)

		// Find the session cookie that exists in our store
		var sessionCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "session" {
				// Check if this session exists in our store
				if _, ok := sessionStore[c.Value]; ok {
					sessionCookie = c
					log.Printf("[AUTH] Found valid session in store: %s", shorten(c.Value))
					break
				}
			}
		}

		if sessionCookie == nil {
			log.Printf("[AUTH] No valid session cookie found")
			log.Printf("[AUTH] Stored sessions:")
			for k, v := range sessionStore {
				log.Printf("[AUTH]   - %s (expires: %d)", shorten(k), v)
			}
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		log.Printf("[AUTH] Using session: %s", shorten(sessionCookie.Value))

		expiry, ok := sessionStore[sessionCookie.Value]
		if !ok {
			log.Printf("[AUTH] Session not found in store!")
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}
		if time.Now().Unix() > expiry {
			log.Printf("[AUTH] Session expired")
			delete(sessionStore, sessionCookie.Value)
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}

		log.Printf("[AUTH] Session valid, allowing: %s", r.URL.Path)
		next.ServeHTTP(w, r)
	}
}

func handleCheckSetup(w http.ResponseWriter, r *http.Request) {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	json.NewEncoder(w).Encode(map[string]bool{"setup": count > 0})
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Setup    bool   `json:"setup"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			log.Printf("[LOGIN] Failed to decode JSON: %v", err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("[LOGIN] Attempting login for user: %s", input.Username)

		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		log.Printf("[LOGIN] Admin users in DB: %d", count)

		if count == 0 {
			log.Printf("[LOGIN] No admin users, creating new user: %s", input.Username)
			hashed := hashPassword(input.Password)
			log.Printf("[LOGIN] Password hash: %s", hashed)
			_, err := db.Exec("INSERT INTO admin_users (username, password) VALUES (?, ?)", input.Username, hashed)
			if err != nil {
				log.Printf("[LOGIN] Failed to insert user: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			var user AdminUser
			db.QueryRow("SELECT id, username FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username)
			log.Printf("[LOGIN] Created user with ID: %d", user.ID)

			sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", user.ID, user.Username, time.Now().Unix()))))
			sessionStore[sessionID] = time.Now().Add(24 * time.Hour).Unix()
			log.Printf("[LOGIN] Session created: %s", shorten(sessionID))

			cookie := &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"}
			http.SetCookie(w, cookie)
			log.Printf("[LOGIN] Cookie set: session=%s", shorten(sessionID))

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
			return
		}

		log.Printf("[LOGIN] Looking up user: %s", input.Username)
		var user AdminUser
		err := db.QueryRow("SELECT id, username, password FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username, &user.Password)
		if err != nil {
			log.Printf("[LOGIN] User not found: %v", err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		log.Printf("[LOGIN] Found user ID: %d, stored password hash: %s", user.ID, shorten(user.Password))

		inputHash := hashPassword(input.Password)
		log.Printf("[LOGIN] Input password hash: %s", shorten(inputHash))

		if inputHash != user.Password {
			log.Printf("[LOGIN] Password mismatch!")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Printf("[LOGIN] Password verified successfully")
		sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", user.ID, user.Username, time.Now().Unix()))))
		sessionStore[sessionID] = time.Now().Add(24 * time.Hour).Unix()
		log.Printf("[LOGIN] Session created: %s", shorten(sessionID))

		cookie := &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"}
		http.SetCookie(w, cookie)
		log.Printf("[LOGIN] Cookie set: session=%s", shorten(sessionID))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	http.ServeFile(w, r, "/app/static/login.html")
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session")
	if err == nil {
		delete(sessionStore, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: "session", Value: "", MaxAge: -1})
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func getRacers(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers ORDER BY rank ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var racers []Racer
	for rows.Next() {
		var r Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		racers = append(racers, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(racers)
}

func updateRacer(w http.ResponseWriter, r *http.Request) {
	var racer Racer
	if err := json.NewDecoder(r.Body).Decode(&racer); err != nil {
		log.Printf("[RACER] Failed to decode: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[RACER] Updating racer: ID=%d, Name=%s, Picture=%s, Car=%s (%s), Points=%d, Rank=%d, Position=%d",
		racer.ID, racer.Name, racer.ProfilePicture, racer.CarName, racer.CarColor, racer.Points, racer.Rank, racer.Position)

	if racer.ID == 0 {
		_, err := db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank, position) VALUES (?, ?, ?, ?, ?, ?, ?)",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position)
		if err != nil {
			log.Printf("[RACER] Insert failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[RACER] Created new racer")
	} else {
		_, err := db.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, rank=?, position=? WHERE id=?",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position, racer.ID)
		if err != nil {
			log.Printf("[RACER] Update failed: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Printf("[RACER] Updated racer ID=%d", racer.ID)
	}
	broadcastRacers()
	w.WriteHeader(http.StatusOK)
}

func deleteRacer(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, _ := strconv.Atoi(idStr)
	log.Printf("[RACER] Deleting racer ID=%d", id)
	_, err := db.Exec("DELETE FROM racers WHERE id=?", id)
	if err != nil {
		log.Printf("[RACER] Delete failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("[RACER] Deleted racer ID=%d", id)
	broadcastRacers()
	w.WriteHeader(http.StatusOK)
}

func getRaceInfo(w http.ResponseWriter, r *http.Request) {
	var ri RaceInfo
	err := db.QueryRow("SELECT country, track, COALESCE(track_id, 'monza'), laps FROM race_info ORDER BY id DESC LIMIT 1").
		Scan(&ri.Country, &ri.Track, &ri.TrackID, &ri.Laps)
	if err != nil {
		ri = RaceInfo{Country: "Italy", Track: "Monza", TrackID: "monza", Laps: 53}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ri)
}

func updateRaceInfo(w http.ResponseWriter, r *http.Request) {
	var ri RaceInfo
	if err := json.NewDecoder(r.Body).Decode(&ri); err != nil {
		log.Printf("[RACEINFO] Failed to decode: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if ri.TrackID == "" {
		ri.TrackID = "monza"
	}

	_, err := db.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		ri.Country, ri.Track, ri.TrackID, ri.Laps)
	if err != nil {
		log.Printf("[RACEINFO] Insert failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	log.Printf("[UPLOAD] Upload request received")
	if r.Method != "POST" {
		log.Printf("[UPLOAD] Wrong method: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		log.Printf("[UPLOAD] Failed to get form file: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Printf("[UPLOAD] File received: %s, size: %d", header.Filename, header.Size)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		log.Printf("[UPLOAD] Invalid file type: %s", ext)
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("%d%s", time.Now().Unix(), ext)
	uploadPath := filepath.Join(basePath, "static/images", filename)
	log.Printf("[UPLOAD] Saving to: %s", uploadPath)

	out, err := os.Create(uploadPath)
	if err != nil {
		log.Printf("[UPLOAD] Failed to create file: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	data, _ := io.ReadAll(file)
	out.Write(data)

	staticCache["/static/images/"+filename] = data

	log.Printf("[UPLOAD] Success! URL: /static/images/%s", filename)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": "/static/images/" + filename})
}

func getTracks(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name, country, geojson, length_km, lap_record FROM tracks ORDER BY name")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		if err := rows.Scan(&t.ID, &t.Name, &t.Country, &t.GeoJSON, &t.Length, &t.LapRecord); err != nil {
			continue
		}
		tracks = append(tracks, t)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tracks)
}

func getRaceHistory(w http.ResponseWriter, r *http.Request) {
	raceID := r.URL.Query().Get("id")

	var query string
	var args []interface{}

	if raceID != "" {
		query = `SELECT rh.id, rh.race_date, rh.country, rh.track, rh.track_id, rh.total_laps,
				 COALESCE(GROUP_CONCAT(rr.racer_id || ':' || rr.racer_name || ':' || rr.position || ':' || rr.points || ':' || rr.fastest_lap, '|'), '') as results
				 FROM race_history rh
				 LEFT JOIN race_results rr ON rh.id = rr.race_id
				 WHERE rh.id = ?
				 GROUP BY rh.id`
		args = []interface{}{raceID}
	} else {
		query = `SELECT id, race_date, country, track, track_id, total_laps FROM race_history ORDER BY race_date DESC LIMIT 20`
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type HistoryRow struct {
		ID        int          `json:"id"`
		Date      string       `json:"race_date"`
		Country   string       `json:"country"`
		Track     string       `json:"track"`
		TrackID   string       `json:"track_id"`
		TotalLaps int          `json:"total_laps"`
		Results   []RaceResult `json:"results,omitempty"`
	}

	var history []HistoryRow
	for rows.Next() {
		var h HistoryRow
		var resultsStr string
		if raceID != "" {
			rows.Scan(&h.ID, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &resultsStr)
			if resultsStr != "" {
				for _, r := range strings.Split(resultsStr, "|") {
					parts := strings.Split(r, ":")
					if len(parts) >= 5 {
						rid, _ := strconv.Atoi(parts[0])
						pos, _ := strconv.Atoi(parts[2])
						pts, _ := strconv.Atoi(parts[3])
						fl, _ := strconv.Atoi(parts[4])
						h.Results = append(h.Results, RaceResult{
							RacerID:    rid,
							RacerName:  parts[1],
							Position:   pos,
							Points:     pts,
							FastestLap: fl == 1,
						})
					}
				}
			}
		} else {
			rows.Scan(&h.ID, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps)
		}
		history = append(history, h)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func saveRaceToHistory(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RaceDate  string `json:"race_date"`
		Country   string `json:"country"`
		Track     string `json:"track"`
		TrackID   string `json:"track_id"`
		TotalLaps int    `json:"total_laps"`
		Results   []struct {
			RacerID    int    `json:"racer_id"`
			RacerName  string `json:"racer_name"`
			Position   int    `json:"position"`
			Points     int    `json:"points"`
			FastestLap bool   `json:"fastest_lap"`
			Finished   bool   `json:"finished"`
		} `json:"results"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if input.RaceDate == "" {
		input.RaceDate = time.Now().Format("2006-01-02")
	}

	result, err := db.Exec("INSERT INTO race_history (race_date, country, track, track_id, total_laps) VALUES (?, ?, ?, ?, ?)",
		input.RaceDate, input.Country, input.Track, input.TrackID, input.TotalLaps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	raceID, _ := result.LastInsertId()

	for _, res := range input.Results {
		db.Exec("INSERT INTO race_results (race_id, racer_id, racer_name, position, points, fastest_lap) VALUES (?, ?, ?, ?, ?, ?)",
			raceID, res.RacerID, res.RacerName, res.Position, res.Points, boolToInt(res.FastestLap))

		db.Exec(`INSERT INTO racer_stats (racer_id, races, wins, podiums, fastest_laps, dnf) VALUES (?, 1, ?, ?, ?, ?)
				 ON CONFLICT(racer_id) DO UPDATE SET
				 races = races + 1,
				 wins = wins + excluded.wins,
				 podiums = podiums + excluded.podiums,
				 fastest_laps = fastest_laps + excluded.fastest_laps,
				 dnf = dnf + excluded.dnf`,
			res.RacerID, boolToInt(res.Position == 1), boolToInt(res.Position <= 3), boolToInt(res.FastestLap), boolToInt(!res.Finished))
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int64{"id": raceID})
}

func deleteRaceHistory(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "ID required", http.StatusBadRequest)
		return
	}
	db.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	db.Exec("DELETE FROM race_history WHERE id = ?", id)
	w.WriteHeader(http.StatusOK)
}

func getRacerStats(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		var stats []RacerStats
		rows, _ := db.Query("SELECT id, racer_id, races, wins, podiums, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf FROM racer_stats")
		for rows.Next() {
			var s RacerStats
			rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Podiums, &s.FastestLaps, &s.Points, &s.DNF)
			stats = append(stats, s)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
		return
	}

	var s RacerStats
	err := db.QueryRow("SELECT id, racer_id, races, wins, podiums, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf FROM racer_stats WHERE racer_id = ?", id).Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Podiums, &s.FastestLaps, &s.Points, &s.DNF)
	if err != nil {
		s = RacerStats{RacerID: 0, Races: 0, Wins: 0, Podiums: 0, FastestLaps: 0, Points: 0, DNF: 0}
	}

	var rInfo Racer
	db.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", id).Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"stats": s, "racer": rInfo})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func getQuotes(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, text, author, created_at FROM quotes ORDER BY id")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var quotes []Quote
	for rows.Next() {
		var q Quote
		if err := rows.Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt); err != nil {
			continue
		}
		quotes = append(quotes, q)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(quotes)
}

func getRandomQuote(w http.ResponseWriter, r *http.Request) {
	var q Quote
	err := db.QueryRow("SELECT id, text, author, created_at FROM quotes ORDER BY RANDOM() LIMIT 1").Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt)
	if err != nil {
		q = Quote{Text: "The engines roar as these legends battle for glory!", Author: "Commentator"}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(q)
}

func handleQuotes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getQuotes(w, r)
	case "POST":
		var q Quote
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if q.Text == "" {
			http.Error(w, "Quote text is required", http.StatusBadRequest)
			return
		}
		if q.Author == "" {
			q.Author = "Commentator"
		}
		result, err := db.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := result.LastInsertId()
		q.ID = int(id)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(q)
	case "PUT":
		var q Quote
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if q.ID == 0 {
			http.Error(w, "Quote ID is required", http.StatusBadRequest)
			return
		}
		_, err := db.Exec("UPDATE quotes SET text = ?, author = ? WHERE id = ?", q.Text, q.Author, q.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(q)
	case "DELETE":
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Quote ID is required", http.StatusBadRequest)
			return
		}
		_, err := db.Exec("DELETE FROM quotes WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
