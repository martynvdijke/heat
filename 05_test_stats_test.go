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

func TestGetRacerStatsSeasonFallback(t *testing.T) {
	testServer.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (1, 5, 3, 3, 1, 0, 2, 0, 0)")
	testServer.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (2, 5, 1, 1, 2, 1, 0, 1, 0)")
	defer testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id IN (1, 2)")

	r := gin.New()
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

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
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

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
	r.POST("/api/racer-stats", testHandler.UpdateRacerStats)
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

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
	testServer.DB.QueryRow("SELECT id FROM racer_stats WHERE racer_id = 3").Scan(&statsID)

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

func TestGetTrackStats(t *testing.T) {
	r := gin.New()
	r.GET("/api/track-stats", testHandler.GetTrackStats)

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

func TestGetRacerStatsAll(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	testServer.DB.Exec("INSERT INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns) VALUES (1, 5, 3, 3, 1, 0, 2, 0, 0)")

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

	testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id = 1")
}

func TestDeeperStats(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/qualifying-delta", testHandler.GetQualifyingRaceDelta)
	r.GET("/api/stats/consistency", testHandler.GetConsistencyRatings)
	r.GET("/api/stats/incidents", testHandler.GetRaceIncidentsReport)
	r.GET("/api/stats/pace-heatmap", testHandler.GetPaceHeatmap)

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

}
