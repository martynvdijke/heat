package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"heat/app"
)

// identityBotServer builds a telegram test server with the identity tables the
// bot reads directly (telegram_links).
func identityBotServer(t *testing.T) *app.Server {
	t.Helper()
	s := testServer(t)
	if _, err := s.DB.Exec(`CREATE TABLE IF NOT EXISTS telegram_links (
		chat_id TEXT PRIMARY KEY,
		racer_id INTEGER NOT NULL,
		linked_at TEXT NOT NULL DEFAULT (datetime('now'))
	)`); err != nil {
		t.Fatalf("telegram_links schema: %v", err)
	}
	return s
}

func newIdentityBot(s *app.Server) *Bot {
	return &Bot{
		s:             s,
		sentReminders: map[string]time.Time{},
		pendingLogins: map[int64]pendingLogin{},
		pendingQuotes: map[int64]pendingQuote{},
	}
}

func TestLoginFlowLifecycle(t *testing.T) {
	b := newIdentityBot(identityBotServer(t))

	out := b.loginCommand(cmdContext{name: "/login", chatID: 5})
	if !strings.Contains(out, "email address") {
		t.Fatalf("expected email prompt, got %q", out)
	}
	if _, ok := b.loginFlow(5); !ok {
		t.Fatal("expected pending login flow")
	}

	b.clearLoginFlow(5)
	if _, ok := b.loginFlow(5); ok {
		t.Fatal("expected cleared login flow")
	}
}

func TestLoginFlowExpires(t *testing.T) {
	b := newIdentityBot(identityBotServer(t))
	b.mu.Lock()
	b.pendingLogins[6] = pendingLogin{expiresAt: time.Now().Add(-time.Minute)}
	b.mu.Unlock()

	if _, ok := b.loginFlow(6); ok {
		t.Fatal("expected expired login flow to be dropped")
	}
}

func TestStartLoginInvalidEmail(t *testing.T) {
	b := newIdentityBot(identityBotServer(t))
	out := b.startLogin(cmdContext{chatID: 5}, "not-an-email")
	if !strings.Contains(out, "doesn't look like an email") {
		t.Fatalf("expected invalid email reply, got %q", out)
	}
}

func TestStartLoginCallsApi(t *testing.T) {
	var gotToken, gotChatID, gotEmail string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotToken = r.Header.Get("X-Bot-Token")
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		gotChatID = body["chat_id"]
		gotEmail = body["email"]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","found":true,"sent":true}`))
	}))
	defer ts.Close()

	b := &Bot{
		s:             identityBotServer(t),
		http:          ts.Client(),
		baseURL:       ts.URL,
		token:         "bot-secret",
		sentReminders: map[string]time.Time{},
		pendingLogins: map[int64]pendingLogin{},
	}
	b.setLoginFlow(77)
	out := b.startLogin(cmdContext{chatID: 77}, "rider@example.com")
	if gotToken != "bot-secret" {
		t.Errorf("expected X-Bot-Token header, got %q", gotToken)
	}
	if gotChatID != "77" || gotEmail != "rider@example.com" {
		t.Errorf("unexpected payload chat_id=%q email=%q", gotChatID, gotEmail)
	}
	if !strings.Contains(out, "Check your inbox") {
		t.Fatalf("expected success reply, got %q", out)
	}
	if _, ok := b.loginFlow(77); ok {
		t.Fatal("expected login flow cleared after startLogin")
	}
}

func TestStartLoginNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","found":false,"sent":false}`))
	}))
	defer ts.Close()

	b := &Bot{
		s:             identityBotServer(t),
		http:          ts.Client(),
		baseURL:       ts.URL,
		token:         "tok",
		sentReminders: map[string]time.Time{},
		pendingLogins: map[int64]pendingLogin{},
	}
	out := b.startLogin(cmdContext{chatID: 8}, "nobody@example.com")
	if !strings.Contains(out, "No racer is registered") {
		t.Fatalf("expected not-found reply, got %q", out)
	}
}

