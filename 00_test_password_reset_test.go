package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func resetFlowRouter() *gin.Engine {
	r := gin.New()
	r.POST("/api/login", testHandler.HandleLogin)
	r.POST("/api/forgot-password", testHandler.RequestPasswordReset)
	r.GET("/api/reset-password/validate", testHandler.ValidateResetToken)
	r.POST("/api/reset-password", testHandler.ResetPassword)
	return r
}

func postJSON(r *gin.Engine, path, body string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func getJSON(r *gin.Engine, path string) *httptest.ResponseRecorder {
	req, _ := http.NewRequest("GET", path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// seedResetAdmin inserts an admin directly (avoids disturbing other tests that
// rely on the setup-created admin) and returns its email.
func seedResetAdmin(t *testing.T) string {
	t.Helper()
	testServer.DB.Exec("DELETE FROM admin_users WHERE username = 'resetadmin'")
	_, err := testServer.DB.Exec("INSERT INTO admin_users (username, password, email) VALUES (?, ?, ?)",
		"resetadmin", hashPassword("oldpass123"), "resetadmin@example.com")
	if err != nil {
		t.Fatalf("failed to seed admin: %v", err)
	}
	return "resetadmin@example.com"
}

func fetchResetToken(t *testing.T) string {
	t.Helper()
	var token string
	var expiresAt string
	var used int
	err := testServer.DB.QueryRow("SELECT token, expires_at, used FROM password_reset_tokens WHERE admin_user_id = (SELECT id FROM admin_users WHERE username = 'resetadmin') ORDER BY id DESC LIMIT 1").
		Scan(&token, &expiresAt, &used)
	if err != nil {
		t.Fatalf("no reset token found: %v", err)
	}
	exp, err := time.Parse("2006-01-02 15:04:05", expiresAt)
	if err != nil {
		t.Fatalf("failed to parse expires_at %q: %v", expiresAt, err)
	}
	if !time.Now().Before(exp) {
		t.Errorf("expected token to be unexpired, expires_at=%q", expiresAt)
	}
	if used != 0 {
		t.Errorf("expected token to be unused, got used=%d", used)
	}
	return token
}

func TestForgotPasswordFlow(t *testing.T) {
	r := resetFlowRouter()
	email := seedResetAdmin(t)

	t.Run("RequestReset", func(t *testing.T) {
		rr := postJSON(r, "/api/forgot-password", `{"email":"`+email+`"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	token := fetchResetToken(t)

	t.Run("ValidateToken", func(t *testing.T) {
		rr := getJSON(r, "/api/reset-password/validate?token="+token)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]bool
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["valid"] != true {
			t.Errorf("expected valid=true, got %v", resp)
		}
	})

	t.Run("ResetPassword", func(t *testing.T) {
		rr := postJSON(r, "/api/reset-password", `{"token":"`+token+`","password":"newpass123"}`)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		if sc := rr.Header().Get("Set-Cookie"); !strings.Contains(sc, "session=") {
			t.Errorf("expected session cookie to be set, got Set-Cookie: %q", sc)
		}
	})

	t.Run("OldPasswordFails", func(t *testing.T) {
		rr := postJSON(r, "/api/login", `{"username":"resetadmin","password":"oldpass123"}`)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	t.Run("NewPasswordWorks", func(t *testing.T) {
		rr := postJSON(r, "/api/login", `{"username":"resetadmin","password":"newpass123"}`)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("TokenReuseRejected", func(t *testing.T) {
		rr := postJSON(r, "/api/reset-password", `{"token":"`+token+`","password":"anotherpass123"}`)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

func TestForgotPasswordUnknownEmail(t *testing.T) {
	r := resetFlowRouter()
	testServer.DB.Exec("DELETE FROM password_reset_tokens")

	rr := postJSON(r, "/api/forgot-password", `{"email":"nobody@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 (anti-enumeration), got %d: %s", rr.Code, rr.Body.String())
	}

	var count int
	testServer.DB.QueryRow("SELECT COUNT(*) FROM password_reset_tokens").Scan(&count)
	if count != 0 {
		t.Errorf("expected no token rows for unknown email, got %d", count)
	}
}

func TestForgotPasswordEmptyEmail(t *testing.T) {
	r := resetFlowRouter()
	rr := postJSON(r, "/api/forgot-password", `{"email":""}`)
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("empty email must not 500, got %d", rr.Code)
	}
}

func TestValidateInvalidToken(t *testing.T) {
	r := resetFlowRouter()
	rr := getJSON(r, "/api/reset-password/validate?token=bogustoken")
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResetPasswordShortPassword(t *testing.T) {
	r := resetFlowRouter()
	email := seedResetAdmin(t)
	postJSON(r, "/api/forgot-password", `{"email":"`+email+`"}`)
	token := fetchResetToken(t)

	rr := postJSON(r, "/api/reset-password", `{"token":"`+token+`","password":"short"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestResetPasswordExpiredToken(t *testing.T) {
	r := resetFlowRouter()
	email := seedResetAdmin(t)
	postJSON(r, "/api/forgot-password", `{"email":"`+email+`"}`)
	token := fetchResetToken(t)

	if _, err := testServer.DB.Exec("UPDATE password_reset_tokens SET expires_at = datetime('now', '-1 hour') WHERE token = ?", token); err != nil {
		t.Fatalf("failed to expire token: %v", err)
	}

	rr := getJSON(r, "/api/reset-password/validate?token="+token)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from validate, got %d", rr.Code)
	}

	rr = postJSON(r, "/api/reset-password", `{"token":"`+token+`","password":"newpass123"}`)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 from reset, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSetupStoresAdminEmail(t *testing.T) {
	r := resetFlowRouter()

	// The setup flow only works with zero admins. Wipe and restore after.
	testServer.DB.Exec("DELETE FROM password_reset_tokens")
	testServer.DB.Exec("DELETE FROM admin_users")
	defer func() {
		// Restore the standard admin so later tests are unaffected.
		testServer.DB.Exec("DELETE FROM admin_users")
		testServer.DB.Exec("INSERT INTO admin_users (username, password) VALUES (?, ?)", "admin", hashPassword("admin123"))
	}()

	rr := postJSON(r, "/api/login", `{"username":"admin","password":"admin123","setup":true,"email":"setup@example.com"}`)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 from setup, got %d: %s", rr.Code, rr.Body.String())
	}

	var email string
	err := testServer.DB.QueryRow("SELECT email FROM admin_users WHERE username = 'admin'").Scan(&email)
	if err != nil {
		t.Fatalf("failed to read admin email: %v", err)
	}
	if email != "setup@example.com" {
		t.Errorf("expected stored email setup@example.com, got %q", email)
	}
}
