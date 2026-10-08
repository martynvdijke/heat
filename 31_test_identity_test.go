package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/middleware"
)

const identityTestEmail = "identity-flow@example.com"
const identityTestBotToken = "identity-bot-token"

func identityTestRouter() *gin.Engine {
	r := gin.New()
	r.POST("/api/telegram/link/start", testHandler.StartTelegramLink)
	r.POST("/api/me/request-link", testHandler.RequestRacerLink)
	r.GET("/api/telegram/verify/validate", testHandler.ValidateRacerLink)
	r.POST("/api/telegram/verify", testHandler.VerifyRacerLink)
	r.GET("/api/racer-recent-results", testHandler.RacerRecentResults)

	me := r.Group("/api/me")
	me.Use(middleware.RacerAuthMiddleware(testServer))
	me.GET("", testHandler.MeRacer)
	me.GET("/upgrades", testHandler.MeRacerUpgrades)
	me.POST("/upgrades/buy", testHandler.MeBuyUpgrade)
	me.PUT("/upgrades/toggle", testHandler.MeToggleUpgrade)
	me.POST("/logout", testHandler.MeLogout)
	return r
}

func identityJSONRequest(method, path, body string, cookie *http.Cookie) *http.Request {
	req, _ := http.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	return req
}

func identityCookie(rr *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rr.Result().Cookies() {
		if c.Name == app.RacerSessionCookie {
			return c
		}
	}
	return nil
}

// seedIdentityRacer ensures there is a racer with the test email on file.
func seedIdentityRacer(t *testing.T) (int, string) {
	t.Helper()
	var racerID int
	var racerName string
	if err := testServer.DB.QueryRow("SELECT id, name FROM racers ORDER BY id LIMIT 1").Scan(&racerID, &racerName); err != nil {
		t.Fatalf("no seeded racer available: %v", err)
	}
	if _, err := testServer.DB.Exec("DELETE FROM racer_emails WHERE racer_id = ? OR LOWER(email) = LOWER(?)", racerID, identityTestEmail); err != nil {
		t.Fatalf("clear racer emails: %v", err)
	}
	if _, err := testServer.DB.Exec("INSERT INTO racer_emails (racer_id, email) VALUES (?, ?)", racerID, identityTestEmail); err != nil {
		t.Fatalf("insert racer email: %v", err)
	}
	testServer.DB.Exec("DELETE FROM telegram_links WHERE racer_id = ?", racerID)
	return racerID, racerName
}

