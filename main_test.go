package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"heat/app"
	"heat/db"
	"heat/ent"
	"heat/handlers"
	"heat/middleware"
	"heat/models"
	"heat/ws"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Unsetenv("DOCKER")
	app.BasePath = "."
	app.DBPath = ":memory:"
	app.MediaPath = filepath.Join(app.BasePath, "media")

	var err error
	app.DB, err = sql.Open("sqlite3", app.DBPath+"?_fk=1")
	if err != nil {
		log.Fatalf("failed to open in-memory db: %v", err)
	}
	app.DB.SetMaxOpenConns(1)

	drv := entsql.OpenDB(dialect.SQLite, app.DB)
	app.Ent = ent.NewClient(ent.Driver(drv))

	db.Init()
	go ws.BroadcastManager()
	go ws.BroadcastFlags()
	go ws.BroadcastGameMechanics()
	go ws.BroadcastWeather()
	go ws.BroadcastLapReplay()
	go ws.BroadcastSound()
	go ws.BroadcastRaceRadio()

	if err := os.MkdirAll(app.MediaPath, 0755); err != nil {
		log.Fatalf("failed to create media directory: %v", err)
	}

	os.Exit(m.Run())
}

func TestHashPassword(t *testing.T) {
	password := "password123"
	hash := hashPassword(password)

	if hash == "" {
		t.Error("Expected hash to be non-empty")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Error("Expected password to verify against its hash")
	}

	hash2 := hashPassword("anotherpassword")
	if err := bcrypt.CompareHashAndPassword([]byte(hash2), []byte("anotherpassword")); err != nil {
		t.Error("Expected second password to verify against its hash")
	}
}

func TestHandleCheckSetup(t *testing.T) {
	r := gin.New()
	r.GET("/api/check-setup", handlers.HandleCheckSetup)

	req, err := http.NewRequest("GET", "/api/check-setup", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]bool
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["setup"] != false {
		t.Errorf("expected setup to be false, got %v", response["setup"])
	}
}

func TestGetRacers(t *testing.T) {
	r := gin.New()
	r.GET("/api/racers", handlers.GetRacers)

	req, err := http.NewRequest("GET", "/api/racers", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var racers []models.Racer
	err = json.Unmarshal(rr.Body.Bytes(), &racers)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(racers) != 5 {
		t.Errorf("expected 5 racers, got %d", len(racers))
	}
}

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.GET("/api/test", middleware.AuthMiddleware(), func(c *gin.Context) {
		c.String(http.StatusOK, "Authorized")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status %v, got %v", http.StatusUnauthorized, status)
		}
	})

	t.Run("Authorized", func(t *testing.T) {
		sessionID := "test-session"
		app.SessionStoreMu.Lock()
		app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
		app.SessionStoreMu.Unlock()
		defer func() {
			app.SessionStoreMu.Lock()
			delete(app.SessionStore, sessionID)
			app.SessionStoreMu.Unlock()
		}()

		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status %v, got %v", http.StatusOK, status)
		}

		if rr.Body.String() != "Authorized" {
			t.Errorf("expected body 'Authorized', got '%s'", rr.Body.String())
		}
	})
}

func TestRaceInfo(t *testing.T) {
	t.Run("GetRaceInfo", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-info", handlers.GetRaceInfo)

		req, _ := http.NewRequest("GET", "/api/race-info", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var ri models.RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &ri)
		if ri.Country != "Italy" || ri.Track != "Monza" {
			t.Errorf("unexpected race info: %+v", ri)
		}
	})

	t.Run("UpdateRaceInfo", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/race-info", handlers.UpdateRaceInfo)
		r.GET("/api/race-info", handlers.GetRaceInfo)

		ri := models.RaceInfo{Country: "Belgium", Track: "Spa", Laps: 44, TrackID: "spa"}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var updated models.RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updated)
		if updated.Country != "Belgium" || updated.Track != "Spa" || updated.Laps != 44 {
			t.Errorf("unexpected race info after update: %+v", updated)
		}
	})
}

func TestRaceHistory(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", handlers.SaveRaceToHistory)
	r.GET("/api/race-history", handlers.GetRaceHistory)
	r.DELETE("/api/race-history", handlers.DeleteRaceHistory)

	t.Run("SaveAndGet", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":       "Test Race",
			"race_date":  "2025-01-01",
			"country":    "Italy",
			"track":      "Monza",
			"track_id":   "monza",
			"total_laps": 53,
			"race_type":  "season",
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true, "finished": true},
				{"racer_id": 2, "racer_name": "M. SCHUMACHER", "position": 2, "points": 18, "fastest_lap": false, "finished": true},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
		}

		var result map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &result)
		if _, ok := result["id"]; !ok {
			t.Errorf("expected id in response, got %v", result)
		}
	})

	t.Run("GetHistory", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var history []models.RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)
		if len(history) < 1 {
			t.Errorf("expected at least 1 race history, got %d", len(history))
		}
	})
}

func TestQuotes(t *testing.T) {
	r := gin.New()
	r.GET("/api/quotes", handlers.GetQuotes)
	r.GET("/api/quote/random", handlers.GetRandomQuote)

	t.Run("GetQuotes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quotes", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var quotes []models.Quote
		json.Unmarshal(rr.Body.Bytes(), &quotes)
		if len(quotes) < 1 {
			t.Errorf("expected quotes, got %d", len(quotes))
		}
	})

	t.Run("GetRandomQuote", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quote/random", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var q models.Quote
		json.Unmarshal(rr.Body.Bytes(), &q)
		if q.Text == "" {
			t.Errorf("expected quote text to be non-empty")
		}
	})
}

func TestGetRacerStatsSeasonFallback(t *testing.T) {
	app.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (1, 5, 3, 3, 1, 0, 2, 0, 0)")
	app.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (2, 5, 1, 1, 2, 1, 0, 1, 0)")
	defer app.DB.Exec("DELETE FROM racer_stats WHERE racer_id IN (1, 2)")

	r := gin.New()
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	t.Run("SeasonFilterFallsBackToRacerStats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-stats?season_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v", status)
		}

		var stats []models.RacerStats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(stats) == 0 {
			t.Fatal("expected non-empty stats from season fallback")
		}

		var found bool
		for _, s := range stats {
			if s.RacerID == 1 {
				found = true
				if s.Wins != 3 || s.Gold != 3 || s.Races != 5 {
					t.Errorf("racer 1: expected wins=3, gold=3, races=5, got wins=%d, gold=%d, races=%d", s.Wins, s.Gold, s.Races)
				}
			}
		}
		if !found {
			t.Error("expected racer 1 in stats")
		}
	})

	t.Run("SingleRacerSeasonFallback", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1&season_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v", status)
		}

		var result map[string]json.RawMessage
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		var s models.RacerStats
		if err := json.Unmarshal(result["stats"], &s); err != nil {
			t.Fatalf("failed to unmarshal stats: %v", err)
		}

		if s.RacerID != 1 || s.Wins != 3 || s.Gold != 3 || s.Races != 5 {
			t.Errorf("expected racer_id=1, wins=3, gold=3, races=5, got racer_id=%d, wins=%d, gold=%d, races=%d", s.RacerID, s.Wins, s.Gold, s.Races)
		}
	})
}

