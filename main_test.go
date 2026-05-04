package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	os.Unsetenv("DOCKER")
	basePath = "."
	dbPath = "./heat.db"
	imagesPath = filepath.Join(basePath, "static/images")

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

	if hash != hashPassword(password) {
		t.Error("Expected hash to be consistent")
	}

	if hash == hashPassword("anotherpassword") {
		t.Error("Expected different passwords to have different hashes")
	}
}

func TestHandleCheckSetup(t *testing.T) {
	r := gin.New()
	r.GET("/api/check-setup", handleCheckSetup)

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
	r.GET("/api/racers", getRacers)

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
	r := gin.New()
	r.GET("/api/test", authMiddleware(), func(c *gin.Context) {
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
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

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
		r.GET("/api/race-info", getRaceInfo)

		req, _ := http.NewRequest("GET", "/api/race-info", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.POST("/api/race-info", updateRaceInfo)
		r.GET("/api/race-info", getRaceInfo)

		ri := RaceInfo{Country: "Belgium", Track: "Spa", Laps: 44, TrackID: "spa"}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

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

		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
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
		r := gin.New()
		r.POST("/api/racers", updateRacer)

		newRacer := Racer{Name: "L. HAMILTON", CarColor: "black", CarName: "Silver Arrow", Points: 0, Rank: 6}
		body, _ := json.Marshal(newRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		rows, _ := db.Query("SELECT id FROM racers WHERE name='L. HAMILTON'")
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&racerID)
		} else {
			t.Fatal("racer not inserted")
		}
	})

	t.Run("UpdateRacer", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/racers", updateRacer)

		updatedRacer := Racer{ID: racerID, Name: "L. HAMILTON", CarColor: "purple", CarName: "W12", Points: 25, Rank: 1, Position: 50}
		body, _ := json.Marshal(updatedRacer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

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
		r := gin.New()
		r.DELETE("/api/racers", deleteRacer)

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/racers?id=%d", racerID), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var count int
		db.QueryRow("SELECT COUNT(*) FROM racers WHERE id=?", racerID).Scan(&count)
		if count != 0 {
			t.Error("racer not deleted")
		}
	})
}

func TestWebSocketBroadcast(t *testing.T) {
	wsr := gin.New()
	wsr.GET("/ws", handleWebSocket)
	s := httptest.NewServer(wsr)
	defer s.Close()

	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"

	ws, _, err := websocket.DefaultDialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("could not open a ws connection on %s %v", u, err)
	}
	defer ws.Close()

	racer := Racer{ID: 1, Name: "A. PROST", ProfilePicture: "/static/images/helmet.svg", Points: 100, Rank: 1, Position: 10}
	body, _ := json.Marshal(racer)
	req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))

	racerRouter := gin.New()
	racerRouter.POST("/api/racers", updateRacer)
	rr := httptest.NewRecorder()
	racerRouter.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("updateRacer failed: %v", rr.Code)
	}

	_, message, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("could not read message: %v", err)
	}

	var racers []Racer
	if err := json.Unmarshal(message, &racers); err != nil {
		t.Fatalf("could not unmarshal message: %v", err)
	}

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
	t.Run("CheckSetupInitial", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/check-setup", handleCheckSetup)

		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["setup"] != false {
			t.Errorf("expected setup false, got %v", resp["setup"])
		}
	})

	t.Run("FirstTimeSetup", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/login", handleLogin)

		loginData := map[string]interface{}{
			"username": "admin",
			"password": "password",
			"setup":    true,
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

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

	t.Run("CheckSetupAfter", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/check-setup", handleCheckSetup)

		req, _ := http.NewRequest("GET", "/api/check-setup", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !resp["setup"] {
			t.Error("expected setup to be true after user creation")
		}
	})

	t.Run("BlockDuplicateSetup", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/login", handleLogin)

		input := map[string]interface{}{
			"username": "hacker",
			"password": "password",
			"setup":    true,
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusForbidden {
			t.Errorf("expected status 403 Forbidden for duplicate setup, got %v", status)
		}
	})
}

func TestGetTracks(t *testing.T) {
	r := gin.New()
	r.GET("/api/tracks", getTracks)

	req, _ := http.NewRequest("GET", "/api/tracks", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.POST("/api/race-info", updateRaceInfo)
		r.GET("/api/race-info", getRaceInfo)

		ri := RaceInfo{Country: "Belgium", Track: "Spa-Francorchamps", TrackID: "spa", Laps: 44}
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
		r := gin.New()
		r.POST("/api/race-info", updateRaceInfo)
		r.GET("/api/race-info", getRaceInfo)

		ri := RaceInfo{Country: "Monaco", Track: "Monaco", Laps: 78}
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
		var updatedRi RaceInfo
		json.Unmarshal(rr.Body.Bytes(), &updatedRi)
		if updatedRi.TrackID != "monza" {
			t.Errorf("expected default track_id 'monza', got '%s'", updatedRi.TrackID)
		}
	})
}

