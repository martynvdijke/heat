package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"heat/app"
	"heat/db"
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
	app.DB, err = sql.Open("sqlite3", app.DBPath)
	if err != nil {
		log.Fatalf("failed to open in-memory db: %v", err)
	}
	app.DB.SetMaxOpenConns(1)

	db.Init()
	go ws.BroadcastManager()

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
		RacerID: 3, Races: 10, Wins: 4, Podiums: 7, FastestLaps: 3, DNF: 1, DNS: 2,
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
	if s.Races != 10 || s.Wins != 4 || s.Podiums != 7 || s.FastestLaps != 3 || s.DNF != 1 || s.DNS != 2 {
		t.Errorf("expected stats (10,4,7,3,1,2), got (%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Podiums, s.FastestLaps, s.DNF, s.DNS)
	}

	// Find the actual DB id for the update
	var statsID int
	app.DB.QueryRow("SELECT id FROM racer_stats WHERE racer_id = 3").Scan(&statsID)

	// Update existing stats
	updateBody, _ := json.Marshal(models.RacerStats{
		ID: statsID, RacerID: 3, Races: 20, Wins: 8, Podiums: 15, FastestLaps: 6, DNF: 2, DNS: 1,
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
	if s.Races != 20 || s.Wins != 8 || s.Podiums != 15 || s.FastestLaps != 6 || s.DNF != 2 || s.DNS != 1 {
		t.Errorf("expected updated stats (20,8,15,6,2,1), got (%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Podiums, s.FastestLaps, s.DNF, s.DNS)
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

	// Reset settings back for other tests
	body, _ := json.Marshal(models.BackupSettings{Enabled: true, IntervalHrs: 24})
	req, _ := http.NewRequest("POST", "/api/backup-settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("reset: expected status 200, got %v", rr.Code)
	}
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
