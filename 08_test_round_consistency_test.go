package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"heat/racing"
)

// roundHarness holds the isolated state for a round-consistency test: a unique
// season, three unique racers, and a draft round containing only those racers.
type roundHarness struct {
	router   *gin.Engine
	seasonID int
	racerIDs []int
	roundID  int
	scoreIDs map[int]int // racer_id -> round_snapshot_scores.id
}

type scoreUpdate struct {
	ID         int  `json:"id"`
	Position   int  `json:"position"`
	DNF        bool `json:"dnf"`
	DNS        bool `json:"dns"`
	Spins      int  `json:"spins"`
	Overheated int  `json:"overheated"`
}

// snapshotRacerState captures points/rank for every racer and every racer_stats
// row so a test can restore the shared in-memory DB afterwards. This keeps the
// global rank recalculation from leaking into unrelated tests.
func snapshotRacerState(t *testing.T) func() {
	t.Helper()

	type racerRow struct {
		ID, Points, Rank int
	}
	var racers []racerRow
	rows, err := testServer.DB.Query("SELECT id, points, rank FROM racers")
	if err != nil {
		t.Fatalf("snapshot racers: %v", err)
	}
	for rows.Next() {
		var r racerRow
		if err := rows.Scan(&r.ID, &r.Points, &r.Rank); err == nil {
			racers = append(racers, r)
		}
	}
	rows.Close()

	type statsRow struct {
		ID, RacerID, Races, Wins, Gold, Silver, Bronze, FastestLaps, Points, DNF, DNS, Spins, Overheated int
	}
	var stats []statsRow
	srows, err := testServer.DB.Query("SELECT id, racer_id, races, wins, gold, silver, bronze, fastest_laps, points, dnf, dns, spins, overheated FROM racer_stats")
	if err != nil {
		t.Fatalf("snapshot racer_stats: %v", err)
	}
	for srows.Next() {
		var s statsRow
		if err := srows.Scan(&s.ID, &s.RacerID, &s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.FastestLaps, &s.Points, &s.DNF, &s.DNS, &s.Spins, &s.Overheated); err == nil {
			stats = append(stats, s)
		}
	}
	srows.Close()

	return func() {
		tx, err := testServer.DB.Begin()
		if err != nil {
			return
		}
		defer tx.Rollback()
		for _, r := range racers {
			tx.Exec("UPDATE racers SET points = ?, rank = ? WHERE id = ?", r.Points, r.Rank, r.ID)
		}
		tx.Exec("DELETE FROM racer_stats")
		for _, s := range stats {
			tx.Exec("INSERT INTO racer_stats (id, racer_id, races, wins, gold, silver, bronze, fastest_laps, points, dnf, dns, spins, overheated) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
				s.ID, s.RacerID, s.Races, s.Wins, s.Gold, s.Silver, s.Bronze, s.FastestLaps, s.Points, s.DNF, s.DNS, s.Spins, s.Overheated)
		}
		tx.Commit()
	}
}

