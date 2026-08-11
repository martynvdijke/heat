package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// TestAdminSeasonStatsSpinsOverheated verifies the spins/overheated round-trip
// that the admin Season tab now surfaces: POST /api/racer-stats persists them
// and GET /api/racer-stats returns them (including the DB row itself).
func TestAdminSeasonStatsSpinsOverheated(t *testing.T) {
	r := gin.New()
	r.POST("/api/racer-stats", testHandler.UpdateRacerStats)
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	const racerID = 42
	testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id = ?", racerID)
	defer testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id = ?", racerID)

	t.Run("CreatePersistsSpinsAndOverheated", func(t *testing.T) {
		createBody, _ := json.Marshal(models.RacerStats{
			RacerID: racerID, Races: 4, Wins: 1, Gold: 1, Silver: 1, Bronze: 0,
			FastestLaps: 1, DNF: 0, DNS: 0, Spins: 6, Overheated: 2,
		})
		req, _ := http.NewRequest("POST", "/api/racer-stats", bytes.NewBuffer(createBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("create: expected status 200, got %v: %s", status, rr.Body.String())
		}

		// API returns the values
		req, _ = http.NewRequest("GET", "/api/racer-stats?id=42", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("get after create: expected status 200, got %v", status)
		}
		var result map[string]json.RawMessage
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		var s models.RacerStats
		if err := json.Unmarshal(result["stats"], &s); err != nil {
			t.Fatalf("failed to unmarshal stats: %v", err)
		}
		if s.Spins != 6 || s.Overheated != 2 {
			t.Errorf("expected spins=6, overheated=2, got spins=%d, overheated=%d", s.Spins, s.Overheated)
		}

		// DB row persists the values
		var dbSpins, dbOverheated int
		err := testServer.DB.QueryRow("SELECT spins, overheated FROM racer_stats WHERE racer_id = ?", racerID).Scan(&dbSpins, &dbOverheated)
		if err != nil {
			t.Fatalf("failed to read racer_stats row: %v", err)
		}
		if dbSpins != 6 || dbOverheated != 2 {
			t.Errorf("expected DB spins=6, overheated=2, got spins=%d, overheated=%d", dbSpins, dbOverheated)
		}
	})

	t.Run("UpdatePreservesSpinsAndOverheated", func(t *testing.T) {
		// Update the same racer's stats via the admin save path (races bumped).
		var statsID int
		if err := testServer.DB.QueryRow("SELECT id FROM racer_stats WHERE racer_id = ?", racerID).Scan(&statsID); err != nil {
			t.Fatalf("failed to load stats id: %v", err)
		}
		updateBody, _ := json.Marshal(models.RacerStats{
			ID: statsID, RacerID: racerID, Races: 5, Wins: 1, Gold: 1, Silver: 1, Bronze: 0,
			FastestLaps: 1, DNF: 0, DNS: 0, Spins: 7, Overheated: 3,
		})
		req, _ := http.NewRequest("POST", "/api/racer-stats", bytes.NewBuffer(updateBody))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("update: expected status 200, got %v: %s", status, rr.Body.String())
		}

		req, _ = http.NewRequest("GET", "/api/racer-stats?id=42", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var result map[string]json.RawMessage
		if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}
		var s models.RacerStats
		if err := json.Unmarshal(result["stats"], &s); err != nil {
			t.Fatalf("failed to unmarshal stats: %v", err)
		}
		if s.Races != 5 || s.Spins != 7 || s.Overheated != 3 {
			t.Errorf("expected races=5, spins=7, overheated=3, got races=%d, spins=%d, overheated=%d", s.Races, s.Spins, s.Overheated)
		}
	})
}
