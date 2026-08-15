package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"heat/app"
	"heat/db"
	"heat/ent"
	"heat/handlers"
	"heat/pkg/logger"
)

func trmnlTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/trmnl/summary", testHandler.GetTRMNLSummary)
	return r
}

func trmnlTestSeason(t *testing.T, seasonID int, name, status string) {
	t.Helper()
	if _, err := testServer.DB.Exec("INSERT OR REPLACE INTO seasons (id, name, start_date, status) VALUES (?, ?, '2025-01-01', ?)", seasonID, name, status); err != nil {
		t.Fatalf("insert season: %v", err)
	}
}

func trmnlTestRound(t *testing.T, id, seasonID, round int, name, date, status string) {
	t.Helper()
	if _, err := testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshots (id, season_id, race_name, race_date, round, status) VALUES (?, ?, ?, ?, ?, ?)", id, seasonID, name, date, round, status); err != nil {
		t.Fatalf("insert round: %v", err)
	}
}

func trmnlTestScore(t *testing.T, id, snapshotID, racerID int, racerName string, points, position int) {
	t.Helper()
	if _, err := testServer.DB.Exec("INSERT OR REPLACE INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (?, ?, ?, ?, ?, ?, 0, 0, 0, 0)", id, snapshotID, racerID, racerName, points, position); err != nil {
		t.Fatalf("insert score: %v", err)
	}
}