func setupRoundHarness(t *testing.T) *roundHarness {
	t.Helper()

	restore := snapshotRacerState(t)
	t.Cleanup(restore)

	uniq := time.Now().UnixNano()

	res, err := testServer.DB.Exec("INSERT INTO seasons (name, start_date, status) VALUES (?, date('now'), 'active')", fmt.Sprintf("round-consistency-%d", uniq))
	if err != nil {
		t.Fatalf("insert season: %v", err)
	}
	seasonID64, _ := res.LastInsertId()
	seasonID := int(seasonID64)

	names := []string{
		fmt.Sprintf("CONSISTENCY A %d", uniq),
		fmt.Sprintf("CONSISTENCY B %d", uniq),
		fmt.Sprintf("CONSISTENCY C %d", uniq),
	}
	racerIDs := make([]int, 0, len(names))
	for _, name := range names {
		r, err := testServer.DB.Exec("INSERT INTO racers (name, profile_picture, car_color, car_name, points, rank) VALUES (?, '', 'red', 'Test Car', 0, 0)", name)
		if err != nil {
			t.Fatalf("insert racer: %v", err)
		}
		id, _ := r.LastInsertId()
		racerIDs = append(racerIDs, int(id))
	}

	t.Cleanup(func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id IN (SELECT id FROM round_snapshots WHERE season_id = ?)", seasonID)
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE season_id = ?", seasonID)
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
		for _, id := range racerIDs {
			testServer.DB.Exec("DELETE FROM racer_stats WHERE racer_id = ?", id)
			testServer.DB.Exec("DELETE FROM racers WHERE id = ?", id)
		}
	})

	r := gin.New()
	r.POST("/api/rounds", testHandler.TakeRoundSnapshot)
	r.GET("/api/rounds", testHandler.GetRoundSnapshots)
	r.PATCH("/api/rounds", testHandler.UpdateRoundScores)
	r.PATCH("/api/rounds/finalize", testHandler.FinalizeRound)
	r.DELETE("/api/rounds", testHandler.DeleteRoundSnapshot)

	body, _ := json.Marshal(map[string]any{
		"race_name": fmt.Sprintf("Round Consistency %d", uniq),
		"season_id": seasonID,
	})
	req, _ := http.NewRequest("POST", "/api/rounds", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create round: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	json.Unmarshal(rr.Body.Bytes(), &out)
	roundID := int(out["id"].(float64))

	// TakeRoundSnapshot snapshots every racer. Trim it down to our three test
	// racers so finalize/edit/delete never touch shared seed racer data.
	if _, err := testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE snapshot_id = ? AND racer_id NOT IN (?, ?, ?)",
		roundID, racerIDs[0], racerIDs[1], racerIDs[2]); err != nil {
		t.Fatalf("trim snapshot scores: %v", err)
	}

	scoreIDs, err := scoreIDsForRound(t, roundID)
	if err != nil {
		t.Fatalf("load score ids: %v", err)
	}
	for _, id := range racerIDs {
		if _, ok := scoreIDs[id]; !ok {
			t.Fatalf("missing score for test racer %d in round %d", id, roundID)
		}
	}

	return &roundHarness{
		router:   r,
		seasonID: seasonID,
		racerIDs: racerIDs,
		roundID:  roundID,
		scoreIDs: scoreIDs,
	}
}

func scoreIDsForRound(t *testing.T, roundID int) (map[int]int, error) {
	t.Helper()
	rows, err := testServer.DB.Query("SELECT id, racer_id FROM round_snapshot_scores WHERE snapshot_id = ?", roundID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[int]int)
	for rows.Next() {
		var scoreID, racerID int
		if err := rows.Scan(&scoreID, &racerID); err == nil {
			out[racerID] = scoreID
		}
	}
	return out, rows.Err()
}

