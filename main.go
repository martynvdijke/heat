package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"
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
	ID             string `json:"id"`
	Name           string `json:"name"`
	Country        string `json:"country"`
	GeoJSON        string `json:"geojson"`
	Length         int    `json:"length_km"`
	LapRecord      string `json:"lap_record"`
	UseMapImage    bool   `json:"use_map_image"`
	MapImageURL    string `json:"map_image_url"`
	RefreshGeoJSON bool   `json:"refresh_geojson"`
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

type RaceHistory struct {
	ID        int          `json:"id"`
	Name      string       `json:"name"`
	Date      string       `json:"race_date"`
	Country   string       `json:"country"`
	Track     string       `json:"track"`
	TrackID   string       `json:"track_id"`
	TotalLaps int          `json:"total_laps"`
	RaceType  string       `json:"race_type,omitempty"`
	Results   []RaceResult `json:"results,omitempty"`
}

const currentSchemaVersion = 9
const currentVersion = "1.9.0"

type NotificationSettings struct {
	ID              int    `json:"id"`
	GotiFyURL       string `json:"gotify_url"`
	GotiFyToken     string `json:"gotify_token"`
	NotifyWinner    bool   `json:"notify_winner"`
	NotifyRaceStart bool   `json:"notify_race_start"`
	NotifyPodium    bool   `json:"notify_podium"`
}

type AdminUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

type AISettings struct {
	ID              int    `json:"id"`
	TrackExtractURL string `json:"track_extract_url"`
	APIKey          string `json:"api_key"`
	Enabled         bool   `json:"enabled"`
}

var (
	db           *sql.DB
	sessionStore = make(map[string]int64)
	staticCache  = make(map[string][]byte)
	upgrader     = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clients    = make(map[*websocket.Conn]bool)
	broadcast  = make(chan []Racer)
	basePath   = "/app"
	dbPath     = "/db/heat.db"
	imagesPath = "/app/images"
)

func handleWebSocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
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

func isAuthorized(c *gin.Context) bool {
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name == "session" {
			if expiry, ok := sessionStore[cookie.Value]; ok {
				if time.Now().Unix() <= expiry {
					return true
				}
				delete(sessionStore, cookie.Value)
			}
		}
	}
	return false
}

func main() {
	if os.Getenv("DOCKER") != "true" {
		basePath = "."
		dbPath = "./heat.db"
	}
	imagesPath = filepath.Join(basePath, "static/images")

	if err := os.MkdirAll(imagesPath, 0755); err != nil {
		log.Printf("Warning: could not create images directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Printf("Warning: could not create database directory: %v", err)
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()
	go broadcastManager()

	r := gin.Default()

	r.GET("/ws", handleWebSocket)

	r.POST("/api/login", handleLogin)
	r.POST("/api/logout", handleLogout)
	r.GET("/api/check-setup", handleCheckSetup)

	r.GET("/api/racers", getRacers)
	r.POST("/api/racers", authMiddleware(), updateRacer)
	r.DELETE("/api/racers", authMiddleware(), deleteRacer)

	r.GET("/api/race-info", getRaceInfo)
	r.POST("/api/race-info", authMiddleware(), updateRaceInfo)

	r.POST("/api/upload", authMiddleware(), handleUpload)

	r.GET("/api/tracks", getTracks)
	r.POST("/api/tracks", authMiddleware(), saveTrack)
	r.DELETE("/api/tracks", authMiddleware(), deleteTrack)

	r.POST("/api/tracks/ai-extract", authMiddleware(), handleAIExtract)

	r.GET("/api/race-history", getRaceHistory)
	r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)
	r.DELETE("/api/race-history", authMiddleware(), deleteRaceHistory)

	r.GET("/api/racer-stats", getRacerStats)
	r.POST("/api/racer-stats", authMiddleware(), updateRacerStats)

	r.GET("/api/notification-settings", getNotificationSettings)
	r.POST("/api/notification-settings", authMiddleware(), saveNotificationSettings)

	r.POST("/api/test-notification", authMiddleware(), testNotification)

	r.GET("/api/ai-settings", getAISettings)
	r.POST("/api/ai-settings", authMiddleware(), saveAISettings)

	r.GET("/api/oneoff-races", getOneOffRaces)
	r.DELETE("/api/oneoff-races", authMiddleware(), deleteOneOffRace)

	r.GET("/api/track-stats", getTrackStats)

	r.GET("/api/quotes", getQuotes)
	r.POST("/api/quotes", authMiddleware(), handleQuotes)
	r.PUT("/api/quotes", authMiddleware(), handleQuotes)
	r.DELETE("/api/quotes", authMiddleware(), handleQuotes)

	r.GET("/api/quote/random", getRandomQuote)

	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": currentVersion})
	})

	r.GET("/api-docs", func(c *gin.Context) {
		c.File(filepath.Join(basePath, "static/swagger.json"))
	})

	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!DOCTYPE html>
