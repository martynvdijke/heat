package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("failed to open in-memory db: %v", err)
	}
	defer db.Close()

	initDB()
	go broadcastManager()

	os.Exit(m.Run())
}

func TestHashPassword(t *testing.T) {
	password := "password123"

	hash := hashPassword(password)

	if hash == "" {
		t.Error("Expected hash to be non-empty")
	}

	// Test consistency
	if hash != hashPassword(password) {
		t.Error("Expected hash to be consistent")
	}

	// Test difference
	if hash == hashPassword("anotherpassword") {
		t.Error("Expected different passwords to have different hashes")
	}
}

func TestHandleCheckSetup(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/check-setup", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(handleCheckSetup)

	handler.ServeHTTP(rr, req)

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
	req, err := http.NewRequest("GET", "/api/racers", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(getRacers)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var racers []Racer
	err = json.Unmarshal(rr.Body.Bytes(), &racers)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(racers) != 5 {
		t.Errorf("expected 5 racers, got %d", len(racers))
	}
}

func TestAuthMiddleware(t *testing.T) {
	// Create a dummy handler that should only be called if authorized
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Authorized")
	})

	handler := authMiddleware(dummyHandler)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status %v, got %v", http.StatusUnauthorized, status)
		}
	})

	t.Run("Authorized", func(t *testing.T) {
		sessionID := "test-session"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

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
		req, _ := http.NewRequest("GET", "/api/race-info", nil)
		rr := httptest.NewRecorder()
		getRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var ri RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &ri)
		if ri.Country != "Italy" || ri.Track != "Monza" {
			t.Errorf("unexpected race info: %+v", ri)
		}
	})

	t.Run("UpdateRaceInfo", func(t *testing.T) {
		ri := RaceInfo{Country: "Belgium", Track: "Spa", Laps: 44}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update
		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.Country != "Belgium" || updatedRi.Track != "Spa" || updatedRi.Laps != 44 {
			t.Errorf("race info not updated correctly: %+v", updatedRi)
		}
	})
}

func TestUpdateAndDeleteRacer(t *testing.T) {
	var racerID int

	t.Run("InsertRacer", func(t *testing.T) {
		newRacer := Racer{Name: "L. HAMILTON", CarColor: "black", CarName: "Silver Arrow", Points: 0, Rank: 6}
		body, _ := json.Marshal(newRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Get the inserted racer to find its ID
		rows, _ := db.Query("SELECT id FROM racers WHERE name='L. HAMILTON'")
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&racerID)
		} else {
			t.Fatal("racer not inserted")
		}
	})

	t.Run("UpdateRacer", func(t *testing.T) {
		updatedRacer := Racer{ID: racerID, Name: "L. HAMILTON", CarColor: "purple", CarName: "W12", Points: 25, Rank: 1, Position: 50}
		body, _ := json.Marshal(updatedRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update
		var name string
		var pos int
		db.QueryRow("SELECT name, position FROM racers WHERE id=?", racerID).Scan(&name, &pos)
		if name != "L. HAMILTON" {
			t.Errorf("expected name L. HAMILTON, got %s", name)
		}
		if pos != 50 {
			t.Errorf("expected position 50, got %d", pos)
		}
	})

	t.Run("DeleteRacer", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/racers?id=%d", racerID), nil)
		rr := httptest.NewRecorder()
		deleteRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify deletion
		var count int
		db.QueryRow("SELECT COUNT(*) FROM racers WHERE id=?", racerID).Scan(&count)
		if count != 0 {
			t.Error("racer not deleted")
		}
	})
}

func TestWebSocketBroadcast(t *testing.T) {
	// Create a test server
	s := httptest.NewServer(http.HandlerFunc(handleWebSocket))
	defer s.Close()

	// Convert http URL to ws URL
	u := "ws" + strings.TrimPrefix(s.URL, "http")

	// Connect to the server
	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", u, err)
	}
	defer ws.Close()

	// Trigger a broadcast by updating a racer
	racer := Racer{ID: 1, Name: "A. PROST", Points: 100, Rank: 1, Position: 10}
	body, _ := json.Marshal(racer)
	req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	updateRacer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("updateRacer failed: %v", rr.Code)
	}

	// Read from WebSocket
	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("could not read message: %v", err)
	}

	var racers []Racer
	if err := json.Unmarshal(message, &racers); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}

	// Verify the updated racer is in the broadcast
	found := false
	for _, r := range racers {
		if r.ID == 1 && r.Position == 10 {
			found = true
			break
		}
	}
	if !found {
		t.Error("updated racer not found in WebSocket broadcast")
	}
}