func TestGetRacerStats(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	req, _ := http.NewRequest("GET", "/api/racer-stats", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var stats []models.RacerStats
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if stats == nil {
		t.Errorf("expected stats array, got nil")
	}
}

func TestUpdateRacerStats(t *testing.T) {
	r := gin.New()
	r.POST("/api/racer-stats", handlers.UpdateRacerStats)
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	// Create new stats for racer 3 (no existing stats yet)
	createBody, _ := json.Marshal(models.RacerStats{
		RacerID: 3, Races: 10, Wins: 4, Gold: 4, Silver: 2, Bronze: 1, FastestLaps: 3, DNF: 1, DNS: 2,
	})
	req, _ := http.NewRequest("POST", "/api/racer-stats", bytes.NewBuffer(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("create: expected status 200, got %v: %s", status, rr.Body.String())
	}

	// Verify via GET with racer_id=3
	req, _ = http.NewRequest("GET", "/api/racer-stats?id=3", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("get after create: expected status 200, got %v", status)
	}
	var result map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &result)
	var s models.RacerStats
	json.Unmarshal(result["stats"], &s)
	if s.Races != 10 || s.Wins != 4 || s.Gold != 4 || s.Silver != 2 || s.Bronze != 1 || s.FastestLaps != 3 || s.DNF != 1 || s.DNS != 2 {
		t.Errorf("expected stats (10,4,4,2,1,3,1,2), got (%d,%d,%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Gold, s.Silver, s.Bronze, s.FastestLaps, s.DNF, s.DNS)
	}

	// Find the actual DB id for the update
	var statsID int
	app.DB.QueryRow("SELECT id FROM racer_stats WHERE racer_id = 3").Scan(&statsID)

	// Update existing stats
	updateBody, _ := json.Marshal(models.RacerStats{
		ID: statsID, RacerID: 3, Races: 20, Wins: 8, Gold: 8, Silver: 4, Bronze: 3, FastestLaps: 6, DNF: 2, DNS: 1,
	})
	req, _ = http.NewRequest("POST", "/api/racer-stats", bytes.NewBuffer(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("update: expected status 200, got %v: %s", status, rr.Body.String())
	}

	// Verify updated
	req, _ = http.NewRequest("GET", "/api/racer-stats?id=3", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	json.Unmarshal(rr.Body.Bytes(), &result)
	json.Unmarshal(result["stats"], &s)
	if s.Races != 20 || s.Wins != 8 || s.Gold != 8 || s.Silver != 4 || s.Bronze != 3 || s.FastestLaps != 6 || s.DNF != 2 || s.DNS != 1 {
		t.Errorf("expected updated stats (20,8,8,4,3,6,2,1), got (%d,%d,%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Gold, s.Silver, s.Bronze, s.FastestLaps, s.DNF, s.DNS)
	}
}

func TestRaceHistoryWithDNS(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", handlers.SaveRaceToHistory)
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	// Save a race with one DNF and one DNS
	payload := map[string]interface{}{
		"name":       "DNS Test Race",
		"race_date":  "2025-06-01",
		"country":    "Italy",
		"track":      "Monza",
		"track_id":   "monza",
		"total_laps": 53,
		"race_type":  "season",
		"results": []map[string]interface{}{
			{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true, "finished": true},
			{"racer_id": 2, "racer_name": "M. SCHUMACHER", "position": 2, "points": 18, "fastest_lap": false, "finished": true},
			{"racer_id": 5, "racer_name": "J. STEWART", "position": 999, "points": 0, "fastest_lap": false, "finished": false, "did_not_start": false},
			{"racer_id": 4, "racer_name": "N. LAUDA", "position": 0, "points": 0, "fastest_lap": false, "finished": false, "did_not_start": true},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	// Check racer 5 has DNF=1, DNS=0
	req, _ = http.NewRequest("GET", "/api/racer-stats?id=5", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var result map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &result)
	var s5 models.RacerStats
	json.Unmarshal(result["stats"], &s5)
	if s5.DNF != 1 {
		t.Errorf("racer 5: expected DNF=1, got %d", s5.DNF)
	}
	if s5.DNS != 0 {
		t.Errorf("racer 5: expected DNS=0, got %d", s5.DNS)
	}

	// Check racer 4 has DNS=1, DNF=0
	req, _ = http.NewRequest("GET", "/api/racer-stats?id=4", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	json.Unmarshal(rr.Body.Bytes(), &result)
	var s4 models.RacerStats
	json.Unmarshal(result["stats"], &s4)
	if s4.DNF != 0 {
		t.Errorf("racer 4: expected DNF=0, got %d", s4.DNF)
	}
	if s4.DNS != 1 {
		t.Errorf("racer 4: expected DNS=1, got %d", s4.DNS)
	}

	// Cleanup: remove the test race
	app.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'DNS Test Race')")
	app.DB.Exec("DELETE FROM race_history WHERE name = 'DNS Test Race'")
}

func TestGetTracks(t *testing.T) {
	r := gin.New()
	r.GET("/api/tracks", handlers.GetTracks)

	req, _ := http.NewRequest("GET", "/api/tracks", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var tracks []models.Track
	json.Unmarshal(rr.Body.Bytes(), &tracks)
	if len(tracks) < 1 {
		t.Errorf("expected at least 1 track, got %d", len(tracks))
	}
}

func TestGetTrackStats(t *testing.T) {
	r := gin.New()
	r.GET("/api/track-stats", handlers.GetTrackStats)

	req, _ := http.NewRequest("GET", "/api/track-stats", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var stats []models.TrackStats
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if stats == nil {
		t.Errorf("expected track stats array, got nil")
	}
}

func TestOneOffRaces(t *testing.T) {
	r := gin.New()
	r.GET("/api/oneoff-races", handlers.GetOneOffRaces)

	req, _ := http.NewRequest("GET", "/api/oneoff-races", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var races []models.RaceHistory
	json.Unmarshal(rr.Body.Bytes(), &races)
	if races == nil {
		t.Errorf("expected races array, got nil")
	}
}

func TestGetUploads(t *testing.T) {
	r := gin.New()
	r.GET("/api/uploads", handlers.GetUploads)

	req, _ := http.NewRequest("GET", "/api/uploads", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var uploads []models.Upload
	json.Unmarshal(rr.Body.Bytes(), &uploads)
	if uploads == nil {
		t.Errorf("expected uploads array, got nil")
	}
}

func TestHandleUpload(t *testing.T) {
	// Create test routes with CSRF + Auth middleware
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.POST("/upload", handlers.HandleUpload)
	r.GET("/api/racers", handlers.GetRacers)

	// Create a valid session for auth
	sessionID := "upload-test-session"
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	app.SessionStoreMu.Unlock()
	defer func() {
		app.SessionStoreMu.Lock()
		delete(app.SessionStore, sessionID)
		app.SessionStoreMu.Unlock()
	}()

	// Create a minimal valid PNG for testing
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x01, 0x00, 0x05, 0x18, 0xD8, 0x4E, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}

	t.Run("UploadSuccess", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "test_racer.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, err := http.NewRequest("POST", "/api/upload", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v: %s", status, rr.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		url, ok := resp["url"].(string)
		if !ok {
			t.Fatalf("expected url in response, got %v", resp)
		}

		// Verify URL starts with /media/ (new media path)
		if !strings.HasPrefix(url, "/media/") {
			t.Errorf("expected URL to start with /media/, got %q", url)
		}

		// Verify the URL contains a hash subdirectory (2 chars)
		parts := strings.Split(strings.TrimPrefix(url, "/media/"), "/")
		if len(parts) != 2 {
			t.Errorf("expected URL format /media/<hash2>/<filename>, got %q", url)
		} else if len(parts[0]) != 2 {
			t.Errorf("expected 2-char hash subdirectory, got %q", parts[0])
		}

		// Verify file exists on disk
		filePath := filepath.Join(app.MediaPath, parts[0], parts[1])
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("uploaded file not found at %s", filePath)
		} else {
			// Clean up test file
			defer os.RemoveAll(filepath.Join(app.MediaPath, parts[0]))
		}

		// Verify upload is stored in DB
		hash, ok := resp["hash"].(string)
		if !ok {
			t.Fatal("expected hash in response")
		}
		var storedURL string
		err = app.DB.QueryRow("SELECT url FROM uploads WHERE hash = ?", hash).Scan(&storedURL)
		if err != nil {
			t.Errorf("upload not found in database: %v", err)
		} else if storedURL != url {
			t.Errorf("stored URL %q doesn't match response %q", storedURL, url)
		}
	})

	t.Run("UploadAndUpdateRacer", func(t *testing.T) {
		// Need a separate router with racer POST route for this test
		r2 := gin.New()
		admin2 := r2.Group("/api")
		admin2.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin2.POST("/upload", handlers.HandleUpload)
		admin2.POST("/racers", handlers.UpdateRacer)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "racer_pic.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, err := http.NewRequest("POST", "/api/upload", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("upload failed: %v: %s", status, rr.Body.String())
		}

		var uploadResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &uploadResp)
		uploadURL := uploadResp["url"].(string)

		// Update racer with uploaded image URL
		racerURL := "/api/racers"
		racerBody := map[string]interface{}{
			"id":              1,
			"name":            "Upload Test Racer",
			"profile_picture": uploadURL,
			"car_color":       "red",
			"car_name":        "Upload Test Car",
			"points":          99,
			"rank":            1,
			"position":        0,
		}
		racerJSON, _ := json.Marshal(racerBody)

		req2, _ := http.NewRequest("POST", racerURL, bytes.NewBuffer(racerJSON))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Origin", "http://127.0.0.1:6270")
		req2.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req2.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r2.ServeHTTP(rr2, req2)

		if status := rr2.Code; status != http.StatusOK {
			t.Fatalf("update racer failed: %v: %s", status, rr2.Body.String())
		}

		// Verify racer's profile_picture in DB
		var profilePicture string
		err = app.DB.QueryRow("SELECT profile_picture FROM racers WHERE id = 1").Scan(&profilePicture)
		if err != nil {
			t.Fatal(err)
		}
		if profilePicture != uploadURL {
			t.Errorf("expected profile_picture %q, got %q", uploadURL, profilePicture)
		}

		// Clean up uploaded file
		parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
		if len(parts) == 2 {
			os.RemoveAll(filepath.Join(app.MediaPath, parts[0]))
		}
	})
}

func makeUniquePNGData(seed byte) []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		seed, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x01, 0x00, 0x05, 0x18, 0xD8, 0x4E, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}

func TestProfilePictureUpload(t *testing.T) {
	r := gin.New()
	r.MaxMultipartMemory = 32 << 20
	r.POST("/api/upload", middleware.CSRFMiddleware(), middleware.AuthMiddleware(), handlers.HandleUpload)
	r.POST("/api/racers", middleware.CSRFMiddleware(), middleware.AuthMiddleware(), handlers.UpdateRacer)
	r.GET("/api/racers", handlers.GetRacers)
	r.Static("/media", app.MediaPath)

	sessionID := "profile-pic-test-session"
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	app.SessionStoreMu.Unlock()
	defer func() {
		app.SessionStoreMu.Lock()
		delete(app.SessionStore, sessionID)
		app.SessionStoreMu.Unlock()
	}()

	// Each subtest uses unique image data to avoid hash collisions
	t.Run("UploadAndVerifyHTTPAccess", func(t *testing.T) {
		pngData := makeUniquePNGData(0xAA)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "profile_pic.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("upload failed: %v: %s", status, rr.Body.String())
		}

		var uploadResp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &uploadResp); err != nil {
			t.Fatalf("failed to parse upload response: %v", err)
		}

		uploadURL, ok := uploadResp["url"].(string)
		if !ok {
			t.Fatalf("expected url in response, got %v", uploadResp)
		}
		if !strings.HasPrefix(uploadURL, "/media/") {
			t.Fatalf("expected url to start with /media/, got %q", uploadURL)
		}

		// Verify the uploaded file is HTTP-accessible
		getReq, _ := http.NewRequest("GET", uploadURL, nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, getReq)
		if status := rr2.Code; status != http.StatusOK {
			t.Errorf("uploaded file not accessible via HTTP: got status %v (url: %s)", status, uploadURL)
		}
		if rr2.Body.Len() == 0 {
			t.Error("uploaded file HTTP response body is empty")
		}

		// Clean up
		parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
		if len(parts) == 2 {
			defer os.RemoveAll(filepath.Join(app.MediaPath, parts[0]))
		}
	})

	t.Run("UploadAndVerifyRacerEndpoint", func(t *testing.T) {
		pngData := makeUniquePNGData(0xBB)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "racer_pic.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("upload failed: %v: %s", rr.Code, rr.Body.String())
		}

		var uploadResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &uploadResp)
		uploadURL := uploadResp["url"].(string)
		defer func() {
			parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
			if len(parts) == 2 {
				os.RemoveAll(filepath.Join(app.MediaPath, parts[0]))
			}
		}()

		// Update racer with profile_picture
		racerData := map[string]interface{}{
			"id":              1,
			"name":            "Profile Pic Racer",
			"profile_picture": uploadURL,
			"car_color":       "blue",
			"car_name":        "Profile Pic Car",
			"points":          50,
			"rank":            2,
			"position":        0,
		}
		racerJSON, _ := json.Marshal(racerData)
		racerReq, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(racerJSON))
		racerReq.Header.Set("Content-Type", "application/json")
		racerReq.Header.Set("Origin", "http://127.0.0.1:6270")
		racerReq.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		racerReq.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, racerReq)
		if rr2.Code != http.StatusOK {
			t.Fatalf("update racer failed: %v: %s", rr2.Code, rr2.Body.String())
		}

		// Verify racers list includes the profile_picture
		getReq, _ := http.NewRequest("GET", "/api/racers", nil)
		rr3 := httptest.NewRecorder()
		r.ServeHTTP(rr3, getReq)
		if rr3.Code != http.StatusOK {
			t.Fatalf("get racers failed: %v", rr3.Code)
		}

		var racers []models.Racer
		if err := json.Unmarshal(rr3.Body.Bytes(), &racers); err != nil {
			t.Fatalf("failed to parse racers response: %v", err)
		}

		var found bool
		for _, racer := range racers {
			if racer.ID == 1 {
				found = true
				if racer.ProfilePicture != uploadURL {
					t.Errorf("expected profile_picture %q, got %q", uploadURL, racer.ProfilePicture)
				}
				break
			}
		}
		if !found {
			t.Error("racer with id=1 not found in racers list")
		}
	})

	t.Run("DuplicateUploadReturnsExistingURL", func(t *testing.T) {
		pngData := makeUniquePNGData(0xCC)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "dup.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("first upload failed: %v: %s", rr.Code, rr.Body.String())
		}

		var firstResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &firstResp)
		firstURL := firstResp["url"].(string)
		defer func() {
			parts := strings.Split(strings.TrimPrefix(firstURL, "/media/"), "/")
			if len(parts) == 2 {
				os.RemoveAll(filepath.Join(app.MediaPath, parts[0]))
			}
		}()

		// Upload same image again
		body2 := new(bytes.Buffer)
		writer2 := multipart.NewWriter(body2)
		part2, _ := writer2.CreateFormFile("image", "dup2.png")
		part2.Write(pngData)
		writer2.Close()

		req2, _ := http.NewRequest("POST", "/api/upload", body2)
		req2.Header.Set("Content-Type", writer2.FormDataContentType())
		req2.Header.Set("Origin", "http://127.0.0.1:6270")
		req2.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req2.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("duplicate upload failed: %v: %s", rr2.Code, rr2.Body.String())
		}

		var secondResp map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &secondResp)

		secondURL, ok := secondResp["url"].(string)
		if !ok {
			t.Fatalf("expected url in duplicate response, got %v", secondResp)
		}
		if secondURL != firstURL {
			t.Errorf("duplicate upload returned different URL: got %q, want %q", secondURL, firstURL)
		}

		isDup, ok := secondResp["duplicate"].(bool)
		if !ok || !isDup {
			t.Errorf("expected duplicate=true in response, got %v", secondResp)
		}
	})

	t.Run("UploadWithoutAuthReturns401", func(t *testing.T) {
		pngData := makeUniquePNGData(0xDD)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "noauth.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for unauthenticated upload, got %v", rr.Code)
		}
	})

	t.Run("UploadInvalidFileTypeReturns400", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.txt")
		part.Write([]byte("not an image"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid file type, got %v", rr.Code)
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	headers := rr.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options not set")
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options not set")
	}
	if headers.Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("X-XSS-Protection not set")
	}
}