// patchPositions assigns the given positions to the harness racers in order
// (racerIDs[0] gets positions[0], etc.) and asserts the request succeeded.
func (h *roundHarness) patchPositions(t *testing.T, positions ...int) {
	t.Helper()
	updates := make([]scoreUpdate, 0, len(positions))
	for i, pos := range positions {
		updates = append(updates, scoreUpdate{ID: h.scoreIDs[h.racerIDs[i]], Position: pos})
	}
	body, _ := json.Marshal(updates)
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/rounds?id=%d", h.roundID), bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("patch scores: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func (h *roundHarness) finalize(t *testing.T) {
	t.Helper()
	req, _ := http.NewRequest("PATCH", fmt.Sprintf("/api/rounds/finalize?id=%d", h.roundID), nil)
	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("finalize: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func racerPointsRank(t *testing.T, racerID int) (points, rank int) {
	t.Helper()
	if err := testServer.DB.QueryRow("SELECT points, rank FROM racers WHERE id = ?", racerID).Scan(&points, &rank); err != nil {
		t.Fatalf("load racer %d: %v", racerID, err)
	}
	return points, rank
}

type statsSnapshot struct {
	Races, Wins, Gold, Silver, Bronze, DNF, DNS, Spins, Overheated int
}

func racerStatsFor(t *testing.T, racerID int) statsSnapshot {
	t.Helper()
	var s statsSnapshot
	err := testServer.DB.QueryRow(`SELECT races, wins, gold, silver, bronze, dnf, dns, spins, overheated FROM racer_stats WHERE racer_id = ?`, racerID).
		Scan(&s.Races, &s.Wins, &s.Gold, &s.Silver, &s.Bronze, &s.DNF, &s.DNS, &s.Spins, &s.Overheated)
	if err != nil {
		t.Fatalf("load racer_stats for %d: %v", racerID, err)
	}
	return s
}

func assertStats(t *testing.T, label string, got, want statsSnapshot) {
	t.Helper()
	if got != want {
		t.Errorf("%s: stats = %+v, want %+v", label, got, want)
	}
}

// TestFinalizeRoundAppliesPodiumStats verifies that finalizing a round awards
// points, counts the race, and records wins/gold/silver/bronze per position,
// and that ranks are recomputed from points.
func TestFinalizeRoundAppliesPodiumStats(t *testing.T) {
	h := setupRoundHarness(t)

	h.patchPositions(t, 1, 2, 3)
	h.finalize(t)

	wantPoints := []int{9, 6, 4}
	for i, id := range h.racerIDs {
		points, _ := racerPointsRank(t, id)
		if points != wantPoints[i] {
			t.Errorf("racer %d points = %d, want %d", id, points, wantPoints[i])
		}
	}

	assertStats(t, "P1", racerStatsFor(t, h.racerIDs[0]), statsSnapshot{Races: 1, Wins: 1, Gold: 1})
	assertStats(t, "P2", racerStatsFor(t, h.racerIDs[1]), statsSnapshot{Races: 1, Silver: 1})
	assertStats(t, "P3", racerStatsFor(t, h.racerIDs[2]), statsSnapshot{Races: 1, Bronze: 1})

	_, rank1 := racerPointsRank(t, h.racerIDs[0])
	_, rank2 := racerPointsRank(t, h.racerIDs[1])
	_, rank3 := racerPointsRank(t, h.racerIDs[2])
	if !(rank1 < rank2 && rank2 < rank3) {
		t.Errorf("ranks should follow points order: P1=%d P2=%d P3=%d", rank1, rank2, rank3)
	}

	// Every racer's rank must match the points/name ordering formula.
	var mismatches int
	err := testServer.DB.QueryRow(`SELECT COUNT(*) FROM racers r WHERE r.rank != (SELECT COUNT(*) FROM racers r2 WHERE r2.points > r.points OR (r2.points = r.points AND r2.name < r.name)) + 1`).Scan(&mismatches)
	if err != nil {
		t.Fatalf("check ranks: %v", err)
	}
	if mismatches != 0 {
		t.Errorf("expected 0 rank mismatches after finalize, got %d", mismatches)
	}
}

// TestEditFinalizedRoundAppliesPodiumDeltas verifies that swapping podium
// positions on a finalized round adjusts points and podium counters without
// counting an extra race.
func TestEditFinalizedRoundAppliesPodiumDeltas(t *testing.T) {
	h := setupRoundHarness(t)

	h.patchPositions(t, 1, 2, 3)
	h.finalize(t)

	// Swap P1 and P2.
	h.patchPositions(t, 2, 1, 3)

	points0, _ := racerPointsRank(t, h.racerIDs[0])
	points1, _ := racerPointsRank(t, h.racerIDs[1])
	if points0 != 6 || points1 != 9 {
		t.Errorf("after swap points = %d/%d, want 6/9", points0, points1)
	}

	assertStats(t, "swapped P2 racer", racerStatsFor(t, h.racerIDs[0]), statsSnapshot{Races: 1, Silver: 1})
	assertStats(t, "swapped P1 racer", racerStatsFor(t, h.racerIDs[1]), statsSnapshot{Races: 1, Wins: 1, Gold: 1})
	assertStats(t, "P3 racer", racerStatsFor(t, h.racerIDs[2]), statsSnapshot{Races: 1, Bronze: 1})
}

// TestDeleteFinalizedRoundReversesAll verifies that deleting a finalized round
// removes points, race count, and all podium counters.
func TestDeleteFinalizedRoundReversesAll(t *testing.T) {
	h := setupRoundHarness(t)

	h.patchPositions(t, 1, 2, 3)
	h.finalize(t)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/rounds?id=%d", h.roundID), nil)
	rr := httptest.NewRecorder()
	h.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("delete round: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	for i, id := range h.racerIDs {
		points, _ := racerPointsRank(t, id)
		if points != 0 {
			t.Errorf("racer %d points after delete = %d, want 0", id, points)
		}
		assertStats(t, fmt.Sprintf("racer %d after delete", i), racerStatsFor(t, id), statsSnapshot{})
	}
}

// TestRoundMutationsInvalidateStatsCache verifies that round mutations bump the
// stats cache so stale aggregates are not served.
func TestRoundMutationsInvalidateStatsCache(t *testing.T) {
	prevCache := testServer.StatsCache
	testServer.StatsCache = racing.NewCache(time.Minute)
	defer func() { testServer.StatsCache = prevCache }()

	h := setupRoundHarness(t)

	testServer.StatsCache.Set("stats:sentinel", "cached")
	if _, ok := testServer.StatsCache.Get("stats:sentinel"); !ok {
		t.Fatal("sentinel should be cached before finalize")
	}

	h.patchPositions(t, 1, 2, 3)
	h.finalize(t)

	if _, ok := testServer.StatsCache.Get("stats:sentinel"); ok {
		t.Error("expected stats cache to be invalidated after finalize")
	}
}
