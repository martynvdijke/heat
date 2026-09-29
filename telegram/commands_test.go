package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCommandRegistryConsistent(t *testing.T) {
	seen := map[string]bool{}
	for _, cmd := range commandRegistry {
		if !strings.HasPrefix(cmd.name, "/") {
			t.Errorf("command %q must start with /", cmd.name)
		}
		if seen[cmd.name] {
			t.Errorf("duplicate command %q", cmd.name)
		}
		seen[cmd.name] = true
		if cmd.usage == "" || cmd.desc == "" {
			t.Errorf("command %q missing usage/description", cmd.name)
		}
		if _, ok := categoryHeadings[cmd.category]; !ok {
			t.Errorf("command %q has unknown category %q", cmd.name, cmd.category)
		}
		if cmd.run == nil {
			t.Errorf("command %q has no handler", cmd.name)
		}
	}
}

func TestFindCommandAliases(t *testing.T) {
	if cmd, ok := findCommand("/start"); !ok || cmd.name != "/help" {
		t.Errorf("findCommand(/start) = %+v, %v; want /help", cmd, ok)
	}
	if _, ok := findCommand("/nope"); ok {
		t.Error("findCommand(/nope) should not resolve")
	}
	if !commandShowsNav("/results") || commandShowsNav("/history") {
		t.Error("showNav flags are wrong")
	}
}

func TestHelpTextCoversRegistry(t *testing.T) {
	help := helpText()
	for _, cmd := range commandRegistry {
		if !strings.Contains(help, escapeHTML(cmd.usage)) {
			t.Errorf("help missing usage %q", cmd.usage)
		}
		if !strings.Contains(help, escapeHTML(cmd.desc)) {
			t.Errorf("help missing description %q", cmd.desc)
		}
	}
	for _, heading := range categoryHeadings {
		if !strings.Contains(help, heading) {
			t.Errorf("help missing heading %q", heading)
		}
	}
	if !strings.Contains(help, "/good-bot") || !strings.Contains(help, "/addquote") {
		t.Error("help should document /good-bot and /addquote")
	}
}

func TestBotCommandsMatchRegistry(t *testing.T) {
	cmds := botCommands()
	if len(cmds) != len(commandRegistry) {
		t.Fatalf("botCommands = %d entries, registry = %d", len(cmds), len(commandRegistry))
	}
	for i, bc := range cmds {
		want := strings.TrimPrefix(commandRegistry[i].name, "/")
		if bc.Command != want {
			t.Errorf("botCommands[%d].Command = %q, want %q", i, bc.Command, want)
		}
		if len(bc.Command) > 32 || len(bc.Description) > 256 {
			t.Errorf("command %q exceeds Telegram limits: %d/%d", bc.Command, len(bc.Command), len(bc.Description))
		}
	}
}

func TestExecuteUnknownCommand(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	if out := b.execute(cmdContext{name: "/nope", chatID: 1}); !strings.Contains(out, "Unknown command") {
		t.Errorf("unknown reply: %q", out)
	}
}

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in       string
		wantName string
		wantArgs string
	}{
		{"/results", "/results", ""},
		{"/Results", "/results", ""},
		{"/addquote Hello — Alice", "/addquote", "Hello — Alice"},
		{"/help@HeatRacingBot", "/help", ""},
		{"/next@HeatRacingBot now", "/next", "now"},
	}
	for _, tc := range cases {
		name, args := splitCommand(tc.in)
		if name != tc.wantName || args != tc.wantArgs {
			t.Errorf("splitCommand(%q) = (%q, %q), want (%q, %q)", tc.in, name, args, tc.wantName, tc.wantArgs)
		}
	}
}

