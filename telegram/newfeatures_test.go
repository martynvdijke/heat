package telegram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"heat/app"
)

// featureServer extends the identity test server with the tables the new
// features read directly.
func featureServer(t *testing.T) *app.Server {
	t.Helper()
	s := identityBotServer(t)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS telegram_notify_prefs (
			chat_id TEXT PRIMARY KEY,
			personal INTEGER NOT NULL DEFAULT 0,
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS telegram_rsvp (
			race_date TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			racer_id INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			updated_at TEXT NOT NULL DEFAULT (datetime('now')),
			PRIMARY KEY (race_date, chat_id)
		)`,
		`CREATE TABLE IF NOT EXISTS racer_achievements (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			racer_id INTEGER NOT NULL,
			code TEXT NOT NULL,
			label TEXT NOT NULL,
			detail TEXT NOT NULL DEFAULT '',
			race_id INTEGER NOT NULL DEFAULT 0,
			achieved_at TEXT NOT NULL DEFAULT (datetime('now')),
			UNIQUE(racer_id, code)
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.DB.Exec(stmt); err != nil {
			t.Fatalf("feature schema: %v", err)
		}
	}
	return s
}

func TestNewCommandsRegistered(t *testing.T) {
	for _, name := range []string{"/elo", "/streaks", "/h2h", "/career", "/trackperf", "/calendar", "/rsvp", "/notify", "/badges"} {
		if _, ok := findCommand(name); !ok {
			t.Errorf("expected command %s to be registered", name)
		}
	}
}

func TestSplitH2HArgs(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"Alice vs Bob", 2},
		{"Alice VS Bob", 2},
		{"Alice, Bob", 2},
		{"Alice — Bob", 2},
		{"Alice", 0},
		{"", 0},
	}
	for _, tc := range cases {
		if got := len(splitH2HArgs(tc.in)); got != tc.want {
			t.Errorf("splitH2HArgs(%q) = %d parts, want %d", tc.in, got, tc.want)
		}
	}
}

func TestEvaluateAchievements(t *testing.T) {
	facts := achievementFacts{
		races: 25, wins: 5, podiums: 10, points: 120,
		fastestLaps: 4, bestStreak: 3, elo: 1650,
		isLeader: true, cleanSweep: true,
	}
	got := map[string]bool{}
	for _, a := range evaluateAchievements(facts) {
		got[a.code] = true
	}
	for _, code := range []string{"first_win", "first_podium", "fastest_lap", "podium_streak",
		"five_wins", "podium_machine", "clean_sweep", "centurion", "veteran", "elo_1600", "points_leader"} {
		if !got[code] {
			t.Errorf("expected achievement %s to be unlocked", code)
		}
	}
	if got["elo_1700"] {
		t.Error("elo_1700 should not unlock at rating 1650")
	}
	// A rookie unlocks nothing.
	if len(evaluateAchievements(achievementFacts{})) != 0 {
		t.Error("empty facts should unlock nothing")
	}
}

func TestPersonalResultMessage(t *testing.T) {
	race := &apiRace{
		Name:    "Spa",
		Results: []apiRaceResult{{RacerName: "Alice", Position: 1, Points: 25}, {RacerName: "Bob", Position: 2, Points: 18}},
	}
	standings := []apiStanding{{RacerName: "Alice", Points: 120}, {RacerName: "Bob", Points: 90}}
	out := personalResultMessage(race, "Alice", standings, "Season 1")
	for _, want := range []string{"Your result", "Spa", "🥇", "Alice", "25 pts", "Season 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("personalResultMessage missing %q in:\n%s", want, out)
		}
	}
	if out := personalResultMessage(race, "Nobody", standings, ""); out != "" {
		t.Errorf("non-participant should render empty, got %q", out)
	}
}

func TestPersonalNotifyToggle(t *testing.T) {
	s := featureServer(t)
	b := newIdentityBot(s)
	if b.personalNotifyEnabled(5) {
		t.Fatal("personal DMs should default to off")
	}
	if err := b.setPersonalNotify(5, true); err != nil {
		t.Fatalf("setPersonalNotify: %v", err)
	}
	if !b.personalNotifyEnabled(5) {
		t.Fatal("expected personal DMs on")
	}
	if err := b.setPersonalNotify(5, false); err != nil {
		t.Fatalf("setPersonalNotify off: %v", err)
	}
	if b.personalNotifyEnabled(5) {
		t.Fatal("expected personal DMs off")
	}
}

func TestNotifyCommandUnlinkedPromptsLogin(t *testing.T) {
	b := newIdentityBot(featureServer(t))
	if out := b.notifyCommand(cmdContext{chatID: 5, args: "on"}); !strings.Contains(out, "/login") {
		t.Fatalf("unlinked /notify reply: %q", out)
	}
}

func TestNotifyCommandLinked(t *testing.T) {
	s := featureServer(t)
	res, _ := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	racerID, _ := res.LastInsertId()
	s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('5', ?)", racerID)
	b := newIdentityBot(s)

	if out := b.notifyCommand(cmdContext{chatID: 5}); !strings.Contains(out, "off") {
		t.Errorf("default state reply: %q", out)
	}
	if out := b.notifyCommand(cmdContext{chatID: 5, args: "on"}); !strings.Contains(out, "Personal DMs on") {
		t.Errorf("notify on reply: %q", out)
	}
	if !b.personalNotifyEnabled(5) {
		t.Error("expected opt-in enabled")
	}
	if out := b.notifyCommand(cmdContext{chatID: 5, args: "off"}); !strings.Contains(out, "off") {
		t.Errorf("notify off reply: %q", out)
	}
}

