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
}

type RaceInfo struct {
	ID          int    `json:"id"`
	Country     string `json:"country"`
	Track       string `json:"track"`
	Date        string `json:"date"`
	Days        int    `json:"days"`
	Hours       int    `json:"hours"`
	Temperature int    `json:"temperature"`
	Length      string `json:"length"`
	Laps        int    `json:"laps"`
}

type AdminUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"-"`
}

var (
	db           *sql.DB
	sessionStore = make(map[string]int64)
	staticCache  = make(map[string][]byte)
)

func main() {
	if err := os.MkdirAll("/app/static/images", 0755); err != nil {
		log.Printf("Warning: could not create images directory: %v", err)
	}

	var err error
	db, err = sql.Open("sqlite3", "/app/heat.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	initDB()

	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/logout", handleLogout)
	http.HandleFunc("/api/check-setup", handleCheckSetup)

	http.HandleFunc("/admin.html", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[ADMIN] Access attempt to admin.html")

		// Debug: print all cookies
		for _, c := range r.Cookies() {
			log.Printf("[ADMIN] Cookie: %s=%s", c.Name, c.Value[:16]+"...")
		}

		cookie, err := r.Cookie("session")
		if err != nil {
			log.Printf("[ADMIN] No session cookie, redirecting to login")
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		log.Printf("[ADMIN] Session cookie found: %s", cookie.Value[:16]+"...")
		expiry, ok := sessionStore[cookie.Value]
		if !ok {
			log.Printf("[ADMIN] Session not found in store, redirecting to login")
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		if time.Now().Unix() > expiry {
			log.Printf("[ADMIN] Session expired, redirecting to login")
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		log.Printf("[ADMIN] Session valid, serving admin.html")
		http.ServeFile(w, r, "/app/static/admin.html")
	})

	http.HandleFunc("/api/racers", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRacers(w, r)
		case "POST":
			updateRacer(w, r)
		case "DELETE":
			deleteRacer(w, r)
		}
	}))

	http.HandleFunc("/api/race-info", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getRaceInfo(w, r)
		case "POST":
			updateRaceInfo(w, r)
		}
	}))

	http.HandleFunc("/api/upload", authMiddleware(handleUpload))

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
		http.ServeFile(w, r, "/app/static/login.html")
	})

	http.HandleFunc("/setup", func(w http.ResponseWriter, r *http.Request) {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
		if count > 0 {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		http.ServeFile(w, r, "/app/static/setup.html")
	})

	fs := http.FileServer(http.Dir("/app/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "/app/static/index.html")
	})

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
		rank INTEGER
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

	createAdminTable := `
	CREATE TABLE IF NOT EXISTS admin_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT
	);`

	_, err := db.Exec(createRacersTable)
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

	db.Exec("INSERT INTO race_info (country, track, date, days, hours, temperature, length, laps) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		"ITALY", "Autodromo Nazionale Monza", "SEPTEMBER 01-03, 2026", 4, 12, 24, "5.793 KM", 53)
}

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session")
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		expiry, ok := sessionStore[cookie.Value]
		if !ok || time.Now().Unix() > expiry {
			delete(sessionStore, cookie.Value)
			http.Error(w, "Session expired", http.StatusUnauthorized)
			return
		}

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
			log.Printf("[LOGIN] Session created: %s", sessionID[:16]+"...")

			cookie := &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"}
			http.SetCookie(w, cookie)
			log.Printf("[LOGIN] Cookie set: session=%s", sessionID[:16]+"...")

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
		log.Printf("[LOGIN] Found user ID: %d, stored password hash: %s", user.ID, user.Password[:20]+"...")

		inputHash := hashPassword(input.Password)
		log.Printf("[LOGIN] Input password hash: %s", inputHash[:20]+"...")

		if inputHash != user.Password {
			log.Printf("[LOGIN] Password mismatch!")
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		log.Printf("[LOGIN] Password verified successfully")
		sessionID := fmt.Sprintf("%x", sha256.Sum256([]byte(fmt.Sprintf("%d-%s-%d", user.ID, user.Username, time.Now().Unix()))))
		sessionStore[sessionID] = time.Now().Add(24 * time.Hour).Unix()
		log.Printf("[LOGIN] Session created: %s", sessionID[:16]+"...")

		cookie := &http.Cookie{Name: "session", Value: sessionID, HttpOnly: true, Path: "/"}
		http.SetCookie(w, cookie)
		log.Printf("[LOGIN] Cookie set: session=%s", sessionID[:16]+"...")

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
	rows, err := db.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank FROM racers ORDER BY rank ASC")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var racers []Racer
	for rows.Next() {
		var r Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank)
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
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if racer.ID == 0 {
		_, err := db.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank) VALUES (?, ?, ?, ?, ?, ?)",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		_, err := db.Exec("UPDATE racers SET name=?, profile_picture=?, car_color=?, car_name=?, points=?, rank=? WHERE id=?",
			racer.Name, racer.ProfilePicture, racer.CarColor, racer.CarName, racer.Points, racer.Rank, racer.ID)
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

func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		http.Error(w, "Invalid file type", http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf("%d%s", time.Now().Unix(), ext)
	uploadPath := filepath.Join("/app/static/images", filename)

	out, err := os.Create(uploadPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer out.Close()

	data, _ := io.ReadAll(file)
	out.Write(data)

	staticCache["/static/images/"+filename] = data

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": "/static/images/" + filename})
}