<html>
<head>
    <title>HEAT Racing API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "/api-docs",
                dom_id: '#swagger-ui',
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
                layout: "BaseLayout"
            });
        };
    </script>
</body>
</html>`))
	})

	r.Static("/static", filepath.Join(basePath, "static"))

	r.GET("/admin.html", func(c *gin.Context) {
		log.Printf("[ADMIN] Access attempt to admin.html")

		var validSession string
		for _, cookie := range c.Request.Cookies() {
			if cookie.Name == "session" {
				if _, ok := sessionStore[cookie.Value]; ok {
					validSession = cookie.Value
					break
				}
			}
		}

		if validSession == "" {
			log.Printf("[ADMIN] No valid session, redirecting to login")
			c.Redirect(http.StatusFound, "/login.html")
			return
		}

		log.Printf("[ADMIN] Session valid: %s, serving admin.html", shorten(validSession))
		c.File(filepath.Join(basePath, "static/admin.html"))
	})

	r.GET("/login.html", func(c *gin.Context) {
		cookie, err := c.Request.Cookie("session")
		if err == nil {
			expiry, ok := sessionStore[cookie.Value]
			if ok && time.Now().Unix() <= expiry {
				c.Redirect(http.StatusFound, "/admin.html")
				return
			}
		}
		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		if count == 0 {
			c.Redirect(http.StatusFound, "/setup")
			return
		}
		c.File(filepath.Join(basePath, "static/login.html"))
	})

	r.GET("/setup", func(c *gin.Context) {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		if count > 0 {
			c.Redirect(http.StatusFound, "/login.html")
			return
		}
		c.File(filepath.Join(basePath, "static/setup.html"))
	})

	r.GET("/controller.html", func(c *gin.Context) {
		var validSession string
		for _, cookie := range c.Request.Cookies() {
			if cookie.Name == "session" {
				if _, ok := sessionStore[cookie.Value]; ok {
					validSession = cookie.Value
					break
				}
			}
		}
		if validSession == "" {
			c.Redirect(http.StatusFound, "/login.html")
			return
		}
		c.File(filepath.Join(basePath, "static/controller.html"))
	})

	r.GET("/stats.html", func(c *gin.Context) {
		c.File(filepath.Join(basePath, "static/stats.html"))
	})

	r.GET("/trophies.html", func(c *gin.Context) {
		c.File(filepath.Join(basePath, "static/trophies.html"))
	})

	r.GET("/chat.html", func(c *gin.Context) {
		c.File(filepath.Join(basePath, "static/chat.html"))
	})

	r.GET("/", func(c *gin.Context) {
		c.File(filepath.Join(basePath, "static/index.html"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "6270"
	}

	log.Printf("Server starting on port %s...", port)
	r.Run(":" + port)
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
		db.Exec("DELETE FROM schema_version")
		db.Exec("INSERT INTO schema_version (version) VALUES (?)", version)
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
			name TEXT,
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
		)`)
		seedTracks()
	case 2:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			author TEXT DEFAULT 'Commentator',
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`)
		seedQuotes()
	case 3:
		_, err := db.Exec("SELECT name FROM race_history LIMIT 0")
		if err != nil {
			_, _ = db.Exec("ALTER TABLE race_history ADD COLUMN name TEXT")
		}
	case 4:
		_, _ = db.Exec("ALTER TABLE tracks ADD COLUMN use_map_image INTEGER DEFAULT 0")
		_, _ = db.Exec("ALTER TABLE tracks ADD COLUMN map_image_url TEXT")
		_, _ = db.Exec("ALTER TABLE tracks ADD COLUMN refresh_geojson INTEGER DEFAULT 1")
	case 5:
		_, _ = db.Exec(`ALTER TABLE race_history ADD COLUMN race_type TEXT DEFAULT 'season'`)
	case 6:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS notification_settings (
			id INTEGER PRIMARY KEY,
			gotify_url TEXT,
			gotify_token TEXT,
			notify_winner INTEGER DEFAULT 1,
			notify_race_start INTEGER DEFAULT 0,
			notify_podium INTEGER DEFAULT 0
		)`)
		_, _ = db.Exec("INSERT INTO notification_settings (id, notify_winner, notify_race_start, notify_podium) VALUES (1, 1, 0, 0)")
	case 7:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS uploads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT UNIQUE,
			ext TEXT,
			url TEXT,
			resized_url TEXT,
			thumbnail_url TEXT,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`)
	case 8:
		_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS ai_settings (
			id INTEGER PRIMARY KEY,
			track_extract_url TEXT,
			api_key TEXT,
			enabled INTEGER DEFAULT 0
		)`)
		_, _ = db.Exec("INSERT INTO ai_settings (id, enabled) VALUES (1, 0)")
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
		{Name: "A. PROST", ProfilePicture: "/static/images/helmet.svg", CarColor: "red", CarName: "Red Beast", Points: 78, Rank: 1},
		{Name: "M. SCHUMACHER", ProfilePicture: "/static/images/helmet.svg", CarColor: "blue", CarName: "Blue Bolt", Points: 62, Rank: 2},
		{Name: "A. SENNA", ProfilePicture: "/static/images/helmet.svg", CarColor: "green", CarName: "Green Machine", Points: 85, Rank: 3},
		{Name: "N. LAUDA", ProfilePicture: "/static/images/helmet.svg", CarColor: "yellow", CarName: "Yellow Flash", Points: 45, Rank: 4},
		{Name: "J. STEWART", ProfilePicture: "/static/images/helmet.svg", CarColor: "grey", CarName: "Grey Ghost", Points: 38, Rank: 5},
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

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("[AUTH] Checking session for: %s", c.Request.URL.Path)

		cookies := c.Request.Cookies()
		log.Printf("[AUTH] All cookies: %v", cookies)

		var sessionCookie string
		for _, cookie := range cookies {
			if cookie.Name == "session" {
				if _, ok := sessionStore[cookie.Value]; ok {
					sessionCookie = cookie.Value
					log.Printf("[AUTH] Found valid session in store: %s", shorten(cookie.Value))
					break
				}
			}
		}

		if sessionCookie == "" {
			log.Printf("[AUTH] No valid session cookie found")
			log.Printf("[AUTH] Stored sessions:")
			for k, v := range sessionStore {
				log.Printf("[AUTH]   - %s (expires: %d)", shorten(k), v)
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		log.Printf("[AUTH] Using session: %s", shorten(sessionCookie))

		expiry, ok := sessionStore[sessionCookie]
		if !ok {
			log.Printf("[AUTH] Session not found in store!")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}
		if time.Now().Unix() > expiry {
			log.Printf("[AUTH] Session expired")
			delete(sessionStore, sessionCookie)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			return
		}

		log.Printf("[AUTH] Session valid, allowing: %s", c.Request.URL.Path)
		c.Next()
	}
}