func TestRsvpFlow(t *testing.T) {
	s := featureServer(t)
	res, _ := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	racerID, _ := res.LastInsertId()
	s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('5', ?)", racerID)
	b := newIdentityBot(s)

	reply, date, ok := b.handleRsvpCallback(cmdContext{chatID: 5}, "rsvp:2026-10-10:in")
	if !ok || date != "2026-10-10" {
		t.Fatalf("rsvp callback = (%q, %q, %v)", reply, date, ok)
	}
	if !strings.Contains(reply, "In: 1") || !strings.Contains(reply, "Bot Racer") {
		t.Errorf("rsvp tally missing entry: %q", reply)
	}

	reply, _, ok = b.handleRsvpCallback(cmdContext{chatID: 6}, "rsvp:2026-10-10:maybe")
	if !ok || !strings.Contains(reply, "Maybe: 1") {
		t.Errorf("second rsvp reply: %q (ok=%v)", reply, ok)
	}

	// Updating the same chat overwrites rather than duplicating.
	reply, _, _ = b.handleRsvpCallback(cmdContext{chatID: 5}, "rsvp:2026-10-10:out")
	if !strings.Contains(reply, "Can't: 1") || strings.Contains(reply, "In: 1") {
		t.Errorf("expected overwrite to Can't: 1, got: %q", reply)
	}

	if _, _, ok := b.handleRsvpCallback(cmdContext{chatID: 5}, "rsvp:2026-10-10:bogus"); ok {
		t.Error("invalid status should not be accepted")
	}
	if _, _, ok := b.handleRsvpCallback(cmdContext{chatID: 5}, "rsvp:nodate"); ok {
		t.Error("malformed callback data should not be accepted")
	}
}

func TestRsvpKeyboard(t *testing.T) {
	b := newIdentityBot(featureServer(t))
	flat := flattenKeyboard(b.rsvpKeyboard("2026-10-10"))
	if len(flat) != 3 {
		t.Fatalf("rsvp keyboard has %d buttons, want 3", len(flat))
	}
	for _, btn := range flat {
		if !strings.HasPrefix(btn.CallbackData, callbackRsvpPrefix+"2026-10-10:") {
			t.Errorf("unexpected rsvp callback data %q", btn.CallbackData)
		}
	}
}

func TestAwardAchievementAndBadges(t *testing.T) {
	s := featureServer(t)
	res, _ := s.DB.Exec("INSERT INTO racers (name) VALUES ('Bot Racer')")
	racerID, _ := res.LastInsertId()
	s.DB.Exec("INSERT INTO telegram_links (chat_id, racer_id) VALUES ('7', ?)", racerID)
	b := newIdentityBot(s)

	a := achievementByCode["first_win"]
	if !b.awardAchievement(int(racerID), a) {
		t.Fatal("first award should report a new achievement")
	}
	if b.awardAchievement(int(racerID), a) {
		t.Fatal("duplicate award should be ignored")
	}

	out := b.badgesCommand(cmdContext{chatID: 7})
	if !strings.Contains(out, "First Win") || !strings.Contains(out, "1/12 unlocked") {
		t.Errorf("badges render: %q", out)
	}
	if out := b.badgesCommand(cmdContext{chatID: 99}); !strings.Contains(out, "/login") {
		t.Errorf("unlinked badges reply: %q", out)
	}
}

func TestCheckAchievementsIntegration(t *testing.T) {
	s := featureServer(t)
	s.DB.Exec(`INSERT INTO telegram_settings (id, bot_token, enabled, default_chat_id) VALUES (1, 'tok', 1, '')`)
	res, _ := s.DB.Exec("INSERT INTO racers (name) VALUES ('Alice')")
	racerID, _ := res.LastInsertId()

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/racers":
			w.Write([]byte(`[{"id":1,"name":"Alice"}]`))
		case r.URL.Path == "/api/stats/elo":
			w.Write([]byte(`[{"racer_id":1,"racer_name":"Alice","rating":1650,"races":5}]`))
		case r.URL.Path == "/api/stats/streaks":
			w.Write([]byte(`[{"racer_name":"Alice","streak_type":"podium","current_value":3,"best_value":3}]`))
		case r.URL.Path == "/api/telegram/summary":
			w.Write([]byte(`{"standings":[{"racer_name":"Alice","points":120}],"season":{"id":1,"name":"Season 1"}}`))
		case r.URL.Path == "/api/race-history":
			w.Write([]byte(`[{"id":1,"name":"Spa","results":[{"racer_name":"Alice","position":1,"fastest_lap":true}]}]`))
		case strings.HasPrefix(r.URL.Path, "/api/racer-stats"):
			w.Write([]byte(`{"stats":{"races":25,"wins":5,"gold":5,"silver":2,"bronze":3,"fastest_laps":4,"points":120},"racer":{"id":1,"name":"Alice"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer api.Close()

	b := &Bot{s: s, http: api.Client(), baseURL: api.URL, sentReminders: map[string]time.Time{}, pendingLogins: map[int64]pendingLogin{}, pendingQuotes: map[int64]pendingQuote{}}
	b.checkAchievements()

	var n int
	s.DB.QueryRow("SELECT COUNT(*) FROM racer_achievements WHERE racer_id = ?", racerID).Scan(&n)
	if n != 11 {
		t.Errorf("expected 11 achievements, got %d", n)
	}

	// Running again must not add duplicates.
	b.checkAchievements()
	s.DB.QueryRow("SELECT COUNT(*) FROM racer_achievements WHERE racer_id = ?", racerID).Scan(&n)
	if n != 11 {
		t.Errorf("expected still 11 achievements after re-run, got %d", n)
	}

	var leader int
	s.DB.QueryRow("SELECT COUNT(*) FROM racer_achievements WHERE racer_id = ? AND code = 'points_leader'", racerID).Scan(&leader)
	if leader != 1 {
		t.Error("expected points_leader achievement")
	}
}