func TestLoginAndSetup(t *testing.T) {
	// 1. Initial check-setup (should be false)
	t.Run("CheckSetupInitial", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		handleCheckSetup(rr, req)
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["setup"] != false {
			t.Errorf("expected setup false, got %v", resp["setup"])
		}
	})

	// 2. Perform first-time setup (create admin)
	t.Run("FirstTimeSetup", func(t *testing.T) {
		loginData := map[string]interface{}{
			"username": "admin",
			"password": "password",
			"setup":    true,
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		handleLogin(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Check if cookie is set
		cookies := rr.Result().Cookies()
		found := false
		for _, c := range cookies {
			if c.Name == "session" {
				found = true
				break
			}
		}
		if !found {
			t.Error("session cookie not found after setup login")
		}
	})

	// 3. check-setup (should be true now)
	t.Run("CheckSetupAfter", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		handleCheckSetup(rr, req)
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["setup"] != true {
			t.Errorf("expected setup true, got %v", resp["setup"])
		}
	})
}

func TestGetTracks(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/tracks", nil)
	rr := httptest.NewRecorder()
	getTracks(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var tracks []Track
	err := json.Unmarshal(rr.Body.Bytes(), &tracks)
	if err != nil {
		t.Fatalf("failed to unmarshal tracks: %v", err)
	}

	if len(tracks) < 5 {
		t.Errorf("expected at least 5 tracks, got %d", len(tracks))
	}

	expectedTracks := map[string]string{
		"monza":       "Monza",
		"spa":         "Spa-Francorchamps",
		"silverstone": "Silverstone",
		"monaco":      "Monaco",
		"interlagos":  "Interlagos",
	}

	for _, track := range tracks {
		if name, ok := expectedTracks[track.ID]; ok {
			if track.Name != name {
				t.Errorf("expected track %s to have name %s, got %s", track.ID, name, track.Name)
			}
		}
	}
}

func TestRaceInfoWithTrackID(t *testing.T) {
	t.Run("UpdateRaceInfoWithTrackID", func(t *testing.T) {
		ri := RaceInfo{Country: "Belgium", Track: "Spa-Francorchamps", TrackID: "spa", Laps: 44}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update includes track_id
		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.TrackID != "spa" {
			t.Errorf("expected track_id 'spa', got '%s'", updatedRi.TrackID)
		}
		if updatedRi.Country != "Belgium" {
			t.Errorf("expected country 'Belgium', got '%s'", updatedRi.Country)
		}
	})

	t.Run("UpdateRaceInfoWithoutTrackIDDefaultsToMonza", func(t *testing.T) {
		ri := RaceInfo{Country: "Monaco", Track: "Monaco", Laps: 78}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify track_id defaults to monza
		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.TrackID != "monza" {
			t.Errorf("expected default track_id 'monza', got '%s'", updatedRi.TrackID)
		}
	})
}

func TestRaceHistory(t *testing.T) {
	t.Run("GetRaceHistoryEmpty", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-history", nil)
		rr := httptest.NewRecorder()
		getRaceHistory(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var history []map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &history)
		if err != nil {
			t.Fatalf("failed to unmarshal history: %v", err)
		}

		if len(history) != 0 {
			t.Errorf("expected empty history, got %d entries", len(history))
		}
	})

	t.Run("SaveRaceToHistory", func(t *testing.T) {
		// Create a session for auth
		sessionID := "test-session-history"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		input := map[string]interface{}{
			"race_date":  "2026-04-15",
			"country":    "Italy",
			"track":      "Monza",
			"track_id":   "monza",
			"total_laps": 53,
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true},
				{"racer_id": 2, "racer_name": "M. SCHUMACHER", "position": 2, "points": 18, "fastest_lap": false},
				{"racer_id": 3, "racer_name": "A. SENNA", "position": 3, "points": 15, "fastest_lap": false},
			},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		saveRaceToHistory(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify history was saved
		req, _ = http.NewRequest("GET", "/api/race-history", nil)
		rr = httptest.NewRecorder()
		getRaceHistory(rr, req)

		var history []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) != 1 {
			t.Errorf("expected 1 history entry, got %d", len(history))
		}

		if history[0]["country"] != "Italy" {
			t.Errorf("expected country 'Italy', got '%v'", history[0]["country"])
		}
	})

	t.Run("GetRaceHistoryWithResults", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-history", nil)
		rr := httptest.NewRecorder()
		getRaceHistory(rr, req)

		var history []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) == 0 {
			t.Fatal("no history entries found")
		}
	})

	t.Run("UnauthorizedSaveRaceHistory", func(t *testing.T) {
		input := map[string]interface{}{
			"race_date":  "2026-04-16",
			"country":    "Belgium",
			"track":      "Spa",
			"total_laps": 44,
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		authMiddleware(saveRaceToHistory)(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestRacerStats(t *testing.T) {
	t.Run("GetAllRacerStats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-stats", nil)
		rr := httptest.NewRecorder()
		getRacerStats(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var stats []RacerStats
		err := json.Unmarshal(rr.Body.Bytes(), &stats)
		if err != nil {
			t.Fatalf("failed to unmarshal stats: %v", err)
		}
	})

	t.Run("GetSpecificRacerStats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		getRacerStats(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var data map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &data)
		if err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		// Should have both stats and racer
		if data["stats"] == nil {
			t.Error("expected stats in response")
		}
		if data["racer"] == nil {
			t.Error("expected racer in response")
		}
	})

	t.Run("StatsUpdatedAfterRaceArchived", func(t *testing.T) {
		// First, get initial stats
		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		getRacerStats(rr, req)

		var initialData map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &initialData)
		_ = initialData["stats"].(map[string]interface{})

		// Archive a race with this racer
		sessionID := "test-session-stats"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		input := map[string]interface{}{
			"race_date":  "2026-04-16",
			"country":    "Belgium",
			"track":      "Spa",
			"track_id":   "spa",
			"total_laps": 44,
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true},
			},
		}
		body, _ := json.Marshal(input)
		req, _ = http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		saveRaceToHistory(rr, req)

		// Check stats were updated
		req, _ = http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr = httptest.NewRecorder()
		getRacerStats(rr, req)

		var updatedData map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &updatedData)
		updatedStats := updatedData["stats"].(map[string]interface{})

		if updatedStats["races"] == nil {
			t.Error("expected races count to be updated")
		}
	})
}