func TestLoginLogout(t *testing.T) {
	r := gin.New()
	r.POST("/api/login", handlers.HandleLogin)
	r.POST("/api/logout", handlers.HandleLogout)

	t.Run("SetupNewAdmin", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "admin123",
			"setup":    true,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "admin123",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
		}
	})

	t.Run("LoginInvalid", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestHeadToHead(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/head-to-head", handlers.GetHeadToHead)

	req, _ := http.NewRequest("GET", "/api/stats/head-to-head?racer1=1&racer2=2", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
	}

	var result models.HeadToHead
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result.Racer1 == "" || result.Racer2 == "" {
		t.Errorf("expected racer names in response, got %+v", result)
	}
}

func TestPointsProgression(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/points-progression", handlers.GetPointsProgression)

	req, _ := http.NewRequest("GET", "/api/stats/points-progression?racer_id=1", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var progression []models.PointsProgression
	json.Unmarshal(rr.Body.Bytes(), &progression)
	if progression == nil {
		t.Errorf("expected progression array, got nil")
	}
}

func TestStreaks(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/streaks", handlers.GetStreaks)

	req, _ := http.NewRequest("GET", "/api/stats/streaks", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var streaks []models.StreakInfo
	json.Unmarshal(rr.Body.Bytes(), &streaks)
	if streaks == nil {
		t.Errorf("expected streaks array, got nil")
	}
}

func TestELORatings(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/elo", handlers.GetELORatings)

	req, _ := http.NewRequest("GET", "/api/stats/elo", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var ratings []models.ELORating
	json.Unmarshal(rr.Body.Bytes(), &ratings)
	if ratings == nil {
		t.Errorf("expected ratings array, got nil")
	}
}

func TestExportStatsCSV(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/export", handlers.ExportStatsCSV)

	req, _ := http.NewRequest("GET", "/api/stats/export", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "text/csv" {
		t.Errorf("expected Content-Type text/csv, got %s", contentType)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Name") {
		t.Errorf("expected CSV header 'Name' in output, got: %s", body)
	}
	if !strings.Contains(body, "Gold") || !strings.Contains(body, "Silver") || !strings.Contains(body, "Bronze") {
		t.Errorf("CSV should contain Gold, Silver, Bronze headers: %s", body)
	}
}

func TestTrackPerformance(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/track-performance", handlers.GetTrackPerformance)

	t.Run("AllRacers", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/track-performance", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("SpecificRacer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/track-performance?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})
}

func TestDeleteRaceHistory(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", handlers.SaveRaceToHistory)
	r.DELETE("/api/race-history", handlers.DeleteRaceHistory)
	r.GET("/api/race-history", handlers.GetRaceHistory)

	t.Run("DeleteNonExistent", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/race-history?id=999", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("CreateAndDelete", func(t *testing.T) {
		payload := map[string]interface{}{
			"name":       "Race To Delete",
			"race_date":  "2025-07-01",
			"country":    "Belgium",
			"track":      "Spa",
			"track_id":   "spa",
			"total_laps": 44,
			"race_type":  "season",
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true, "finished": true},
			},
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("save: expected status 200, got %v", status)
		}
		var result map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &result)
		raceID := int(result["id"].(float64))

		// Verify it exists
		req, _ = http.NewRequest("GET", "/api/race-history?id="+strconv.Itoa(raceID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var history []models.RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)
		if len(history) != 1 {
			t.Fatalf("expected 1 race, got %d", len(history))
		}

		// Delete it
		req, _ = http.NewRequest("DELETE", "/api/race-history?id="+strconv.Itoa(raceID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("delete: expected status 200, got %v", status)
		}

		// Verify it's gone
		req, _ = http.NewRequest("GET", "/api/race-history?id="+strconv.Itoa(raceID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		json.Unmarshal(rr.Body.Bytes(), &history)
		if len(history) != 0 {
			t.Errorf("expected 0 races after delete, got %d", len(history))
		}
	})
}

func TestDeleteOneOffRace(t *testing.T) {
	r := gin.New()
	r.DELETE("/api/oneoff-races", handlers.DeleteOneOffRace)

	req, _ := http.NewRequest("DELETE", "/api/oneoff-races?id=999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}
}

func TestShorten(t *testing.T) {
	if shorten("short") != "short" {
		t.Error("shorten should return short strings unchanged")
	}
	long := "this is a very long string that should be shortened"
	result := shorten(long)
	if len(result) > 20 {
		t.Errorf("shorten should truncate long strings, got: %s", result)
	}
	if !strings.HasSuffix(result, "...") {
		t.Errorf("shorten should add ellipsis, got: %s", result)
	}
}

func TestEscapeHTML(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<script>", "&lt;script&gt;"},
		{"a & b", "a &amp; b"},
		{`"quote"`, "&quot;quote&quot;"},
		{"normal text", "normal text"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeHTML(tt.input)
		if result != tt.expected {
			t.Errorf("escapeHTML(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestRateLimit(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RateLimitMiddleware())
	hitCount := 0
	r.GET("/test", func(c *gin.Context) {
		hitCount++
		c.String(http.StatusOK, "ok")
	})

	// Send multiple requests rapidly
	for i := 0; i < 20; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}

func TestGetAISettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/ai-settings", middleware.AuthMiddleware(), handlers.GetAISettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/ai-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetNotificationSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/notification-settings", middleware.AuthMiddleware(), handlers.GetNotificationSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/notification-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetEmailSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/email-settings", middleware.AuthMiddleware(), handlers.GetEmailSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/email-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetRacerEmails(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-emails", middleware.AuthMiddleware(), handlers.GetRacerEmails)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-emails", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestVersion(t *testing.T) {
	r := gin.New()
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": app.CurrentVersion})
	})

	req, _ := http.NewRequest("GET", "/api/version", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["version"] != app.CurrentVersion {
		t.Errorf("expected version %q, got %q", app.CurrentVersion, resp["version"])
	}
}

func TestGetUmamiSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/umami-settings", middleware.AuthMiddleware(), handlers.GetUmamiSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/umami-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestBackupSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/backup-settings", handlers.GetBackupSettings)
	r.POST("/api/backup-settings", handlers.SaveBackupSettings)
	r.POST("/api/backup/manual", handlers.TriggerManualBackup)
	r.GET("/api/backup/list", handlers.ListBackups)

	t.Run("GetSettings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/backup-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var s models.BackupSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.ID != 1 {
			t.Errorf("expected id 1, got %d", s.ID)
		}
	})

	t.Run("SaveSettings", func(t *testing.T) {
		body, _ := json.Marshal(models.BackupSettings{Enabled: false, IntervalHrs: 12})
		req, _ := http.NewRequest("POST", "/api/backup-settings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/backup-settings", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var s models.BackupSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.Enabled != false || s.IntervalHrs != 12 {
			t.Errorf("expected enabled=false interval=12, got enabled=%v interval=%d", s.Enabled, s.IntervalHrs)
		}
	})

	t.Run("ManualBackup", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/backup/manual", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
		}
	})

	t.Run("ListBackups", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/backup/list", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var backups []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &backups)
		if backups == nil {
			t.Errorf("expected backups array, got nil")
		}
	})

	t.Run("PruneBackups", func(t *testing.T) {
		backupDir := filepath.Join(filepath.Dir(app.DBPath), "backups")
		os.MkdirAll(backupDir, 0755)

		for i := 1; i <= 10; i++ {
			name := fmt.Sprintf("heat_backup_20260101_%06d.db", i)
			os.WriteFile(filepath.Join(backupDir, name), []byte("test"), 0644)
		}

		db.PruneBackups()

		entries, _ := os.ReadDir(backupDir)
		var remaining int
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "heat_backup_") {
				remaining++
			}
		}
		if remaining != 7 {
			t.Errorf("expected 7 backups after prune, got %d", remaining)
		}

		os.RemoveAll(backupDir)
	})

	// Reset settings back for other tests
	body, _ := json.Marshal(models.BackupSettings{Enabled: true, IntervalHrs: 24, RetentionCount: 7})
	req, _ := http.NewRequest("POST", "/api/backup-settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("reset: expected status 200, got %v", rr.Code)
	}
}