func TestRacerIdentityFlow(t *testing.T) {
	racerID, racerName := seedIdentityRacer(t)

	var oldToken string
	testServer.DB.QueryRow("SELECT COALESCE(bot_token, '') FROM telegram_settings WHERE id = 1").Scan(&oldToken)
	if _, err := testServer.DB.Exec("UPDATE telegram_settings SET bot_token = ? WHERE id = 1", identityTestBotToken); err != nil {
		t.Fatalf("set bot token: %v", err)
	}
	defer testServer.DB.Exec("UPDATE telegram_settings SET bot_token = ? WHERE id = 1", oldToken)

	r := identityTestRouter()

	t.Run("link start requires bot token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		body := `{"email":"` + identityTestEmail + `","chat_id":"4242"}`
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/telegram/link/start", body, nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 without bot token, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("link start rejects missing chat id", func(t *testing.T) {
		req := identityJSONRequest("POST", "/api/telegram/link/start", `{"email":"`+identityTestEmail+`"}`, nil)
		req.Header.Set("X-Bot-Token", identityTestBotToken)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 without chat_id, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	var linkToken string
	t.Run("link start issues token", func(t *testing.T) {
		req := identityJSONRequest("POST", "/api/telegram/link/start", `{"email":"`+identityTestEmail+`","chat_id":"4242"}`, nil)
		req.Header.Set("X-Bot-Token", identityTestBotToken)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Found bool `json:"found"`
			Sent  bool `json:"sent"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !resp.Found {
			t.Fatalf("expected racer to be found: %s", rr.Body.String())
		}
		// SMTP is not configured in tests, so the email is not sent but the
		// single-use token must still be persisted for verification.
		if err := testServer.DB.QueryRow(
			"SELECT token FROM telegram_link_tokens WHERE racer_id = ? ORDER BY id DESC LIMIT 1", racerID).Scan(&linkToken); err != nil {
			t.Fatalf("expected stored link token: %v", err)
		}
	})

	t.Run("validate token", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("GET", "/api/telegram/verify/validate?token="+linkToken, "", nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Valid     bool   `json:"valid"`
			RacerName string `json:"racer_name"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !resp.Valid || resp.RacerName != racerName {
			t.Fatalf("unexpected validate response: %s", rr.Body.String())
		}
	})

	var racerSession *http.Cookie
	t.Run("verify consumes token and links chat", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/telegram/verify", `{"token":"`+linkToken+`"}`, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		racerSession = identityCookie(rr)
		if racerSession == nil {
			t.Fatalf("expected racer_session cookie: %s", rr.Body.String())
		}
		var linked int
		if err := testServer.DB.QueryRow("SELECT racer_id FROM telegram_links WHERE chat_id = '4242'").Scan(&linked); err != nil {
			t.Fatalf("expected telegram link: %v", err)
		}
		if linked != racerID {
			t.Fatalf("expected chat linked to racer %d, got %d", racerID, linked)
		}
	})

	t.Run("token is single use", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/telegram/verify", `{"token":"`+linkToken+`"}`, nil))
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for reused token, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("me requires racer session", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("GET", "/api/me", "", nil))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("me returns personal stats", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("GET", "/api/me", "", racerSession))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]json.RawMessage
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if _, ok := resp["racer"]; !ok {
			t.Fatalf("expected racer in response: %s", rr.Body.String())
		}
		if _, ok := resp["stats"]; !ok {
			t.Fatalf("expected stats in response: %s", rr.Body.String())
		}
		if _, ok := resp["recent"]; !ok {
			t.Fatalf("expected recent in response: %s", rr.Body.String())
		}
	})

	t.Run("admin session cannot access racer routes", func(t *testing.T) {
		adminSession := createAdminSession(t)
		defer removeAdminSession(adminSession)
		req := identityJSONRequest("GET", "/api/me", "", &http.Cookie{Name: "session", Value: adminSession})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for admin session, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("buy and list upgrades", func(t *testing.T) {
		res, err := testServer.DB.Exec(
			"INSERT INTO upgrade_cards (name, description, card_type, cost, effects, extension_id) VALUES ('Identity Wing', 'test upgrade', 'upgrade', 1, '{}', 0)")
		if err != nil {
			t.Fatalf("insert upgrade card: %v", err)
		}
		upgradeID, _ := res.LastInsertId()

		rr := httptest.NewRecorder()
		body := `{"upgrade_id":` + strconv.FormatInt(upgradeID, 10) + `,"season_id":0,"round":0}`
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/me/upgrades/buy", body, racerSession))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 buying upgrade, got %d: %s", rr.Code, rr.Body.String())
		}

		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("GET", "/api/me/upgrades", "", racerSession))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 listing upgrades, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp struct {
			Owned []map[string]json.RawMessage `json:"owned"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.Owned) == 0 {
			t.Fatalf("expected at least one owned upgrade: %s", rr.Body.String())
		}
	})

	t.Run("toggle rejects upgrades not owned", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("PUT", "/api/me/upgrades/toggle", `{"id":999999,"equipped":true}`, racerSession))
		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("request-link is enumeration safe", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/me/request-link", `{"email":"nobody@example.com"}`, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for unknown email, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("logout clears session", func(t *testing.T) {
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("POST", "/api/me/logout", "", racerSession))
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 logging out, got %d: %s", rr.Code, rr.Body.String())
		}
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, identityJSONRequest("GET", "/api/me", "", racerSession))
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 after logout, got %d: %s", rr.Code, rr.Body.String())
		}
	})
}

func TestRacerRecentResultsValidation(t *testing.T) {
	r := identityTestRouter()
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, identityJSONRequest("GET", "/api/racer-recent-results", "", nil))
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 without racer_id, got %d: %s", rr.Code, rr.Body.String())
	}
}