func handleCheckSetup(c *gin.Context) {
	var count int
	db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	c.JSON(http.StatusOK, gin.H{"setup": count > 0})
}

func handleLogin(c *gin.Context) {
	var input struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Setup    bool   `json:"setup"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[LOGIN] Failed to decode JSON: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[LOGIN] Attempting login for user: %s", input.Username)

	var count int
	db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
	log.Printf("[LOGIN] Admin users in DB: %d", count)

	if input.Setup && count > 0 {
		log.Printf("[LOGIN] Setup attempt blocked: Admin user already exists")
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Setup already completed"})
		return
	}

	if count == 0 {
		log.Printf("[LOGIN] No admin users, creating new user: %s", input.Username)
		hashed := hashPassword(input.Password)
		log.Printf("[LOGIN] Password hash: %s", hashed)
		_, err := db.Exec("INSERT INTO admin_users (username, password) VALUES (?, ?)", input.Username, hashed)
		if err != nil {
			log.Printf("[LOGIN] Failed to insert user: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var user AdminUser
		db.QueryRow("SELECT id, username FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username)
		log.Printf("[LOGIN] Created user with ID: %d", user.ID)

		sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", user.ID, user.Username, time.Now().Unix()))))
		sessionStore[sessionID] = time.Now().Add(24 * time.Hour).Unix()
		log.Printf("[LOGIN] Session created: %s", shorten(sessionID))

		http.SetCookie(c.Writer, &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"})
		log.Printf("[LOGIN] Cookie set: session=%s", shorten(sessionID))

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	log.Printf("[LOGIN] Looking up user: %s", input.Username)
	var user AdminUser
	err := db.QueryRow("SELECT id, username, password FROM admin_users WHERE username = ?", input.Username).Scan(&user.ID, &user.Username, &user.Password)
	if err != nil {
		log.Printf("[LOGIN] User not found: %v", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	log.Printf("[LOGIN] Found user ID: %d, stored password hash: %s", user.ID, shorten(user.Password))

	inputHash := hashPassword(input.Password)
	log.Printf("[LOGIN] Input password hash: %s", shorten(inputHash))

	if inputHash != user.Password {
		log.Printf("[LOGIN] Password mismatch!")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	log.Printf("[LOGIN] Password verified successfully")
	sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", user.ID, user.Username, time.Now().Unix()))))
	sessionStore[sessionID] = time.Now().Add(24 * time.Hour).Unix()
	log.Printf("[LOGIN] Session created: %s", shorten(sessionID))

	http.SetCookie(c.Writer, &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"})
	log.Printf("[LOGIN] Cookie set: session=%s", shorten(sessionID))

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleLogout(c *gin.Context) {
	cookie, err := c.Request.Cookie("session")
	if err == nil {
		delete(sessionStore, cookie.Value)
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: "session", Value: "", MaxAge: -1})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getRacers(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers ORDER BY rank ASC")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var racers []Racer
	for rows.Next() {
		var r Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		racers = append(racers, r)
	}

	c.JSON(http.StatusOK, racers)
}

func updateRacer(c *gin.Context) {
	var racer Racer
	if err := c.ShouldBindJSON(&racer); err != nil {
		log.Printf("[RACER] Failed to decode: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[RACER] Updating racer: ID=%d, Name=%s, Picture=%s, Car=%s (%s), Points=%d, Rank=%d, Position=%d",
		racer.ID, racer.Name, racer.ProfilePicture, racer.CarName, racer.CarColor, racer.Points, racer.Rank, racer.Position)

	if racer.ID == 0 {
		_, err := db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank, position) VALUES (?, ?, ?, ?, ?, ?, ?)",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position)
		if err != nil {
			log.Printf("[RACER] Insert failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[RACER] Created new racer")
	} else {
		_, err := db.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, rank=?, position=? WHERE id=?",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.Position, racer.ID)
		if err != nil {
			log.Printf("[RACER] Update failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[RACER] Updated racer ID=%d", racer.ID)
	}
	broadcastRacers()
	c.Status(http.StatusOK)
}

func deleteRacer(c *gin.Context) {
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	log.Printf("[RACER] Deleting racer ID=%d", id)
	_, err := db.Exec("DELETE FROM racers WHERE id=?", id)
	if err != nil {
		log.Printf("[RACER] Delete failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	log.Printf("[RACER] Deleted racer ID=%d", id)
	broadcastRacers()
	c.Status(http.StatusOK)
}

func getRaceInfo(c *gin.Context) {
	var ri RaceInfo
	err := db.QueryRow("SELECT country, track, COALESCE(track_id, 'monza'), laps FROM race_info ORDER BY id DESC LIMIT 1").
		Scan(&ri.Country, &ri.Track, &ri.TrackID, &ri.Laps)
	if err != nil {
		ri = RaceInfo{Country: "Italy", Track: "Monza", TrackID: "monza", Laps: 53}
	}
	c.JSON(http.StatusOK, ri)
}

func updateRaceInfo(c *gin.Context) {
	var ri RaceInfo
	if err := c.ShouldBindJSON(&ri); err != nil {
		log.Printf("[RACEINFO] Failed to decode: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ri.TrackID == "" {
		ri.TrackID = "monza"
	}

	_, err := db.Exec("INSERT INTO race_info (country, track, track_id, laps) VALUES (?, ?, ?, ?)",
		ri.Country, ri.Track, ri.TrackID, ri.Laps)
	if err != nil {
		log.Printf("[RACEINFO] Insert failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

func handleUpload(c *gin.Context) {
	log.Printf("[UPLOAD] Upload request received")

	header, err := c.FormFile("image")
	if err != nil {
		log.Printf("[UPLOAD] Failed to get form file: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("[UPLOAD] File received: %s, size: %d", header.Filename, header.Size)

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		log.Printf("[UPLOAD] Invalid file type: %s", ext)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	file, err := header.Open()
	if err != nil {
		log.Printf("[UPLOAD] Failed to open uploaded file: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		log.Printf("[UPLOAD] Failed to read uploaded file: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	var existingURL string
	err = db.QueryRow("SELECT url FROM uploads WHERE hash = ?", hashStr).Scan(&existingURL)
	if err == nil {
		log.Printf("[UPLOAD] Duplicate found: %s", hashStr)
		c.JSON(http.StatusOK, gin.H{"url": existingURL, "duplicate": true})
		return
	}

	saveExt := ext
	if ext == ".jpeg" {
		saveExt = ".jpg"
	}

	filename := hashStr + saveExt
	uploadPath := filepath.Join(imagesPath, filename)

	if err := os.WriteFile(uploadPath, data, 0644); err != nil {
		log.Printf("[UPLOAD] Failed to save file: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resizedFilename := hashStr + "_resized" + saveExt
	thumbFilename := hashStr + "_thumb" + saveExt
	resizedPath := filepath.Join(imagesPath, resizedFilename)
	thumbPath := filepath.Join(imagesPath, thumbFilename)

	src, err := imaging.Open(uploadPath)
	if err == nil {
		resized := imaging.Fit(src, 1200, 1200, imaging.Lanczos)
		if err := imaging.Save(resized, resizedPath); err != nil {
			log.Printf("[UPLOAD] Failed to save resized: %v", err)
		} else {
			resizedData, _ := os.ReadFile(resizedPath)
			staticCache["/static/images/"+resizedFilename] = resizedData
		}

		thumb := imaging.Thumbnail(src, 150, 150, imaging.Lanczos)
		if err := imaging.Save(thumb, thumbPath); err != nil {
			log.Printf("[UPLOAD] Failed to save thumbnail: %v", err)
		} else {
			thumbData, _ := os.ReadFile(thumbPath)
			staticCache["/static/images/"+thumbFilename] = thumbData
		}
	} else {
		log.Printf("[UPLOAD] Failed to open image for processing: %v", err)
	}

	url := "/static/images/" + filename
	resizedURL := "/static/images/" + resizedFilename
	thumbURL := "/static/images/" + thumbFilename

	db.Exec("INSERT INTO uploads (hash, ext, url, resized_url, thumbnail_url) VALUES (?, ?, ?, ?, ?)",
		hashStr, ext, url, resizedURL, thumbURL)

	staticCache[url] = data

	log.Printf("[UPLOAD] Success! URL: %s", url)
	c.JSON(http.StatusOK, gin.H{
		"url":           url,
		"resized_url":   resizedURL,
		"thumbnail_url": thumbURL,
		"hash":          hashStr,
	})
}

func getTracks(c *gin.Context) {
	rows, err := db.Query("SELECT id, name, country, geojson, length_km, lap_record, COALESCE(use_map_image, 0), COALESCE(map_image_url, ''), COALESCE(refresh_geojson, 1) FROM tracks ORDER BY name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tracks []Track
	for rows.Next() {
		var t Track
		var useMapImage, refreshGeoJSON int
		if err := rows.Scan(&t.ID, &t.Name, &t.Country, &t.GeoJSON, &t.Length, &t.LapRecord, &useMapImage, &t.MapImageURL, &refreshGeoJSON); err != nil {
			log.Printf("[TRACK] Scan error: %v", err)
			continue
		}
		t.UseMapImage = useMapImage == 1
		t.RefreshGeoJSON = refreshGeoJSON == 1
		tracks = append(tracks, t)
	}
	c.JSON(http.StatusOK, tracks)
}

func saveTrack(c *gin.Context) {
	var t Track
	if err := c.ShouldBindJSON(&t); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO tracks (id, name, country, geojson, length_km, lap_record, use_map_image, map_image_url, refresh_geojson) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Country, t.ID, t.Length, t.LapRecord, boolToInt(t.UseMapImage), t.MapImageURL, boolToInt(t.RefreshGeoJSON))
	if err != nil {
		log.Printf("[TRACK] Failed to save track: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, t)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func deleteTrack(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}

	_, err := db.Exec("DELETE FROM tracks WHERE id = ?", id)
	if err != nil {
		log.Printf("[TRACK] Failed to delete track: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func handleAIExtract(c *gin.Context) {
	log.Printf("[AI] Track extraction requested")

	aiURL := os.Getenv("AI_TRACK_EXTRACT_URL")
	if aiURL == "" {
		var dbURL string
		var enabled bool
		err := db.QueryRow("SELECT track_extract_url, enabled FROM ai_settings WHERE id = 1").Scan(&dbURL, &enabled)
		if err == nil && enabled && dbURL != "" {
			aiURL = dbURL
		}
	}
	if aiURL == "" {
		log.Printf("[AI] No endpoint configured")
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "AI endpoint not configured"})
		return
	}

	var imageData []byte
	var contentType string

	file, header, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		imageData, err = io.ReadAll(file)
		if err != nil {
			log.Printf("[AI] Failed to read uploaded image: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
			return
		}
		contentType = header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		log.Printf("[AI] Image received: %s, %d bytes", header.Filename, len(imageData))
	} else {
		var input struct {
			ImageURL string `json:"image_url"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.ImageURL == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No image provided. Send multipart form with 'image' field or JSON with 'image_url'"})
			return
		}
		localPath := filepath.Join(basePath, input.ImageURL)
		imageData, err = os.ReadFile(localPath)
		if err != nil {
			log.Printf("[AI] Failed to read image from path %s: %v", localPath, err)
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid image URL"})
			return
		}
		contentType = http.DetectContentType(imageData)
		log.Printf("[AI] Image read from: %s, %d bytes", input.ImageURL, len(imageData))
	}

	reqBody := &bytes.Buffer{}
	writer := multipart.NewWriter(reqBody)
	part, _ := writer.CreateFormFile("image", "track.png")
	part.Write(imageData)
	writer.Close()

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", aiURL, reqBody)
	if err != nil {
		log.Printf("[AI] Failed to create request: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create AI request"})
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add API key if available
	var apiKey string
	db.QueryRow("SELECT COALESCE(api_key, '') FROM ai_settings WHERE id = 1").Scan(&apiKey)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[AI] Request to %s failed: %v", aiURL, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "AI request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[AI] Failed to read AI response: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read AI response"})
		return
	}

	if resp.StatusCode >= 400 {
		log.Printf("[AI] AI returned status %d: %s", resp.StatusCode, string(bodyBytes))
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("AI returned status %d", resp.StatusCode)})
		return
	}

	var parsedResponse interface{}
	if err := json.Unmarshal(bodyBytes, &parsedResponse); err != nil {
		log.Printf("[AI] AI response is not valid JSON, returning raw")
		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}

	log.Printf("[AI] Successfully extracted track data from AI")
	c.Data(http.StatusOK, "application/json", bodyBytes)
}

