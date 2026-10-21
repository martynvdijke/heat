package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/handlers"
)

// OIDC disabled by default in tests (no OIDC_* env): routes stay inert,
// password login unaffected (add-authelia-oidc fallback requirement).
func setupOIDCRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/auth/oidc/config", testHandler.HandleOIDCConfig)
	r.GET("/api/auth/oidc/login", testHandler.HandleOIDCLogin)
	r.GET("/api/auth/oidc/callback", testHandler.HandleOIDCCallback)
	r.GET("/api/auth/oidc/logout", testHandler.HandleOIDCLogout)
	r.POST("/api/auth/oidc/mobile/start", testHandler.HandleOIDCMobileStart)
	r.GET("/api/auth/oidc/mobile/status", testHandler.HandleOIDCMobileStatus)
	r.GET("/api/auth/oidc/mobile/done", testHandler.HandleOIDCMobileDone)
	return r
}

func TestOIDCDisabled(t *testing.T) {
	testServer.OIDC.Enabled = false
	r := setupOIDCRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/auth/oidc/config", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || w.Body.String() != `{"enabled":false}` {
		t.Fatalf("config = %d %s, want 200 {\"enabled\":false}", w.Code, w.Body.String())
	}

	for _, path := range []string{"/api/auth/oidc/login", "/api/auth/oidc/callback"} {
		w = httptest.NewRecorder()
		req, _ = http.NewRequest("GET", path, nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusNotFound {
			t.Fatalf("GET %s = %d, want 404 when OIDC disabled", path, w.Code)
		}
	}

	// Logout with OIDC disabled clears session and redirects home.
	sid := createAdminSession(t)
	defer removeAdminSession(sid)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/logout", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sid})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusFound {
		t.Fatalf("logout = %d, want 302", w.Code)
	}
	testServer.SessionStoreMu.RLock()
	_, ok := testServer.SessionStore[sid]
	testServer.SessionStoreMu.RUnlock()
	if ok {
		t.Fatal("logout did not clear local session")
	}
}

func TestOIDCLinkOrProvision(t *testing.T) {
	email := "oidc-provision@example.com"
	sub := "https://authelia.example.com|test-sub-1"
	testServer.DB.Exec("DELETE FROM admin_users WHERE email = ?", email)
	defer testServer.DB.Exec("DELETE FROM admin_users WHERE email = ?", email)

	// Existing password user links by verified email.
	testServer.DB.Exec("INSERT INTO admin_users (username, password, email) VALUES (?, ?, ?)",
		"oidclink", hashPassword("password123"), email)
	if err := testHandler.LinkOrProvisionOIDCUser(sub, &handlers.OIDCClaims{Email: email, EmailVerified: true}); err != nil {
		t.Fatalf("link = %v", err)
	}
	var gotSub, gotMethod string
	if err := testServer.DB.QueryRow("SELECT oidc_sub, auth_method FROM admin_users WHERE email = ?", email).Scan(&gotSub, &gotMethod); err != nil {
		t.Fatalf("select = %v", err)
	}
	if gotSub != sub || gotMethod != "oidc" {
		t.Fatalf("linked = (%q, %q), want (%q, oidc)", gotSub, gotMethod, sub)
	}

	// Returning OIDC user matches by sub even if email changed.
	if err := testHandler.LinkOrProvisionOIDCUser(sub, &handlers.OIDCClaims{Email: "oidc-new@example.com", EmailVerified: true}); err != nil {
		t.Fatalf("relink = %v", err)
	}
	var count int
	testServer.DB.QueryRow("SELECT COUNT(*) FROM admin_users WHERE oidc_sub = ?", sub).Scan(&count)
	if count != 1 {
		t.Fatalf("rows with sub = %d, want 1 (no duplicate)", count)
	}
	testServer.DB.Exec("DELETE FROM admin_users WHERE email = ?", "oidc-new@example.com")

	// First-time login auto-provisions.
	testServer.DB.Exec("DELETE FROM admin_users WHERE oidc_sub = ?", sub)
	newEmail := "oidc-first@example.com"
	defer testServer.DB.Exec("DELETE FROM admin_users WHERE email = ?", newEmail)
	if err := testHandler.LinkOrProvisionOIDCUser("https://authelia.example.com|brand-new", &handlers.OIDCClaims{Email: newEmail, EmailVerified: true, PreferredUsername: "oidcfirst"}); err != nil {
		t.Fatalf("provision = %v", err)
	}
	var username, password string
	if err := testServer.DB.QueryRow("SELECT username, password FROM admin_users WHERE email = ?", newEmail).Scan(&username, &password); err != nil {
		t.Fatalf("select new = %v", err)
	}
	if username != "oidcfirst" || password != "" {
		t.Fatalf("provisioned = (%q, pwlen %d), want (oidcfirst, empty pw)", username, len(password))
	}
}

func TestOIDCMobileDeviceFlow(t *testing.T) {
	testServer.OIDC.Enabled = true
	testServer.OIDC.RedirectURL = "https://heat.example.com/heat/api/auth/oidc/callback"
	defer func() {
		testServer.OIDC.Enabled = false
		testServer.OIDC.RedirectURL = ""
	}()
	r := setupOIDCRouter()

	// The app asks for a device code and browser URL.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/auth/oidc/mobile/start", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("start = %d %s, want 200", w.Code, w.Body.String())
	}
	var start struct {
		DeviceCode string `json:"device_code"`
		AuthURL    string `json:"auth_url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &start); err != nil {
		t.Fatalf("decode start = %v", err)
	}
	if start.DeviceCode == "" {
		t.Fatal("start returned an empty device_code")
	}
	wantAuth := "https://heat.example.com/heat/api/auth/oidc/login?device=" + url.QueryEscape(start.DeviceCode)
	if start.AuthURL != wantAuth {
		t.Fatalf("auth_url = %q, want %q", start.AuthURL, wantAuth)
	}

	// Until the browser callback completes the app sees "pending".
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/mobile/status?device="+url.QueryEscape(start.DeviceCode), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":"pending"`) {
		t.Fatalf("pending status = %d %s", w.Code, w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("status Cache-Control = %q, want no-store", got)
	}

	// Completing the browser flow hands the session to the app exactly once.
	testServer.PutOIDCPendingLogin(start.DeviceCode, "mobile-session-id")
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/mobile/status?device="+url.QueryEscape(start.DeviceCode), nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"session":"mobile-session-id"`) {
		t.Fatalf("ok status = %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/mobile/status?device="+url.QueryEscape(start.DeviceCode), nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"status":"pending"`) {
		t.Fatalf("session was returned more than once: %s", w.Body.String())
	}

	// Browser-side failures are surfaced instead of polling forever. The
	// callback reads the device code from the cookie set by /login.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/callback?error=access_denied&error_description=denied", nil)
	req.AddCookie(&http.Cookie{Name: "oidc_device", Value: start.DeviceCode})
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("callback error = %d, want 400", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/mobile/status?device="+url.QueryEscape(start.DeviceCode), nil)
	r.ServeHTTP(w, req)
	if !strings.Contains(w.Body.String(), `"status":"error"`) || !strings.Contains(w.Body.String(), "access_denied") {
		t.Fatalf("error status = %s", w.Body.String())
	}

	// Codes are opaque tokens; arbitrary query values are rejected.
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/mobile/status?device=../etc/passwd", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid device status = %d, want 400", w.Code)
	}
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/auth/oidc/login?device=not%20a%20token", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("invalid login device = %d, want 400", w.Code)
	}
}