func TestSchemaMigrations(t *testing.T) {
	t.Run("SchemaVersionTableExists", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM schema_version").Scan(&count)
		if err != nil {
			t.Errorf("schema_version table should exist: %v", err)
		}
		if count != 1 {
			t.Errorf("expected 1 schema version row, got %d", count)
		}
	})

	t.Run("TracksTablePopulated", func(t *testing.T) {
		var count int
		db.QueryRow("SELECT COUNT(*) FROM tracks").Scan(&count)
		if count < 5 {
			t.Errorf("expected at least 5 tracks, got %d", count)
		}
	})

	t.Run("RaceHistoryTablesExist", func(t *testing.T) {
		var raceHistoryCount, raceResultsCount int
		db.QueryRow("SELECT COUNT(*) FROM race_history").Scan(&raceHistoryCount)
		db.QueryRow("SELECT COUNT(*) FROM race_results").Scan(&raceResultsCount)

		if raceHistoryCount == 0 {
			t.Error("race_history table should exist")
		}
		if raceResultsCount == 0 {
			t.Error("race_results table should exist")
		}
	})

	t.Run("RacerStatsTableExists", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM racer_stats").Scan(&count)
		if err != nil {
			t.Errorf("racer_stats table should exist: %v", err)
		}
	})
}

