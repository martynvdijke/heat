package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"heat/middleware"
	"heat/models"
)

func setupWLEDRouter() *gin.Engine {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/wled-settings", testHandler.GetWLEDSettings)
	admin.POST("/wled-settings", testHandler.SaveWLEDSettings)
	admin.POST("/wled-settings/test", testHandler.TestWLED)
	admin.GET("/wled/status", testHandler.GetWLEDStatus)
	admin.POST("/wled/arm", testHandler.SetWLEDArmed)
	return r
}

func TestWLEDSettings(t *testing.T) {
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)
	r := setupWLEDRouter()

	t.Run("get defaults", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/wled-settings", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var s models.WLEDSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.ID != 1 {
			t.Errorf("expected id 1, got %d", s.ID)
		}
		if s.Presets == nil {
			t.Errorf("expected non-nil presets map")
		}
	})

	t.Run("save and reload", func(t *testing.T) {
		body, _ := json.Marshal(models.WLEDSettings{
			URL:     "http://wled.local",
			Enabled: true,
			Presets: map[string]int{"red": 3, "blue": 5},
		})
		req := newAdminRequest("POST", "/api/wled-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}

		req = newAdminRequest("GET", "/api/wled-settings", nil, sessionID)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var s models.WLEDSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.URL != "http://wled.local" || !s.Enabled {
			t.Errorf("expected url/enabled saved, got %q %v", s.URL, s.Enabled)
		}
		if s.Presets["red"] != 3 || s.Presets["blue"] != 5 {
			t.Errorf("expected presets saved, got %v", s.Presets)
		}
	})

	t.Run("status reflects enabled", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/wled/status", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !resp["enabled"] {
			t.Errorf("expected enabled=true after save")
		}
	})

	t.Run("arm toggle", func(t *testing.T) {
		body, _ := json.Marshal(map[string]bool{"armed": true})
		req := newAdminRequest("POST", "/api/wled/arm", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("arm: expected 200, got %d", rr.Code)
		}
		if !testServer.WLEDArmed.Load() {
			t.Errorf("expected armed=true")
		}

		body, _ = json.Marshal(map[string]bool{"armed": false})
		req = newAdminRequest("POST", "/api/wled/arm", body, sessionID)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if testServer.WLEDArmed.Load() {
			t.Errorf("expected armed=false")
		}
	})

	t.Run("test without url fails", func(t *testing.T) {
		// Clear URL to force the not-configured path.
		body, _ := json.Marshal(models.WLEDSettings{Enabled: true, Presets: map[string]int{}})
		req := newAdminRequest("POST", "/api/wled-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		req = newAdminRequest("POST", "/api/wled-settings/test", nil, sessionID)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 without url, got %d", rr.Code)
		}
	})

	t.Run("unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/wled-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})
}

func TestWLEDBroadcastFanout(t *testing.T) {
	// A flag sent to FlagBroadcast should also land on WLEDBroadcast. Other
	// tests share these channels, so drain until our command shows up.
	cmd := models.FlagCommand{Type: "flag", Flag: "safety", State: "on"}
	testServer.FlagBroadcast <- cmd

	deadline := time.After(2 * time.Second)
	for {
		select {
		case got := <-testServer.WLEDBroadcast:
			if got.Flag == "safety" && got.State == "on" {
				return
			}
		case <-deadline:
			t.Fatal("expected flag to fan out to WLEDBroadcast")
		}
	}
}
