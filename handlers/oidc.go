package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"heat/app"
)

// OIDCClaims is the verified ID token payload Heat cares about.
type OIDCClaims struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Groups            []string `json:"groups"`
	Name              string   `json:"name"`
	PreferredUsername string   `json:"preferred_username"`
}

// Cached OIDC provider setup, re-initialized when config changes.
var (
	oidcMu       sync.Mutex
	oidcProvider *oidc.Provider
	oidcVerifier *oidc.IDTokenVerifier
	oidcOAuth2   *oauth2.Config
	oidcIssuer   string
	oidcClientID string
)

func oidcSetup(ctx context.Context, h *Handler) (*oidc.Provider, *oidc.IDTokenVerifier, *oauth2.Config, error) {
	cfg := h.S.OIDC
	oidcMu.Lock()
	defer oidcMu.Unlock()
	if oidcProvider != nil && oidcIssuer == cfg.IssuerURL && oidcClientID == cfg.ClientID {
		return oidcProvider, oidcVerifier, oidcOAuth2, nil
	}
	provider, err := oidc.NewProvider(ctx, cfg.IssuerURL)
	if err != nil {
		return nil, nil, nil, err
	}
	oidcProvider = provider
	oidcVerifier = provider.Verifier(&oidc.Config{ClientID: cfg.ClientID})
	oidcOAuth2 = &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       cfg.Scopes,
	}
	oidcIssuer = cfg.IssuerURL
	oidcClientID = cfg.ClientID
	return oidcProvider, oidcVerifier, oidcOAuth2, nil
}

func oidcRandom(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func oidcTempCookie(c *gin.Context, name, value string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func oidcClearCookie(c *gin.Context, name string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func oidcInGroups(groups []string, want string) bool {
	for _, g := range groups {
		if g == want {
			return true
		}
	}
	return false
}

// @Summary OIDC provider status
// @Description Returns whether OIDC login is enabled
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]bool
// @Router /api/auth/oidc/config [get]
func (h *Handler) HandleOIDCConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"enabled": h.S.OIDC.Enabled})
}

// @Summary Start OIDC login
// @Description Redirect to Authelia authorize with PKCE S256 + state/nonce cookies
// @Tags Auth
// @Success 302 {string} string "Redirect to IdP"
// @Router /api/auth/oidc/login [get]
func (h *Handler) HandleOIDCLogin(c *gin.Context) {
	if !h.S.OIDC.Enabled {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "OIDC disabled"})
		return
	}
	_, _, oauth2Cfg, err := oidcSetup(c.Request.Context(), h)
	if err != nil {
		log.Printf("[OIDC] provider discovery failed: %v", err)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "OIDC provider unavailable"})
		return
	}
	state := oidcRandom(32)
	nonce := oidcRandom(32)
	verifier := oauth2.GenerateVerifier()
	if state == "" || nonce == "" || verifier == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to generate OIDC parameters"})
		return
	}
	oidcTempCookie(c, "oidc_state", state, h.S.SecureCookies)
	oidcTempCookie(c, "oidc_nonce", nonce, h.S.SecureCookies)
	oidcTempCookie(c, "oidc_pkce", verifier, h.S.SecureCookies)
	c.Redirect(http.StatusFound, oauth2Cfg.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier)))
}

