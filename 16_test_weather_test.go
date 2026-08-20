package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWeatherBackend(t *testing.T) {
	r := gin.New()
	r.GET("/api/stats/pace-heatmap", testHandler.GetPaceHeatmap)
	r.POST("/api/weather", testHandler.SetWeather)
	r.GET("/api/weather", testHandler.GetWeather)

	testServer.DB.Exec("DELETE FROM lap_records")
	testServer.DB.Exec("DELETE FROM weather_conditions")
	testServer.DB.Exec("DELETE FROM commentary")

	t.Run("pace-heatmap annotates weather condition and grip", func(t *testing.T) {
		// Insert lap records for racer 1: laps 1-5.
		for lap := 1; lap <= 5; lap++ {
			testServer.DB.Exec("INSERT INTO lap_records (race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used) VALUES (0, 1, ?, ?, 3, 50, 0)", lap, lap)
		}
		// Wet from lap 3 to lap 4 (inclusive window).
		testServer.DB.Exec("INSERT INTO weather_conditions (race_id, condition, lap_start, lap_end, grip_modifier) VALUES (0, 'wet', 3, 4, 0.7)")

		req, _ := http.NewRequest("GET", "/api/stats/pace-heatmap?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var points []struct {
			Lap          int     `json:"lap"`
			Condition    string  `json:"condition"`
			GripModifier float64 `json:"grip_modifier"`
		}
		json.Unmarshal(rr.Body.Bytes(), &points)
		if len(points) != 5 {
			t.Fatalf("expected 5 points, got %d", len(points))
		}
		for _, p := range points {
			if p.Lap == 3 || p.Lap == 4 {
				if p.Condition != "wet" {
					t.Errorf("lap %d: expected wet, got %q", p.Lap, p.Condition)
				}
				if p.GripModifier != 0.7 {
					t.Errorf("lap %d: expected grip 0.7, got %v", p.Lap, p.GripModifier)
				}
			} else {
				if p.Condition != "dry" {
					t.Errorf("lap %d: expected dry, got %q", p.Lap, p.Condition)
				}
				if p.GripModifier != 1.0 {
					t.Errorf("lap %d: expected grip 1.0, got %v", p.Lap, p.GripModifier)
				}
			}
		}
	})

	t.Run("pace-heatmap defaults to dry with no weather", func(t *testing.T) {
		testServer.DB.Exec("DELETE FROM lap_records")
		testServer.DB.Exec("DELETE FROM weather_conditions")
		testServer.DB.Exec("INSERT INTO lap_records (race_id, racer_id, lap_number, position, gear_used, heat_generated, turbo_used) VALUES (0, 2, 1, 1, 3, 50, 0)")

		req, _ := http.NewRequest("GET", "/api/stats/pace-heatmap?racer_id=2", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var points []struct {
			Condition    string  `json:"condition"`
			GripModifier float64 `json:"grip_modifier"`
		}
		json.Unmarshal(rr.Body.Bytes(), &points)
		if len(points) != 1 {
			t.Fatalf("expected 1 point, got %d", len(points))
		}
		if points[0].Condition != "dry" || points[0].GripModifier != 1.0 {
			t.Errorf("expected dry/1.0 defaults, got %q/%v", points[0].Condition, points[0].GripModifier)
		}
	})

	t.Run("SetWeather persists lap_end", func(t *testing.T) {
		body := `{"race_id":0,"condition":"torrential","lap_start":10,"lap_end":20,"grip_modifier":0.5}`
		req, _ := http.NewRequest("POST", "/api/weather", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		req2, _ := http.NewRequest("GET", "/api/weather?race_id=0", nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr2.Code)
		}
		var entries []struct {
			Condition    string  `json:"condition"`
			LapStart     int     `json:"lap_start"`
			LapEnd       int     `json:"lap_end"`
			GripModifier float64 `json:"grip_modifier"`
		}
		json.Unmarshal(rr2.Body.Bytes(), &entries)
		found := false
		for _, e := range entries {
			if e.Condition == "torrential" && e.LapStart == 10 {
				found = true
				if e.LapEnd != 20 {
					t.Errorf("expected lap_end 20, got %d", e.LapEnd)
				}
				if e.GripModifier != 0.5 {
					t.Errorf("expected grip 0.5, got %v", e.GripModifier)
				}
			}
		}
		if !found {
			t.Error("expected torrential entry with lap_start 10")
		}
	})

	testServer.DB.Exec("DELETE FROM lap_records")
	testServer.DB.Exec("DELETE FROM weather_conditions")
	testServer.DB.Exec("DELETE FROM commentary")
}
