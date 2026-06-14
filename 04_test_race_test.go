package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func TestRaceInfo(t *testing.T) {
	t.Run("GetRaceInfo", func(t *testing.T) {
		r := gin.New()
		r.GET("/api/race-info", testHandler.GetRaceInfo)

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
		r.POST("/api/race-info", testHandler.UpdateRaceInfo)
		r.GET("/api/race-info", testHandler.GetRaceInfo)

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
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)
	r.GET("/api/race-history", testHandler.GetRaceHistory)
	r.DELETE("/api/race-history", testHandler.DeleteRaceHistory)

	t.Run("SaveAndGet", func(t *testing.T) {
		payload := map[string]any{
			"name":       "Test Race",
			"race_date":  "2025-01-01",
			"country":    "Italy",
			"track":      "Monza",
			"track_id":   "monza",
			"total_laps": 53,
			"race_type":  "season",
			"results": []map[string]any{
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

		var result map[string]any
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

func TestRaceHistoryWithDNS(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	// Save a race with one DNF and one DNS
	payload := map[string]any{
		"name":       "DNS Test Race",
		"race_date":  "2025-06-01",
		"country":    "Italy",
		"track":      "Monza",
		"track_id":   "monza",
		"total_laps": 53,
		"race_type":  "season",
		"results": []map[string]any{
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
	testServer.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'DNS Test Race')")
	testServer.DB.Exec("DELETE FROM race_history WHERE name = 'DNS Test Race'")
}

func TestOneOffRaces(t *testing.T) {
	r := gin.New()
	r.GET("/api/oneoff-races", testHandler.GetOneOffRaces)

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

func TestDeleteRaceHistory(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)
	r.DELETE("/api/race-history", testHandler.DeleteRaceHistory)
	r.GET("/api/race-history", testHandler.GetRaceHistory)

	t.Run("DeleteNonExistent", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/race-history?id=999", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("CreateAndDelete", func(t *testing.T) {
		payload := map[string]any{
			"name":       "Race To Delete",
			"race_date":  "2025-07-01",
			"country":    "Belgium",
			"track":      "Spa",
			"track_id":   "spa",
			"total_laps": 44,
			"race_type":  "season",
			"results": []map[string]any{
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
		var result map[string]any
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
	r.DELETE("/api/oneoff-races", testHandler.DeleteOneOffRace)

	req, _ := http.NewRequest("DELETE", "/api/oneoff-races?id=999", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}
}

func TestHeadToHead(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/head-to-head", testHandler.GetHeadToHead)

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
	r.GET("/api/stats/points-progression", testHandler.GetPointsProgression)

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
	r.GET("/api/stats/streaks", testHandler.GetStreaks)

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
	r.GET("/api/stats/elo", testHandler.GetELORatings)

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
	r.GET("/api/stats/export", testHandler.ExportStatsCSV)

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
	r.GET("/api/stats/track-performance", testHandler.GetTrackPerformance)

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

func TestRaceEventAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/race-events", testHandler.GetRaceEvents)
	r.POST("/api/race-events", testHandler.AddRaceEvent)

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

	testServer.DB.Exec("DELETE FROM race_events")
}

func TestSpectatorState(t *testing.T) {
	r := gin.New()
	r.GET("/api/spectator/state", testHandler.GetSpectatorState)

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

func TestRaceHistoryWithGoldSilverBronze(t *testing.T) {
	testServer.DB.Exec("DELETE FROM racer_stats")

	r := gin.New()
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	payload := map[string]any{
		"name":       "Gold Silver Bronze Test",
		"race_date":  "2025-07-01",
		"country":    "Italy",
		"track":      "Monza",
		"track_id":   "monza",
		"total_laps": 53,
		"race_type":  "season",
		"results": []map[string]any{
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

	testServer.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'Gold Silver Bronze Test')")
	testServer.DB.Exec("DELETE FROM race_history WHERE name = 'Gold Silver Bronze Test'")
	testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id IN (1,2,3,4,5)")
}

func TestOneOffRaceDoesNotUpdateStats(t *testing.T) {
	r := gin.New()
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	payload := map[string]any{
		"name":       "One-Off Test",
		"race_date":  "2025-08-01",
		"country":    "France",
		"track":      "Spa",
		"track_id":   "spa",
		"total_laps": 44,
		"race_type":  "oneoff",
		"results": []map[string]any{
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

	testServer.DB.Exec("DELETE FROM race_results WHERE race_id IN (SELECT id FROM race_history WHERE name = 'One-Off Test')")
	testServer.DB.Exec("DELETE FROM race_history WHERE name = 'One-Off Test'")
}