func TestLogoutAndRacerForChat(t *testing.T) {
	s := identityBotServer(t)
	res, err := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	if err != nil {
		t.Fatalf("insert racer: %v", err)
	}
	racerID, _ := res.LastInsertId()
	if _, err := s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('555', ?)", racerID); err != nil {
		t.Fatalf("insert link: %v", err)
	}

	b := newIdentityBot(s)
	gotID, gotName, ok := b.racerForChat(555)
	if !ok || gotID != int(racerID) || gotName != "Bot Racer" {
		t.Fatalf("racerForChat = (%d, %q, %v)", gotID, gotName, ok)
	}

	out := b.logoutCommand(cmdContext{chatID: 555})
	if !strings.Contains(out, "Signed out") {
		t.Fatalf("expected signed-out reply, got %q", out)
	}
	if _, _, ok := b.racerForChat(555); ok {
		t.Fatal("expected link removed after logout")
	}

	out = b.logoutCommand(cmdContext{chatID: 555})
	if !strings.Contains(out, "not signed in") {
		t.Fatalf("expected not-signed-in reply, got %q", out)
	}
}

func TestPersonalCommandsUnlinked(t *testing.T) {
	b := newIdentityBot(identityBotServer(t))
	if out := b.renderMyStats(cmdContext{chatID: 9}); !strings.Contains(out, "/login") {
		t.Fatalf("renderMyStats unlinked reply: %q", out)
	}
	if out := b.renderMyUpgrades(cmdContext{chatID: 9}); !strings.Contains(out, "/login") {
		t.Fatalf("renderMyUpgrades unlinked reply: %q", out)
	}
}

func TestIdentityCommandsRegistered(t *testing.T) {
	has := func(name string) bool {
		for _, c := range commandRegistry {
			if c.name == name {
				return true
			}
			for _, alias := range c.aliases {
				if alias == name {
					return true
				}
			}
		}
		return false
	}
	for _, name := range []string{"/login", "/logout", "/mystats", "/myupgrades"} {
		if !has(name) {
			t.Errorf("expected command %s to be registered", name)
		}
	}
}

func TestRenderMyStatsLinked(t *testing.T) {
	s := identityBotServer(t)
	res, err := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	if err != nil {
		t.Fatalf("insert racer: %v", err)
	}
	racerID, _ := res.LastInsertId()
	s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('777', ?)", racerID)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/racer-stats"):
			w.Write([]byte(`{"stats":{"races":10,"wins":3,"gold":3,"silver":2,"bronze":1,"fastest_laps":4,"points":120,"dnf":1,"dns":0,"spins":2,"overheated":1},"racer":{"id":1,"name":"Bot Racer"}}`))
		case strings.HasPrefix(r.URL.Path, "/api/telegram/summary"):
			w.Write([]byte(`{"standings":[{"racer_name":"Bot Racer","points":88,"wins":2}],"season":{"id":1,"name":"Season 1"}}`))
		case strings.HasPrefix(r.URL.Path, "/api/racer-recent-results"):
			w.Write([]byte(`[{"name":"Monza","race_date":"2026-10-01","track":"Monza","position":1,"points":25,"fastest_lap":true,"race_type":"season"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	b := &Bot{
		s:             s,
		http:          ts.Client(),
		baseURL:       ts.URL,
		sentReminders: map[string]time.Time{},
		pendingLogins: map[int64]pendingLogin{},
	}
	out := b.renderMyStats(cmdContext{chatID: 777})
	for _, want := range []string{"My Stats", "Bot Racer", "Races: 10", "Wins: 3", "120", "Season 1", "Monza"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderMyStats missing %q in:\n%s", want, out)
		}
	}
}

func TestRenderMyUpgradesLinked(t *testing.T) {
	s := identityBotServer(t)
	res, err := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	if err != nil {
		t.Fatalf("insert racer: %v", err)
	}
	racerID, _ := res.LastInsertId()
	s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('778', ?)", racerID)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasPrefix(r.URL.Path, "/api/player-upgrades") {
			w.Write([]byte(`[{"id":1,"racer_id":1,"upgrade_id":2,"season_id":0,"equipped":true,"round_bought":1,"upgrade":{"id":2,"name":"Cooling System","cost":3}}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	b := &Bot{
		s:             s,
		http:          ts.Client(),
		baseURL:       ts.URL,
		sentReminders: map[string]time.Time{},
		pendingLogins: map[int64]pendingLogin{},
	}
	out := b.renderMyUpgrades(cmdContext{chatID: 778})
	for _, want := range []string{"My Upgrades", "Cooling System", "equipped"} {
		if !strings.Contains(out, want) {
			t.Errorf("renderMyUpgrades missing %q in:\n%s", want, out)
		}
	}
}