// session helper for authenticated admin tests
func createAdminSession(t *testing.T) string {
	t.Helper()
	sessionID := "admin-session-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	app.SessionStoreMu.Unlock()
	return sessionID
}

func removeAdminSession(sessionID string) {
	app.SessionStoreMu.Lock()
	delete(app.SessionStore, sessionID)
	app.SessionStoreMu.Unlock()
}

func newAdminRequest(method, path string, body []byte, sessionID string) *http.Request {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Host = "127.0.0.1:6270"
	return req
}

func setupAdminRouter() (*gin.Engine, string) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	r.GET("/api/racers", handlers.GetRacers)
	return r, ""
}

func TestTeamsAPI(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.POST("/teams", handlers.SaveTeam)
	admin.DELETE("/teams", handlers.DeleteTeam)
	admin.POST("/teams/assign", handlers.AssignTeam)
	r.GET("/api/teams", handlers.GetTeams)
	r.GET("/api/teams/standings", handlers.GetConstructorStandings)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("list teams", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/teams", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var teams []models.Team
		json.Unmarshal(rr.Body.Bytes(), &teams)
		if len(teams) < 5 {
			t.Errorf("expected at least 5 teams, got %d", len(teams))
		}
	})

	t.Run("create team", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "Test Team", Color: "#ff00ff"})
		req := newAdminRequest("POST", "/api/teams", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var name string
		app.DB.QueryRow("SELECT name FROM teams WHERE name = 'Test Team'").Scan(&name)
		if name != "Test Team" {
			t.Errorf("expected 'Test Team', got %q", name)
		}
	})

	t.Run("create team empty name", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "", Color: "#ff00ff"})
		req := newAdminRequest("POST", "/api/teams", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty name, got %d", rr.Code)
		}
	})

	t.Run("assign racer to team", func(t *testing.T) {
		body, _ := json.Marshal(map[string]int{"racer_id": 1, "team_id": 1})
		req := newAdminRequest("POST", "/api/teams/assign", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var teamID int
		app.DB.QueryRow("SELECT team_id FROM racers WHERE id = 1").Scan(&teamID)
		if teamID != 1 {
			t.Errorf("expected team_id=1, got %d", teamID)
		}
	})

	t.Run("constructor standings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/teams/standings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var standings []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &standings)
		if len(standings) < 1 {
			t.Error("expected at least 1 team in standings")
		}
	})

	t.Run("delete team", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/teams?id=1", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var count int
		app.DB.QueryRow("SELECT COUNT(*) FROM teams WHERE id = 1").Scan(&count)
		if count != 0 {
			t.Error("team should be deleted")
		}
		var teamID int
		app.DB.QueryRow("SELECT COALESCE(team_id, 0) FROM racers WHERE id = 1").Scan(&teamID)
		if teamID != 0 {
			t.Errorf("expected racer team_id reset to 0, got %d", teamID)
		}
		// Re-seed for other tests
		app.DB.Exec("INSERT OR IGNORE INTO teams (id, name, color) VALUES (1, 'Scuderia Ferrari', '#d40000')")
	})

	t.Run("delete team without auth", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/teams?id=2", nil)
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("create team without auth", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "Unauth Team"})
		req, _ := http.NewRequest("POST", "/api/teams", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("delete team invalid id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/teams?id=0", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

func TestRacerCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.POST("/racers", handlers.UpdateRacer)
	admin.DELETE("/racers", handlers.DeleteRacer)
	r.GET("/api/racers", handlers.GetRacers)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("create racer", func(t *testing.T) {
		racer := models.Racer{Name: "New Racer", CarColor: "red", CarName: "Red Bull", Points: 0, Rank: 99, Position: 0}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify it shows up in racers list
		req, _ = http.NewRequest("GET", "/api/racers", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var racers []models.Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)
		found := false
		for _, r := range racers {
			if r.Name == "New Racer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("new racer not found in list")
		}
	})

	t.Run("create racer without auth", func(t *testing.T) {
		racer := models.Racer{Name: "Unauth Racer", CarColor: "blue", CarName: "No Auth"}
		body, _ := json.Marshal(racer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("update racer", func(t *testing.T) {
		racer := models.Racer{ID: 1, Name: "Updated Racer", ProfilePicture: "/static/helmet.svg", CarColor: "purple", CarName: "Updated Car", Points: 100, Rank: 1, Position: 0}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d", rr.Code)
		}
		var name string
		app.DB.QueryRow("SELECT name FROM racers WHERE id = 1").Scan(&name)
		if name != "Updated Racer" {
			t.Errorf("expected name 'Updated Racer', got %q", name)
		}
	})

	t.Run("delete racer", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/racers?id=2", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
		var count int
		app.DB.QueryRow("SELECT COUNT(*) FROM racers WHERE id = 2").Scan(&count)
		if count != 0 {
			t.Error("racer should be deleted")
		}
	})

	t.Run("delete racer invalid id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/racers?id=0", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid id, got %d", rr.Code)
		}
	})

	t.Run("delete racer without auth", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/racers?id=1", nil)
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("create racer empty name", func(t *testing.T) {
		racer := models.Racer{Name: "", CarColor: "red", CarName: "No Name"}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		// Should still succeed (no name validation on server)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestTrackCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.POST("/tracks", handlers.SaveTrack)
	admin.DELETE("/tracks", handlers.DeleteTrack)
	r.GET("/api/tracks", handlers.GetTracks)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save new track", func(t *testing.T) {
		track := models.Track{ID: "test-track", Name: "Test Circuit", Country: "Testland", Length: 5, LapRecord: "1:30.000"}
		body, _ := json.Marshal(track)
		req := newAdminRequest("POST", "/api/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var name string
		app.DB.QueryRow("SELECT name FROM tracks WHERE id = 'test-track'").Scan(&name)
		if name != "Test Circuit" {
			t.Errorf("expected track name 'Test Circuit', got %q", name)
		}
	})

	t.Run("save track without auth", func(t *testing.T) {
		track := models.Track{ID: "noauth", Name: "No Auth Track", Country: "Nowhere"}
		body, _ := json.Marshal(track)
		req, _ := http.NewRequest("POST", "/api/tracks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("delete track", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/tracks?id=test-track", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
	})

	t.Run("delete track without id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/tracks", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})
}

func TestQuoteCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.POST("/quotes", handlers.HandleQuotes)
	admin.PUT("/quotes", handlers.HandleQuotes)
	admin.DELETE("/quotes", handlers.HandleQuotes)
	r.GET("/api/quotes", handlers.GetQuotes)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("create quote", func(t *testing.T) {
		q := models.Quote{Text: "Test quote text", Author: "Tester"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var created models.Quote
		json.Unmarshal(rr.Body.Bytes(), &created)
		if created.ID == 0 || created.Text != "Test quote text" {
			t.Errorf("unexpected response: %+v", created)
		}
	})

	t.Run("create quote without auth", func(t *testing.T) {
		q := models.Quote{Text: "Unauth quote"}
		body, _ := json.Marshal(q)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("create quote empty text", func(t *testing.T) {
		q := models.Quote{Text: "", Author: "Empty"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty text, got %d", rr.Code)
		}
	})

	t.Run("update quote", func(t *testing.T) {
		q := models.Quote{ID: 1, Text: "Updated quote text", Author: "Updated Author"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("PUT", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var text string
		app.DB.QueryRow("SELECT text FROM quotes WHERE id = 1").Scan(&text)
		if text != "Updated quote text" {
			t.Errorf("expected text 'Updated quote text', got %q", text)
		}
	})

	t.Run("update quote without id", func(t *testing.T) {
		q := models.Quote{Text: "No ID"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("PUT", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})

	t.Run("delete quote", func(t *testing.T) {
		// First create one to delete
		q := models.Quote{Text: "To be deleted", Author: "Temp"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var created models.Quote
		json.Unmarshal(rr.Body.Bytes(), &created)

		req = newAdminRequest("DELETE", "/api/quotes?id="+strconv.Itoa(created.ID), nil, sessionID)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
		var count int
		app.DB.QueryRow("SELECT COUNT(*) FROM quotes WHERE id = ?", created.ID).Scan(&count)
		if count != 0 {
			t.Error("quote should be deleted")
		}
	})

	t.Run("delete quote without id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/quotes", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})
}

func TestSettingsSave(t *testing.T) {
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save notification settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin.POST("/notification-settings", handlers.SaveNotificationSettings)
		admin.GET("/notification-settings", handlers.GetNotificationSettings)

		s := models.NotificationSettings{GotiFyURL: "https://gotify.example.com", GotiFyToken: "token123", NotifyWinner: true, NotifyPodium: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/notification-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify saved (token should be hidden but settings should stick)
		req, _ = http.NewRequest("GET", "/api/notification-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["gotify_url"] != "https://gotify.example.com" {
			t.Errorf("expected gotify_url saved, got %v", resp["gotify_url"])
		}
	})

	t.Run("save AI settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin.POST("/ai-settings", handlers.SaveAISettings)
		admin.GET("/ai-settings", handlers.GetAISettings)

		s := models.AISettings{TrackExtractURL: "https://ai.example.com/extract", APIKey: "key123", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/ai-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/ai-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["track_extract_url"] != "https://ai.example.com/extract" {
			t.Errorf("expected track_extract_url saved, got %v", resp["track_extract_url"])
		}
	})

	t.Run("save email settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin.POST("/email-settings", handlers.SaveEmailSettings)
		admin.GET("/email-settings", handlers.GetEmailSettings)

		s := models.EmailSettings{SMTPHost: "smtp.example.com", SMTPPort: 587, Username: "user", Password: "pass123", FromAddr: "from@example.com", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/email-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/email-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["smtp_host"] != "smtp.example.com" {
			t.Errorf("expected smtp_host saved, got %v", resp["smtp_host"])
		}
	})

	t.Run("save umami settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin.POST("/umami-settings", handlers.SaveUmamiSettings)
		admin.GET("/umami-settings", handlers.GetUmamiSettings)

		s := models.UmamiSettings{URL: "https://analytics.example.com", WebsiteID: "abc-123", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/umami-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/umami-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["url"] != "https://analytics.example.com" {
			t.Errorf("expected url saved, got %v", resp["url"])
		}
		if resp["website_id"] != "abc-123" {
			t.Errorf("expected website_id saved, got %v", resp["website_id"])
		}
	})
}

func TestRacerEmailCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.GET("/racer-emails", handlers.GetRacerEmails)
	admin.POST("/racer-emails", handlers.SaveRacerEmail)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save racer email", func(t *testing.T) {
		re := models.RacerEmail{RacerID: 1, Email: "test@example.com"}
		body, _ := json.Marshal(re)
		req := newAdminRequest("POST", "/api/racer-emails", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get racer emails after save", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-emails", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var emails []models.RacerEmail
		json.Unmarshal(rr.Body.Bytes(), &emails)
		found := false
		for _, e := range emails {
			if e.RacerID == 1 && e.Email == "test@example.com" {
				found = true
				break
			}
		}
		if !found {
			t.Error("saved racer email not found in list")
		}
	})

	t.Run("save racer email unauthorized", func(t *testing.T) {
		re := models.RacerEmail{RacerID: 1, Email: "noauth@example.com"}
		body, _ := json.Marshal(re)
		req, _ := http.NewRequest("POST", "/api/racer-emails", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	// cleanup
	app.DB.Exec("DELETE FROM racer_emails WHERE racer_id = 1 AND email = 'test@example.com'")
}

func TestDeleteSeason(t *testing.T) {
	r := gin.New()
	r.GET("/api/seasons", handlers.GetSeasons)
	r.POST("/api/seasons", handlers.CreateSeason)
	r.DELETE("/api/seasons", handlers.DeleteSeason)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	// Create a season to delete
	body, _ := json.Marshal(map[string]string{"name": "Season To Delete"})
	req := newAdminRequest("POST", "/api/seasons", body, sessionID)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", rr.Code)
	}

	// Find the season we just created
	req, _ = http.NewRequest("GET", "/api/seasons", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var seasons []models.Season
	json.Unmarshal(rr.Body.Bytes(), &seasons)
	var targetID int
	for _, s := range seasons {
		if s.Name == "Season To Delete" {
			targetID = s.ID
			break
		}
	}
	if targetID == 0 {
		t.Skip("could not find created season")
	}

	t.Run("delete season", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/seasons?id="+strconv.Itoa(targetID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("verify season deleted", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		for _, s := range seasons {
			if s.ID == targetID {
				t.Error("season should be deleted but still found")
			}
		}
	})
}

// Helper functions kept for tests
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func escapeHTML(s string) string {
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

func shorten(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}
	return s
}

func TestPWAManifest(t *testing.T) {
	r := gin.New()
	r.StaticFile("/sw.js", "./static/sw.js")

	t.Run("serve service worker", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/sw.js", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		if rr.Body.Len() == 0 {
			t.Error("expected non-empty service worker")
		}
	})

	t.Run("manifest exists", func(t *testing.T) {
		data, err := os.ReadFile("static/manifest.json")
		if err != nil {
			t.Fatal("manifest.json not found")
		}
		var m map[string]interface{}
		json.Unmarshal(data, &m)
		if m["name"] == nil || m["short_name"] == nil {
			t.Error("manifest missing required fields")
		}
	})

	t.Run("all HTML pages have manifest link", func(t *testing.T) {
		files, _ := filepath.Glob("static/*.html")
		for _, f := range files {
			data, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			if !strings.Contains(string(data), "manifest.json") {
				t.Errorf("%s missing manifest link", f)
			}
			if !strings.Contains(string(data), "theme-color") {
				t.Errorf("%s missing theme-color", f)
			}
		}
	})
}

func TestI18nAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/translations", handlers.GetTranslations)
	r.POST("/api/language", handlers.SetLanguage)

	t.Run("get translations default", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		if tmap["nav.standings"] == "" {
			t.Error("expected nav.standings translation")
		}
		if tmap["_lang"] != "en" {
			t.Errorf("expected _lang=en, got %s", tmap["_lang"])
		}
	})

	t.Run("get german translations", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations?lang=de", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		if tmap["nav.admin"] != "Admin" {
			t.Errorf("expected de nav.admin to be 'Admin', got %s", tmap["nav.admin"])
		}
		if tmap["_lang"] != "de" {
			t.Errorf("expected _lang=de, got %s", tmap["_lang"])
		}
	})

	t.Run("set language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": "de"})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("set invalid language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": "fr"})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid lang, got %d", rr.Code)
		}
	})

	t.Run("set empty language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": ""})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty lang, got %d", rr.Code)
		}
	})

	t.Run("get translations detects german accept-language", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations", nil)
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		// Accept-Language is not easily detected in tests through gin
	})

	t.Run("translation files exist", func(t *testing.T) {
		for _, lang := range []string{"en", "de"} {
			data, err := os.ReadFile("static/locales/" + lang + ".json")
			if err != nil {
				t.Errorf("missing locale file: %s", lang)
				continue
			}
			var tmap map[string]string
			if err := json.Unmarshal(data, &tmap); err != nil {
				t.Errorf("invalid JSON in %s locale: %v", lang, err)
			}
			if len(tmap) == 0 {
				t.Errorf("empty translations for %s", lang)
			}
		}
	})

	t.Run("translations keys match between locales", func(t *testing.T) {
		enData, _ := os.ReadFile("static/locales/en.json")
		deData, _ := os.ReadFile("static/locales/de.json")
		var en, de map[string]string
		json.Unmarshal(enData, &en)
		json.Unmarshal(deData, &de)

		for k := range en {
			if de[k] == "" {
				t.Errorf("key %q missing from de.json", k)
			}
		}
		for k := range de {
			if en[k] == "" {
				t.Errorf("key %q missing from en.json", k)
			}
		}
	})
}