func getAISettings(c *gin.Context) {
	var s AISettings
	var enabled int
	err := db.QueryRow("SELECT id, COALESCE(track_extract_url, ''), COALESCE(api_key, ''), COALESCE(enabled, 0) FROM ai_settings WHERE id = 1").
		Scan(&s.ID, &s.TrackExtractURL, &s.APIKey, &enabled)
	if err != nil {
		s = AISettings{ID: 1, Enabled: false}
		c.JSON(http.StatusOK, s)
		return
	}
	s.Enabled = enabled == 1
	c.JSON(http.StatusOK, s)
}

func saveAISettings(c *gin.Context) {
	var s AISettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO ai_settings (id, track_extract_url, api_key, enabled) VALUES (1, ?, ?, ?)`,
		s.TrackExtractURL, s.APIKey, boolToInt(s.Enabled))
	if err != nil {
		log.Printf("[AI] Save settings failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func getRaceHistory(c *gin.Context) {
	raceID := c.Query("id")
	raceType := c.Query("type")

	var query string
	var args []interface{}

	if raceID != "" {
		query = `SELECT rh.id, COALESCE(rh.name, ''), rh.race_date, rh.country, rh.track, rh.track_id, rh.total_laps, COALESCE(rh.race_type, 'season'),
				 COALESCE(GROUP_CONCAT(rr.racer_id || ':' || rr.racer_name || ':' || rr.position || ':' || rr.points || ':' || rr.fastest_lap, '|'), '') as results
				 FROM race_history rh
				 LEFT JOIN race_results rr ON rh.id = rr.race_id
				 WHERE rh.id = ?
				 GROUP BY rh.id`
		args = []interface{}{raceID}
	} else {
		if raceType != "" {
			query = `SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'season')
					 FROM race_history WHERE race_type = ? ORDER BY race_date DESC LIMIT 20`
			args = []interface{}{raceType}
		} else {
			query = `SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'season') FROM race_history ORDER BY race_date DESC LIMIT 20`
		}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []RaceHistory
	for rows.Next() {
		var h RaceHistory
		var resultsStr string
		var raceType string
		if raceID != "" {
			rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &raceType, &resultsStr)
			h.RaceType = raceType
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
			rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &h.RaceType)
		}
		history = append(history, h)
	}
	c.JSON(http.StatusOK, history)
}

func saveRaceToHistory(c *gin.Context) {
	var input struct {
		Name      string `json:"name"`
		RaceDate  string `json:"race_date"`
		Country   string `json:"country"`
		Track     string `json:"track"`
		TrackID   string `json:"track_id"`
		TotalLaps int    `json:"total_laps"`
		RaceType  string `json:"race_type"`
		Results   []struct {
			RacerID    int    `json:"racer_id"`
			RacerName  string `json:"racer_name"`
			Position   int    `json:"position"`
			Points     int    `json:"points"`
			FastestLap bool   `json:"fastest_lap"`
			Finished   bool   `json:"finished"`
		} `json:"results"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if input.RaceDate == "" {
		input.RaceDate = time.Now().Format("2006-01-02")
	}
	if input.Name == "" {
		input.Name = input.RaceDate
	}
	if input.RaceType == "" {
		input.RaceType = "season"
	}

	isOneOff := input.RaceType == "oneoff"

	result, err := db.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES (?, ?, ?, ?, ?, ?, ?)",
		input.Name, input.RaceDate, input.Country, input.Track, input.TrackID, input.TotalLaps, input.RaceType)
	if err != nil {
		log.Printf("[HISTORY] Insert failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	raceID, _ := result.LastInsertId()

	for _, res := range input.Results {
		db.Exec("INSERT INTO race_results (race_id, racer_id, racer_name, position, points, fastest_lap) VALUES (?, ?, ?, ?, ?, ?)",
			raceID, res.RacerID, res.RacerName, res.Position, res.Points, boolToInt(res.FastestLap))

		// Only accumulate stats for season races, not one-off races
		if !isOneOff {
			db.Exec(`INSERT INTO racer_stats (racer_id, races, wins, podiums, fastest_laps, dnf) VALUES (?, 1, ?, ?, ?, ?)
					 ON CONFLICT(racer_id) DO UPDATE SET
					 races = races + 1,
					 wins = wins + excluded.wins,
					 podiums = podiums + excluded.podiums,
					 fastest_laps = fastest_laps + excluded.fastest_laps,
					 dnf = dnf + excluded.dnf`,
				res.RacerID, boolToInt(res.Position == 1), boolToInt(res.Position <= 3), boolToInt(res.FastestLap), boolToInt(!res.Finished))
		}
	}

	// Only send notifications for season races
	if !isOneOff && len(input.Results) > 0 {
		winner := ""
		second := ""
		third := ""
		for _, res := range input.Results {
			if res.Position == 1 {
				winner = res.RacerName
			} else if res.Position == 2 {
				second = res.RacerName
			} else if res.Position == 3 {
				third = res.RacerName
			}
		}
		notifyRaceWinner(winner, input.Track)
		if second != "" && third != "" {
			notifyRacePodium(winner, second, third, input.Track)
		}
	}

	c.JSON(http.StatusOK, gin.H{"id": raceID})
}

func deleteRaceHistory(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}
	db.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	db.Exec("DELETE FROM race_history WHERE id = ?", id)
	c.Status(http.StatusOK)
}