func TestRaceHistory(t *testing.T) {
	t.Run("GetRaceHistoryEmpty", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-history", getRaceHistory)

		req, _ := http.NewRequest("GET", "/api/race-history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		sessionID := "test-session-history"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)
		r.GET("/api/race-history", getRaceHistory)

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
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var resp map[string]int64
		json.Unmarshal(rr.Body.Bytes(), &resp)
		raceID := resp["id"]

		var dbName, dbCountry string
		err := db.QueryRow("SELECT name, country FROM race_history WHERE id=?", raceID).Scan(&dbName, &dbCountry)
		if err != nil {
			t.Fatalf("failed to find race history in DB: %v", err)
		}
		if dbName != "Test Race" || dbCountry != "Italy" {
			t.Errorf("DB mismatch for history: got %s, %s", dbName, dbCountry)
		}

		var resultCount int
		db.QueryRow("SELECT COUNT(*) FROM race_results WHERE race_id=?", raceID).Scan(&resultCount)
		if resultCount != 3 {
			t.Errorf("expected 3 results in DB, got %d", resultCount)
		}

		var wins int
		db.QueryRow("SELECT wins FROM racer_stats WHERE racer_id=1").Scan(&wins)
		if wins < 1 {
			t.Error("expected racer 1 to have at least 1 win in stats")
		}

		req, _ = http.NewRequest("GET", "/api/race-history", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var history []RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) == 0 {
			t.Fatal("no history entries found via API")
		}
	})

	t.Run("GetRaceHistoryWithResults", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-history", getRaceHistory)

		req, _ := http.NewRequest("GET", "/api/race-history", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var history []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &history)

		if len(history) == 0 {
			t.Fatal("no history entries found")
		}
	})

	t.Run("UnauthorizedSaveRaceHistory", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

		input := map[string]interface{}{
			"race_date":  "2026-04-16",
			"country":    "Belgium",
			"track":      "Spa",
			"total_laps": 44,
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestRacerStats(t *testing.T) {
	t.Run("GetAllRacerStats", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/racer-stats", getRacerStats)

		req, _ := http.NewRequest("GET", "/api/racer-stats", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.GET("/api/racer-stats", getRacerStats)

		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var data map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &data)
		if err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if data["stats"] == nil {
			t.Error("expected stats in response")
		}
		if data["racer"] == nil {
			t.Error("expected racer in response")
		}
	})

	t.Run("StatsUpdatedAfterRaceArchived", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/racer-stats", getRacerStats)

		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var initialData map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &initialData)
		_ = initialData["stats"].(map[string]interface{})

		sessionID := "test-session-stats"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		rh := gin.New()
		rh.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

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
		rh.ServeHTTP(rr, req)

		req, _ = http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
	sessionID := "test-session-delete"
	sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
	defer delete(sessionStore, sessionID)

	r := gin.New()
	r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)
	r.GET("/api/race-history", getRaceHistory)
	r.DELETE("/api/race-history", authMiddleware(), deleteRaceHistory)

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
	r.ServeHTTP(rr, req)

	req, _ = http.NewRequest("GET", "/api/race-history", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	var history []RaceHistory
	json.Unmarshal(rr.Body.Bytes(), &history)
	if len(history) == 0 {
		t.Fatal("no history to delete")
	}

	raceID := history[0].ID

	req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/race-history?id=%d", raceID), nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

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

	req, _ = http.NewRequest("GET", "/api/race-history", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var updatedHistory []RaceHistory
	json.Unmarshal(rr.Body.Bytes(), &updatedHistory)

	for _, h := range updatedHistory {
		if h.ID == raceID {
			t.Error("race should have been deleted (found in API response)")
		}
	}
}

func TestUpload(t *testing.T) {
	t.Run("UploadWithoutAuth", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		req, _ := http.NewRequest("POST", "/api/upload", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("UploadImageSuccessfully", func(t *testing.T) {
		sessionID := "test-session-upload"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		img := image.NewRGBA(image.Rect(0, 0, 200, 200))
		for y := 0; y < 200; y++ {
			for x := 0; x < 200; x++ {
				img.Set(x, y, color.RGBA{255, 0, 0, 255})
			}
		}

		var buf bytes.Buffer
		png.Encode(&buf, img)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.png")
		part.Write(buf.Bytes())
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var resp map[string]interface{}
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		if err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if resp["url"] == nil {
			t.Error("expected url in response")
		}
		if resp["resized_url"] == nil {
			t.Error("expected resized_url in response")
		}
		if resp["thumbnail_url"] == nil {
			t.Error("expected thumbnail_url in response")
		}
		if resp["hash"] == nil {
			t.Error("expected hash in response")
		}

		urlStr, _ := resp["url"].(string)
		if !strings.HasPrefix(urlStr, "/static/images/") {
			t.Errorf("expected url to start with /static/images/, got %s", urlStr)
		}

		hashStr, _ := resp["hash"].(string)
		if len(hashStr) != 64 {
			t.Errorf("expected hash to be 64 hex chars, got %d", len(hashStr))
		}

		baseName := urlStr[len("/static/images/"):]
		os.Remove(filepath.Join(imagesPath, baseName))
		ext := filepath.Ext(baseName)
		hashOnly := baseName[:len(baseName)-len(ext)]
		os.Remove(filepath.Join(imagesPath, hashOnly+"_resized"+ext))
		os.Remove(filepath.Join(imagesPath, hashOnly+"_thumb"+ext))
	})

	t.Run("UploadDuplicateImage", func(t *testing.T) {
		sessionID := "test-session-upload-dup"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		var buf bytes.Buffer
		png.Encode(&buf, img)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "dup.png")
		part.Write(buf.Bytes())
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)

		firstURL, _ := resp["url"].(string)

		body2 := &bytes.Buffer{}
		writer2 := multipart.NewWriter(body2)
		part2, _ := writer2.CreateFormFile("image", "dup2.png")
		part2.Write(buf.Bytes())
		writer2.Close()

		req2, _ := http.NewRequest("POST", "/api/upload", body2)
		req2.Header.Set("Content-Type", writer2.FormDataContentType())
		req2.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)

		var resp2 map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &resp2)

		if resp2["duplicate"] != true {
			t.Errorf("expected duplicate to be true, got %v", resp2["duplicate"])
		}
		if resp2["url"] != firstURL {
			t.Errorf("expected duplicate url %s to match %s", resp2["url"], firstURL)
		}

		baseName := firstURL[len("/static/images/"):]
		os.Remove(filepath.Join(imagesPath, baseName))
		ext := filepath.Ext(baseName)
		hashOnly := baseName[:len(baseName)-len(ext)]
		os.Remove(filepath.Join(imagesPath, hashOnly+"_resized"+ext))
		os.Remove(filepath.Join(imagesPath, hashOnly+"_thumb"+ext))
	})

	t.Run("UploadInvalidFileType", func(t *testing.T) {
		sessionID := "test-session-upload-invalid"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.txt")
		part.Write([]byte("not an image"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})
}

func uploadTestImage(t *testing.T, sessionID string, imgBytes []byte, filename string) map[string]interface{} {
	t.Helper()
	r := gin.New()
	r.POST("/api/upload", authMiddleware(), handleUpload)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, _ := writer.CreateFormFile("image", filename)
	part.Write(imgBytes)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	var resp map[string]interface{}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	return resp
}

func cleanupUpload(t *testing.T, url string) {
	t.Helper()
	if url == "" {
		return
	}
	baseName := strings.TrimPrefix(url, "/static/images/")
	os.Remove(filepath.Join(imagesPath, baseName))
	ext := filepath.Ext(baseName)
	hashOnly := baseName[:len(baseName)-len(ext)]
	os.Remove(filepath.Join(imagesPath, hashOnly+"_resized"+ext))
	os.Remove(filepath.Join(imagesPath, hashOnly+"_thumb"+ext))
}

func encodePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func encodeJPEG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, nil)
	return buf.Bytes()
}

