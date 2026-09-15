package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/middleware"
	"heat/models"
)

// resetRaceState returns the singleton race_state row to its initial stopped
// form so tests do not leak state into each other.
func resetRaceState(t *testing.T) {
	t.Helper()
	_, err := testServer.DB.Exec("UPDATE race_state SET state='stopped', started_at='', accumulated_ms=0, current_lap=0, total_laps=0 WHERE id=1")
	if err != nil {
		t.Fatalf("reset race_state: %v", err)
	}
}

func TestRaceStateTransitions(t *testing.T) {
	resetRaceState(t)

	st, err := testServer.GetRaceState()
	if err != nil {
		t.Fatalf("initial GetRaceState: %v", err)
	}
	if st.State != app.RaceStopped {
		t.Fatalf("initial state = %q, want stopped", st.State)
	}

	st, err = testServer.ApplyRaceAction("start", 20)
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if st.State != app.RaceRacing || st.CurrentLap != 1 || st.TotalLaps != 20 {
		t.Fatalf("after start = %+v, want racing lap 1 / 20 laps", st)
	}

	if _, err := testServer.ApplyRaceAction("start", 0); !errors.Is(err, app.ErrInvalidRaceState) {
		t.Fatalf("double start err = %v, want ErrInvalidRaceState", err)
	}

	// Pause folds elapsed into accumulated_ms and preserves lap/total.
	st, err = testServer.ApplyRaceAction("pause", 0)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	if st.State != app.RacePaused || st.CurrentLap != 1 || st.TotalLaps != 20 {
		t.Fatalf("after pause = %+v, want paused lap 1 / 20 laps", st)
	}
	if _, err := testServer.ApplyRaceAction("pause", 0); !errors.Is(err, app.ErrInvalidRaceState) {
		t.Fatalf("double pause err = %v, want ErrInvalidRaceState", err)
	}

	st, err = testServer.ApplyRaceAction("resume", 0)
	if err != nil || st.State != app.RaceRacing {
		t.Fatalf("resume = %+v err %v, want racing", st, err)
	}

	st, err = testServer.ApplyRaceAction("stop", 0)
	if err != nil {
		t.Fatalf("stop: %v", err)
	}
	if st.State != app.RaceStopped || st.CurrentLap != 0 || st.ElapsedMs != 0 {
		t.Fatalf("after stop = %+v, want stopped/0/0", st)
	}

	if _, err := testServer.ApplyRaceAction("bogus", 0); !errors.Is(err, app.ErrInvalidRaceState) {
		t.Fatalf("bogus action err = %v, want ErrInvalidRaceState", err)
	}
}

func TestRaceStateHTTP(t *testing.T) {
	resetRaceState(t)

	r := gin.New()
	r.GET("/api/race/state", testHandler.GetRaceState)
	r.POST("/api/race/state", middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer), testHandler.PostRaceState)
	srv := httptest.NewServer(r)
	defer srv.Close()

	client := srv.Client()

	// Unauthenticated transition is rejected (CSRF origin present, no session).
	req, _ := http.NewRequest("POST", srv.URL+"/api/race/state", bytes.NewBufferString(`{"action":"start","total_laps":10}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.Host = "127.0.0.1:6270"
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unauth POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth POST status = %d, want 401", resp.StatusCode)
	}

	session := createAdminSession(t)
	defer removeAdminSession(session)

	post := func(body string) *http.Response {
		t.Helper()
		req, _ := http.NewRequest("POST", srv.URL+"/api/race/state", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: session})
		req.Host = "127.0.0.1:6270"
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("POST %s: %v", body, err)
		}
		return resp
	}

	resp = post(`{"action":"start","total_laps":10}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start status = %d, want 200", resp.StatusCode)
	}
	var st models.RaceState
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatalf("decode start: %v", err)
	}
	resp.Body.Close()
	if st.State != app.RaceRacing || st.TotalLaps != 10 {
		t.Fatalf("start body = %+v, want racing/10", st)
	}

	// Illegal transition is a 409, not a 500.
	resp = post(`{"action":"start","total_laps":10}`)
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("double start status = %d, want 409", resp.StatusCode)
	}

	// GET persists the transitioned state.
	gresp, err := client.Get(srv.URL + "/api/race/state")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer gresp.Body.Close()
	var got models.RaceState
	if err := json.NewDecoder(gresp.Body).Decode(&got); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	if got.State != app.RaceRacing || got.CurrentLap != 1 || got.TotalLaps != 10 {
		t.Fatalf("GET = %+v, want racing/1/10", got)
	}
}

func TestComputeStandings(t *testing.T) {
	raceID := int(time.Now().UnixNano() % 1_000_000)
	if raceID < 1 {
		raceID = 1
	}

	insertRacer := func(name string, position int) int {
		t.Helper()
		res, err := testServer.DB.Exec("INSERT INTO racers (name, position) VALUES (?, ?)", name, position)
		if err != nil {
			t.Fatalf("insert racer %s: %v", name, err)
		}
		id, _ := res.LastInsertId()
		return int(id)
	}
	leader := insertRacer("Standings Leader", 1)
	second := insertRacer("Standings Second", 2)
	third := insertRacer("Standings Third", 3)

	record := func(racerID, lap, position int) {
		t.Helper()
		if _, err := testServer.DB.Exec(
			"INSERT INTO lap_records (race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used) VALUES (?, ?, ?, ?, 0, 0, 0)",
			raceID, racerID, lap, position); err != nil {
			t.Fatalf("insert lap record: %v", err)
		}
	}
	record(leader, 5, 1)
	record(second, 4, 2)
	record(third, 5, 3)

	t.Cleanup(func() {
		testServer.DB.Exec("DELETE FROM lap_records WHERE race_id = ?", raceID)
		testServer.DB.Exec("DELETE FROM racers WHERE id IN (?, ?, ?)", leader, second, third)
	})

	standings, err := testServer.ComputeStandings(raceID)
	if err != nil {
		t.Fatalf("ComputeStandings: %v", err)
	}

	byID := map[int]models.Standing{}
	positions := map[int]int{}
	for i, s := range standings {
		byID[s.RacerID] = s
		positions[s.RacerID] = i
	}

	if byID[leader].Gap != "LEAD" {
		t.Errorf("leader gap = %q, want LEAD", byID[leader].Gap)
	}
	if byID[second].Gap != "+1" {
		t.Errorf("second gap = %q, want +1", byID[second].Gap)
	}
	if byID[third].Gap != "" {
		t.Errorf("third gap = %q, want empty", byID[third].Gap)
	}
	if byID[leader].Lap != 5 || byID[second].Lap != 4 {
		t.Errorf("laps = leader %d / second %d, want 5 / 4", byID[leader].Lap, byID[second].Lap)
	}
	if !(positions[leader] < positions[second] && positions[second] < positions[third]) {
		t.Errorf("standings not ordered by position: %v", positions)
	}

	// No lap data for a fresh race: all gaps empty, no leader.
	empty, err := testServer.ComputeStandings(raceID + 999999)
	if err != nil {
		t.Fatalf("ComputeStandings empty: %v", err)
	}
	for _, s := range empty {
		if s.Gap != "" || s.Lap != 0 {
			t.Errorf("empty-race standing = %+v, want no gap / lap 0", s)
		}
	}
}
