package main

import (
	"net/http"
	"net/http/httptest"
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