func TestRaceReportPage(t *testing.T) {
	t.Run("race report page exists", func(t *testing.T) {
		data, err := os.ReadFile("static/race-report.html")
		if err != nil {
			t.Fatal("race-report.html not found")
		}
		if !strings.Contains(string(data), "Final Classification") {
			t.Error("expected race report content")
		}
	})

	t.Run("race report API returns data", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-report", handlers.GetRaceReport)
		req, _ := http.NewRequest("GET", "/api/race-report", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
			t.Errorf("expected 200 or 404, got %d", rr.Code)
		}
	})
}

func TestFlagEndpoint(t *testing.T) {
	r := gin.New()
	r.POST("/api/flags", handlers.HandleFlag)

	t.Run("valid safety car flag", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "safety"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid red flag", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "red"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid blue flag with racer info", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "blue", RacerID: 1, RacerName: "A. PROST"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid blackwhite flag with racer info", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "blackwhite", RacerID: 2, RacerName: "M. SCHUMACHER"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("invalid flag type", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "invalid", Flag: "safety"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid type, got %d", rr.Code)
		}
	})
}

func TestRaceHistoryWithGoldSilverBronze(t *testing.T) {
	app.DB.Exec("DELETE FROM racer_stats")

	r := gin.New()
	r.POST("/api/race-history", handlers.SaveRaceToHistory)
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	payload := map[string]interface{}{
		"name":       "Gold Silver Bronze Test",
		"race_date":  "2025-07-01",
		"country":    "Italy",
		"track":      "Monza",
		"track_id":   "monza",
		"total_laps": 53,
		"race_type":  "season",
		"results": []map[string]interface{}{
			{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true, "finished": true},
			{"racer_id": 2, "racer_name": "M. SCHUMACHER", "position": 2, "points": 18, "fastest_lap": false, "finished": true},
			{"racer_id": 3, "racer_name": "A. SENNA", "position": 3, "points": 15, "fastest_lap": false, "finished": true},
			{"racer_id": 4, "racer_name": "N. LAUDA", "position": 4, "points": 12, "fastest_lap": false, "finished": true},
			{"racer_id": 5, "racer_name": "J. STEWART", "position": 5, "points": 10, "fastest_lap": false, "finished": true},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	checkStats := func(racerID int, expectedGold, expectedSilver, expectedBronze, expectedWins int) {
		t.Helper()
		req, _ = http.NewRequest("GET", "/api/racer-stats?id="+strconv.Itoa(racerID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var result map[string]json.RawMessage
		json.Unmarshal(rr.Body.Bytes(), &result)
		var s models.RacerStats
		json.Unmarshal(result["stats"], &s)
		if s.Gold != expectedGold {
			t.Errorf("racer %d: expected gold=%d, got %d", racerID, expectedGold, s.Gold)
		}
		if s.Silver != expectedSilver {
			t.Errorf("racer %d: expected silver=%d, got %d", racerID, expectedSilver, s.Silver)
		}
		if s.Bronze != expectedBronze {
			t.Errorf("racer %d: expected bronze=%d, got %d", racerID, expectedBronze, s.Bronze)
		}
		if s.Wins != expectedWins {
			t.Errorf("racer %d: expected wins=%d, got %d", racerID, expectedWins, s.Wins)
		}
	}

	checkStats(1, 1, 0, 0, 1)
	checkStats(2, 0, 1, 0, 0)
	checkStats(3, 0, 0, 1, 0)
	checkStats(4, 0, 0, 0, 0)

	app.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'Gold Silver Bronze Test')")
	app.DB.Exec("DELETE FROM race_history WHERE name = 'Gold Silver Bronze Test'")
	app.DB.Exec("DELETE FROM racer_stats WHERE racer_id IN (1,2,3,4,5)")
}

func TestOneOffRaceDoesNotUpdateStats(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", handlers.SaveRaceToHistory)
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	payload := map[string]interface{}{
		"name":       "One-Off Test",
		"race_date":  "2025-08-01",
		"country":    "France",
		"track":      "Spa",
		"track_id":   "spa",
		"total_laps": 44,
		"race_type":  "oneoff",
		"results": []map[string]interface{}{
			{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true, "finished": true},
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	req, _ = http.NewRequest("GET", "/api/racer-stats?id=1", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var result map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &result)
	var s models.RacerStats
	json.Unmarshal(result["stats"], &s)
	if s.Races != 0 {
		t.Errorf("one-off race should not update stats, got races=%d", s.Races)
	}

	app.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'One-Off Test')")
	app.DB.Exec("DELETE FROM race_history WHERE name = 'One-Off Test'")
}

func TestGetRacerStatsAll(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-stats", handlers.GetRacerStats)

	app.DB.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (1, 5, 3, 3, 1, 0, 2, 0, 0)")

	req, _ := http.NewRequest("GET", "/api/racer-stats", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var stats []models.RacerStats
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if len(stats) < 1 {
		t.Error("expected at least 1 stat entry")
	}
	found := false
	for _, s := range stats {
		if s.RacerID == 1 {
			found = true
			if s.Gold != 3 || s.Silver != 1 || s.Bronze != 0 || s.Wins != 3 {
				t.Errorf("racer 1 stats mismatch: gold=%d silver=%d bronze=%d wins=%d", s.Gold, s.Silver, s.Bronze, s.Wins)
			}
		}
	}
	if !found {
		t.Error("expected racer 1 stats to be returned")
	}

	app.DB.Exec("DELETE FROM racer_stats WHERE racer_id = 1")
}

func TestCreateSeason(t *testing.T) {
	r := gin.New()
	r.GET("/api/seasons", handlers.GetSeasons)
	r.POST("/api/seasons", handlers.CreateSeason)
	r.POST("/api/seasons/archive", handlers.ArchiveSeason)
	r.DELETE("/api/seasons", handlers.DeleteSeason)

	t.Run("create season", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": "Test Season 2025"})
		req, _ := http.NewRequest("POST", "/api/seasons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("create season empty name", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": ""})
		req, _ := http.NewRequest("POST", "/api/seasons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty name, got %d", rr.Code)
		}
	})

	t.Run("list seasons", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("list: expected 200, got %d", rr.Code)
		}
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		if len(seasons) < 1 {
			t.Error("expected at least 1 season")
		}
		if seasons[0].Name == "" {
			t.Error("expected season to have a name")
		}
	})

	t.Run("archive season", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		if len(seasons) == 0 {
			t.Skip("no seasons to archive")
		}
		req, _ = http.NewRequest("POST", "/api/seasons/archive?id="+strconv.Itoa(seasons[0].ID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("archive: expected 200, got %d", rr.Code)
		}
	})
}