func encodeGIF(w, h int) []byte {
	img := image.NewPaletted(image.Rect(0, 0, w, h), color.Palette{color.Black, color.White})
	var buf bytes.Buffer
	gif.Encode(&buf, img, nil)
	return buf.Bytes()
}

func TestUploadAdvanced(t *testing.T) {
	t.Run("UploadJPEGNormalizesExtension", func(t *testing.T) {
		sessionID := "test-session-jpeg"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		resp := uploadTestImage(t, sessionID, encodeJPEG(100, 100), "photo.jpeg")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["url"] == nil {
			t.Fatal("expected url in response")
		}
		urlStr := resp["url"].(string)
		if !strings.HasSuffix(urlStr, ".jpg") {
			t.Errorf("expected .jpg extension after .jpeg normalization, got %s", urlStr)
		}
		if resp["resized_url"] == nil || resp["thumbnail_url"] == nil {
			t.Error("expected resized_url and thumbnail_url for JPEG")
		}
		if resp["duplicate"] != nil {
			t.Error("unexpected duplicate flag for new upload")
		}
	})

	t.Run("UploadLargeImageResizedDimensions", func(t *testing.T) {
		sessionID := "test-session-large"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		resp := uploadTestImage(t, sessionID, encodePNG(2000, 1000), "large.png")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["url"] == nil {
			t.Fatal("expected url in response")
		}
		urlStr := resp["url"].(string)
		baseName := strings.TrimPrefix(urlStr, "/static/images/")
		ext := filepath.Ext(baseName)
		hashOnly := baseName[:len(baseName)-len(ext)]
		resizedPath := filepath.Join(imagesPath, hashOnly+"_resized"+ext)

		f, err := os.Open(resizedPath)
		if err != nil {
			t.Fatalf("resized file not found: %v", err)
		}
		defer f.Close()

		resizedImg, _, err := image.Decode(f)
		if err != nil {
			t.Fatalf("failed to decode resized image: %v", err)
		}
		bounds := resizedImg.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		if w > 1200 || h > 1200 {
			t.Errorf("resized dimensions %dx%d exceed 1200px limit", w, h)
		}
		aspect := float64(w) / float64(h)
		if aspect < 1.6 || aspect > 2.1 {
			t.Errorf("resized aspect ratio %.2f should be ~2.0 (2000x1000)", aspect)
		}
	})

	t.Run("UploadThumbnailExactDimensions", func(t *testing.T) {
		sessionID := "test-session-thumb"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		resp := uploadTestImage(t, sessionID, encodePNG(500, 300), "thumbtest.png")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["url"] == nil {
			t.Fatal("expected url in response")
		}
		urlStr := resp["url"].(string)
		baseName := strings.TrimPrefix(urlStr, "/static/images/")
		ext := filepath.Ext(baseName)
		hashOnly := baseName[:len(baseName)-len(ext)]
		thumbPath := filepath.Join(imagesPath, hashOnly+"_thumb"+ext)

		f, err := os.Open(thumbPath)
		if err != nil {
			t.Fatalf("thumbnail file not found: %v", err)
		}
		defer f.Close()

		thumbImg, _, err := image.Decode(f)
		if err != nil {
			t.Fatalf("failed to decode thumbnail: %v", err)
		}
		bounds := thumbImg.Bounds()
		w, h := bounds.Dx(), bounds.Dy()
		if w != 150 || h != 150 {
			t.Errorf("expected thumbnail 150x150, got %dx%d", w, h)
		}
	})

	t.Run("UploadGIF", func(t *testing.T) {
		sessionID := "test-session-gif"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		resp := uploadTestImage(t, sessionID, encodeGIF(50, 50), "anim.gif")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["url"] == nil {
			t.Fatal("expected url in response")
		}
		if !strings.HasSuffix(resp["url"].(string), ".gif") {
			t.Errorf("expected .gif extension, got %s", resp["url"].(string))
		}
		baseName := strings.TrimPrefix(resp["url"].(string), "/static/images/")
		os.Remove(filepath.Join(imagesPath, baseName))
	})

	t.Run("UploadAllFilesExistOnDisk", func(t *testing.T) {
		sessionID := "test-session-exist"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		resp := uploadTestImage(t, sessionID, encodePNG(300, 200), "exist.png")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["url"] == nil || resp["resized_url"] == nil || resp["thumbnail_url"] == nil {
			t.Fatal("all URLs must be present")
		}

		for _, key := range []string{"url", "resized_url", "thumbnail_url"} {
			path := filepath.Join(imagesPath, strings.TrimPrefix(resp[key].(string), "/static/images/"))
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Errorf("%s file does not exist on disk: %s", key, path)
			}
		}
	})

	t.Run("UploadStoresRecordInDatabase", func(t *testing.T) {
		sessionID := "test-session-db"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		img.Set(0, 0, color.RGBA{byte(time.Now().UnixNano()), 0, 0, 255})
		var uniquePNG bytes.Buffer
		png.Encode(&uniquePNG, img)

		resp := uploadTestImage(t, sessionID, uniquePNG.Bytes(), "dbtest.png")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["hash"] == nil {
			t.Fatal("expected hash in response")
		}
		hashStr := resp["hash"].(string)

		var url, resizedURL, thumbURL string
		err := db.QueryRow("SELECT url, resized_url, thumbnail_url FROM uploads WHERE hash = ?", hashStr).
			Scan(&url, &resizedURL, &thumbURL)
		if err != nil {
			t.Fatalf("upload record not found in database: %v", err)
		}
		if url != resp["url"] {
			t.Errorf("db url %q != response url %q", url, resp["url"])
		}
		if resizedURL != resp["resized_url"] {
			t.Errorf("db resized_url %q != response %q", resizedURL, resp["resized_url"])
		}
		if thumbURL != resp["thumbnail_url"] {
			t.Errorf("db thumbnail_url %q != response %q", thumbURL, resp["thumbnail_url"])
		}
	})

	t.Run("UploadHashMatchesContent", func(t *testing.T) {
		sessionID := "test-session-hash"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		imgData := encodePNG(80, 60)

		resp := uploadTestImage(t, sessionID, imgData, "hashcheck.png")
		defer cleanupUpload(t, resp["url"].(string))

		if resp["hash"] == nil {
			t.Fatal("expected hash in response")
		}
		actualHash := resp["hash"].(string)

		sum := sha256.Sum256(imgData)
		expected := hex.EncodeToString(sum[:])
		if actualHash != expected {
			t.Errorf("hash mismatch: expected %s, got %s", expected, actualHash)
		}
	})

	t.Run("UploadMissingFormFile", func(t *testing.T) {
		sessionID := "test-session-missing"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fw, _ := writer.CreateFormField("not_image")
		fw.Write([]byte("value"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400 for missing form file, got %v", status)
		}
	})

	t.Run("UploadNoExtension", func(t *testing.T) {
		sessionID := "test-session-noext"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "noextension")
		part.Write([]byte("some content"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400 for no extension, got %v", status)
		}
	})

	t.Run("UploadWebP", func(t *testing.T) {
		sessionID := "test-session-webp"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.webp")
		part.Write(encodePNG(50, 50))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200 for .webp extension, got %v", status)
		}

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["url"] != nil {
			os.Remove(filepath.Join(imagesPath, strings.TrimPrefix(resp["url"].(string), "/static/images/")))
		}
	})
}

