package telegram

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/ent"
)

func testServer(t *testing.T) *app.Server {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_fk=1")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	_, err = db.Exec(`
		CREATE TABLE telegram_settings (
			id INTEGER PRIMARY KEY,
			bot_token TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 0,
			default_chat_id TEXT NOT NULL DEFAULT '',
			notify_results INTEGER NOT NULL DEFAULT 1,
			notify_next_race INTEGER NOT NULL DEFAULT 1,
			reminder_days TEXT NOT NULL DEFAULT '7,1',
			reminder_hour INTEGER NOT NULL DEFAULT 18
		);
		CREATE TABLE telegram_subscribers (
			chat_id TEXT PRIMARY KEY,
			username TEXT NOT NULL DEFAULT '',
			first_name TEXT NOT NULL DEFAULT '',
			subscribed INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)`)
	if err != nil {
		t.Fatalf("schema: %v", err)
	}
	drv := entsql.OpenDB(dialect.SQLite, db)
	client := ent.NewClient(ent.Driver(drv))
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("ent schema: %v", err)
	}
	return &app.Server{DB: db, Ent: client}
}

func TestLoadSettings(t *testing.T) {
	s := testServer(t)
	if _, err := s.DB.Exec(`INSERT INTO telegram_settings (id, bot_token, enabled, default_chat_id, notify_results, notify_next_race, reminder_days, reminder_hour) VALUES (1, 'tok', 1, '-100', 1, 0, '7,3,0', 20)`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	st, err := LoadSettings(s)
	if err != nil {
		t.Fatalf("LoadSettings: %v", err)
	}
	if st.BotToken != "tok" || !st.Enabled || st.DefaultChatID != "-100" {
		t.Errorf("unexpected settings: %+v", st)
	}
	if !st.NotifyResults || st.NotifyNextRace {
		t.Errorf("toggle mismatch: %+v", st)
	}
	if st.ReminderDays != "7,3,0" || st.ReminderHour != 20 {
		t.Errorf("reminder mismatch: %+v", st)
	}
}

func TestReminderDueAt(t *testing.T) {
	now := time.Date(2026, 9, 28, 20, 0, 0, 0, time.Local)
	cases := []struct {
		name     string
		raceDate string
		days     int
		hour     int
		daysCSV  string
		want     bool
	}{
		{"seven days out after hour", "2026-10-05", 7, 18, "7,1", true},
		{"not a lead day", "2026-10-02", 4, 18, "7,1", false},
		{"before reminder hour", "2026-09-29", 1, 21, "7,1", false},
		{"race day", "2026-09-28", 0, 18, "7,1,0", true},
		{"past race ignored", "2026-09-20", 0, 18, "7,1,0", false},
		{"empty config", "2026-10-05", 7, 18, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := reminderDueAt(tc.raceDate, tc.days, tc.hour, tc.daysCSV, now); got != tc.want {
				t.Errorf("reminderDueAt = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestEscapeHTML(t *testing.T) {
	got := escapeHTML("A & B <script>")
	want := "A &amp; B &lt;script&gt;"
	if got != want {
		t.Errorf("escapeHTML = %q, want %q", got, want)
	}
}

func TestRenderRace(t *testing.T) {
	out := renderRace(&apiRace{
		Name:    "Monaco GP",
		Track:   "Monte Carlo",
		Round:   3,
		Results: []apiRaceResult{{RacerName: "Alice", Team: "Red", Position: 1, Points: 25}, {RacerName: "Bob", Position: 2, Points: 18}},
	}, "Season 1")
	if !strings.Contains(out, "Monaco GP") || !strings.Contains(out, "Alice") || !strings.Contains(out, "25 pts") {
		t.Errorf("unexpected render: %q", out)
	}
	if !strings.Contains(out, "🥇") || !strings.Contains(out, "🥈") {
		t.Errorf("missing medals: %q", out)
	}
	if !strings.Contains(out, "Season 1") || !strings.Contains(out, "Round 3") || !strings.Contains(out, divider) {
		t.Errorf("missing season/round/divider context: %q", out)
	}
}

func TestRenderRaceIncludesSpinsOverheated(t *testing.T) {
	out := renderRace(&apiRace{
		Name:    "Test GP",
		Results: []apiRaceResult{{RacerName: "Alice", Position: 1, Points: 25, Spins: 2, Overheated: 1}},
	}, "")
	if !strings.Contains(out, "2 spins") {
		t.Errorf("expected '2 spins' in %q", out)
	}
	if !strings.Contains(out, "1 overheated") {
		t.Errorf("expected '1 overheated' in %q", out)
	}
	out2 := renderRace(&apiRace{
		Name:    "Test GP",
		Results: []apiRaceResult{{RacerName: "Bob", Position: 2, Points: 18, Spins: 1}},
	}, "")
	if !strings.Contains(out2, "1 spin") {
		t.Errorf("expected '1 spin' in %q", out2)
	}
	if strings.Contains(out2, "1 spins") {
		t.Errorf("should not contain '1 spins' in %q", out2)
	}
}

func TestExecuteResultsUsesPublicAPI(t *testing.T) {
	var gotPath string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(apiSummary{
			LatestRace: &apiRace{Name: "Spa", Results: []apiRaceResult{{RacerName: "Carol", Position: 1, Points: 25}}},
		})
	}))
	defer api.Close()

	b := &Bot{s: testServer(t), http: api.Client(), baseURL: api.URL, sentReminders: map[string]time.Time{}}
	out := b.execute(cmdContext{name: "/results", chatID: 1})
	if gotPath != "/api/telegram/summary" {
		t.Errorf("public API path = %q, want /api/telegram/summary", gotPath)
	}
	if !strings.Contains(out, "Carol") {
		t.Errorf("unexpected reply: %q", out)
	}
}

func TestSubscribeUnsubscribe(t *testing.T) {
	s := testServer(t)
	b := &Bot{s: s, sentReminders: map[string]time.Time{}}

	c := cmdContext{name: "/subscribe", chatID: -42, username: "dave", firstName: "Dave"}
	if out := b.execute(c); !strings.Contains(out, "Subscribed") {
		t.Fatalf("subscribe reply: %q", out)
	}
	if ids := b.subscriberChatIDs(); len(ids) != 1 || ids[0] != "-42" {
		t.Fatalf("subscriberChatIDs = %v", ids)
	}
	c.name = "/unsubscribe"
	if out := b.execute(c); !strings.Contains(out, "Unsubscribed") {
		t.Fatalf("unsubscribe reply: %q", out)
	}
	if ids := b.subscriberChatIDs(); len(ids) != 0 {
		t.Fatalf("expected no subscribers, got %v", ids)
	}
}

func TestSendTest(t *testing.T) {
	var path string
	tg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		if strings.HasSuffix(r.URL.Path, "/getMe") {
			_, _ = w.Write([]byte(`{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"Test"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":1,"date":0,"chat":{"id":1,"type":"private"}}}`))
	}))
	defer tg.Close()

	b := &Bot{http: tg.Client(), apiServerURL: tg.URL, sentReminders: map[string]time.Time{}}
	if err := b.SendTest("token", "123", "hi"); err != nil {
		t.Fatalf("SendTest: %v", err)
	}
	if !strings.HasSuffix(path, "/bottoken/sendMessage") {
		t.Errorf("unexpected API path %q", path)
	}
}