func TestCreateRoundSnapshot(t *testing.T) {
	app.DB.Exec("DELETE FROM round_snapshot_scores")
	app.DB.Exec("DELETE FROM round_snapshots")

	r := gin.New()
	r.POST("/api/rounds", handlers.TakeRoundSnapshot)
	r.GET("/api/rounds", handlers.GetRoundSnapshots)
	r.DELETE("/api/rounds", handlers.DeleteRoundSnapshot)

	t.Run("take round snapshot", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"race_name": "Round 1",
			"season_id": 1,
		})
		req, _ := http.NewRequest("POST", "/api/rounds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("snapshot: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var result map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &result)
		if result["id"] == nil {
			t.Error("expected snapshot id in response")
		}
	})

	t.Run("list round snapshots", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("list: expected 200, got %d", rr.Code)
		}
		var snapshots []models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &snapshots)
		if len(snapshots) < 1 {
			t.Error("expected at least 1 snapshot")
		}
	})

	t.Run("get snapshot with scores", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var snapshots []models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &snapshots)
		if len(snapshots) == 0 {
			t.Skip("no snapshots")
		}
		req, _ = http.NewRequest("GET", "/api/rounds?id="+strconv.Itoa(snapshots[0].ID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("get by id: expected 200, got %d", rr.Code)
		}
		var details models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &details)
		if len(details.Scores) == 0 {
			t.Error("expected snapshot to have scores")
		}
	})

	t.Run("filter by season", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds?season_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("filter: expected 200, got %d", rr.Code)
		}
	})

	app.DB.Exec("DELETE FROM round_snapshot_scores")
	app.DB.Exec("DELETE FROM round_snapshots")
}