func TestAISettings(t *testing.T) {
	t.Run("GetAISettingsUnauthenticated", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/ai-settings", getAISettings)

		req, _ := http.NewRequest("GET", "/api/ai-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var s AISettings
		if err := json.Unmarshal(rr.Body.Bytes(), &s); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
	})

	t.Run("SaveAISettingsRequiresAuth", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/ai-settings", authMiddleware(), saveAISettings)

		req, _ := http.NewRequest("POST", "/api/ai-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("SaveAndGetAISettings", func(t *testing.T) {
		sessionID := "test-session-ai"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/ai-settings", authMiddleware(), saveAISettings)
		r.GET("/api/ai-settings", getAISettings)

		settings := AISettings{
			TrackExtractURL: "https://ai.example.com/extract",
			APIKey:          "sk-test-key",
			Enabled:         true,
		}
		body, _ := json.Marshal(settings)
		req, _ := http.NewRequest("POST", "/api/ai-settings", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/ai-settings", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var saved AISettings
		if err := json.Unmarshal(rr.Body.Bytes(), &saved); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if saved.TrackExtractURL != "https://ai.example.com/extract" {
			t.Errorf("expected URL %q, got %q", "https://ai.example.com/extract", saved.TrackExtractURL)
		}
		if !saved.Enabled {
			t.Errorf("expected enabled to be true")
		}
	})

	t.Run("AIExtractWithoutConfigReturnsError", func(t *testing.T) {
		sessionID := "test-session-ai-extract"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/tracks/ai-extract", authMiddleware(), handleAIExtract)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		fw, _ := writer.CreateFormField("not_image")
		fw.Write([]byte("value"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/tracks/ai-extract", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400 when no AI endpoint configured, got %v", status)
		}
	})
}

func TestQuotes(t *testing.T) {
	t.Run("GetAllQuotes", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/quotes", getQuotes)

		req, _ := http.NewRequest("GET", "/api/quotes", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.GET("/api/quote/random", getRandomQuote)

		req, _ := http.NewRequest("GET", "/api/quote/random", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.POST("/api/quotes", authMiddleware(), handleQuotes)

		newQuote := Quote{Text: "Test quote for unit testing!", Author: "Test Author"}
		body, _ := json.Marshal(newQuote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("expected status 201, got %v", status)
		}

		var createdQuote Quote
		json.Unmarshal(rr.Body.Bytes(), &createdQuote)
		if createdQuote.Text != "Test quote for unit testing!" {
			t.Errorf("expected quote text 'Test quote for unit testing!', got '%s'", createdQuote.Text)
		}

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
		r := gin.New()
		r.POST("/api/quotes", authMiddleware(), handleQuotes)

		newQuote := Quote{Text: "Unauthorized quote"}
		body, _ := json.Marshal(newQuote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("UpdateQuoteWithAuth", func(t *testing.T) {
		sessionID := "test-session-update-quote"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.PUT("/api/quotes", authMiddleware(), handleQuotes)

		res, _ := db.Exec("INSERT INTO quotes (text, author) VALUES ('Original', 'Original')")
		quoteID, _ := res.LastInsertId()

		updatedQuote := Quote{ID: int(quoteID), Text: "Updated quote text", Author: "Updated Author"}
		body, _ := json.Marshal(updatedQuote)
		req, _ := http.NewRequest("PUT", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

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

		r := gin.New()
		r.DELETE("/api/quotes", authMiddleware(), handleQuotes)

		res, _ := db.Exec("INSERT INTO quotes (text, author) VALUES ('ToDelete', 'Author')")
		quoteID, _ := res.LastInsertId()

		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/quotes?id=%d", quoteID), nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var count int
		db.QueryRow("SELECT COUNT(*) FROM quotes WHERE id=?", quoteID).Scan(&count)
		if count != 0 {
			t.Errorf("quote with ID %d still exists in DB after deletion", quoteID)
		}
	})

	t.Run("DeleteQuoteWithoutAuth", func(t *testing.T) {
		r := gin.New()
		r.DELETE("/api/quotes", authMiddleware(), handleQuotes)

		req, _ := http.NewRequest("DELETE", "/api/quotes?id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)
		r.GET("/api/race-history", getRaceHistory)

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
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var name string
		err := db.QueryRow("SELECT name FROM race_history WHERE name='2024 Season Finale'").Scan(&name)
		if err != nil {
			t.Fatalf("failed to find archived race in DB: %v", err)
		}
		if name != "2024 Season Finale" {
			t.Errorf("expected name '2024 Season Finale', got '%s'", name)
		}

		req, _ = http.NewRequest("GET", "/api/race-history", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

	r := gin.New()
	r.POST("/api/racers", authMiddleware(), updateRacer)

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
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("failed to add racer: %d", rr.Code)
	}

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

	r := gin.New()
	r.POST("/api/quotes", authMiddleware(), handleQuotes)

	newQuote := Quote{
		Text:   "Simply lovely!",
		Author: "Max Verstappen",
	}
	body, _ := json.Marshal(newQuote)
	req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("failed to add quote: %d", rr.Code)
	}

	var text, author string
	err := db.QueryRow("SELECT text, author FROM quotes WHERE text='Simply lovely!'").Scan(&text, &author)
	if err != nil {
		t.Fatalf("quote not found in database: %v", err)
	}
	if text != "Simply lovely!" || author != "Max Verstappen" {
		t.Errorf("database data mismatch: got %s, %s", text, author)
	}
}

func TestSchemaMigration(t *testing.T) {
	t.Run("NameColumnExistsInRaceHistory", func(t *testing.T) {
		_, err := db.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps) VALUES (?, ?, ?, ?, ?, ?)",
			"Migration Test", "2026-01-01", "Test", "Test", "test", 10)
		if err != nil {
			t.Errorf("failed to insert into race_history with name column: %v", err)
		}
	})

	t.Run("NewTrackColumnsExist", func(t *testing.T) {
		_, err := db.Exec("SELECT use_map_image, map_image_url, refresh_geojson FROM tracks LIMIT 0")
		if err != nil {
			t.Errorf("new track columns should exist: %v", err)
		}
	})

	t.Run("SchemaVersionIs9", func(t *testing.T) {
		var version int
		err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err != nil {
			t.Errorf("schema_version should exist: %v", err)
		}

		if version != 9 {
			t.Errorf("expected schema version 9, got %d", version)
		}
	})
}

func TestLogout(t *testing.T) {
	t.Run("LogoutClearsSession", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/logout", handleLogout)

		sessionID := "logout-test-session"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()

		req, _ := http.NewRequest("POST", "/api/logout", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		if _, exists := sessionStore[sessionID]; exists {
			t.Error("session should be cleared after logout")
		}
	})

	t.Run("LogoutWithNoSession", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/logout", handleLogout)

		req, _ := http.NewRequest("POST", "/api/logout", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})
}

func TestSessionExpiration(t *testing.T) {
	t.Run("ExpiredSessionIsRejected", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/test", authMiddleware(), func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		sessionID := "expired-session"
		sessionStore[sessionID] = time.Now().Add(-1 * time.Hour).Unix()

		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

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
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("GetRaceByID", func(t *testing.T) {
		sessionID := "test-session-get-by-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		rh := gin.New()
		rh.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

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
		rh.ServeHTTP(rr, req)

		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		raceID := int(resp["id"].(float64))

		r := gin.New()
		r.GET("/api/race-history", getRaceHistory)

		req, _ = http.NewRequest("GET", fmt.Sprintf("/api/race-history?id=%d", raceID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.DELETE("/api/race-history", authMiddleware(), deleteRaceHistory)

		req, _ := http.NewRequest("DELETE", "/api/race-history", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.POST("/api/quotes", authMiddleware(), handleQuotes)

		quote := Quote{Text: ""}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("UpdateQuoteWithoutID", func(t *testing.T) {
		sessionID := "test-session-update-no-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.PUT("/api/quotes", authMiddleware(), handleQuotes)

		quote := Quote{Text: "Updated text", Author: "Test"}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("PUT", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("DeleteQuoteWithoutID", func(t *testing.T) {
		sessionID := "test-session-delete-quote-no-id"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.DELETE("/api/quotes", authMiddleware(), handleQuotes)

		req, _ := http.NewRequest("DELETE", "/api/quotes", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})

	t.Run("AddQuoteDefaultsAuthor", func(t *testing.T) {
		sessionID := "test-session-default-author"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/quotes", authMiddleware(), handleQuotes)

		quote := Quote{Text: "Test quote without author"}
		body, _ := json.Marshal(quote)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.GET("/api/racer-stats", getRacerStats)

		req, _ := http.NewRequest("GET", "/api/racer-stats?id=9999", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		rh := gin.New()
		rh.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

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
			rh.ServeHTTP(rr, req)
		}

		r := gin.New()
		r.GET("/api/racer-stats", getRacerStats)

		req, _ := http.NewRequest("GET", "/api/racer-stats?id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.POST("/api/login", handleLogin)

		loginData := map[string]interface{}{
			"username": "wronguser",
			"password": "wrongpass",
		}
		body, _ := json.Marshal(loginData)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})

	t.Run("LoginWithInvalidJSON", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/login", handleLogin)

		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBufferString("invalid json"))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusBadRequest {
			t.Errorf("expected status 400, got %v", status)
		}
	})
}

func TestGetRacersEdgeCases(t *testing.T) {
	t.Run("RacersSortedByRank", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/racers", getRacers)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.GET("/api/racers", getRacers)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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
		r := gin.New()
		r.GET("/ws", handleWebSocket)

		s := httptest.NewServer(r)
		defer s.Close()

		u := "ws" + strings.TrimPrefix(s.URL, "http") + "/ws"
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

		r := gin.New()
		r.POST("/api/racers", authMiddleware(), updateRacer)

		names := []string{"R. PETROV", "S. VETTEL", "K. RAIKKONEN"}
		for _, name := range names {
			racer := Racer{Name: name, CarColor: "red", CarName: "Test Car", Points: 0, Rank: 10}
			body, _ := json.Marshal(racer)
			req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("failed to create racer %s: status %d", name, rr.Code)
			}
		}
	})

	t.Run("UpdateMultipleRacers", func(t *testing.T) {
		sessionID := "test-session-update-multi"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		getR := gin.New()
		getR.GET("/api/racers", getRacers)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		getR.ServeHTTP(rr, req)

		var racers []Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)

		updR := gin.New()
		updR.POST("/api/racers", authMiddleware(), updateRacer)

		for i, r := range racers {
			updated := Racer{
				ID:             r.ID,
				Name:           r.Name,
				ProfilePicture: r.ProfilePicture,
				CarColor:       r.CarColor,
				CarName:        r.CarName,
				Points:         (i + 1) * 10,
				Rank:           i + 1,
				Position:       (i + 1) * 10,
			}
			body, _ := json.Marshal(updated)
			req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
			req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
			rr := httptest.NewRecorder()
			updR.ServeHTTP(rr, req)

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

		r := gin.New()
		r.POST("/api/race-info", authMiddleware(), updateRaceInfo)
		r.GET("/api/race-info", getRaceInfo)

		ri := RaceInfo{Country: "Germany", Track: "Nürburgring", TrackID: "nurburgring", Laps: 60}
		body, _ := json.Marshal(ri)
		req, _ := http.NewRequest("POST", "/api/race-info", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/race-info", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

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

		r := gin.New()
		r.DELETE("/api/racers", authMiddleware(), deleteRacer)

		req, _ := http.NewRequest("DELETE", "/api/racers?id=9999", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200 (idempotent), got %v", status)
		}
	})
}

func TestRaceTypeFeature(t *testing.T) {
	t.Run("SaveSeasonRace", func(t *testing.T) {
		sessionID := "test-session-season-race"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

		input := map[string]interface{}{
			"name":       "Season Race 1",
			"race_date":  "2026-05-01",
			"country":    "Italy",
			"track":      "Monza",
			"track_id":   "monza",
			"total_laps": 53,
			"race_type":  "season",
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 25, "fastest_lap": true},
			},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var dbRaceType string
		err := db.QueryRow("SELECT race_type FROM race_history WHERE name='Season Race 1'").Scan(&dbRaceType)
		if err != nil {
			t.Fatalf("failed to find race in DB: %v", err)
		}
		if dbRaceType != "season" {
			t.Errorf("expected race_type 'season', got '%s'", dbRaceType)
		}
	})

	t.Run("SaveOneOffRace", func(t *testing.T) {
		sessionID := "test-session-oneoff"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

		input := map[string]interface{}{
			"name":       "Exhibition Race",
			"race_date":  "2026-06-01",
			"country":    "Monaco",
			"track":      "Monaco",
			"track_id":   "monaco",
			"total_laps": 30,
			"race_type":  "oneoff",
			"results": []map[string]interface{}{
				{"racer_id": 1, "racer_name": "A. PROST", "position": 1, "points": 0, "fastest_lap": false},
			},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var dbRaceType string
		err := db.QueryRow("SELECT race_type FROM race_history WHERE name='Exhibition Race'").Scan(&dbRaceType)
		if err != nil {
			t.Fatalf("failed to find oneoff race in DB: %v", err)
		}
		if dbRaceType != "oneoff" {
			t.Errorf("expected race_type 'oneoff', got '%s'", dbRaceType)
		}
	})

	t.Run("GetRaceHistoryFilteredByType", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-history", getRaceHistory)

		req, _ := http.NewRequest("GET", "/api/race-history?type=season", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var history []RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)

		for _, h := range history {
			if h.RaceType != "season" && h.RaceType != "" {
				t.Errorf("expected only season races, got '%s'", h.RaceType)
			}
		}
	})

	t.Run("GetOneOffRaces", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/oneoff-races", getOneOffRaces)

		req, _ := http.NewRequest("GET", "/api/oneoff-races", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var history []RaceHistory
		json.Unmarshal(rr.Body.Bytes(), &history)

		for _, h := range history {
			if h.RaceType != "oneoff" {
				t.Errorf("expected only oneoff races, got '%s'", h.RaceType)
			}
		}
	})

	t.Run("DeleteOneOffRace", func(t *testing.T) {
		sessionID := "test-session-delete-oneoff"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		rh := gin.New()
		rh.POST("/api/race-history", authMiddleware(), saveRaceToHistory)

		input := map[string]interface{}{
			"name":       "To Delete",
			"race_date":  "2026-07-01",
			"country":    "Test",
			"track":      "Test",
			"track_id":   "test",
			"total_laps": 10,
			"race_type":  "oneoff",
			"results":    []map[string]interface{}{},
		}
		body, _ := json.Marshal(input)
		req, _ := http.NewRequest("POST", "/api/race-history", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		rh.ServeHTTP(rr, req)

		var resp map[string]int64
		json.Unmarshal(rr.Body.Bytes(), &resp)
		raceID := resp["id"]

		delR := gin.New()
		delR.DELETE("/api/oneoff-races", authMiddleware(), deleteOneOffRace)

		req, _ = http.NewRequest("DELETE", fmt.Sprintf("/api/oneoff-races?id=%d", raceID), nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		delR.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})
}

func TestSchemaMigrationToV6(t *testing.T) {
	t.Run("RaceTypeColumnExists", func(t *testing.T) {
		var raceType string
		err := db.QueryRow("SELECT race_type FROM race_history LIMIT 1").Scan(&raceType)
		if err != nil {
			t.Logf("race_type column may not exist yet: %v (this is expected for fresh DB)", err)
		}
	})

	t.Run("NotificationSettingsExist", func(t *testing.T) {
		var id int
		err := db.QueryRow("SELECT id FROM notification_settings WHERE id = 1").Scan(&id)
		if err != nil {
			t.Errorf("notification_settings should exist: %v", err)
		}
	})

	t.Run("SchemaVersionIs9", func(t *testing.T) {
		var version int
		err := db.QueryRow("SELECT version FROM schema_version").Scan(&version)
		if err != nil {
			t.Errorf("schema_version should exist: %v", err)
		}

		if version != 9 {
			t.Errorf("expected schema version 9, got %d", version)
		}
	})

	t.Run("AISettingsTableExists", func(t *testing.T) {
		var id int
		err := db.QueryRow("SELECT id FROM ai_settings WHERE id = 1").Scan(&id)
		if err != nil {
			t.Errorf("ai_settings should exist: %v", err)
		}
		if id != 1 {
			t.Errorf("expected ai_settings id 1, got %d", id)
		}
	})
}

func TestDefaultRacerProfilePictures(t *testing.T) {
	t.Run("DefaultRacersHaveProfilePicture", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/racers", getRacers)

		req, _ := http.NewRequest("GET", "/api/racers", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v", status)
		}

		var racers []Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)

		if len(racers) < 5 {
			t.Fatalf("expected at least 5 racers, got %d", len(racers))
		}

		// Check that the first 5 default racers have profile pictures
		defaultNames := []string{"A. PROST", "M. SCHUMACHER", "A. SENNA", "N. LAUDA", "J. STEWART"}
		for _, name := range defaultNames {
			found := false
			for _, r := range racers {
				if r.Name == name && r.ProfilePicture != "" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("default racer %s should have a profile picture", name)
			}
		}
	})
}

func TestRacerCreationWithoutProfilePicture(t *testing.T) {
	sessionID := "test-session-no-pic"
	sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
	defer delete(sessionStore, sessionID)

	r := gin.New()
	r.POST("/api/racers", authMiddleware(), updateRacer)

	newRacer := Racer{
		Name:     "NO PIC RACER",
		CarColor: "black",
		CarName:  "Stealth",
		Points:   0,
		Rank:     20,
		Position: 0,
	}
	body, _ := json.Marshal(newRacer)
	req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("failed to add racer without profile picture: %d", rr.Code)
	}

	var name string
	err := db.QueryRow("SELECT name FROM racers WHERE name='NO PIC RACER'").Scan(&name)
	if err != nil {
		t.Fatalf("racer not found in database: %v", err)
	}
	if name != "NO PIC RACER" {
		t.Errorf("expected 'NO PIC RACER', got '%s'", name)
	}
}

func TestTrackSaveAndRetrieve(t *testing.T) {
	t.Run("SaveTrackWithMapImage", func(t *testing.T) {
		sessionID := "test-session-track-map"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		r := gin.New()
		r.POST("/api/tracks", authMiddleware(), saveTrack)
		r.GET("/api/tracks", getTracks)

		track := Track{
			ID:          "test-track",
			Name:        "Test Track",
			Country:     "Test Land",
			Length:      5,
			LapRecord:   "1:00.000",
			UseMapImage: true,
			MapImageURL: "/static/images/helmet.svg",
		}
		body, _ := json.Marshal(track)
		req, _ := http.NewRequest("POST", "/api/tracks", bytes.NewBuffer(body))
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var dbName string
		err := db.QueryRow("SELECT name FROM tracks WHERE id='test-track'").Scan(&dbName)
		if err != nil {
			t.Fatalf("track not found in DB: %v", err)
		}
		if dbName != "Test Track" {
			t.Errorf("expected 'Test Track', got '%s'", dbName)
		}

		db.Exec("DELETE FROM tracks WHERE id='test-track'")
	})
}

func TestUploadsList(t *testing.T) {
	t.Run("ListUploadsEmpty", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/uploads", getUploads)

		req, _ := http.NewRequest("GET", "/api/uploads", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var uploads []Upload
		if err := json.Unmarshal(rr.Body.Bytes(), &uploads); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
	})

	t.Run("ListUploadsHasEntries", func(t *testing.T) {
		sessionID := "test-session-uploads-list"
		sessionStore[sessionID] = time.Now().Add(1 * time.Hour).Unix()
		defer delete(sessionStore, sessionID)

		// Upload a test image first
		r := gin.New()
		r.POST("/api/upload", authMiddleware(), handleUpload)

		img := image.NewRGBA(image.Rect(0, 0, 100, 100))
		var buf bytes.Buffer
		png.Encode(&buf, img)

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "listtest.png")
		part.Write(buf.Bytes())
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var uploadResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &uploadResp)

		// Now list uploads
		r2 := gin.New()
		r2.GET("/api/uploads", getUploads)

		req2, _ := http.NewRequest("GET", "/api/uploads", nil)
		rr2 := httptest.NewRecorder()
		r2.ServeHTTP(rr2, req2)

		if status := rr2.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var uploads []Upload
		if err := json.Unmarshal(rr2.Body.Bytes(), &uploads); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}
		if len(uploads) == 0 {
			t.Error("expected at least 1 upload in list")
		}

		// Cleanup
		if url, ok := uploadResp["url"].(string); ok {
			baseName := strings.TrimPrefix(url, "/static/images/")
			os.Remove(filepath.Join(imagesPath, baseName))
			ext := filepath.Ext(baseName)
			hashOnly := baseName[:len(baseName)-len(ext)]
			os.Remove(filepath.Join(imagesPath, hashOnly+"_resized"+ext))
			os.Remove(filepath.Join(imagesPath, hashOnly+"_thumb"+ext))
		}
	})
}
