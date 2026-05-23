package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"heat/app"
	"heat/middleware"
)

func TestHashPassword(t *testing.T) {
	password := "password123"
	hash := hashPassword(password)

	if hash == "" {
		t.Error("Expected hash to be non-empty")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		t.Error("Expected password to verify against its hash")
	}

	hash2 := hashPassword("anotherpassword")
	if err := bcrypt.CompareHashAndPassword([]byte(hash2), []byte("anotherpassword")); err != nil {
		t.Error("Expected second password to verify against its hash")
	}
}

func TestHandleCheckSetup(t *testing.T) {
	r := gin.New()
	r.GET("/api/check-setup", testHandler.HandleCheckSetup)

	req, err := http.NewRequest("GET", "/api/check-setup", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var response map[string]bool
	err = json.Unmarshal(rr.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response["setup"] != false {
		t.Errorf("expected setup to be false, got %v", response["setup"])
	}
}

func TestAuthMiddleware(t *testing.T) {
	r := gin.New()
	r.GET("/api/test", middleware.AuthMiddleware(testServer), func(c *gin.Context) {
		c.String(http.StatusOK, "Authorized")
	})

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status %v, got %v", http.StatusUnauthorized, status)
		}
	})

	t.Run("Authorized", func(t *testing.T) {
		sessionID := "test-session"
		testServer.SessionStoreMu.Lock()
		testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
		testServer.SessionStoreMu.Unlock()
		defer func() {
			testServer.SessionStoreMu.Lock()
			delete(testServer.SessionStore, sessionID)
			testServer.SessionStoreMu.Unlock()
		}()

		req, _ := http.NewRequest("GET", "/api/test", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status %v, got %v", http.StatusOK, status)
		}

		if rr.Body.String() != "Authorized" {
			t.Errorf("expected body 'Authorized', got '%s'", rr.Body.String())
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	r := gin.New()
	r.Use(middleware.SecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	headers := rr.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("X-Content-Type-Options not set")
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Error("X-Frame-Options not set")
	}
	if headers.Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("X-XSS-Protection not set")
	}
}

func TestLoginLogout(t *testing.T) {
	r := gin.New()
	r.POST("/api/login", testHandler.HandleLogin)
	r.POST("/api/logout", testHandler.HandleLogout)

	t.Run("SetupNewAdmin", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "admin123",
			"setup":    true,
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}
	})

	t.Run("Login", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "admin123",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
		}
	})

	t.Run("LoginInvalid", func(t *testing.T) {
		payload := map[string]interface{}{
			"username": "admin",
			"password": "wrongpassword",
		}
		body, _ := json.Marshal(payload)
		req, _ := http.NewRequest("POST", "/api/login", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestPlayerLogin(t *testing.T) {
	r := gin.New()
	r.POST("/api/player/login", testHandler.PlayerLogin)
	r.GET("/api/player/validate", testHandler.ValidatePlayerToken)
	r.POST("/api/player/logout", testHandler.PlayerLogout)

	var token string
	t.Run("login valid racer", func(t *testing.T) {
		body := `{"racer_id":1,"device_name":"TestPhone"}`
		req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Token     string `json:"token"`
			RacerID   int    `json:"racer_id"`
			RacerName string `json:"racer_name"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.Token == "" {
			t.Error("expected non-empty token")
		}
		if resp.RacerID != 1 {
			t.Errorf("expected racer_id=1, got %d", resp.RacerID)
		}
		token = resp.Token
	})

	t.Run("validate token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/validate", nil)
		req.Header.Set("X-Player-Token", token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var resp struct {
			RacerID   int    `json:"racer_id"`
			RacerName string `json:"racer_name"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp.RacerName == "" {
			t.Error("expected non-empty racer name")
		}
	})

	t.Run("reject invalid token", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/validate", nil)
		req.Header.Set("X-Player-Token", "invalid_token_12345")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("login nonexistent racer", func(t *testing.T) {
		body := `{"racer_id":999,"device_name":"Test"}`
		req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}

func TestRateLimit(t *testing.T) {
	r := gin.New()
	r.Use(middleware.RateLimitMiddleware(testServer))
	hitCount := 0
	r.GET("/test", func(c *gin.Context) {
		hitCount++
		c.String(http.StatusOK, "ok")
	})

	// Send multiple requests rapidly
	for i := 0; i < 20; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
	}
}

func TestVersion(t *testing.T) {
	r := gin.New()
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": testServer.CurrentVersion})
	})

	req, _ := http.NewRequest("GET", "/api/version", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var resp map[string]string
	json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp["version"] != testServer.CurrentVersion {
		t.Errorf("expected version %q, got %q", testServer.CurrentVersion, resp["version"])
	}
}
