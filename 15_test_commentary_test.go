package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func TestCommentaryAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/commentary", testHandler.GetCommentary)
	r.POST("/api/commentary", testHandler.AddCommentary)
	r.POST("/api/race-events", testHandler.AddRaceEvent)
	r.POST("/api/weather", testHandler.SetWeather)

	testServer.DB.Exec("DELETE FROM commentary")

	t.Run("manual entry round-trip", func(t *testing.T) {
		body := `{"race_id":0,"lap":3,"racer_id":1,"message":"Ladies and gentlemen, what a race!"}`
		req, _ := http.NewRequest("POST", "/api/commentary", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var entry models.Commentary
		json.Unmarshal(rr.Body.Bytes(), &entry)
		if entry.ID == 0 {
			t.Error("expected non-zero id")
		}
		if entry.Message != "Ladies and gentlemen, what a race!" {
			t.Errorf("expected message, got %q", entry.Message)
		}
		if entry.RacerName == "" {
			t.Error("expected racer name")
		}
		if entry.TemplateKey != "" {
			t.Errorf("expected empty template_key for manual entry, got %q", entry.TemplateKey)
		}
	})

	t.Run("manual entry empty message rejected", func(t *testing.T) {
		body := `{"race_id":0,"lap":1,"message":""}`
		req, _ := http.NewRequest("POST", "/api/commentary", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty message, got %d", rr.Code)
		}
	})

	t.Run("get returns newest-first", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/commentary?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var entries []models.Commentary
		json.Unmarshal(rr.Body.Bytes(), &entries)
		if len(entries) < 1 {
			t.Fatal("expected at least 1 entry")
		}
		if entries[0].Message != "Ladies and gentlemen, what a race!" {
			t.Errorf("expected newest-first ordering, got %q", entries[0].Message)
		}
	})

	t.Run("since filter returns only newer entries", func(t *testing.T) {
		// Capture the current max id, then add a new entry.
		var maxID int
		testServer.DB.QueryRow("SELECT COALESCE(MAX(id), 0) FROM commentary").Scan(&maxID)

		body := `{"race_id":0,"lap":4,"message":"And now for something completely different."}`
		req, _ := http.NewRequest("POST", "/api/commentary", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		req2, _ := http.NewRequest("GET", "/api/commentary?race_id=0&since="+strconv.Itoa(maxID), nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr2.Code)
		}
		var entries []models.Commentary
		json.Unmarshal(rr2.Body.Bytes(), &entries)
		if len(entries) != 1 {
			t.Fatalf("expected exactly 1 entry after since=%d, got %d", maxID, len(entries))
		}
		if entries[0].Message != "And now for something completely different." {
			t.Errorf("expected the new entry, got %q", entries[0].Message)
		}
	})

	t.Run("limit filter caps response", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/commentary?race_id=0&limit=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var entries []models.Commentary
		json.Unmarshal(rr.Body.Bytes(), &entries)
		if len(entries) != 1 {
			t.Errorf("expected 1 entry with limit=1, got %d", len(entries))
		}
	})

	t.Run("race event auto-generates commentary with driver substitution", func(t *testing.T) {
		body := `{"race_id":0,"lap":5,"event_type":"overtake","racer_id":1,"racer_id2":2}`
		req, _ := http.NewRequest("POST", "/api/race-events", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		req2, _ := http.NewRequest("GET", "/api/commentary?race_id=0&limit=1", nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr2.Code)
		}
		var entries []models.Commentary
		json.Unmarshal(rr2.Body.Bytes(), &entries)
		if len(entries) < 1 {
			t.Fatal("expected auto-generated commentary entry")
		}
		entry := entries[0]
		if entry.TemplateKey != "overtake" {
			t.Errorf("expected template_key 'overtake', got %q", entry.TemplateKey)
		}
		if !strings.Contains(entry.Message, "lap 5") {
			t.Errorf("expected lap substitution in %q", entry.Message)
		}
		// Driver name should be substituted (racer 1 exists in seeded data).
		var name string
		testServer.DB.QueryRow("SELECT name FROM racers WHERE id = 1").Scan(&name)
		if name != "" && !strings.Contains(entry.Message, name) {
			t.Errorf("expected driver name %q substituted in %q", name, entry.Message)
		}
	})

	t.Run("weather change auto-generates commentary", func(t *testing.T) {
		body := `{"race_id":0,"condition":"wet","lap_start":6,"lap_end":999,"grip_modifier":0.7}`
		req, _ := http.NewRequest("POST", "/api/weather", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}

		req2, _ := http.NewRequest("GET", "/api/commentary?race_id=0&limit=1", nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr2.Code)
		}
		var entries []models.Commentary
		json.Unmarshal(rr2.Body.Bytes(), &entries)
		if len(entries) < 1 {
			t.Fatal("expected weather commentary entry")
		}
		entry := entries[0]
		if entry.TemplateKey != "weather_wet" {
			t.Errorf("expected template_key 'weather_wet', got %q", entry.TemplateKey)
		}
		if !strings.Contains(entry.Message, "Wet") {
			t.Errorf("expected condition name in %q", entry.Message)
		}
	})

	testServer.DB.Exec("DELETE FROM commentary")
}