func getRacerStats(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		var stats []RacerStats
		rows, _ := db.Query("SELECT id, racer_id, races, wins, podiums, fastest_laps, (SELECT SUM(points) FROM racers WHERE id = racer_id) as pts, dnf FROM racer_stats")
		if rows != nil {
			for rows.Next() {
				var s RacerStats
				rows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Podiums, &s.FastestLaps, &s.Points, &s.DNF)
				stats = append(stats, s)
			}
			rows.Close()
		}
		c.JSON(http.StatusOK, stats)
		return
	}

	var s RacerStats
	err := db.QueryRow("SELECT id, racer_id, races, wins, podiums, fastest_laps, COALESCE((SELECT SUM(points) FROM racers WHERE id = racer_id), 0) as pts, dnf FROM racer_stats WHERE racer_id = ?", id).Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Podiums, &s.FastestLaps, &s.Points, &s.DNF)
	if err != nil {
		s = RacerStats{RacerID: 0, Races: 0, Wins: 0, Podiums: 0, FastestLaps: 0, Points: 0, DNF: 0}
	}

	var rInfo Racer
	db.QueryRow("SELECT id, name, profile_picture, car_color, car_name, points FROM racers WHERE id = ?", id).Scan(&rInfo.ID, &rInfo.Name, &rInfo.ProfilePicture, &rInfo.CarColor, &rInfo.CarName, &rInfo.Points)

	c.JSON(http.StatusOK, gin.H{"stats": s, "racer": rInfo})
}