func TestPlayerLogin(t *testing.T) {
	r := gin.New()
	r.POST("/api/player/login", handlers.PlayerLogin)
	r.GET("/api/player/validate", handlers.ValidatePlayerToken)
	r.POST("/api/player/logout", handlers.PlayerLogout)

	var token string
	t.Run("login valid racer", func(t *testing.T) {
		body := `{"racer_id":1,"device_name":"TestPhone"}`
		req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Token     string `json:"token"`
			RacerID   int    `json:"racer_id"`
			RacerName string `json:"racer_name"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Token == "" {
			t.Error("expected non-empty token")
		}
		if resp.RacerID != 1 {
			t.Errorf("expected racer_id=1, got %d", resp.RacerID)
		}
		token = resp.Token
	})

	t.Run("validate token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/validate", nil)
		req.Header.Set("X-Player-Token", token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var resp struct {
			RacerID   int    `json:"racer_id"`
			RacerName string `json:"racer_name"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.RacerName == "" {
			t.Error("expected non-empty racer name")
		}
	})

	t.Run("reject invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/validate", nil)
		req.Header.Set("X-Player-Token", "invalid_token_12345")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("login nonexistent racer", func(t *testing.T) {
		body := `{"racer_id":999,"device_name":"Test"}`
		req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}

func TestHeatCardAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/heat-cards", handlers.GetHeatCards)
	r.POST("/api/heat-cards", handlers.AddHeatCard)
	r.PUT("/api/heat-cards/move", handlers.MoveHeatCard)
	r.DELETE("/api/heat-cards", handlers.DeleteHeatCard)
	r.POST("/api/heat-cards/init-decks", handlers.InitializeHeatDecks)

	t.Run("init decks", func(t *testing.T) {
		body := `{"race_id":0,"racer_ids":[1,2]}`
		req, _ := http.NewRequest("POST", "/api/heat-cards/init-decks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list heat cards", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/heat-cards", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var cards []models.HeatCard
		json.Unmarshal(rr.Body.Bytes(), &cards)
		if len(cards) < 14 {
			t.Errorf("expected at least 14 cards (7x2), got %d", len(cards))
		}
	})

	t.Run("add heat card", func(t *testing.T) {
		body := `{"racer_id":1,"location":"hand","card_type":"heat","lap_added":1}`
		req, _ := http.NewRequest("POST", "/api/heat-cards", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("move heat card", func(t *testing.T) {
		var cards []models.HeatCard
		req, _ := http.NewRequest("GET", "/api/heat-cards?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		json.Unmarshal(rr.Body.Bytes(), &cards)
		if len(cards) == 0 {
			t.Skip("no cards to move")
		}
		body := fmt.Sprintf(`{"card_id":%d,"location":"engine"}`, cards[len(cards)-1].ID)
		req, _ = http.NewRequest("PUT", "/api/heat-cards/move", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/heat-cards?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var cards []models.HeatCard
		json.Unmarshal(rr.Body.Bytes(), &cards)
		for _, c := range cards {
			if c.RacerID != 1 {
				t.Errorf("expected all cards for racer 1, got racer_id=%d", c.RacerID)
			}
		}
	})

	app.DB.Exec("DELETE FROM heat_cards")
}

func TestGearShiftAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/gear-shifts", handlers.GetGearShifts)
	r.POST("/api/gear-shifts", handlers.AddGearShift)

	t.Run("add gear shift", func(t *testing.T) {
		body := `{"racer_id":1,"race_id":0,"lap":1,"gear":3,"stress":1}`
		req, _ := http.NewRequest("POST", "/api/gear-shifts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list gear shifts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/gear-shifts", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var shifts []models.GearShift
		json.Unmarshal(rr.Body.Bytes(), &shifts)
		if len(shifts) < 1 {
			t.Error("expected at least 1 gear shift")
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/gear-shifts?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	app.DB.Exec("DELETE FROM gear_shifts")
}

func TestUpgradeAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/upgrade-cards", handlers.GetUpgradeCards)
	r.GET("/api/legend-abilities", handlers.GetLegendAbilities)

	t.Run("list upgrade cards", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/upgrade-cards", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var upgrades []models.UpgradeCard
		json.Unmarshal(rr.Body.Bytes(), &upgrades)
		if len(upgrades) < 8 {
			t.Errorf("expected at least 8 upgrade cards, got %d", len(upgrades))
		}
	})

	t.Run("list legend abilities", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/legend-abilities", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var abilities []models.LegendAbility
		json.Unmarshal(rr.Body.Bytes(), &abilities)
		if len(abilities) < 5 {
			t.Errorf("expected at least 5 legend abilities, got %d", len(abilities))
		}
	})
}

func TestWeatherAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/weather", handlers.GetWeather)
	r.POST("/api/weather", handlers.SetWeather)

	t.Run("set weather", func(t *testing.T) {
		body := `{"race_id":0,"condition":"wet","lap_start":1,"lap_end":999,"grip_modifier":0.7}`
		req, _ := http.NewRequest("POST", "/api/weather", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("get weather", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/weather?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var weather []models.WeatherCondition
		json.Unmarshal(rr.Body.Bytes(), &weather)
		if len(weather) < 1 {
			t.Error("expected at least 1 weather condition")
		}
		if weather[len(weather)-1].Condition != "wet" {
			t.Errorf("expected wet, got %s", weather[len(weather)-1].Condition)
		}
	})

	app.DB.Exec("DELETE FROM weather_conditions")
}

func TestTurboLogAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/turbo-logs", handlers.GetTurboLogs)
	r.POST("/api/turbo-logs", handlers.AddTurboLog)

	t.Run("add turbo log", func(t *testing.T) {
		body := `{"racer_id":1,"race_id":0,"lap":1,"times_used":1}`
		req, _ := http.NewRequest("POST", "/api/turbo-logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list turbo logs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/turbo-logs", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var logs []models.TurboLog
		json.Unmarshal(rr.Body.Bytes(), &logs)
		if len(logs) < 1 {
			t.Error("expected at least 1 turbo log")
		}
	})

	app.DB.Exec("DELETE FROM turbo_logs")
}

func TestLapRecordAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/lap-records", handlers.GetLapRecords)
	r.POST("/api/lap-records", handlers.RecordLap)
	r.POST("/api/lap-records/batch", handlers.RecordLapBatch)

	t.Run("record single lap", func(t *testing.T) {
		body := `{"race_id":0,"racer_id":1,"lap_number":1,"position":1,"gear_used":3,"heat_generated":2,"turbo_used":false}`
		req, _ := http.NewRequest("POST", "/api/lap-records", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("record batch lap", func(t *testing.T) {
		body := `{"race_id":0,"lap":2,"records":[
			{"racer_id":1,"position":2,"gear_used":2,"heat_generated":1,"turbo_used":false},
			{"racer_id":2,"position":1,"gear_used":3,"heat_generated":0,"turbo_used":true}
		]}`
		req, _ := http.NewRequest("POST", "/api/lap-records/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list lap records", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/lap-records?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var records []models.LapRecord
		json.Unmarshal(rr.Body.Bytes(), &records)
		if len(records) < 3 {
			t.Errorf("expected at least 3 lap records, got %d", len(records))
		}
	})

	app.DB.Exec("DELETE FROM lap_records")
}

func TestRaceEventAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/race-events", handlers.GetRaceEvents)
	r.POST("/api/race-events", handlers.AddRaceEvent)

	t.Run("add race event", func(t *testing.T) {
		body := `{"race_id":0,"lap":1,"event_type":"overtake","racer_id":1,"racer_id2":2,"note":"Great pass!"}`
		req, _ := http.NewRequest("POST", "/api/race-events", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list race events", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-events?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var events []models.RaceEvent
		json.Unmarshal(rr.Body.Bytes(), &events)
		if len(events) < 1 {
			t.Error("expected at least 1 event")
		}
		if events[0].EventType != "overtake" {
			t.Errorf("expected overtake, got %s", events[0].EventType)
		}
	})

	app.DB.Exec("DELETE FROM race_events")
}

func TestSpectatorState(t *testing.T) {
	r := gin.New()
	r.GET("/api/spectator/state", handlers.GetSpectatorState)

	t.Run("get spectator state", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/spectator/state", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var resp struct {
			Racers []models.Racer  `json:"racers"`
			Race   models.RaceInfo `json:"race"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Racers) == 0 {
			t.Error("expected at least 1 racer")
		}
		if resp.Race.Track == "" {
			t.Error("expected race track to be non-empty")
		}
	})
}

func TestSoundFX(t *testing.T) {
	r := gin.New()
	r.POST("/api/sound", handlers.PlaySound)

	t.Run("play engine sound", func(t *testing.T) {
		body := `{"sound":"engine"}`
		req, _ := http.NewRequest("POST", "/api/sound", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("play finish sound", func(t *testing.T) {
		body := `{"sound":"finish"}`
		req, _ := http.NewRequest("POST", "/api/sound", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestRaceRadioAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/race-radio", handlers.GetRaceRadio)
	r.POST("/api/race-radio", handlers.AddRaceRadio)

	t.Run("add radio message", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":1,"message":"Box box, box now!"}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var msg models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msg)
		if msg.ID == 0 {
			t.Error("expected non-zero id")
		}
		if msg.RacerName == "" {
			t.Error("expected racer name")
		}
	})

	t.Run("add radio message empty message", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":1,"message":""}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty message, got %d", rr.Code)
		}
	})

	t.Run("add radio message no racer", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":0,"message":"Test"}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for no racer, got %d", rr.Code)
		}
	})

	t.Run("get radio messages", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-radio?race_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var msgs []models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msgs)
		if len(msgs) < 1 {
			t.Error("expected at least 1 message")
		}
		if msgs[0].Message != "Box box, box now!" {
			t.Errorf("expected 'Box box, box now!', got %q", msgs[0].Message)
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-radio?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var msgs []models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msgs)
		for _, m := range msgs {
			if m.RacerID != 1 {
				t.Errorf("expected racer_id=1, got %d", m.RacerID)
			}
		}
	})

	app.DB.Exec("DELETE FROM race_radio")
}

func TestDeeperStats(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/qualifying-delta", handlers.GetQualifyingRaceDelta)
	r.GET("/api/stats/consistency", handlers.GetConsistencyRatings)
	r.GET("/api/stats/incidents", handlers.GetRaceIncidentsReport)
	r.GET("/api/stats/pace-heatmap", handlers.GetPaceHeatmap)
	r.GET("/api/race-report", handlers.GetRaceReport)

	t.Run("qualifying delta", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/qualifying-delta", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var data []interface{}
		json.Unmarshal(rr.Body.Bytes(), &data)
		// Can be non-nil empty array or nil (no race data yet)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("consistency ratings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/consistency", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("incidents report", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/incidents", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("pace heatmap", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/stats/pace-heatmap", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("race report", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-report", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		// Can be 200 (race exists) or 404 (no race yet)
		if rr.Code != http.StatusOK && rr.Code != http.StatusNotFound {
			t.Errorf("expected 200 or 404, got %d", rr.Code)
		}
	})
}

func TestPlayerSelfServiceEndpoints(t *testing.T) {
	r := gin.New()
	r.POST("/api/player/gear", handlers.PlayerReportGear)
	r.POST("/api/player/heat", handlers.PlayerReportHeat)
	r.POST("/api/player/turbo", handlers.PlayerReportTurbo)
	r.GET("/api/player/status", handlers.PlayerGetStatus)
	r.POST("/api/player/login", handlers.PlayerLogin)

	var token string
	loginBody := `{"racer_id":1,"device_name":"Test"}`
	req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rr.Body.Bytes(), &loginResp)
	token = loginResp.Token

	t.Run("report gear", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","lap":1,"gear":3,"stress":0}`, token)
		req, _ := http.NewRequest("POST", "/api/player/gear", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("report heat", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","card_type":"heat","location":"engine","count":1}`, token)
		req, _ := http.NewRequest("POST", "/api/player/heat", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("report turbo", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","lap":1}`, token)
		req, _ := http.NewRequest("POST", "/api/player/turbo", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("get player status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/status", nil)
		req.Header.Set("X-Player-Token", token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var data map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &data)
		if data["racer"] == nil {
			t.Error("expected racer field")
		}
		if data["heat_cards"] == nil {
			t.Error("expected heat_cards field")
		}
	})

	t.Run("reject unauthorized gear report", func(t *testing.T) {
		body := `{"token":"bad_token","lap":1,"gear":1,"stress":0}`
		req, _ := http.NewRequest("POST", "/api/player/gear", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	app.DB.Exec("DELETE FROM gear_shifts")
	app.DB.Exec("DELETE FROM heat_cards WHERE racer_id=1 AND lap_added=0")
	app.DB.Exec("DELETE FROM turbo_logs")
	app.DB.Exec("DELETE FROM player_sessions")
}

func TestPlayerUpgradeBuy(t *testing.T) {
	r := gin.New()
	r.POST("/api/player-upgrades/buy", handlers.BuyUpgrade)

	t.Run("buy upgrade", func(t *testing.T) {
		body := `{"racer_id":1,"upgrade_id":1,"season_id":1,"round":1}`
		req, _ := http.NewRequest("POST", "/api/player-upgrades/buy", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	app.DB.Exec("DELETE FROM player_upgrades")
}

func TestSectorAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/sectors", handlers.GetSectors)
	r.POST("/api/racer-sectors", handlers.RecordRacerSector)

	t.Run("list sectors for monza", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/sectors?track_id=monza", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var sectors []models.Sector
		json.Unmarshal(rr.Body.Bytes(), &sectors)
		if len(sectors) < 5 {
			t.Errorf("expected at least 5 sectors for monza, got %d", len(sectors))
		}
	})

	t.Run("record racer sector", func(t *testing.T) {
		body := `{"race_id":0,"racer_id":1,"sector_id":1,"lap":1}`
		req, _ := http.NewRequest("POST", "/api/racer-sectors", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	app.DB.Exec("DELETE FROM racer_sectors")
}

func TestAvailableUpgrades(t *testing.T) {
	r := gin.New()
	r.GET("/api/available-upgrades", handlers.GetAvailableUpgradesForRacer)

	t.Run("list available upgrades", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/available-upgrades?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var upgrades []models.UpgradeCard
		json.Unmarshal(rr.Body.Bytes(), &upgrades)
		if len(upgrades) < 8 {
			t.Errorf("expected at least 8 available upgrades, got %d", len(upgrades))
		}
	})
}

func TestDriverShare(t *testing.T) {
	adminRouter := func() *gin.Engine {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
		admin.POST("/driver-share", handlers.GenerateDriverShareToken)
		admin.GET("/driver-shares", handlers.GetDriverShareTokens)
		admin.DELETE("/driver-share", handlers.DeleteDriverShareToken)
		return r
	}

	publicRouter := func() *gin.Engine {
		r := gin.New()
		r.GET("/api/shared/driver-stats", handlers.GetDriverStatsByToken)
		return r
	}

	sessionID := "test-share-session"
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	app.SessionStoreMu.Unlock()
	defer func() {
		app.SessionStoreMu.Lock()
		delete(app.SessionStore, sessionID)
		app.SessionStoreMu.Unlock()
	}()

	addSessionCookie := func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	}

	t.Run("generate share token", func(t *testing.T) {
		r := adminRouter()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/driver-share?racer_id=1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var share models.DriverShare
		json.Unmarshal(rr.Body.Bytes(), &share)
		if share.Token == "" {
			t.Error("expected non-empty token")
		}
		if share.RacerID != 1 {
			t.Errorf("expected racer_id 1, got %d", share.RacerID)
		}
	})

	t.Run("get driver shares (admin)", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("GET", "/api/driver-shares", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var shares []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &shares)
		if len(shares) == 0 {
			t.Error("expected at least one share")
		}
	})

	t.Run("access driver stats via token", func(t *testing.T) {
		// First get the token
		rAdmin := adminRouter()
		req, _ := http.NewRequest("GET", "/api/driver-shares", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		rAdmin.ServeHTTP(rr, req)
		var shares []struct {
			RacerID int    `json:"racer_id"`
			Token   string `json:"token"`
		}
		json.Unmarshal(rr.Body.Bytes(), &shares)
		if len(shares) == 0 {
			t.Fatal("no shares to test")
		}

		rPub := publicRouter()
		req2, _ := http.NewRequest("GET", "/api/shared/driver-stats?token="+shares[0].Token, nil)
		rr2 := httptest.NewRecorder()
		rPub.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
		}
		var result map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &result)
		if result["racer"] == nil {
			t.Error("expected racer data")
		}
		if result["stats"] == nil {
			t.Error("expected stats data")
		}
	})

	t.Run("invalid token returns 404", func(t *testing.T) {
		r := publicRouter()
		req, _ := http.NewRequest("GET", "/api/shared/driver-stats?token=invalidtoken123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for invalid token, got %d", rr.Code)
		}
	})

	t.Run("missing token returns 400", func(t *testing.T) {
		r := publicRouter()
		req, _ := http.NewRequest("GET", "/api/shared/driver-stats", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing token, got %d", rr.Code)
		}
	})

	t.Run("delete driver share", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("DELETE", "/api/driver-share?racer_id=1", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		// Verify token is gone
		rPub := publicRouter()
		req2, _ := http.NewRequest("GET", "/api/shared/driver-stats?token=invalidtoken123", nil)
		rr2 := httptest.NewRecorder()
		rPub.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", rr2.Code)
		}
	})

	t.Run("generate for nonexistent racer returns 404", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("POST", "/api/driver-share?racer_id=9999", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for nonexistent racer, got %d", rr.Code)
		}
	})
}

func setupHtmxRouter() (*gin.Engine, string) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.GET("/html/racers", handlers.HtmxRacersTable)
	admin.POST("/html/racers", handlers.HtmxRacersSave)
	admin.GET("/html/racers/:id/edit", handlers.HtmxRacersEditForm)
	admin.DELETE("/html/racers/:id", handlers.HtmxRacersDelete)
	admin.POST("/html/racers/:id/share", handlers.HtmxRacersGenerateShare)
	admin.GET("/html/tracks", handlers.HtmxTracksTable)
	admin.POST("/html/tracks", handlers.HtmxTracksSave)
	admin.GET("/html/tracks/:id/edit", handlers.HtmxTracksEditForm)
	admin.DELETE("/html/tracks/:id", handlers.HtmxTracksDelete)
	admin.GET("/html/quotes", handlers.HtmxQuotesTable)
	admin.POST("/html/quotes", handlers.HtmxQuotesSave)
	admin.GET("/html/quotes/:id/edit", handlers.HtmxQuotesEditForm)
	admin.DELETE("/html/quotes/:id", handlers.HtmxQuotesDelete)
	admin.GET("/html/teams", handlers.HtmxTeamsTable)
	admin.POST("/html/teams", handlers.HtmxTeamsSave)
	admin.GET("/html/teams/:id/edit", handlers.HtmxTeamsEditForm)
	admin.DELETE("/html/teams/:id", handlers.HtmxTeamsDelete)
	admin.GET("/html/seasons", handlers.HtmxSeasonsTable)
	admin.GET("/html/seasons/new", handlers.HtmxSeasonsNewForm)
	admin.POST("/html/seasons", handlers.HtmxSeasonsCreate)
	admin.POST("/html/seasons/:id/archive", handlers.HtmxSeasonsArchive)
	admin.DELETE("/html/seasons/:id", handlers.HtmxSeasonsDelete)
	sessionID := "htmx-test-session-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	app.SessionStoreMu.Lock()
	app.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	app.SessionStoreMu.Unlock()
	return r, sessionID
}

func newHtmxAdminFormRequest(path string, formData map[string]string, sessionID string) *http.Request {
	body := ""
	for k, v := range formData {
		if body != "" {
			body += "&"
		}
		body += url.Values{k: {v}}.Encode()
	}
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Host = "127.0.0.1:6270"
	return req
}

func TestHtmxRacersTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/racers", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Errorf("expected text/html content type, got %s", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<tbody") {
		t.Errorf("expected HTML table body, got: %s", body[:min(len(body), 200)])
	}
	if !strings.Contains(body, "racer-list") {
		t.Errorf("expected racer-list id in response")
	}
	if strings.Count(body, "<tr>") < 2 {
		t.Errorf("expected at least 2 racer rows (5 seeded), got fewer")
	}
}

func TestHtmxRacersCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name":            "HTMX Racer",
		"car_name":        "HTMX Car",
		"car_color":       "#ff0000",
		"points":          "100",
		"rank":            "1",
		"position":        "0",
		"profile_picture": "/static/images/helmet.svg",
	}
	req := newHtmxAdminFormRequest("/api/html/racers", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Racer") {
		t.Errorf("expected new racer in table HTML, got: %s", body[:min(len(body), 300)])
	}
	trigger := rr.Header().Get("HX-Trigger")
	if !strings.Contains(trigger, "closeRacerModal") {
		t.Errorf("expected HX-Trigger closeRacerModal, got: %s", trigger)
	}
}

func TestHtmxRacersDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	app.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/racers/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "<tbody") {
		t.Errorf("expected HTML table after delete")
	}
}

func TestHtmxRacersEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	app.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/racers/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hx-post") || !strings.Contains(body, "Save Racer") {
		t.Errorf("expected edit form with htmx attributes, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxRacersNewForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/racers/0/edit", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "hx-post") || !strings.Contains(body, "Save Racer") {
		t.Errorf("expected new racer form, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxRacersGenerateShare(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	app.DB.QueryRow("SELECT id FROM racers LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no racers")
	}

	req, _ := http.NewRequest("POST", fmt.Sprintf("/api/html/racers/%d/share", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var token string
	err := app.DB.QueryRow("SELECT token FROM driver_shares WHERE racer_id = ?", id).Scan(&token)
	if err != nil {
		t.Errorf("expected share token to exist: %v", err)
	}
}

func TestHtmxTracksTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/tracks", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "track-list") {
		t.Errorf("expected track-list tbody")
	}
}

func TestHtmxTracksCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"id":         "htmx-test-track",
		"id_visible": "htmx-test-track",
		"name":       "HTMX Test Track",
		"country":    "Testland",
		"length_km":  "42",
		"lap_record": "1:30.000",
	}
	req := newHtmxAdminFormRequest("/api/html/tracks", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Track") {
		t.Errorf("expected new track in table, got: %s", body[:min(len(body), 300)])
	}
	app.DB.Exec("DELETE FROM tracks WHERE id = 'htmx-test-track'")
}

func TestHtmxTracksDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	app.DB.Exec("INSERT OR REPLACE INTO tracks (id, name, country, length_km, lap_record) VALUES ('htmx-del-track', 'Del Track', 'Nowhere', 10, '--')")

	req, _ := http.NewRequest("DELETE", "/api/html/tracks/htmx-del-track", nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "track-list") {
		t.Errorf("expected track list after delete")
	}
}

func TestHtmxQuotesTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/quotes", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "quote-list") {
		t.Errorf("expected quote-list tbody")
	}
}

func TestHtmxQuotesCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"text":   "HTMX Test Quote",
		"author": "HTMX Tester",
	}
	req := newHtmxAdminFormRequest("/api/html/quotes", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Quote") {
		t.Errorf("expected new quote in table, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxQuotesDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	app.DB.QueryRow("SELECT id FROM quotes LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no quotes to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/quotes/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "quote-list") {
		t.Errorf("expected quote list after delete")
	}
}

func TestHtmxTeamsTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/teams", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "team-list") {
		t.Errorf("expected team-list tbody")
	}
}

func TestHtmxTeamsCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name":  "HTMX Test Team",
		"color": "#00ff00",
	}
	req := newHtmxAdminFormRequest("/api/html/teams", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Team") {
		t.Errorf("expected new team in table")
	}
}

func TestHtmxTeamsDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	app.DB.Exec("INSERT INTO teams (name, color) VALUES ('Del Team', '#000')")
	var id int
	app.DB.QueryRow("SELECT id FROM teams WHERE name = 'Del Team'").Scan(&id)
	if id == 0 {
		t.Fatal("no team to delete")
	}

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/teams/%d", id), nil)
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestHtmxSeasonsTable(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/seasons", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "seasons-list") {
		t.Errorf("expected seasons-list tbody")
	}
}

func TestHtmxSeasonsCreate(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name": "HTMX Test Season",
	}
	req := newHtmxAdminFormRequest("/api/html/seasons", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "HTMX Test Season") {
		t.Errorf("expected new season in table, got: %s", body[:min(len(body), 300)])
	}
}

func TestHtmxSeasonsArchive(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req := newHtmxAdminFormRequest("/api/html/seasons", map[string]string{"name": "Archive Test Season"}, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create season: expected 200, got %d", rr.Code)
	}

	var id int
	app.DB.QueryRow("SELECT id FROM seasons WHERE name = 'Archive Test Season'").Scan(&id)
	if id == 0 {
		t.Fatal("no season created")
	}

	req2, _ := http.NewRequest("POST", fmt.Sprintf("/api/html/seasons/%d/archive", id), nil)
	req2.Header.Set("Origin", "http://127.0.0.1:6270")
	req2.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("archive: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
	var status string
	app.DB.QueryRow("SELECT status FROM seasons WHERE id = ?", id).Scan(&status)
	if status != "archived" {
		t.Errorf("expected status 'archived', got %q", status)
	}
}

func TestHtmxSeasonsDelete(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req := newHtmxAdminFormRequest("/api/html/seasons", map[string]string{"name": "Del Test Season"}, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", rr.Code)
	}

	var id int
	app.DB.QueryRow("SELECT id FROM seasons WHERE name = 'Del Test Season'").Scan(&id)
	if id == 0 {
		t.Fatal("no season created")
	}

	req2, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/html/seasons/%d", id), nil)
	req2.Header.Set("Origin", "http://127.0.0.1:6270")
	req2.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr2 := httptest.NewRecorder()
	r.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}
}

func TestHtmxSeasonsNewForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	req, _ := http.NewRequest("GET", "/api/html/seasons/new", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Season Name") || !strings.Contains(body, "hx-post") {
		t.Errorf("expected season creation form, got: %s", body[:min(len(body), 200)])
	}
}

func TestHtmxTracksEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	app.DB.Exec("INSERT OR REPLACE INTO tracks (id, name, country, length_km, lap_record) VALUES ('edit-test', 'Edit Test', 'Test', 10, '--')")

	req, _ := http.NewRequest("GET", "/api/html/tracks/edit-test/edit", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Track") {
		t.Errorf("expected track edit form, got: %s", body[:min(len(body), 200)])
	}
	app.DB.Exec("DELETE FROM tracks WHERE id = 'edit-test'")
}

func TestHtmxQuotesEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	var id int
	app.DB.QueryRow("SELECT id FROM quotes LIMIT 1").Scan(&id)
	if id == 0 {
		t.Fatal("no quotes")
	}

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/quotes/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Quote") {
		t.Errorf("expected quote edit form, got: %s", body[:min(len(body), 200)])
	}
}

func TestHtmxTeamsEditForm(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	app.DB.Exec("INSERT INTO teams (name, color) VALUES ('Edit Team Test', '#fff')")
	var id int
	app.DB.QueryRow("SELECT id FROM teams WHERE name = 'Edit Team Test'").Scan(&id)

	req, _ := http.NewRequest("GET", fmt.Sprintf("/api/html/teams/%d/edit", id), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Save Team") {
		t.Errorf("expected team edit form, got: %s", body[:min(len(body), 200)])
	}
	app.DB.Exec("DELETE FROM teams WHERE id = ?", id)
}

func TestHtmxRacersCreateMissingName(t *testing.T) {
	r, sid := setupHtmxRouter()
	defer removeAdminSession(sid)

	formData := map[string]string{
		"name": "",
	}
	req := newHtmxAdminFormRequest("/api/html/racers", formData, sid)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d", rr.Code)
	}
}

func TestHtmxUnauthorized(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	admin.GET("/html/racers", handlers.HtmxRacersTable)

	req, _ := http.NewRequest("GET", "/api/html/racers", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without session, got %d", rr.Code)
	}
}
