package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"heat/app"
	"heat/db"
	"heat/ent"
	"heat/handlers"
	"heat/pkg/logger"
)

func trmnlNextRaceTestRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/trmnl/next-race", testHandler.GetTRMNLNextRace)
	return r
}

func TestTRMNLNextRacePopulated(t *testing.T) {
	seasonID := 8893
	trmnlTestSeason(t, seasonID, "Next Race Season", "active")

	// A finalized round with four racers; the payload must cap results at 3.
	trmnlTestRound(t, 88931, seasonID, 1, "Round 1", "2025-01-05", "final")
	trmnlTestScore(t, 889311, 88931, 1, "A. PROST", 25, 1)
	trmnlTestScore(t, 889312, 88931, 2, "M. SCHUMACHER", 18, 2)
	trmnlTestScore(t, 889313, 88931, 3, "A. SENNA", 15, 3)
	trmnlTestScore(t, 889314, 88931, 4, "N. LAUDA", 12, 4)

	// Configure an upcoming race ~10 days out.
	raceDate := time.Now().AddDate(0, 0, 10).Format("2006-01-02")
	if _, err := testServer.DB.Exec("UPDATE race_info SET country = 'UK', track = 'Silverstone', track_id = 'silverstone', laps = 52, next_race_date = ? WHERE id = (SELECT MAX(id) FROM race_info)", raceDate); err != nil {
		t.Fatalf("update race_info: %v", err)
	}

	defer func() {
		testServer.DB.Exec("UPDATE race_info SET next_race_date = '' WHERE next_race_date = ?", raceDate)
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id IN (889311, 889312, 889313, 889314)")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id = 88931")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := trmnlNextRaceTestRouter()
	req, _ := http.NewRequest("GET", "/api/trmnl/next-race", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}

	var nextRace struct {
		RaceDate      string `json:"race_date"`
		Track         string `json:"track"`
		Country       string `json:"country"`
		TrackID       string `json:"track_id"`
		TotalLaps     int    `json:"total_laps"`
		DaysRemaining int    `json:"days_remaining"`
	}
	if err := json.Unmarshal(body["next_race"], &nextRace); err != nil {
		t.Fatalf("unmarshal next_race: %v", err)
	}
	if nextRace.RaceDate != raceDate || nextRace.Track != "Silverstone" || nextRace.Country != "UK" ||
		nextRace.TrackID != "silverstone" || nextRace.TotalLaps != 52 {
		t.Errorf("next_race: unexpected payload %+v", nextRace)
	}
	if nextRace.DaysRemaining != 10 && nextRace.DaysRemaining != 11 {
		t.Errorf("expected days_remaining ~10, got %d", nextRace.DaysRemaining)
	}

	var latestRace struct {
		Results []struct {
			RacerName string `json:"racer_name"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body["latest_race"], &latestRace); err != nil {
		t.Fatalf("unmarshal latest_race: %v", err)
	}
	if len(latestRace.Results) != 3 {
		t.Errorf("expected 3 latest race results, got %d", len(latestRace.Results))
	}

	var standings []json.RawMessage
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) == 0 {
		t.Errorf("expected non-empty standings, got 0 entries")
	}

	var season struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(body["season"], &season); err != nil {
		t.Fatalf("unmarshal season: %v", err)
	}
	if season.ID != seasonID || season.Name != "Next Race Season" {
		t.Errorf("expected season %d/Next Race Season, got %d/%s", seasonID, season.ID, season.Name)
	}
}

func TestTRMNLNextRaceNotScheduled(t *testing.T) {
	seasonID := 8894
	trmnlTestSeason(t, seasonID, "No Date Season", "active")
	trmnlTestRound(t, 88941, seasonID, 1, "Round 1", "2025-01-05", "final")
	trmnlTestScore(t, 889411, 88941, 1, "A. PROST", 25, 1)

	// No next_race_date configured on the seed race_info row.
	if _, err := testServer.DB.Exec("UPDATE race_info SET next_race_date = '' WHERE id = (SELECT MAX(id) FROM race_info)"); err != nil {
		t.Fatalf("clear next_race_date: %v", err)
	}

	defer func() {
		testServer.DB.Exec("DELETE FROM round_snapshot_scores WHERE id = 889411")
		testServer.DB.Exec("DELETE FROM round_snapshots WHERE id = 88941")
		testServer.DB.Exec("DELETE FROM seasons WHERE id = ?", seasonID)
	}()

	r := trmnlNextRaceTestRouter()
	req, _ := http.NewRequest("GET", "/api/trmnl/next-race", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if string(body["next_race"]) != "null" {
		t.Errorf("expected next_race null when not scheduled, got %s", body["next_race"])
	}
	if string(body["latest_race"]) == "null" {
		t.Errorf("expected latest_race present even without a next race")
	}
	var standings []json.RawMessage
	if err := json.Unmarshal(body["standings"], &standings); err != nil {
		t.Fatalf("unmarshal standings: %v", err)
	}
	if len(standings) == 0 {
		t.Errorf("expected non-empty standings even without a next race")
	}
}

func TestTRMNLNextRaceEmpty(t *testing.T) {
	r, _ := newTRMNLIsolatedNextRace(t)

	req, _ := http.NewRequest("GET", "/api/trmnl/next-race", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %v: %s", rr.Code, rr.Body.String())
	}

	var body map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal body: %v", err)
	}
	if string(body["next_race"]) != "null" {
		t.Errorf("expected next_race null on empty data, got %s", body["next_race"])
	}
	if string(body["latest_race"]) != "null" {
		t.Errorf("expected latest_race null on empty data, got %s", body["latest_race"])
	}
}

// newTRMNLIsolatedNextRace builds a fresh in-memory server with only the
// next-race endpoint registered.
func newTRMNLIsolatedNextRace(t *testing.T) (*gin.Engine, *sql.DB) {
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
	r.GET("/api/trmnl/next-race", handlers.New(srv).GetTRMNLNextRace)
	return r, conn
}
