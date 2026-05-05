package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	r.DELETE("/api/race-history", handlers.DeleteRaceHistory)

	req, _ := http.NewRequest("DELETE", "/api/race-history?id=999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}
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