func TestRenderNextRaceCountdown(t *testing.T) {
	cases := []struct {
		days int
		want string
	}{
		{-2, "race day! 🚦"},
		{0, "race day! 🚦"},
		{1, "in 1 day"},
		{9, "in 9 days"},
	}
	for _, tc := range cases {
		out := renderNextRace(&apiNextRace{RaceDate: "2026-10-05", Track: "Spa", DaysRemaining: tc.days}, "Season 1")
		if !strings.Contains(out, tc.want) {
			t.Errorf("renderNextRace(days=%d) missing %q: %q", tc.days, tc.want, out)
		}
		if !strings.Contains(out, "Season 1") || !strings.Contains(out, divider) {
			t.Errorf("renderNextRace(days=%d) missing season/divider: %q", tc.days, out)
		}
	}
	if out := renderNextRace(nil, ""); !strings.Contains(out, "No upcoming race") {
		t.Errorf("nil next race: %q", out)
	}
}

func TestRenderGoodBot(t *testing.T) {
	got := renderGoodBot()
	for _, reply := range goodBotReplies {
		if reply == got {
			return
		}
	}
	t.Errorf("renderGoodBot returned %q, not one of the known replies", got)
}

func TestRenderSeason(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/telegram/summary" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(apiSummary{
			Season: &apiSeason{ID: 1, Name: "Season 2"},
			Standings: []apiStanding{
				{RacerName: "Bob", TeamName: "Blue", Points: 40, Wins: 2},
				{RacerName: "Alice", Points: 10},
			},
		})
	}))
	defer api.Close()

	b := &Bot{s: testServer(t), http: api.Client(), baseURL: api.URL, sentReminders: map[string]time.Time{}}
	out := b.renderSeason()
	if !strings.Contains(out, "Season 2") || !strings.Contains(out, "Leader: <b>Bob</b>") {
		t.Errorf("unexpected season render: %q", out)
	}
	if !strings.Contains(out, "2 racers") {
		t.Errorf("missing racer count: %q", out)
	}
}

func TestRenderQuotes(t *testing.T) {
	s := testServer(t)
	b := &Bot{s: s, sentReminders: map[string]time.Time{}}

	if out := b.renderQuotes(); !strings.Contains(out, "No quotes yet") {
		t.Errorf("empty quotes render: %q", out)
	}

	if _, err := s.DB.Exec(`INSERT INTO quotes (text, author) VALUES ('First quote', 'Alice'), ('Second quote', 'Bob')`); err != nil {
		t.Fatalf("insert: %v", err)
	}
	out := b.renderQuotes()
	if !strings.Contains(out, "#2") || !strings.Contains(out, "Second quote") || !strings.Contains(out, "#1") || !strings.Contains(out, "First quote") {
		t.Errorf("unexpected quotes render: %q", out)
	}
}

func TestFastestLapRacer(t *testing.T) {
	got := fastestLapRacer([]apiHistoryResult{{RacerName: "Alice"}, {RacerName: "Bob", FastestLap: true}})
	if got != "Bob" {
		t.Errorf("fastestLapRacer = %q, want Bob", got)
	}
	if got := fastestLapRacer(nil); got != "" {
		t.Errorf("fastestLapRacer(nil) = %q, want empty", got)
	}
}

func TestRenderArchivedRaceFastestLap(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/race-history" || r.URL.Query().Get("id") != "4" {
			t.Errorf("unexpected request: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode([]apiHistory{{
			ID: 4, Name: "Spa", Track: "Spa-Francorchamps",
			Results: []apiHistoryResult{
				{RacerName: "Alice", Position: 1, Points: 25, FastestLap: true},
				{RacerName: "Bob", Position: 2, Points: 18},
			},
		}})
	}))
	defer api.Close()

	b := &Bot{s: testServer(t), http: api.Client(), baseURL: api.URL, sentReminders: map[string]time.Time{}}
	out := b.renderArchivedRace(4)
	if !strings.Contains(out, "Spa") || !strings.Contains(out, "⚡ <b>Fastest lap:</b> Alice") {
		t.Errorf("unexpected archived render: %q", out)
	}
	if out := b.renderArchivedRace(0); out != "" {
		t.Errorf("race id 0 should render nothing, got %q", out)
	}
}
