package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/middleware"
	"heat/models"
)

func TestGetTracks(t *testing.T) {
	r := gin.New()
	r.GET("/api/tracks", testHandler.GetTracks)

	req, _ := http.NewRequest("GET", "/api/tracks", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var tracks []models.Track
	json.Unmarshal(rr.Body.Bytes(), &tracks)
	if len(tracks) < 1 {
		t.Errorf("expected at least 1 track, got %d", len(tracks))
	}
}

func TestTrackCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/tracks", testHandler.SaveTrack)
	admin.DELETE("/tracks", testHandler.DeleteTrack)
	r.GET("/api/tracks", testHandler.GetTracks)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save new track", func(t *testing.T) {
		track := models.Track{ID: "test-track", Name: "Test Circuit", Country: "Testland", Length: 5, LapRecord: "1:30.000"}
		body, _ := json.Marshal(track)
		req := newAdminRequest("POST", "/api/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var name string
		testServer.DB.QueryRow("SELECT name FROM tracks WHERE id = 'test-track'").Scan(&name)
		if name != "Test Circuit" {
			t.Errorf("expected track name 'Test Circuit', got %q", name)
		}
	})

	t.Run("save track without auth", func(t *testing.T) {
		track := models.Track{ID: "noauth", Name: "No Auth Track", Country: "Nowhere"}
		body, _ := json.Marshal(track)
		req, _ := http.NewRequest("POST", "/api/tracks", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("delete track", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/tracks?id=test-track", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
	})

	t.Run("delete track without id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/tracks", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})
}

func TestQuotes(t *testing.T) {
	r := gin.New()
	r.GET("/api/quotes", testHandler.GetQuotes)
	r.GET("/api/quote/random", testHandler.GetRandomQuote)

	t.Run("GetQuotes", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quotes", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var quotes []models.Quote
		json.Unmarshal(rr.Body.Bytes(), &quotes)
		if len(quotes) < 1 {
			t.Errorf("expected quotes, got %d", len(quotes))
		}
	})

	t.Run("GetRandomQuote", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/quote/random", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var q models.Quote
		json.Unmarshal(rr.Body.Bytes(), &q)
		if q.Text == "" {
			t.Errorf("expected quote text to be non-empty")
		}
	})
}

func TestQuoteCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/quotes", testHandler.HandleQuotes)
	admin.PUT("/quotes", testHandler.HandleQuotes)
	admin.DELETE("/quotes", testHandler.HandleQuotes)
	r.GET("/api/quotes", testHandler.GetQuotes)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("create quote", func(t *testing.T) {
		q := models.Quote{Text: "Test quote text", Author: "Tester"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
		}
		var created models.Quote
		json.Unmarshal(rr.Body.Bytes(), &created)
		if created.ID == 0 || created.Text != "Test quote text" {
			t.Errorf("unexpected response: %+v", created)
		}
	})

	t.Run("create quote without auth", func(t *testing.T) {
		q := models.Quote{Text: "Unauth quote"}
		body, _ := json.Marshal(q)
		req, _ := http.NewRequest("POST", "/api/quotes", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("create quote empty text", func(t *testing.T) {
		q := models.Quote{Text: "", Author: "Empty"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty text, got %d", rr.Code)
		}
	})

	t.Run("update quote", func(t *testing.T) {
		q := models.Quote{ID: 1, Text: "Updated quote text", Author: "Updated Author"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("PUT", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var text string
		testServer.DB.QueryRow("SELECT text FROM quotes WHERE id = 1").Scan(&text)
		if text != "Updated quote text" {
			t.Errorf("expected text 'Updated quote text', got %q", text)
		}
	})

	t.Run("update quote without id", func(t *testing.T) {
		q := models.Quote{Text: "No ID"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("PUT", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})

	t.Run("delete quote", func(t *testing.T) {
		// First create one to delete
		q := models.Quote{Text: "To be deleted", Author: "Temp"}
		body, _ := json.Marshal(q)
		req := newAdminRequest("POST", "/api/quotes", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var created models.Quote
		json.Unmarshal(rr.Body.Bytes(), &created)

		req = newAdminRequest("DELETE", "/api/quotes?id="+strconv.Itoa(created.ID), nil, sessionID)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM quotes WHERE id = ?", created.ID).Scan(&count)
		if count != 0 {
			t.Error("quote should be deleted")
		}
	})

	t.Run("delete quote without id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/quotes", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without id, got %d", rr.Code)
		}
	})
}

func TestTeamsAPI(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/teams", testHandler.SaveTeam)
	admin.DELETE("/teams", testHandler.DeleteTeam)
	admin.POST("/teams/assign", testHandler.AssignTeam)
	r.GET("/api/teams", testHandler.GetTeams)
	r.GET("/api/teams/standings", testHandler.GetConstructorStandings)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("list teams", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/teams", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var teams []models.Team
		json.Unmarshal(rr.Body.Bytes(), &teams)
		if len(teams) < 5 {
			t.Errorf("expected at least 5 teams, got %d", len(teams))
		}
	})

	t.Run("create team", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "Test Team", Color: "#ff00ff"})
		req := newAdminRequest("POST", "/api/teams", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var name string
		testServer.DB.QueryRow("SELECT name FROM teams WHERE name = 'Test Team'").Scan(&name)
		if name != "Test Team" {
			t.Errorf("expected 'Test Team', got %q", name)
		}
	})

	t.Run("create team empty name", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "", Color: "#ff00ff"})
		req := newAdminRequest("POST", "/api/teams", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty name, got %d", rr.Code)
		}
	})

	t.Run("assign racer to team", func(t *testing.T) {
		body, _ := json.Marshal(map[string]int{"racer_id": 1, "team_id": 1})
		req := newAdminRequest("POST", "/api/teams/assign", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var teamID int
		testServer.DB.QueryRow("SELECT team_id FROM racers WHERE id = 1").Scan(&teamID)
		if teamID != 1 {
			t.Errorf("expected team_id=1, got %d", teamID)
		}
	})

	t.Run("constructor standings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/teams/standings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var standings []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &standings)
		if len(standings) < 1 {
			t.Error("expected at least 1 team in standings")
		}
	})

	t.Run("delete team", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/teams?id=1", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM teams WHERE id = 1").Scan(&count)
		if count != 0 {
			t.Error("team should be deleted")
		}
		var teamID int
		testServer.DB.QueryRow("SELECT COALESCE(team_id, 0) FROM racers WHERE id = 1").Scan(&teamID)
		if teamID != 0 {
			t.Errorf("expected racer team_id reset to 0, got %d", teamID)
		}
		// Re-seed for other tests
		testServer.DB.Exec("INSERT OR IGNORE INTO teams (id, name, color) VALUES (1, 'Scuderia Ferrari', '#d40000')")
	})

	t.Run("delete team without auth", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/teams?id=2", nil)
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("create team without auth", func(t *testing.T) {
		body, _ := json.Marshal(models.Team{Name: "Unauth Team"})
		req, _ := http.NewRequest("POST", "/api/teams", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("delete team invalid id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/teams?id=0", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}
