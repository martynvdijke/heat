package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/models"
)

func TestRaceHistorySeasonMigration(t *testing.T) {
	database := testServer.DB

	cleanup := func() {
		database.Exec("DELETE FROM race_history")
		database.Exec("DELETE FROM round_snapshots")
		database.Exec("DELETE FROM round_snapshot_scores")
		database.Exec("DELETE FROM seasons")
	}
	cleanup()
	defer cleanup()

	// Column exists after Init; exercise the helper directly (idempotent).
	if err := db.EnsureRaceHistorySeasonColumn(database); err != nil {
		t.Fatalf("EnsureRaceHistorySeasonColumn: %v", err)
	}
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM pragma_table_info('race_history') WHERE name = 'season_id'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("season_id column missing (count=%d err=%v)", count, err)
	}

	database.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('Mig Season A', '2026-01-01', '2026-02-01', 'archived')")
	database.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('Mig Season B', '2026-03-01', '2026-04-01', 'active')")
	var seasonA, seasonB int
	database.QueryRow("SELECT id FROM seasons WHERE name = 'Mig Season A'").Scan(&seasonA)
	database.QueryRow("SELECT id FROM seasons WHERE name = 'Mig Season B'").Scan(&seasonB)
	if seasonA == 0 || seasonB == 0 {
		t.Fatalf("failed to seed seasons (A=%d B=%d)", seasonA, seasonB)
	}

	database.Exec("INSERT INTO round_snapshots (race_name, race_date, round, created_at, season_id, status) VALUES ('GP Alpha', '2026-01-10', 1, '', ?, 'final')", seasonA)
	database.Exec("INSERT INTO round_snapshots (race_name, race_date, round, created_at, season_id, status) VALUES ('GP Beta', '2026-03-10', 1, '', ?, 'final')", seasonB)
	database.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES ('GP Alpha', '2026-01-10', 'Italy', 'Monza', 'monza', 10, 'season')")
	database.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES ('GP Beta', '2026-03-10', 'Italy', 'Monza', 'monza', 10, 'season')")
	database.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES ('Mig Oneoff', '2026-05-01', 'Italy', 'Monza', 'monza', 10, 'oneoff')")

	if err := db.BackfillRaceHistorySeasons(database); err != nil {
		t.Fatalf("BackfillRaceHistorySeasons: %v", err)
	}

	var gotA, gotB int
	var oneoffSeason sql.NullInt64
	database.QueryRow("SELECT COALESCE(season_id, 0) FROM race_history WHERE name = 'GP Alpha'").Scan(&gotA)
	database.QueryRow("SELECT COALESCE(season_id, 0) FROM race_history WHERE name = 'GP Beta'").Scan(&gotB)
	database.QueryRow("SELECT season_id FROM race_history WHERE name = 'Mig Oneoff'").Scan(&oneoffSeason)

	if gotA != seasonA {
		t.Errorf("GP Alpha: expected season %d, got %d", seasonA, gotA)
	}
	if gotB != seasonB {
		t.Errorf("GP Beta: expected season %d, got %d", seasonB, gotB)
	}
	if oneoffSeason.Valid {
		t.Errorf("unmatched race should stay NULL, got %d", oneoffSeason.Int64)
	}

	// Idempotent: re-running both helpers must not change anything.
	if err := db.EnsureRaceHistorySeasonColumn(database); err != nil {
		t.Fatalf("second EnsureRaceHistorySeasonColumn: %v", err)
	}
	if err := db.BackfillRaceHistorySeasons(database); err != nil {
		t.Fatalf("second BackfillRaceHistorySeasons: %v", err)
	}
	database.QueryRow("SELECT COALESCE(season_id, 0) FROM race_history WHERE name = 'GP Alpha'").Scan(&gotA)
	if gotA != seasonA {
		t.Errorf("idempotent backfill changed GP Alpha season: %d -> %d", seasonA, gotA)
	}
}

func TestSeasonScopedStatsAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-stats", testHandler.GetRacerStats)
	r.POST("/api/race-history", testHandler.SaveRaceToHistory)

	database := testServer.DB
	cleanup := func() {
		database.Exec("DELETE FROM race_results")
		database.Exec("DELETE FROM race_history")
		database.Exec("DELETE FROM round_snapshot_scores")
		database.Exec("DELETE FROM round_snapshots")
		database.Exec("DELETE FROM seasons")
	}
	cleanup()
	defer cleanup()

	// Seed two seasons with finalized snapshots and scores:
	// Season A: racer 1 wins both rounds; Season B: racer 2 wins the round.
	database.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('Scope A', '2026-01-01', '2026-02-01', 'archived')")
	database.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('Scope B', '2026-03-01', '', 'active')")
	var seasonA, seasonB int
	database.QueryRow("SELECT id FROM seasons WHERE name = 'Scope A'").Scan(&seasonA)
	database.QueryRow("SELECT id FROM seasons WHERE name = 'Scope B'").Scan(&seasonB)

	insertSnapshot := func(name string, seasonID int, round int) int64 {
		res, err := database.Exec("INSERT INTO round_snapshots (race_name, race_date, round, created_at, season_id, status) VALUES (?, '2026-06-01', ?, '', ?, 'final')", name, round, seasonID)
		if err != nil {
			t.Fatalf("insert snapshot: %v", err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	insertScore := func(snapshotID int64, racerID, position, points int) {
		database.Exec("INSERT INTO round_snapshot_scores (snapshot_id, racer_id, racer_name, position, points, dnf, dns, spins, overheated) VALUES (?, ?, ?, ?, ?, 0, 0, 0, 0)",
			snapshotID, racerID, fmt.Sprintf("Racer %d", racerID), position, points)
	}

	snapA1 := insertSnapshot("Scope Race 1", seasonA, 1)
	insertScore(snapA1, 1, 1, 25)
	insertScore(snapA1, 2, 2, 18)
	snapA2 := insertSnapshot("Scope Race 2", seasonA, 2)
	insertScore(snapA2, 1, 1, 25)
	insertScore(snapA2, 2, 2, 18)
	snapB1 := insertSnapshot("Scope Race 3", seasonB, 1)
	insertScore(snapB1, 1, 2, 18)
	insertScore(snapB1, 2, 1, 25)

	fetchStats := func(query string) ([]models.RacerStats, int) {
		req, _ := http.NewRequest("GET", "/api/racer-stats"+query, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var stats []models.RacerStats
		json.Unmarshal(rr.Body.Bytes(), &stats)
		return stats, rr.Code
	}

	t.Run("season_id alias scopes to a single season", func(t *testing.T) {
		stats, code := fetchStats(fmt.Sprintf("?season_id=%d", seasonA))
		if code != http.StatusOK || len(stats) != 2 {
			t.Fatalf("expected 200 with 2 racers, got %d (%d entries)", code, len(stats))
		}
		if stats[0].RacerID != 1 || stats[0].Points != 50 || stats[0].Races != 2 {
			t.Errorf("season A leader mismatch: %+v", stats[0])
		}
	})

	t.Run("multi-season scope aggregates sums", func(t *testing.T) {
		stats, code := fetchStats(fmt.Sprintf("?season_ids=%d,%d", seasonA, seasonB))
		if code != http.StatusOK || len(stats) != 2 {
			t.Fatalf("expected 200 with 2 racers, got %d (%d entries)", code, len(stats))
		}
		byRacer := map[int]models.RacerStats{}
		for _, s := range stats {
			byRacer[s.RacerID] = s
		}
		if byRacer[1].Points != 68 || byRacer[1].Wins != 2 || byRacer[1].Races != 3 {
			t.Errorf("racer 1 aggregate mismatch: %+v", byRacer[1])
		}
		if byRacer[2].Points != 61 || byRacer[2].Wins != 1 || byRacer[2].Races != 3 {
			t.Errorf("racer 2 aggregate mismatch: %+v", byRacer[2])
		}
	})

	t.Run("absent scope means all seasons", func(t *testing.T) {
		stats, code := fetchStats("")
		if code != http.StatusOK || len(stats) != 2 {
			t.Fatalf("expected 200 with 2 racers, got %d (%d entries)", code, len(stats))
		}
		total := 0
		for _, s := range stats {
			total += s.Points
		}
		if total != 129 {
			t.Errorf("all-season total points = %d, want 129", total)
		}
	})

	t.Run("explicit scope with no data returns empty list", func(t *testing.T) {
		database.Exec("INSERT INTO seasons (name, start_date, end_date, status) VALUES ('Scope Empty', '2026-05-01', '', 'archived')")
		var emptySeason int
		database.QueryRow("SELECT id FROM seasons WHERE name = 'Scope Empty'").Scan(&emptySeason)
		stats, code := fetchStats(fmt.Sprintf("?season_ids=%d", emptySeason))
		if code != http.StatusOK {
			t.Fatalf("expected 200, got %d", code)
		}
		if len(stats) != 0 {
			t.Errorf("expected empty result, got %d entries", len(stats))
		}
	})

	t.Run("invalid season_ids returns 400", func(t *testing.T) {
		_, code := fetchStats("?season_ids=abc")
		if code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", code)
		}
		_, code = fetchStats("?season_ids=0")
		if code != http.StatusBadRequest {
			t.Errorf("expected 400 for non-positive id, got %d", code)
		}
	})

	t.Run("SaveRace links season explicitly, resolves active, NULLs oneoffs", func(t *testing.T) {
		postRace := func(body string) (int64, int) {
			req, _ := http.NewRequest("POST", "/api/race-history", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)
			var resp struct {
				ID int64 `json:"id"`
			}
			json.Unmarshal(rr.Body.Bytes(), &resp)
			return resp.ID, rr.Code
		}

		// Explicit season_id wins even though Scope B is active.
		id, code := postRace(fmt.Sprintf(`{"name":"Explicit Link","race_type":"season","season_id":%d,"results":[]}`, seasonA))
		if code != http.StatusOK {
			t.Fatalf("explicit save failed: %d", code)
		}
		var got int
		database.QueryRow("SELECT COALESCE(season_id, 0) FROM race_history WHERE id = ?", id).Scan(&got)
		if got != seasonA {
			t.Errorf("explicit season link: expected %d, got %d", seasonA, got)
		}

		// No season_id + season type resolves the active season (Scope B).
		id, code = postRace(`{"name":"Resolved Link","race_type":"season","results":[]}`)
		if code != http.StatusOK {
			t.Fatalf("resolved save failed: %d", code)
		}
		database.QueryRow("SELECT COALESCE(season_id, 0) FROM race_history WHERE id = ?", id).Scan(&got)
		if got != seasonB {
			t.Errorf("active season resolution: expected %d, got %d", seasonB, got)
		}

		// Oneoff stays unlinked.
		id, code = postRace(`{"name":"Oneoff Link","race_type":"oneoff","results":[]}`)
		if code != http.StatusOK {
			t.Fatalf("oneoff save failed: %d", code)
		}
		var nullSeason sql.NullInt64
		database.QueryRow("SELECT season_id FROM race_history WHERE id = ?", id).Scan(&nullSeason)
		if nullSeason.Valid {
			t.Errorf("oneoff should stay NULL, got %d", nullSeason.Int64)
		}
	})
}
