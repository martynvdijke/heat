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
		ri := RaceInfo{Country: "Belgium", Track: "Spa", Laps: 44, TrackID: "spa"}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Verify update in DB
		var dbCountry, dbTrack, dbTrackID string
		var dbLaps int
		err := db.QueryRow("SELECT country, track, track_id, laps FROM race_info ORDER BY id DESC LIMIT 1").
			Scan(&dbCountry, &dbTrack, &dbTrackID, &dbLaps)
		if err != nil {
			t.Fatalf("failed to find race info in DB: %v", err)
		}
		if dbCountry != "Belgium" || dbTrack != "Spa" || dbTrackID != "spa" || dbLaps != 44 {
			t.Errorf("DB data mismatch: got %s, %s, %s, %d", dbCountry, dbTrack, dbTrackID, dbLaps)
		}

		// Verify update via API
		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.Country != "Belgium" || updatedRi.Track != "Spa" || updatedRi.Laps != 44 {
			t.Errorf("race info not updated correctly via API: %+v", updatedRi)
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
		if !resp["setup"] {
			t.Error("expected setup to be true after user creation")
		}
	})

	t.Run("BlockDuplicateSetup", func(t *testing.T) {
		// Attempt setup again even though admin exists
		input := map[string]interface{}{
			"username": "hacker",
			"password": "password",
			"setup":    true,
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		handleLogin(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("expected status 403 Forbidden for duplicate setup, got %v", status)
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
			"name":       "Test Race",
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

		// Get the inserted ID
		var resp map[string]int64
		json.Unmarshal(rr.Body.Bytes(), &resp)
		raceID := resp["id"]

		// Check race_history in DB
		var dbName, dbCountry string
		err := db.QueryRow("SELECT name, country FROM race_history WHERE id=?", raceID).Scan(&dbName, &dbCountry)
		if err != nil {
			t.Fatalf("failed to find race history in DB: %v", err)
		}
		if dbName != "Test Race" || dbCountry != "Italy" {
			t.Errorf("DB mismatch for history: got %s, %s", dbName, dbCountry)
		}

		// Check race_results in DB
		var resultCount int
		db.QueryRow("SELECT COUNT(*) FROM race_results WHERE race_id=?", raceID).Scan(&resultCount)
		if resultCount != 3 {
			t.Errorf("expected 3 results in DB, got %d", resultCount)
		}

		// Check racer_stats in DB
		var wins int
		db.QueryRow("SELECT wins FROM racer_stats WHERE racer_id=1").Scan(&wins)
		if wins < 1 {
			t.Error("expected racer 1 to have at least 1 win in stats")
		}

		// Verify history was saved via API
		req, _ = http.NewRequest("GET", "/api/race-history", nil)
		rr = httptest.NewRecorder()
		getRaceHistory(rr, req)

		var history []RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) == 0 {
			t.Fatal("no history entries found via API")
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
			"name":       "Test Race 2",
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
		"name":       "Silverstone Test",
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

	var history []RaceHistory
	json.Unmarshal(rr.Body.Bytes(), &history)
	if len(history) == 0 {
		t.Fatal("no history to delete")
	}

	raceID := history[0].ID

	// Delete the race
	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/race-history?id=%d", raceID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr = httptest.NewRecorder()
	deleteRaceHistory(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	// Verify deletion in DB
	var historyCount int
	db.QueryRow("SELECT COUNT(*) FROM race_history WHERE id=?", raceID).Scan(&historyCount)
	if historyCount != 0 {
		t.Errorf("race_history entry with ID %d still exists in DB", raceID)
	}

	var resultsCount int
	db.QueryRow("SELECT COUNT(*) FROM race_results WHERE race_id=?", raceID).Scan(&resultsCount)
	if resultsCount != 0 {
		t.Errorf("race_results entries for race_id %d still exist in DB", raceID)
	}

	// Verify deletion via API
	req, _ = http.NewRequest("GET", "/api/race-history", nil)
	rr = httptest.NewRecorder()
	getRaceHistory(rr, req)
	var updatedHistory []RaceHistory
	json.Unmarshal(rr.Body.Bytes(), &updatedHistory)

	for _, h := range updatedHistory {
		if h.ID == raceID {
			t.Error("race should have been deleted (found in API response)")
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

		// Check DB
		var dbText string
		err := db.QueryRow("SELECT text FROM quotes WHERE id=?", createdQuote.ID).Scan(&dbText)
		if err != nil {
			t.Fatalf("failed to find quote in DB: %v", err)
		}
		if dbText != "Test quote for unit testing!" {
			t.Errorf("expected DB text 'Test quote for unit testing!', got '%s'", dbText)
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

		// Create a quote first to get a valid ID
		res, _ := db.Exec("INSERT INTO quotes (text, author) VALUES ('Original', 'Original')")
		quoteID, _ := res.LastInsertId()

		updatedQuote := Quote{ID: int(quoteID), Text: "Updated quote text", Author: "Updated Author"}
		body, _ := json.Marshal(updatedQuote)
		req, _ := http.NewRequest("PUT", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Check DB
		var dbText, dbAuthor string
		err := db.QueryRow("SELECT text, author FROM quotes WHERE id=?", quoteID).Scan(&dbText, &dbAuthor)
		if err != nil {
			t.Fatalf("failed to find updated quote in DB: %v", err)
		}
		if dbText != "Updated quote text" || dbAuthor != "Updated Author" {
			t.Errorf("DB update check failed: got %s, %s", dbText, dbAuthor)
		}
	})

	t.Run("DeleteQuoteWithAuth", func(t *testing.T) {
		sessionID := "test-session-delete-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		// Create a quote first to get a valid ID
		res, _ := db.Exec("INSERT INTO quotes (text, author) VALUES ('ToDelete', 'Author')")
		quoteID, _ := res.LastInsertId()

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/quotes?id=%d", quoteID), nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		// Check DB
		var count int
		db.QueryRow("SELECT COUNT(*) FROM quotes WHERE id=?", quoteID).Scan(&count)
		if count != 0 {
			t.Errorf("quote with ID %d still exists in DB after deletion", quoteID)
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

func TestSaveRaceToHistoryWithName(t *testing.T) {
	t.Run("SaveRaceToHistoryWithCustomName", func(t *testing.T) {
		sessionID := "test-session-history-name"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		input := map[string]interface{}{
			"name":       "2024 Season Finale",
			"race_date":  "2026-04-15",
			"country":    "Italy",
			"track":      "Monza",
			"track_id":   "monza",
			"total_laps": 53,
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true},
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

		// Verify history was saved with the custom name in the database
		var name string
		err := db.QueryRow("SELECT name FROM race_history WHERE name='2024 Season Finale'").Scan(&name)
		if err != nil {
			t.Fatalf("failed to find archived race in DB: %v", err)
		}
		if name != "2024 Season Finale" {
			t.Errorf("expected name '2024 Season Finale', got '%s'", name)
		}

		// Verify through API
		req, _ = http.NewRequest("GET", "/api/race-history", nil)
		rr = httptest.NewRecorder()
		getRaceHistory(rr, req)

		var history []RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)

		found := false
		for _, h := range history {
			if h.Name == "2024 Season Finale" {
				found = true
				break
			}
		}
		if !found {
			t.Error("custom archive name not found in history via API")
		}
	})
}

func TestAdminAddRacerAndCheckDB(t *testing.T) {
	sessionID := "admin-test-session"
	sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
	defer delete(sessionStore, sessionID)

	newRacer := Racer{
		Name:           "M. VERSTAPPEN",
		ProfilePicture: "/static/images/max.png",
		CarColor:       "blue",
		CarName:        "RB20",
		Points:         100,
		Rank:           1,
		Position:       0,
	}
	body, _ := json.Marshal(newRacer)
	req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	updateRacer(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("failed to add racer: %d", rr.Code)
	}

	// Check DB
	var name, car string
	err := db.QueryRow("SELECT name, car_name FROM racers WHERE name='M. VERSTAPPEN'").Scan(&name, &car)
	if err != nil {
		t.Fatalf("racer not found in database: %v", err)
	}
	if name != "M. VERSTAPPEN" || car != "RB20" {
		t.Errorf("database data mismatch: got %s, %s", name, car)
	}
}

func TestAdminAddQuoteAndCheckDB(t *testing.T) {
	sessionID := "admin-test-session-quote"
	sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
	defer delete(sessionStore, sessionID)

	newQuote := Quote{
		Text:   "Simply lovely!",
		Author: "Max Verstappen",
	}
	body, _ := json.Marshal(newQuote)
	req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	handleQuotes(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to add quote: %d", rr.Code)
	}

	// Check DB
	var text, author string
	err := db.QueryRow("SELECT text, author FROM quotes WHERE text='Simply lovely!'").Scan(&text, &author)
	if err != nil {
		t.Fatalf("quote not found in database: %v", err)
	}
	if text != "Simply lovely!" || author != "Max Verstappen" {
		t.Errorf("database data mismatch: got %s, %s", text, author)
	}
}

func TestSchemaMigrationToV5(t *testing.T) {
	t.Run("NameColumnExistsInRaceHistory", func(t *testing.T) {
		// Try to insert a row with the name column to verify it exists
		_, err := db.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps) VALUES (?, ?, ?, ?, ?, ?)",
			"Migration Test", "2026-01-01", "Test", "Test", "test", 10)
		if err != nil {
			t.Errorf("failed to insert into race_history with name column: %v", err)
		}
	})

	t.Run("NewTrackColumnsExist", func(t *testing.T) {
		// Check if new columns exist in tracks table
		_, err := db.Exec("SELECT use_map_image, map_image_url, refresh_geojson FROM tracks LIMIT 0")
		if err != nil {
			t.Errorf("new track columns should exist: %v", err)
		}
	})

	t.Run("SchemaVersionIs5", func(t *testing.T) {
		var version int
		err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err != nil {
			t.Errorf("schema_version should exist: %v", err)
		}

		if version != 5 {
			t.Errorf("expected schema version 5, got %d", version)
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("LogoutClearsSession", func(t *testing.T) {
		sessionID := "logout-test-session"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()

		req, _ := http.NewRequest("POST", "/api/logout", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleLogout(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		if _, exists := sessionStore[sessionID]; exists {
			t.Error("session should be cleared after logout")
		}
	})

	t.Run("LogoutWithNoSession", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/logout", nil)
		rr := httptest.NewRecorder()
		handleLogout(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})
}

func TestSessionExpiration(t *testing.T) {
	t.Run("ExpiredSessionIsRejected", func(t *testing.T) {
		sessionID := "expired-session"
		sessionStore[sessionID] = time.Now().Add(-1 * time.Hour).Unix()

		dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
		handler := authMiddleware(dummyHandler)

		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}

		if _, exists := sessionStore[sessionID]; exists {
			t.Error("expired session should be removed from store")
		}
	})
}

func TestRaceHistoryEdgeCases(t *testing.T) {
	t.Run("SaveRaceWithEmptyResults", func(t *testing.T) {
		sessionID := "test-session-empty-results"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		input := map[string]interface{}{
			"name":       "Hungary GP",
			"race_date":  "2026-04-25",
			"country":    "Hungary",
			"track":      "Hungaroring",
			"track_id":   "hungary",
			"total_laps": 70,
			"results":    []map[string]interface{}{},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		saveRaceToHistory(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("GetRaceByID", func(t *testing.T) {
		sessionID := "test-session-get-by-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		input := map[string]interface{}{
			"name":       "Spain GP",
			"race_date":  "2026-04-26",
			"country":    "Spain",
			"track":      "Catalunya",
			"total_laps": 66,
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25},
			},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		saveRaceToHistory(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		raceID := int(resp["id"].(float64))

		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/race-history?id=%d", raceID), nil)
		rr = httptest.NewRecorder()
		getRaceHistory(rr, req)

		var history []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) != 1 {
			t.Errorf("expected 1 race, got %d", len(history))
		}
		if history[0]["country"] != "Spain" {
			t.Errorf("expected country 'Spain', got '%v'", history[0]["country"])
		}
	})

	t.Run("DeleteRaceWithoutID", func(t *testing.T) {
		sessionID := "test-session-delete-no-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("DELETE", "/api/race-history", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		deleteRaceHistory(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})
}

func TestQuoteEdgeCases(t *testing.T) {
	t.Run("AddQuoteWithEmptyText", func(t *testing.T) {
		sessionID := "test-session-empty-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		quote := Quote{Text: ""}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("UpdateQuoteWithoutID", func(t *testing.T) {
		sessionID := "test-session-update-no-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		quote := Quote{Text: "Updated text", Author: "Test"}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("PUT", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("DeleteQuoteWithoutID", func(t *testing.T) {
		sessionID := "test-session-delete-quote-no-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("DELETE", "/api/quotes", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("AddQuoteDefaultsAuthor", func(t *testing.T) {
		sessionID := "test-session-default-author"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		quote := Quote{Text: "Test quote without author"}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		handleQuotes(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("expected status 201, got %v", status)
		}

		var created Quote
		json.Unmarshal(rr.Body.Bytes(), &created)
		if created.Author != "Commentator" {
			t.Errorf("expected default author 'Commentator', got '%s'", created.Author)
		}
	})
}

func TestRacerStatsEdgeCases(t *testing.T) {
	t.Run("GetStatsForNonexistentRacer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-stats?id=9999", nil)
		rr := httptest.NewRecorder()
		getRacerStats(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var data map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &data)
		stats := data["stats"].(map[string]interface{})

		racesVal := stats["races"]
		if racesVal != nil {
			races := int(racesVal.(float64))
			if races != 0 {
				t.Errorf("expected 0 races for nonexistent racer, got %v", stats["races"])
			}
		}
	})

	t.Run("StatsAccumulateWithMultipleRaces", func(t *testing.T) {
		sessionID := "test-session-multiple-races"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		for i := 0; i < 3; i++ {
			input := map[string]interface{}{
				"name":       fmt.Sprintf("Race %d", i),
				"race_date":  fmt.Sprintf("2026-05-%02d", i+1),
				"country":    "Test",
				"track":      "Test",
				"total_laps": 50,
				"results": []map[string]interface{}{
					{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true},
				},
			}
			body, _ := json.Marshal(input)
			req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
			rr := httptest.NewRecorder()
			saveRaceToHistory(rr, req)
		}

		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		getRacerStats(rr, req)

		var data map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &data)
		stats := data["stats"].(map[string]interface{})

		racesVal := stats["races"]
		if racesVal != nil {
			races := int(racesVal.(float64))
			if races < 1 {
				t.Errorf("expected at least 1 race, got %v", stats["races"])
			}
		}
	})
}

func TestLoginEdgeCases(t *testing.T) {
	t.Run("InvalidLoginCredentials", func(t *testing.T) {
		loginData := map[string]interface{}{
			"username": "wronguser",
			"password": "wrongpass",
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		handleLogin(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("LoginWithInvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()
		handleLogin(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})
}

func TestGetRacersEdgeCases(t *testing.T) {
	t.Run("RacersSortedByRank", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		getRacers(rr, req)

		var racers []Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)

		for i := 1; i < len(racers); i++ {
			if racers[i].Rank < racers[i-1].Rank {
				t.Error("racers should be sorted by rank")
				break
			}
		}
	})

	t.Run("RacerFieldsAreComplete", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		getRacers(rr, req)

		var racers []Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)

		hasRacers := false
		for _, r := range racers {
			if r.Name != "" {
				hasRacers = true
			}
		}

		if !hasRacers {
			t.Error("should have at least one racer with a name")
		}
	})
}

func TestWebSocketEdgeCases(t *testing.T) {
	t.Run("WebSocketClientDisconnection", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(handleWebSocket))
		defer s.Close()

		u := "ws" + strings.TrimPrefix(s.URL, "http")
		ws, _, err := websocket.DefaultDialer.Dial(u, nil)
		if err != nil {
			t.Fatalf("could not connect: %v", err)
		}
		defer ws.Close()

		initialClientCount := len(clients)

		ws.Close()
		time.Sleep(100 * time.Millisecond)

		if len(clients) >= initialClientCount {
			t.Error("client should be removed after disconnect")
		}
	})
}

func TestMultipleRacerOperations(t *testing.T) {
	t.Run("CreateMultipleRacers", func(t *testing.T) {
		sessionID := "test-session-multi-racer"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		names := []string{"R. PETROV", "S. VETTEL", "K. RAIKKONEN"}
		for _, name := range names {
			racer := Racer{Name: name, CarColor: "red", CarName: "Test Car", Points: 0, Rank: 10}
			body, _ := json.Marshal(racer)
			req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
			rr := httptest.NewRecorder()
			updateRacer(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("failed to create racer %s: status %d", name, rr.Code)
			}
		}
	})

	t.Run("UpdateMultipleRacers", func(t *testing.T) {
		sessionID := "test-session-update-multi"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		getRacers(rr, req)

		var racers []Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)

		for i, r := range racers {
			updated := Racer{
				ID:       r.ID,
				Name:     r.Name,
				CarColor: r.CarColor,
				CarName:  r.CarName,
				Points:   (i + 1) * 10,
				Rank:     i + 1,
				Position: (i + 1) * 10,
			}
			body, _ := json.Marshal(updated)
			req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
			rr := httptest.NewRecorder()
			updateRacer(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("failed to update racer %d: status %d", r.ID, rr.Code)
			}
		}
	})
}

func TestRaceInfoEdgeCases(t *testing.T) {
	t.Run("UpdateRaceInfoWithAllFields", func(t *testing.T) {
		sessionID := "test-session-full-race-info"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		ri := RaceInfo{Country: "Germany", Track: "Nürburgring", TrackID: "nurburgring", Laps: 60}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		updateRaceInfo(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		getRaceInfo(rr, req)

		var updated RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updated)

		if updated.Country != "Germany" {
			t.Errorf("expected country 'Germany', got '%s'", updated.Country)
		}
		if updated.Track != "Nürburgring" {
			t.Errorf("expected track 'Nürburgring', got '%s'", updated.Track)
		}
		if updated.TrackID != "nurburgring" {
			t.Errorf("expected track_id 'nurburgring', got '%s'", updated.TrackID)
		}
		if updated.Laps != 60 {
			t.Errorf("expected 60 laps, got %d", updated.Laps)
		}
	})
}

func TestDeleteRacerEdgeCases(t *testing.T) {
	t.Run("DeleteNonexistentRacer", func(t *testing.T) {
		sessionID := "test-session-delete-nonexistent"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		req, _ := http.NewRequest("DELETE", "/api/racers?id=9999", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		deleteRacer(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200 (idempotent), got %v", status)
		}
	})
}
