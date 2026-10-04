package main

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExportRacesICS(t *testing.T) {
	// Seed distinctive data
	_, err := testServer.DB.Exec("INSERT INTO race_info (country, track, track_id, laps, next_race_date) VALUES (?, ?, ?, ?, ?)", "TestCountryICAL", "TestTrackICALUpcoming", "test-ical-upcoming", 99, "2099-12-31")
	if err != nil {
		t.Fatalf("seed race_info: %v", err)
	}
	t.Cleanup(func() {
		testServer.DB.Exec("DELETE FROM race_info WHERE track = 'TestTrackICALUpcoming'")
	})
	res, err := testServer.DB.Exec("INSERT INTO race_history (name, race_date, country, track, track_id, total_laps, race_type) VALUES (?, ?, ?, ?, ?, ?, ?)", "TestRaceICALDone", "2099-01-15", "TestCountryICALDone", "TestTrackICALDone", "test-ical-done", 50, "season")
	if err != nil {
		t.Fatalf("seed race_history: %v", err)
	}
	id, _ := res.LastInsertId()
	t.Cleanup(func() {
		testServer.DB.Exec("DELETE FROM race_results WHERE race_id = ?", id)
		testServer.DB.Exec("DELETE FROM race_history WHERE id = ?", id)
	})

	r := gin.New()
	r.GET("/api/races/export.ics", testHandler.ExportRacesICS)

	req, _ := http.NewRequest("GET", "/api/races/export.ics", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/calendar") {
		t.Errorf("expected Content-Type text/calendar, got %q", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "BEGIN:VCALENDAR") {
		t.Error("missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(body, "END:VCALENDAR") {
		t.Error("missing END:VCALENDAR")
	}
	if !strings.Contains(body, "BEGIN:VEVENT") {
		t.Error("missing BEGIN:VEVENT")
	}
	if !strings.Contains(body, "END:VEVENT") {
		t.Error("missing END:VEVENT")
	}
	if !strings.Contains(body, "HEAT Race: TestTrackICALUpcoming") {
		t.Errorf("missing upcoming summary, body: %s", body[:min(2000, len(body))])
	}
	expectedUID := "UID:race-" + strconv.Itoa(int(id)) + "@heat-racer"
	if !strings.Contains(body, expectedUID) {
		t.Errorf("missing done race UID %q", expectedUID)
	}
	if !strings.Contains(body, "DTSTART;VALUE=DATE:") {
		t.Error("missing DTSTART;VALUE=DATE:")
	}
}
