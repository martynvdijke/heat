package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/models"
)

func TestGetRacerStatsSeasonFallback(t *testing.T) {
	testServer.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns, spins, overheated) VALUES (1, 5, 3, 3, 1, 0, 2, 0, 0, 8, 3)")
	testServer.DB.Exec("INSERT OR REPLACE INTO racer_stats (racer_id, races, wins, gold, silver, bronze, fastest_laps, dnf, dns, spins, overheated) VALUES (2, 5, 1, 1, 2, 1, 0, 1, 0, 4, 1)")
	defer testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id IN (1, 2)")
	// Explicit scopes are snapshot-derived only: use a dedicated season with
	// no snapshots so shared state cannot leak into the assertion.
	testServer.DB.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('NoSnapshot Season', '2026-01-01', '', 'archived')")
	var emptySeasonID int
	testServer.DB.QueryRow("SELECT id FROM seasons WHERE name = 'NoSnapshot Season'").Scan(&emptySeasonID)
	defer testServer.DB.Exec("DELETE FROM seasons WHERE name = 'NoSnapshot Season'")
	testServer.StatsCache.InvalidatePrefix("stats:")

	r := gin.New()
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	t.Run("ExplicitScopeWithoutSnapshotsReturnsEmpty", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/racer-stats?season_id=%d", emptySeasonID), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v", status)
		}

		var stats []models.RacerStats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(stats) != 0 {
			t.Errorf("expected empty stats for scope without snapshots (no silent all-time fallback), got %d entries", len(stats))
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
		if s.Spins != 8 {
			t.Errorf("expected spins=8, got %d", s.Spins)
		}
		if s.Overheated != 3 {
			t.Errorf("expected overheated=3, got %d", s.Overheated)
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
		RacerID: 3, Races: 10, Wins: 4, Gold: 4, Silver: 2, Bronze: 1, FastestLaps: 3, DNF: 1, DNS: 2, Spins: 5, Overheated: 2,
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
	if s.Races != 10 || s.Wins != 4 || s.Gold != 4 || s.Silver != 2 || s.Bronze != 1 || s.FastestLaps != 3 || s.DNF != 1 || s.DNS != 2 || s.Spins != 5 || s.Overheated != 2 {
		t.Errorf("expected stats (10,4,4,2,1,3,1,2,5,2), got (%d,%d,%d,%d,%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Gold, s.Silver, s.Bronze, s.FastestLaps, s.DNF, s.DNS, s.Spins, s.Overheated)
	}

	// Find the actual DB id for the update
	var statsID int
	testServer.DB.QueryRow("SELECT id FROM racer_stats WHERE racer_id = 3").Scan(&statsID)

	// Update existing stats
	updateBody, _ := json.Marshal(models.RacerStats{
		ID: statsID, RacerID: 3, Races: 20, Wins: 8, Gold: 8, Silver: 4, Bronze: 3, FastestLaps: 6, DNF: 2, DNS: 1, Spins: 10, Overheated: 4,
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
	if s.Races != 20 || s.Wins != 8 || s.Gold != 8 || s.Silver != 4 || s.Bronze != 3 || s.FastestLaps != 6 || s.DNF != 2 || s.DNS != 1 || s.Spins != 10 || s.Overheated != 4 {
		t.Errorf("expected updated stats (20,8,8,4,3,6,2,1,10,4), got (%d,%d,%d,%d,%d,%d,%d,%d,%d,%d)", s.Races, s.Wins, s.Gold, s.Silver, s.Bronze, s.FastestLaps, s.DNF, s.DNS, s.Spins, s.Overheated)
	}
}

func TestRacerStatsFromRoundSnapshots(t *testing.T) {
	// Insert a dedicated season for this test
	seasonID := 9998
	testServer.DB.Exec("INSERT OR IGNORE INTO seasons (id, name, start_date, status) VALUES (?, 'Snapshot Test Season', '2025-01-01', 'active')", seasonID)

	// Insert two finalized round snapshots
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshots (id, season_id, race_name, race_date, round, status) VALUES (99901, ?, 'Round 1', '2025-01-15', 1, 'final')", seasonID)
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshots (id, season_id, race_name, race_date, round, status) VALUES (99902, ?, 'Round 2', '2025-02-15', 2, 'final')", seasonID)

	// Insert scores: racer 1 gets 25pts (P1) in round 1, 18pts (P2) in round 2
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (999011, 99901, 1, 'A. PROST', 25, 1, 0, 0, 3, 1)")
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (999012, 99902, 1, 'A. PROST', 18, 2, 0, 0, 2, 0)")

	// Racer 2 gets 18pts (P2) in round 1, 15pts (P3) in round 2, with a DNF in round 2
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (999021, 99901, 2, 'M. SCHUMACHER', 18, 2, 0, 0, 5, 2)")
	testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (999022, 99902, 2, 'M. SCHUMACHER', 0, 999, 1, 0, 1, 0)")

	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id IN (999011, 999012, 999021, 999022)")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id IN (99901, 99902)")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := gin.New()
	r.GET("/api/racer-stats", testHandler.GetRacerStats)

	t.Run("AggregateStatsFromSnapshots", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/racer-stats?season_id=%d", seasonID), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v: %s", status, rr.Body.String())
		}

		var stats []models.RacerStats
		if err := json.Unmarshal(rr.Body.Bytes(), &stats); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if len(stats) == 0 {
			t.Fatal("expected non-empty stats from round snapshots")
		}

		for _, s := range stats {
			switch s.RacerID {
			case 1:
				if s.Points != 43 {
					t.Errorf("racer 1: expected points=43, got %d", s.Points)
				}
				if s.Races != 2 {
					t.Errorf("racer 1: expected races=2, got %d", s.Races)
				}
				if s.Wins != 1 || s.Gold != 1 {
					t.Errorf("racer 1: expected wins=1, gold=1, got wins=%d, gold=%d", s.Wins, s.Gold)
				}
				if s.Spins != 5 {
					t.Errorf("racer 1: expected spins=5, got %d", s.Spins)
				}
				if s.Overheated != 1 {
					t.Errorf("racer 1: expected overheated=1, got %d", s.Overheated)
				}
			case 2:
				if s.Points != 18 {
					t.Errorf("racer 2: expected points=18, got %d", s.Points)
				}
				if s.Races != 2 {
					t.Errorf("racer 2: expected races=2, got %d", s.Races)
				}
				if s.DNF != 1 {
					t.Errorf("racer 2: expected dnf=1, got %d", s.DNF)
				}
				if s.Spins != 6 {
					t.Errorf("racer 2: expected spins=6, got %d", s.Spins)
				}
				if s.Overheated != 2 {
					t.Errorf("racer 2: expected overheated=2, got %d", s.Overheated)
				}
			}
		}
	})

	t.Run("SingleRacerSeasonStats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/racer-stats?id=1&season_id=%d", seasonID), nil)
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

		if s.RacerID != 1 || s.Points != 43 || s.Races != 2 {
			t.Errorf("expected racer_id=1, points=43, races=2, got racer_id=%d, points=%d, races=%d", s.RacerID, s.Points, s.Races)
		}
		if s.Spins != 5 {
			t.Errorf("expected spins=5, got %d", s.Spins)
		}
		if s.Overheated != 1 {
			t.Errorf("expected overheated=1, got %d", s.Overheated)
		}
	})
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

	// All-time stats are snapshot-derived (aggregated across every finalized
	// snapshot); legacy racer_stats rows are no longer part of the list view.
	// Seed a dedicated season + finalized snapshot and clean up only those rows.
	testServer.DB.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('All Scope Season', '2026-01-01', '', 'archived')")
	var seasonID int
	testServer.DB.QueryRow("SELECT id FROM seasons WHERE name = 'All Scope Season'").Scan(&seasonID)
	res, err := testServer.DB.Exec("INSERT INTO round_snapshots (race_name, race_date, round, created_at, season_id, status) VALUES ('All Scope Race', '2026-06-01', 1, '', ?, 'final')", seasonID)
	if err != nil {
		t.Fatalf("failed to seed snapshot: %v", err)
	}
	snapID, _ := res.LastInsertId()
	testServer.DB.Exec("INSERT INTO round_snapshot_scores (snapshot_id, racer_id, racer_name, position, points, dnf, dns, spins, overheated) VALUES (?, 7, 'Racer 7', 1, 25, 0, 0, 0, 0)", snapID)
	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id = ?", snapID)
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id = ?", snapID)
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()
	testServer.StatsCache.InvalidatePrefix("stats:")

	req, _ := http.NewRequest("GET", "/api/racer-stats", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	var stats []models.RacerStats
	json.Unmarshal(rr.Body.Bytes(), &stats)
	found := false
	for _, s := range stats {
		if s.RacerID == 7 {
			found = true
			if s.Points != 25 || s.Wins != 1 || s.Races != 1 {
				t.Errorf("racer 7 stats mismatch: points=%d wins=%d races=%d", s.Points, s.Wins, s.Races)
			}
		}
	}
	if !found {
		t.Error("expected racer 7 snapshot-derived stats to be returned")
	}
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
		var data []any
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