// @Summary OIDC callback
// @Description Verify state/nonce/PKCE + ID token, link/provision user, set session cookie
// @Tags Auth
// @Success 302 {string} string "Redirect to app"
// @Router /api/auth/oidc/callback [get]
func (h *Handler) HandleOIDCCallback(c *gin.Context) {
	if !h.S.OIDC.Enabled {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "OIDC disabled"})
		return
	}
	if e := c.Query("error"); e != "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": e + ": " + c.Query("error_description")})
		return
	}
	stateCookie, err := c.Request.Cookie("oidc_state")
	if err != nil || c.Query("state") != stateCookie.Value {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "state mismatch"})
		return
	}
	pkceCookie, err := c.Request.Cookie("oidc_pkce")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "PKCE verifier missing"})
		return
	}
	nonceCookie, err := c.Request.Cookie("oidc_nonce")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "nonce missing"})
		return
	}
	_, verifier, oauth2Cfg, err := oidcSetup(c.Request.Context(), h)
	if err != nil {
		log.Printf("[OIDC] provider discovery failed: %v", err)
		c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{"error": "OIDC provider unavailable"})
		return
	}
	token, err := oauth2Cfg.Exchange(c.Request.Context(), c.Query("code"), oauth2.VerifierOption(pkceCookie.Value))
	if err != nil {
		log.Printf("[OIDC] code exchange failed: %v", err)
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "code exchange failed"})
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "no id_token in response"})
		return
	}
	idToken, err := verifier.Verify(c.Request.Context(), rawIDToken)
	if err != nil {
		log.Printf("[OIDC] token verify failed: %v", err)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid ID token"})
		return
	}
	// Verify() does NOT check nonce — compare manually.
	if idToken.Nonce != nonceCookie.Value {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "nonce mismatch"})
		return
	}
	var claims OIDCClaims
	if err := idToken.Claims(&claims); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "invalid ID token claims"})
		return
	}
	for _, n := range []string{"oidc_state", "oidc_nonce", "oidc_pkce"} {
		oidcClearCookie(c, n, h.S.SecureCookies)
	}
	if !claims.EmailVerified || claims.Email == "" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "verified email required"})
		return
	}
	// admin_users has no admin column — every row is an admin, so OIDC
	// login is gated on the `admins` group (provision + sync in one check).
	if !oidcInGroups(claims.Groups, "admins") {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "not a member of the admins group"})
		return
	}
	sub := h.S.OIDC.IssuerURL + "|" + claims.Sub
	if err := h.LinkOrProvisionOIDCUser(sub, &claims); err != nil {
		log.Printf("[OIDC] user link failed: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to provision user"})
		return
	}
	sessionID := generateSessionID()
	h.S.SessionStoreMu.Lock()
	h.S.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(24 * time.Hour).Unix(), IP: c.ClientIP()}
	h.S.SessionStoreMu.Unlock()
	setSessionCookie(c, sessionID, h.S.SecureCookies)
	c.Redirect(http.StatusFound, "/admin.html")
}

// LinkOrProvisionOIDCUser links oidc_sub by verified email or provisions a
// first-time user. Password hash stays empty so OIDC users can't password-login.
func (h *Handler) LinkOrProvisionOIDCUser(sub string, claims *OIDCClaims) error {
	var id int
	err := h.S.DB.QueryRow("SELECT id FROM admin_users WHERE oidc_sub = ?", sub).Scan(&id)
	if err == nil {
		_, err = h.S.DB.Exec("UPDATE admin_users SET email = ?, auth_method = 'oidc' WHERE id = ?", claims.Email, id)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	err = h.S.DB.QueryRow("SELECT id FROM admin_users WHERE email = ?", claims.Email).Scan(&id)
	if err == nil {
		_, err = h.S.DB.Exec("UPDATE admin_users SET oidc_sub = ?, auth_method = 'oidc' WHERE id = ?", sub, id)
		return err
	}
	if err != sql.ErrNoRows {
		return err
	}
	username := claims.PreferredUsername
	if username == "" {
		username, _, _ = strings.Cut(claims.Email, "@")
	}
	username = strings.TrimSpace(username)
	if username == "" {
		username = "oidc-user"
	}
	base := username
	for i := 1; ; i++ {
		var existing int
		qerr := h.S.DB.QueryRow("SELECT id FROM admin_users WHERE username = ?", username).Scan(&existing)
		if qerr == sql.ErrNoRows {
			break
		}
		if qerr != nil {
			return qerr
		}
		username = fmt.Sprintf("%s%d", base, i)
	}
	_, err = h.S.DB.Exec(
		"INSERT INTO admin_users (username, password, email, oidc_sub, auth_method) VALUES (?, '', ?, ?, 'oidc')",
		username, claims.Email, sub,
	)
	return err
}

// @Summary OIDC logout
// @Description Clear local session and redirect to Authelia logout
// @Tags Auth
// @Success 302 {string} string "Redirect to IdP logout"
// @Router /api/auth/oidc/logout [get]
func (h *Handler) HandleOIDCLogout(c *gin.Context) {
	if cookie, err := c.Request.Cookie("session"); err == nil {
		h.S.SessionStoreMu.Lock()
		delete(h.S.SessionStore, cookie.Value)
		h.S.SessionStoreMu.Unlock()
	}
	c.SetCookie("session", "", -1, "/", "", h.S.SecureCookies, true)
	if !h.S.OIDC.Enabled {
		c.Redirect(http.StatusFound, "/")
		return
	}
	logoutURL := h.S.OIDC.LogoutURL
	if logoutURL == "" && h.S.OIDC.IssuerURL != "" {
		logoutURL = h.S.OIDC.IssuerURL + "/logout"
	}
	if logoutURL == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	u, err := url.Parse(logoutURL)
	if err != nil {
		c.Redirect(http.StatusFound, "/")
		return
	}
	q := u.Query()
	// Authelia stable uses `rd` for post-logout redirect (no OIDC
	// RP-initiated end_session_endpoint yet).
	q.Set("rd", "/")
	u.RawQuery = q.Encode()
	c.Redirect(http.StatusFound, u.String())
}