func trmnlGet(t *testing.T, r *gin.Engine) (int, map[string]json.RawMessage) {
	t.Helper()
	req, _ := http.NewRequest("GET", "/api/trmnl/summary", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}
	if ct := rr.Header().Get("Content-Type"); len(ct) == 0 || ct[:16] != "application/json" {
		t.Errorf("expected application/json Content-Type, got %q", ct)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	return rr.Code, body
}

func TestTRMNLSummaryPopulated(t *testing.T) {
	seasonID := 8888
	trmnlTestSeason(t, seasonID, "TRMNL Season", "active")

	// Round 2 (final, latest) and Round 1 (final, older); a draft round with a
	// later date must NOT be selected as the latest race.
	trmnlTestRound(t, 88801, seasonID, 2, "Round 2", "2025-03-10", "final")
	trmnlTestRound(t, 88802, seasonID, 1, "Round 1", "2025-02-01", "final")
	trmnlTestRound(t, 88803, seasonID, 3, "Round 3 (draft)", "2025-04-01", "draft")

	// Round 1: racer 1 P1 (25), racer 2 P2 (18), racer 3 P3 (15)
	trmnlTestScore(t, 888011, 88802, 1, "A. PROST", 25, 1)
	trmnlTestScore(t, 888012, 88802, 2, "M. SCHUMACHER", 18, 2)
	trmnlTestScore(t, 888013, 88802, 3, "A. SENNA", 15, 3)

	// Round 2: racer 1 P2 (18), racer 2 P1 (25), racer 3 P4 (10)
	trmnlTestScore(t, 888021, 88801, 1, "A. PROST", 18, 2)
	trmnlTestScore(t, 888022, 88801, 2, "M. SCHUMACHER", 25, 1)
	trmnlTestScore(t, 888023, 88801, 3, "A. SENNA", 10, 4)

	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id IN (888011, 888012, 888013, 888021, 888022, 888023)")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id IN (88801, 88802, 88803)")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := trmnlTestRouter()
	_, body := trmnlGet(t, r)

	// Latest race: Round 2 (2025-03-10, round 2), draft round excluded.
	var race struct {
		Name     string `json:"name"`
		RaceDate string `json:"race_date"`
		Round    int    `json:"round"`
		Results  []struct {
			RacerName string `json:"racer_name"`
			Position  int    `json:"position"`
			Points    int    `json:"points"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body["latest_race"], &race); err != nil {
		t.Fatalf("unmarshal latest_race: %v", err)
	}
	if race.Name != "Round 2" || race.Round != 2 || race.RaceDate != "2025-03-10" {
		t.Errorf("latest race: expected Round 2/2025-03-10/round 2, got %s/%s/%d", race.Name, race.RaceDate, race.Round)
	}
	if len(race.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(race.Results))
	}
	// Ordered by position ascending.
	expectResults := []struct {
		name     string
		position int
		points   int
	}{
		{"M. SCHUMACHER", 1, 25},
		{"A. PROST", 2, 18},
		{"A. SENNA", 4, 10},
	}
	for i, exp := range expectResults {
		if race.Results[i].RacerName != exp.name || race.Results[i].Position != exp.position || race.Results[i].Points != exp.points {
			t.Errorf("result[%d]: expected %s P%d %dpts, got %s P%d %dpts",
				i, exp.name, exp.position, exp.points, race.Results[i].RacerName, race.Results[i].Position, race.Results[i].Points)
		}
	}

	// Standings: only finalized rounds count; ordered by points desc.
	// racer 1: 43 pts (25+18, 1 win), racer 2: 43 pts (18+25, 1 win),
	// racer 3: 25 pts (15+10, 0 wins).
	var standings []struct {
		RacerName string `json:"racer_name"`
		Races     int    `json:"races"`
		Wins      int    `json:"wins"`
		Points    int    `json:"points"`
	}
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 3 {
		t.Fatalf("expected 3 standings entries, got %d", len(standings))
	}
	pointsByName := map[string]int{}
	for _, s := range standings {
		pointsByName[s.RacerName] = s.Points
	}
	if pointsByName["A. PROST"] != 43 || pointsByName["M. SCHUMACHER"] != 43 || pointsByName["A. SENNA"] != 25 {
		t.Errorf("unexpected standings points: %v", pointsByName)
	}
	if standings[2].RacerName != "A. SENNA" {
		t.Errorf("expected A. SENNA last (25pts), got %s (%dpts)", standings[2].RacerName, standings[2].Points)
	}

	var season struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body["season"], &season); err != nil {
		t.Fatalf("unmarshal season: %v", err)
	}
	if season.ID != seasonID || season.Name != "TRMNL Season" {
		t.Errorf("expected season %d/TRMNL Season, got %d/%s", seasonID, season.ID, season.Name)
	}
}

func TestTRMNLSummaryDraftOnly(t *testing.T) {
	seasonID := 8889
	trmnlTestSeason(t, seasonID, "Draft Only Season", "active")
	trmnlTestRound(t, 88804, seasonID, 1, "Round 1 (draft)", "2025-01-05", "draft")
	trmnlTestScore(t, 888041, 88804, 1, "A. PROST", 25, 1)

	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id = 888041")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id = 88804")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := trmnlTestRouter()
	_, body := trmnlGet(t, r)

	if string(body["latest_race"]) != "null" {
		t.Errorf("expected latest_race null with only draft rounds, got %s", body["latest_race"])
	}
	var standings []json.RawMessage
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 0 {
		t.Errorf("expected empty standings with only draft rounds, got %d entries", len(standings))
	}
	var season struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(body["season"], &season); err != nil {
		t.Fatalf("unmarshal season: %v", err)
	}
	if season.ID != seasonID {
		t.Errorf("expected season %d still returned, got %d", seasonID, season.ID)
	}
}

func TestTRMNLSummaryNoActiveSeasonFallback(t *testing.T) {
	// Isolated server: the seeded "Season 1" is active, so archive everything
	// to force the most-recent-season fallback path.
	r, dbc := newTRMNLIsolated(t)
	if _, err := dbc.Exec("UPDATE seasons SET status = 'archived'"); err != nil {
		t.Fatalf("archive seeded seasons: %v", err)
	}
	// Two archived seasons; standings must come from the most recent (higher id).
	if _, err := dbc.Exec("INSERT INTO seasons (id, name, start_date, status) VALUES (8891, 'Old Archived Season', '2024-01-01', 'archived'), (8892, 'New Archived Season', '2025-01-01', 'archived')"); err != nil {
		t.Fatalf("insert seasons: %v", err)
	}
	if _, err := dbc.Exec("INSERT INTO round_snapshots (id, season_id, race_name, race_date, round, status) VALUES (88911, 8891, 'Old Round', '2025-01-10', 1, 'final'), (88921, 8892, 'New Round', '2025-02-10', 1, 'final')"); err != nil {
		t.Fatalf("insert rounds: %v", err)
	}
	if _, err := dbc.Exec("INSERT INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES (889111, 88911, 10, 'OLD RACER', 25, 1, 0, 0, 0, 0), (889211, 88921, 11, 'NEW RACER', 18, 1, 0, 0, 0, 0)"); err != nil {
		t.Fatalf("insert scores: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/trmnl/summary", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	var season struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body["season"], &season); err != nil {
		t.Fatalf("unmarshal season: %v", err)
	}
	if season.ID != 8892 {
		t.Errorf("expected fallback to most recent season 8892, got %d", season.ID)
	}

	var standings []struct {
		RacerName string `json:"racer_name"`
	}
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 1 || standings[0].RacerName != "NEW RACER" {
		t.Errorf("expected standings from season 8892 (NEW RACER), got %v", standings)
	}
}

func TestTRMNLSummaryLimits(t *testing.T) {
	seasonID := 8893
	trmnlTestSeason(t, seasonID, "Limit Season", "active")
	trmnlTestRound(t, 88930, seasonID, 1, "Big Round", "2025-01-15", "final")

	// 11 racers in the round: results capped at 10, standings capped at 8.
	for i := 1; i <= 11; i++ {
		trmnlTestScore(t, 889300+i, 88930, 100+i, fmt.Sprintf("RACER %d", i), 25-(i-1)*2, i)
	}

	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id >= 889301 AND id <= 889311")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id = 88930")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := trmnlTestRouter()
	_, body := trmnlGet(t, r)

	var race struct {
		Results []struct {
			ProfilePicture string `json:"profile_picture"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body["latest_race"], &race); err != nil {
		t.Fatalf("unmarshal latest_race: %v", err)
	}
	if len(race.Results) != 10 {
		t.Errorf("expected results capped at 10, got %d", len(race.Results))
	}
	// These racer ids have no row in the racers table, so no profile picture
	// may be resolved and the field must be omitted/empty.
	for i, r := range race.Results {
		if r.ProfilePicture != "" {
			t.Errorf("result[%d]: expected empty profile_picture for unknown racer, got %q", i, r.ProfilePicture)
		}
	}

	var standings []json.RawMessage
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 8 {
		t.Errorf("expected standings capped at 8, got %d", len(standings))
	}
}

// TestTRMNLSummaryProfilePictures verifies the driver profile picture flows
// into the TRMNL payload as an absolute URL (the TRMNL servers fetch images
// server-side, so relative paths would break). Uses an isolated server so the
// seeded racers are in a known state.
func TestTRMNLSummaryProfilePictures(t *testing.T) {
	r, dbc := newTRMNLIsolated(t)

	// The isolated DB seeds racers 1-3 with /static/images/helmet.svg.
	if _, err := dbc.Exec("INSERT INTO seasons (id, name, start_date, status) VALUES (8899, 'Pic Season', '2025-01-01', 'active')"); err != nil {
		t.Fatalf("insert season: %v", err)
	}
	if _, err := dbc.Exec("INSERT INTO round_snapshots (id, season_id, race_name, race_date, round, status) VALUES (88991, 8899, 'Pic Round', '2025-05-10', 1, 'final')"); err != nil {
		t.Fatalf("insert round: %v", err)
	}
	// racer 999 has no row in the racers table: its picture must be empty.
	if _, err := dbc.Exec(`INSERT INTO round_snapshot_scores (id, snapshot_id, racer_id, racer_name, points, position, dnf, dns, spins, overheated) VALUES
		(889911, 88991, 1, 'A. PROST', 25, 1, 0, 0, 0, 0),
		(889912, 88991, 2, 'M. SCHUMACHER', 18, 2, 0, 0, 0, 0),
		(889913, 88991, 999, 'GHOST', 15, 3, 0, 0, 0, 0)`); err != nil {
		t.Fatalf("insert scores: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/trmnl/summary", nil)
	req.Host = "example.com"
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	var race struct {
		Results []struct {
			RacerName      string `json:"racer_name"`
			ProfilePicture string `json:"profile_picture"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body["latest_race"], &race); err != nil {
		t.Fatalf("unmarshal latest_race: %v", err)
	}
	if len(race.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(race.Results))
	}
	const wantPic = "http://example.com/static/images/helmet.svg"
	if race.Results[0].ProfilePicture != wantPic {
		t.Errorf("result[0] (A. PROST): expected profile_picture %q, got %q", wantPic, race.Results[0].ProfilePicture)
	}
	if race.Results[1].ProfilePicture != wantPic {
		t.Errorf("result[1] (M. SCHUMACHER): expected profile_picture %q, got %q", wantPic, race.Results[1].ProfilePicture)
	}
	if race.Results[2].RacerName != "GHOST" || race.Results[2].ProfilePicture != "" {
		t.Errorf("result[2]: expected GHOST with empty profile_picture, got %s / %q", race.Results[2].RacerName, race.Results[2].ProfilePicture)
	}

	var standings []struct {
		RacerName      string `json:"racer_name"`
		ProfilePicture string `json:"profile_picture"`
	}
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 3 {
		t.Fatalf("expected 3 standings entries, got %d", len(standings))
	}
	for _, s := range standings {
		switch s.RacerName {
		case "GHOST":
			if s.ProfilePicture != "" {
				t.Errorf("standing GHOST: expected empty profile_picture (no racer row), got %q", s.ProfilePicture)
			}
		default:
			if s.ProfilePicture != wantPic {
				t.Errorf("standing %s: expected profile_picture %q, got %q", s.RacerName, wantPic, s.ProfilePicture)
			}
		}
	}
}

// newTRMNLIsolated builds a fresh in-memory server so a test does not depend
// on the shared test DB's seeded seasons/rounds.
func newTRMNLIsolated(t *testing.T) (*gin.Engine, *sql.DB) {
	t.Helper()
	srv := app.NewServer()
	srv.BasePath = "."
	srv.DBPath = ":memory:"
	srv.MediaPath = filepath.Join(srv.BasePath, "media")
	conn, err := sql.Open("sqlite3", srv.DBPath+"?_fk=1")
	if err != nil {
		t.Fatalf("failed to open isolated db: %v", err)
	}
	conn.SetMaxOpenConns(1)
	srv.DB = conn
	drv := entsql.OpenDB(dialect.SQLite, conn)
	srv.Ent = ent.NewClient(ent.Driver(drv))
	srv.BroadcastRacers = func() {}
	db.Init(srv)
	srv.Log = logger.New(conn)

	r := gin.New()
	r.GET("/api/trmnl/summary", handlers.New(srv).GetTRMNLSummary)
	return r, conn
}

func TestTRMNLSummaryEmpty(t *testing.T) {
	r, _ := newTRMNLIsolated(t)

	// Fresh DB: no finalized rounds (only the seeded active season exists).
	req, _ := http.NewRequest("GET", "/api/trmnl/summary", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if string(body["latest_race"]) != "null" {
		t.Errorf("expected latest_race null on empty data, got %s", body["latest_race"])
	}
	var standings []json.RawMessage
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) != 0 {
		t.Errorf("expected empty standings on empty data, got %d entries", len(standings))
	}
	var season struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body["season"], &season); err != nil {
		t.Fatalf("unmarshal season: %v", err)
	}
	if season.ID != 1 || season.Name != "Season 1" {
		t.Errorf("expected seeded season 1, got %d/%s", season.ID, season.Name)
	}
}