func updateRacerStats(c *gin.Context) {
	var stats RacerStats
	if err := c.ShouldBindJSON(&stats); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if stats.ID == 0 {
		_, err := db.Exec("INSERT INTO racer_stats (racer_id, races, wins, podiums, fastest_laps, dnf) VALUES (?, ?, ?, ?, ?, ?)",
			stats.RacerID, stats.Races, stats.Wins, stats.Podiums, stats.FastestLaps, stats.DNF)
		if err != nil {
			log.Printf("[STATS] Insert failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		_, err := db.Exec("UPDATE racer_stats SET races = ?, wins = ?, podiums = ?, fastest_laps = ?, dnf = ? WHERE id = ?",
			stats.Races, stats.Wins, stats.Podiums, stats.FastestLaps, stats.DNF, stats.ID)
		if err != nil {
			log.Printf("[STATS] Update failed: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	c.Status(http.StatusOK)
}

func getQuotes(c *gin.Context) {
	rows, err := db.Query("SELECT id, text, author, created_at FROM quotes ORDER BY id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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
	c.JSON(http.StatusOK, quotes)
}

func getRandomQuote(c *gin.Context) {
	var q Quote
	err := db.QueryRow("SELECT id, text, author, created_at FROM quotes ORDER BY RANDOM() LIMIT 1").Scan(&q.ID, &q.Text, &q.Author, &q.CreatedAt)
	if err != nil {
		q = Quote{Text: "The engines roar as these legends battle for glory!", Author: "Commentator"}
	}
	c.JSON(http.StatusOK, q)
}

func handleQuotes(c *gin.Context) {
	switch c.Request.Method {
	case "GET":
		getQuotes(c)
	case "POST":
		var q Quote
		if err := c.ShouldBindJSON(&q); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if q.Text == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote text is required"})
			return
		}
		if q.Author == "" {
			q.Author = "Commentator"
		}
		result, err := db.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", q.Text, q.Author)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := result.LastInsertId()
		q.ID = int(id)
		c.JSON(http.StatusCreated, q)
	case "PUT":
		var q Quote
		if err := c.ShouldBindJSON(&q); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if q.ID == 0 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote ID is required"})
			return
		}
		_, err := db.Exec("UPDATE quotes SET text = ?, author = ? WHERE id = ?", q.Text, q.Author, q.ID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, q)
	case "DELETE":
		id := c.Query("id")
		if id == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Quote ID is required"})
			return
		}
		_, err := db.Exec("DELETE FROM quotes WHERE id = ?", id)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusOK)
	}
}

func getOneOffRaces(c *gin.Context) {
	rows, err := db.Query(`SELECT id, COALESCE(name, ''), race_date, country, track, track_id, total_laps, COALESCE(race_type, 'oneoff')
					   FROM race_history WHERE race_type = 'oneoff' ORDER BY race_date DESC LIMIT 20`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var history []RaceHistory
	for rows.Next() {
		var h RaceHistory
		rows.Scan(&h.ID, &h.Name, &h.Date, &h.Country, &h.Track, &h.TrackID, &h.TotalLaps, &h.RaceType)
		history = append(history, h)
	}
	c.JSON(http.StatusOK, history)
}

func deleteOneOffRace(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}
	db.Exec("DELETE FROM race_results WHERE race_id = ?", id)
	db.Exec("DELETE FROM race_history WHERE id = ? AND race_type = 'oneoff'", id)
	c.Status(http.StatusOK)
}

func getNotificationSettings(c *gin.Context) {
	var s NotificationSettings
	err := db.QueryRow("SELECT id, COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), COALESCE(notify_winner, 0), COALESCE(notify_race_start, 0), COALESCE(notify_podium, 0) FROM notification_settings WHERE id = 1").
		Scan(&s.ID, &s.GotiFyURL, &s.GotiFyToken, &s.NotifyWinner, &s.NotifyRaceStart, &s.NotifyPodium)
	if err != nil {
		s = NotificationSettings{ID: 1, NotifyWinner: true, NotifyRaceStart: false, NotifyPodium: false}
	}
	c.JSON(http.StatusOK, s)
}

func saveNotificationSettings(c *gin.Context) {
	var s NotificationSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec(`INSERT OR REPLACE INTO notification_settings (id, gotify_url, gotify_token, notify_winner, notify_race_start, notify_podium) VALUES (1, ?, ?, ?, ?, ?)`,
		s.GotiFyURL, s.GotiFyToken, boolToInt(s.NotifyWinner), boolToInt(s.NotifyRaceStart), boolToInt(s.NotifyPodium))
	if err != nil {
		log.Printf("[NOTIFY] Save failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func testNotification(c *gin.Context) {
	var s NotificationSettings
	db.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, '') FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken)

	if s.GotiFyURL == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Gotify URL not configured"})
		return
	}

	err := sendGotifyNotification("Test Notification", "HEAT notification test successful!", s.GotiFyURL, s.GotiFyToken)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to send test: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

type TrackStats struct {
	TrackID    string `json:"track_id"`
	TrackName  string `json:"track_name"`
	Country    string `json:"country"`
	RacesCount int    `json:"races_count"`
	Winner     string `json:"winner"`
	FastestLap string `json:"fastest_lap"`
}

func getTrackStats(c *gin.Context) {
	rows, err := db.Query(`
		SELECT rh.track_id, rh.track, rh.country, COUNT(*) as races_count,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh2.id AND rr.position = 1 LIMIT 1), '') as winner,
			COALESCE((SELECT rr.racer_name FROM race_results rr WHERE rr.race_id = rh3.id AND rr.fastest_lap = 1 LIMIT 1), '') as fastest_lap
		FROM race_history rh
		LEFT JOIN race_history rh2 ON rh2.id = rh.id
		LEFT JOIN race_history rh3 ON rh3.id = rh.id
		WHERE rh.race_type = 'season'
		GROUP BY rh.track_id
		ORDER BY races_count DESC
	`)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var stats []TrackStats
	for rows.Next() {
		var s TrackStats
		if err := rows.Scan(&s.TrackID, &s.TrackName, &s.Country, &s.RacesCount, &s.Winner, &s.FastestLap); err != nil {
			log.Printf("[TRACK STATS] Scan error: %v", err)
			continue
		}
		stats = append(stats, s)
	}
	c.JSON(http.StatusOK, stats)
}

func sendGotifyNotification(title, message, gotifyURL, token string) error {
	if gotifyURL == "" || token == "" {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("POST", gotifyURL+"/message", strings.NewReader(fmt.Sprintf(`{"title":"%s","message":"%s","priority":5}`, title, message)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gotify returned status %d", resp.StatusCode)
	}
	return nil
}

func notifyRaceWinner(winnerName, trackName string) {
	var s NotificationSettings
	db.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), notify_winner FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken, &s.NotifyWinner)

	if s.NotifyWinner && s.GotiFyURL != "" {
		go sendGotifyNotification("🏆 Race Winner!", fmt.Sprintf("%s wins at %s!", winnerName, trackName), s.GotiFyURL, s.GotiFyToken)
	}
}

func notifyRacePodium(first, second, third, trackName string) {
	var s NotificationSettings
	db.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), notify_podium FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken, &s.NotifyPodium)

	if s.NotifyPodium && s.GotiFyURL != "" {
		go sendGotifyNotification("🎉 Podium Result", fmt.Sprintf(" podium at %s: 1. %s  2. %s  3. %s", trackName, first, second, third), s.GotiFyURL, s.GotiFyToken)
	}
}

func notifyRaceStart(trackName string) {
	var s NotificationSettings
	db.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), notify_race_start FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken, &s.NotifyRaceStart)

	if s.NotifyRaceStart && s.GotiFyURL != "" {
		go sendGotifyNotification("🏁 Race Starting!", fmt.Sprintf("The race at %s has begun!", trackName), s.GotiFyURL, s.GotiFyToken)
	}
}
