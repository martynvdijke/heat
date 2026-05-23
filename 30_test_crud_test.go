package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/middleware"
	"heat/models"
)

func TestRacerCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/racers", testHandler.UpdateRacer)
	admin.DELETE("/racers", testHandler.DeleteRacer)
	r.GET("/api/racers", testHandler.GetRacers)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("create racer", func(t *testing.T) {
		racer := models.Racer{Name: "New Racer", CarColor: "red", CarName: "Red Bull", Points: 0, Rank: 99, Position: 0}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify it shows up in racers list
		req, _ = http.NewRequest("GET", "/api/racers", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var racers []models.Racer
		json.Unmarshal(rr.Body.Bytes(), &racers)
		found := false
		for _, r := range racers {
			if r.Name == "New Racer" {
				found = true
				break
			}
		}
		if !found {
			t.Error("new racer not found in list")
		}
	})

	t.Run("create racer without auth", func(t *testing.T) {
		racer := models.Racer{Name: "Unauth Racer", CarColor: "blue", CarName: "No Auth"}
		body, _ := json.Marshal(racer)
		req, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("update racer", func(t *testing.T) {
		racer := models.Racer{ID: 1, Name: "Updated Racer", ProfilePicture: "/static/helmet.svg", CarColor: "purple", CarName: "Updated Car", Points: 100, Rank: 1, Position: 0}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d", rr.Code)
		}
		var name string
		testServer.DB.QueryRow("SELECT name FROM racers WHERE id = 1").Scan(&name)
		if name != "Updated Racer" {
			t.Errorf("expected name 'Updated Racer', got %q", name)
		}
	})

	t.Run("delete racer", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/racers?id=2", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d", rr.Code)
		}
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM racers WHERE id = 2").Scan(&count)
		if count != 0 {
			t.Error("racer should be deleted")
		}
	})

	t.Run("delete racer invalid id", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/racers?id=0", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid id, got %d", rr.Code)
		}
	})

	t.Run("delete racer without auth", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/racers?id=1", nil)
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	t.Run("create racer empty name", func(t *testing.T) {
		racer := models.Racer{Name: "", CarColor: "red", CarName: "No Name"}
		body, _ := json.Marshal(racer)
		req := newAdminRequest("POST", "/api/racers", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		// Should still succeed (no name validation on server)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestRacerEmailCRUD(t *testing.T) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/racer-emails", testHandler.GetRacerEmails)
	admin.POST("/racer-emails", testHandler.SaveRacerEmail)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save racer email", func(t *testing.T) {
		re := models.RacerEmail{RacerID: 1, Email: "test@example.com"}
		body, _ := json.Marshal(re)
		req := newAdminRequest("POST", "/api/racer-emails", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("get racer emails after save", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-emails", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var emails []models.RacerEmail
		json.Unmarshal(rr.Body.Bytes(), &emails)
		found := false
		for _, e := range emails {
			if e.RacerID == 1 && e.Email == "test@example.com" {
				found = true
				break
			}
		}
		if !found {
			t.Error("saved racer email not found in list")
		}
	})

	t.Run("save racer email unauthorized", func(t *testing.T) {
		re := models.RacerEmail{RacerID: 1, Email: "noauth@example.com"}
		body, _ := json.Marshal(re)
		req, _ := http.NewRequest("POST", "/api/racer-emails", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 without auth, got %d", rr.Code)
		}
	})

	// cleanup
	testServer.DB.Exec("DELETE FROM racer_emails WHERE racer_id = 1 AND email = 'test@example.com'")
}