func TestDeleteRaceHistory(t *testing.T) {
	// Create a session for auth
	sessionID := "test-session-delete"
	sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
	defer delete(sessionStore, sessionID)

	// First, save a race
	input := map[string]interface{}{
		"race_date":  "2026-04-20",
		"country":    "UK",
		"track":      "Silverstone",
		"track_id":   "silverstone",
		"total_laps": 52,
		"results": []map[string]interface{}{
			{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25},
		},
	}
	body, _ := json.Marshal(input)
	req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	saveRaceToHistory(rr, req)

	// Get the history to find the ID
	req, _ = http.NewRequest("GET", "/api/race-history", nil)
	rr = httptest.NewRecorder()
	getRaceHistory(rr, req)

	var history []map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &history)
	if len(history) == 0 {
		t.Fatal("no history to delete")
	}

	raceID := int(history[0]["id"].(float64))

	// Delete the race
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/race-history?id=%d", raceID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr = httptest.NewRecorder()
	deleteRaceHistory(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	// Verify deletion
	req, _ = http.NewRequest("GET", "/api/race-history", nil)
	rr = httptest.NewRecorder()
	getRaceHistory(rr, req)
	json.Unmarshal(rr.Body.Bytes(), &history)

	for _, h := range history {
		if int(h["id"].(float64)) == raceID {
			t.Error("race should have been deleted")
		}
	}
}

func TestUploadRequiresAuth(t *testing.T) {
	t.Run("UploadWithoutAuth", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/upload", nil)
		rr := httptest.NewRecorder()
		authMiddleware(handleUpload)(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestQuotes(t *testing.T) {
	t.Run("GetAllQuotes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quotes", nil)
		rr := httptest.NewRecorder()
		getQuotes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var quotes []Quote
		err := json.Unmarshal(rr.Body.Bytes(), &quotes)
		if err != nil {
			t.Fatalf("failed to unmarshal quotes: %v", err)
		}

		if len(quotes) < 20 {
			t.Errorf("expected at least 20 quotes, got %d", len(quotes))
		}
	})

	t.Run("GetRandomQuote", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quote/random", nil)
		rr := httptest.NewRecorder()
		getRandomQuote(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var quote Quote
		err := json.Unmarshal(rr.Body.Bytes(), &quote)
		if err != nil {
			t.Fatalf("failed to unmarshal quote: %v", err)
		}

		if quote.Text == "" {
			t.Error("expected quote text to be non-empty")
		}
	})

	t.Run("AddQuoteWithAuth", func(t *testing.T) {
		sessionID := "test-session-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		newQuote := Quote{Text: "Test quote for unit testing!", Author: "Test Author"}
		body, _ := json.Marshal(newQuote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("expected status 201, got %v", status)
		}

		var createdQuote Quote
		json.Unmarshal(rr.Body.Bytes(), &createdQuote)
		if createdQuote.Text != "Test quote for unit testing!" {
			t.Errorf("expected quote text 'Test quote for unit testing!', got '%s'", createdQuote.Text)
		}
	})

	t.Run("AddQuoteWithoutAuth", func(t *testing.T) {
		newQuote := Quote{Text: "Unauthorized quote"}
		body, _ := json.Marshal(newQuote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		authMiddleware(handleQuotes)(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("UpdateQuoteWithAuth", func(t *testing.T) {
		sessionID := "test-session-update-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		updatedQuote := Quote{ID: 1, Text: "Updated quote text", Author: "Updated Author"}
		body, _ := json.Marshal(updatedQuote)
		req, _ := http.NewRequest("PUT", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("DeleteQuoteWithAuth", func(t *testing.T) {
		sessionID := "test-session-delete-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("DELETE", "/api/quotes?id=1", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("DeleteQuoteWithoutAuth", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/quotes?id=1", nil)
		rr := httptest.NewRecorder()
		authMiddleware(handleQuotes)(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestSchemaMigrationToV3(t *testing.T) {
	t.Run("QuotesTableExists", func(t *testing.T) {
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM quotes").Scan(&count)
		if err != nil {
			t.Errorf("quotes table should exist: %v", err)
		}
		if count < 20 {
			t.Errorf("expected at least 20 seeded quotes, got %d", count)
		}
	})

	t.Run("SchemaVersionIs3", func(t *testing.T) {
		var version int
		err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err != nil {
			t.Errorf("schema_version should exist: %v", err)
		}
		if version != 3 {
			t.Errorf("expected schema version 3, got %d", version)
		}
	})
}
