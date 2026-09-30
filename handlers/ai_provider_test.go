package handlers

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/ent"
)

func TestAIProviderUserAgent(t *testing.T) {
	if got := aiProviderUserAgent("1.2.3"); got != "heat/1.2.3" {
		t.Errorf("versioned UA = %q", got)
	}
	for _, v := range []string{"", "   "} {
		if got := aiProviderUserAgent(v); got != "heat/0.0.0-dev" {
			t.Errorf("default UA for %q = %q", v, got)
		}
	}
}

func TestResolveSessionID(t *testing.T) {
	if got := resolveSessionID("  abc123  "); got != "abc123" {
		t.Errorf("trimmed = %q", got)
	}
	got := resolveSessionID("")
	if got == "" {
		t.Fatal("generated session is empty")
	}
	if _, err := hex.DecodeString(got); err != nil {
		t.Errorf("generated session is not hex: %v", err)
	}
}

func TestSetAIProviderHeaders(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	setAIProviderHeaders(req, "sid-1", "9.9.9")
	if got := req.Header.Get("User-Agent"); got != "heat/9.9.9" {
		t.Errorf("User-Agent = %q", got)
	}
	if got := req.Header.Get("x-opencode-session"); got != "sid-1" {
		t.Errorf("x-opencode-session = %q", got)
	}

	req2, _ := http.NewRequest("POST", "http://example.com", nil)
	setAIProviderHeaders(req2, "sid-2", "")
	if got := req2.Header.Get("User-Agent"); got != "heat/0.0.0-dev" {
		t.Errorf("default User-Agent = %q", got)
	}
}

func TestAIRequestURL(t *testing.T) {
	cases := map[string]string{
		"https://api.example.com/v1":  "https://api.example.com/v1/chat/completions",
		"https://api.example.com/v1/": "https://api.example.com/v1/chat/completions",
		"":                            "",
	}
	for base, want := range cases {
		if got := aiRequestURL(base, "/chat/completions"); got != want {
			t.Errorf("aiRequestURL(%q) = %q, want %q", base, got, want)
		}
	}
}

func quoteSuggestTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.SQLite, db)))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("ent schema: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return New(&app.Server{DB: db, Ent: client, CurrentVersion: "9.9.9"})
}

func TestHandleQuoteSuggestProviderHeadersAndEcho(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := quoteSuggestTestHandler(t)

	var gotUA, gotSession string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		gotSession = r.Header.Get("x-opencode-session")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"[{\"text\":\"Go go go!\",\"author\":\"Tester\"}]"}}]}`))
	}))
	defer provider.Close()
	t.Setenv("AI_TEXT_GEN_URL", provider.URL)

	r := gin.New()
	r.POST("/api/quotes/ai-suggest", h.HandleQuoteSuggest)

	req, _ := http.NewRequest("POST", "/api/quotes/ai-suggest", strings.NewReader(`{"count":1}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-opencode-session", "supplied-session")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if gotUA != "heat/9.9.9" {
		t.Errorf("outbound User-Agent = %q", gotUA)
	}
	if gotSession != "supplied-session" {
		t.Errorf("outbound x-opencode-session = %q", gotSession)
	}
	if got := rr.Header().Get("X-Opencode-Session"); got != "supplied-session" {
		t.Errorf("response X-Opencode-Session = %q", got)
	}

	var resp struct {
		Suggestions []struct {
			Text   string `json:"text"`
			Author string `json:"author"`
		} `json:"suggestions"`
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Suggestions) != 1 || resp.Suggestions[0].Text != "Go go go!" {
		t.Errorf("suggestions = %+v", resp.Suggestions)
	}
	if resp.SessionID != "supplied-session" {
		t.Errorf("session_id = %q", resp.SessionID)
	}
}

func TestHandleQuoteSuggestGeneratesStableSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := quoteSuggestTestHandler(t)

	var gotSession string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSession = r.Header.Get("x-opencode-session")
		w.Write([]byte(`{"choices":[{"message":{"content":"just one line"}}]}`))
	}))
	defer provider.Close()
	t.Setenv("AI_TEXT_GEN_URL", provider.URL)

	r := gin.New()
	r.POST("/api/quotes/ai-suggest", h.HandleQuoteSuggest)

	req, _ := http.NewRequest("POST", "/api/quotes/ai-suggest", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	if gotSession == "" {
		t.Fatal("generated session is empty")
	}
	if _, err := hex.DecodeString(gotSession); err != nil {
		t.Errorf("generated session is not hex: %v", err)
	}
	if got := rr.Header().Get("X-Opencode-Session"); got != gotSession {
		t.Errorf("echoed session = %q, outbound = %q", got, gotSession)
	}
}
