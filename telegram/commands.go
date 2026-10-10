package telegram

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strconv"
	"strings"
)

// divider separates a message header from its body for a consistent look.
const divider = "━━━━━━━━━━━━"

// escapeHTML makes user-supplied strings safe for Telegram's HTML parse mode.
func escapeHTML(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}

func positionMedal(position int) string {
	switch position {
	case 1:
		return "🥇"
	case 2:
		return "🥈"
	case 3:
		return "🥉"
	default:
		return strconv.Itoa(position) + "."
	}
}

func rankLabel(rank int) string {
	switch rank {
	case 1:
		return "🥇"
	case 2:
		return "🥈"
	case 3:
		return "🥉"
	default:
		return strconv.Itoa(rank) + "."
	}
}

func pluralize(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// cmdContext carries everything a command handler needs about the sender.
type cmdContext struct {
	name      string
	args      string
	chatID    int64
	username  string
	firstName string
}

// Command categories, in the order /help renders them.
const (
	catRace          = "race"
	catQuotes        = "quotes"
	catNotifications = "notifications"
	catBot           = "bot"
)

var categoryOrder = []string{catRace, catQuotes, catNotifications, catBot}

var categoryHeadings = map[string]string{
	catRace:          "🏁 <b>Race</b>",
	catQuotes:        "💬 <b>Quotes</b>",
	catNotifications: "🔔 <b>Notifications</b>",
	catBot:           "🤖 <b>Bot</b>",
}

// command describes one bot command. The registry below is the single source
// of truth for dispatch, /help and Telegram's native command menu.
type command struct {
	name     string // canonical command, e.g. "/results"
	usage    string // usage line shown in /help
	desc     string // one-line description
	category string
	aliases  []string
	showNav  bool // attach the standard navigation keyboard to replies
	run      func(b *Bot, c cmdContext) string
}

// commandRegistry is the single source of truth for every bot command. It is
// assigned in init so the /help handler can reference it without creating an
// initialization cycle.
var commandRegistry []command

func init() {
	commandRegistry = []command{
		{
			name: "/results", usage: "/results", desc: "latest race finishing order",
			category: catRace, showNav: true,
			run: func(b *Bot, _ cmdContext) string { return b.renderLatestRace() },
		},
		{
			name: "/standings", usage: "/standings", desc: "championship table",
			category: catRace, showNav: true,
			run: func(b *Bot, _ cmdContext) string { return b.renderStandings() },
		},
		{
			name: "/next", usage: "/next", desc: "upcoming race countdown",
			category: catRace, showNav: true,
			run: func(b *Bot, _ cmdContext) string { return b.renderNextRace() },
		},
		{
			name: "/history", usage: "/history", desc: "recent races with fastest laps",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderHistory() },
		},
		{
			name: "/stats", usage: "/stats", desc: "career points leaders",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderStats() },
		},
		{
			name: "/season", usage: "/season", desc: "season summary and leader",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderSeason() },
		},
		{
			name: "/elo", usage: "/elo", desc: "ELO-style skill ratings",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderElo() },
		},
		{
			name: "/streaks", usage: "/streaks", desc: "longest podium streaks",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderStreaks() },
		},
		{
			name: "/h2h", usage: "/h2h <a> vs <b>", desc: "head-to-head between two racers",
			category: catRace,
			run:      func(b *Bot, c cmdContext) string { return b.renderH2H(c) },
		},
		{
			name: "/career", usage: "/career [racer]", desc: "full career record for a racer",
			category: catRace,
			run:      func(b *Bot, c cmdContext) string { return b.renderCareer(c) },
		},
		{
			name: "/trackperf", usage: "/trackperf [racer]", desc: "best tracks for a racer",
			category: catRace,
			run:      func(b *Bot, c cmdContext) string { return b.renderTrackPerf(c) },
		},
		{
			name: "/calendar", usage: "/calendar", desc: "subscribe to the season calendar",
			category: catRace,
			run:      func(b *Bot, _ cmdContext) string { return b.renderCalendar() },
		},
		{
			name: "/rsvp", usage: "/rsvp", desc: "check in for the next race day",
			category: catRace,
			run:      func(b *Bot, c cmdContext) string { return b.startRsvpCommand(c) },
		},
		{
			name: "/quote", usage: "/quote", desc: "random paddock quote",
			category: catQuotes, showNav: true,
			run: func(b *Bot, _ cmdContext) string { return b.renderQuote() },
		},
		{
			name: "/quotes", usage: "/quotes", desc: "latest quotes with IDs",
			category: catQuotes,
			run:      func(b *Bot, _ cmdContext) string { return b.renderQuotes() },
		},
		{
			name: "/addquote", usage: "/addquote <text> — <author>", desc: "add a quote to the web app",
			category: catQuotes, showNav: true,
			run: func(b *Bot, c cmdContext) string { return b.addQuoteCommand(c) },
		},
		{
			name: "/subscribe", usage: "/subscribe", desc: "get results & race reminders pushed to you",
			category: catNotifications,
			run: func(b *Bot, c cmdContext) string {
				if err := b.setSubscription(c.chatID, c.username, c.firstName, true); err != nil {
					b.warnf("subscribe failed: %v", err)
					return "⚠️ Could not subscribe right now."
				}
				return "✅ <b>Subscribed!</b> You'll now get race results and upcoming-race reminders."
			},
		},
		{
			name: "/unsubscribe", usage: "/unsubscribe", desc: "stop pushes",
			category: catNotifications,
			run: func(b *Bot, c cmdContext) string {
				if err := b.setSubscription(c.chatID, c.username, c.firstName, false); err != nil {
					b.warnf("unsubscribe failed: %v", err)
					return "⚠️ Could not unsubscribe right now."
				}
				return "🔕 <b>Unsubscribed.</b> You won't receive any more pushes."
			},
		},
		{
			name: "/help", usage: "/help", desc: "show every command",
			category: catBot, aliases: []string{"/start"},
			run: func(_ *Bot, _ cmdContext) string { return helpText() },
		},
		{
			name: "/status", usage: "/status", desc: "bot status and subscriber count",
			category: catBot,
			run:      func(b *Bot, _ cmdContext) string { return b.renderStatus() },
		},
		{
			name: "/good-bot", usage: "/good-bot", desc: "tell the bot it did well",
			category: catBot,
			run:      func(_ *Bot, _ cmdContext) string { return renderGoodBot() },
		},
		{
			name: "/cancel", usage: "/cancel", desc: "abort a guided login or quote",
			category: catBot,
			run:      func(b *Bot, c cmdContext) string { return b.cancelQuoteCommand(c) },
		},
		{
			name: "/login", usage: "/login [email]", desc: "sign in and link this chat to your racer",
			category: catBot,
			showNav:  true,
			run:      func(b *Bot, c cmdContext) string { return b.loginCommand(c) },
		},
		{
			name: "/logout", usage: "/logout", desc: "unlink this chat from your racer",
			category: catBot,
			run:      func(b *Bot, c cmdContext) string { return b.logoutCommand(c) },
		},
		{
			name: "/mystats", usage: "/mystats", desc: "your personal stats and recent form",
			category: catBot,
			showNav:  true,
			run:      func(b *Bot, c cmdContext) string { return b.renderMyStats(c) },
		},
		{
			name: "/myupgrades", usage: "/myupgrades", desc: "the upgrades you own",
			category: catBot,
			showNav:  true,
			run:      func(b *Bot, c cmdContext) string { return b.renderMyUpgrades(c) },
		},
		{
			name: "/badges", usage: "/badges [racer]", desc: "your unlocked achievements",
			category: catBot,
			run:      func(b *Bot, c cmdContext) string { return b.badgesCommand(c) },
		},
		{
			name: "/notify", usage: "/notify on|off", desc: "personal result DMs on/off",
			category: catBot,
			run:      func(b *Bot, c cmdContext) string { return b.notifyCommand(c) },
		},
	}
}

// findCommand resolves a command name or alias to its registry entry.
func findCommand(name string) (command, bool) {
	for _, cmd := range commandRegistry {
		if cmd.name == name {
			return cmd, true
		}
		for _, alias := range cmd.aliases {
			if alias == name {
				return cmd, true
			}
		}
	}
	return command{}, false
}

// commandShowsNav reports whether replies for the named command carry the
// standard navigation keyboard.
func commandShowsNav(name string) bool {
	cmd, ok := findCommand(name)
	return ok && cmd.showNav
}

// execute answers a single command and returns the reply text ("" = no reply).
func (b *Bot) execute(c cmdContext) string {
	cmd, ok := findCommand(c.name)
	if !ok {
		return "🤖 Unknown command. Try /help"
	}
	return cmd.run(b, c)
}

// helpText renders /help straight from the registry, so it can never drift
// from the commands that are actually dispatched.
func helpText() string {
	var sb strings.Builder
	sb.WriteString("🏎️ <b>HEAT Racing Bot</b>\n")
	sb.WriteString("Everything I can do — tap a button below or type a command.\n")
	for _, cat := range categoryOrder {
		sb.WriteString("\n" + categoryHeadings[cat] + "\n")
		for _, cmd := range commandRegistry {
			if cmd.category != cat {
				continue
			}
			fmt.Fprintf(&sb, "%s — %s\n", escapeHTML(cmd.usage), escapeHTML(cmd.desc))
		}
	}
	sb.WriteString("\nPowered by the HEAT public API.")
	return sb.String()
}

// seasonName returns the current season's name, if any.
func seasonName(sum apiSummary) string {
	if sum.Season == nil {
		return ""
	}
	return sum.Season.Name
}

func (b *Bot) renderLatestRace() string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("summary fetch failed: %v", err)
		return "⚠️ Results are unavailable right now."
	}
	return renderRace(sum.LatestRace, seasonName(sum))
}

func renderRace(r *apiRace, season string) string {
	if r == nil || len(r.Results) == 0 {
		return "🏁 No finalized race results yet."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 <b>%s</b>\n", escapeHTML(r.Name))

	var meta []string
	if season != "" {
		meta = append(meta, escapeHTML(season))
	}
	if r.Round > 0 {
		meta = append(meta, fmt.Sprintf("Round %d", r.Round))
	}
	if r.Track != "" {
		meta = append(meta, escapeHTML(r.Track))
	}
	if r.Country != "" {
		meta = append(meta, escapeHTML(r.Country))
	}
	if r.TotalLaps > 0 {
		meta = append(meta, fmt.Sprintf("%d laps", r.TotalLaps))
	}
	if r.RaceDate != "" {
		meta = append(meta, escapeHTML(r.RaceDate))
	}
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "<i>%s</i>\n", strings.Join(meta, " · "))
	}
	sb.WriteString(divider + "\n")

	for _, res := range r.Results {
		line := fmt.Sprintf("%s <b>%s</b>", positionMedal(res.Position), escapeHTML(res.RacerName))
		if res.Team != "" {
			line += " (" + escapeHTML(res.Team) + ")"
		}
		if res.Points > 0 {
			line += fmt.Sprintf(" — %d pts", res.Points)
		}
		if res.Spins > 0 {
			line += fmt.Sprintf(" · %d %s", res.Spins, pluralize(res.Spins, "spin"))
		}
		if res.Overheated > 0 {
			line += fmt.Sprintf(" · %d overheated", res.Overheated)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (b *Bot) renderStandings() string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("standings fetch failed: %v", err)
		return "⚠️ Standings are unavailable right now."
	}
	if len(sum.Standings) == 0 {
		return "🏆 No championship standings yet."
	}

	title := "🏆 <b>Championship Standings</b>"
	if name := seasonName(sum); name != "" {
		title = "🏆 <b>" + escapeHTML(name) + " · Championship Standings</b>"
	}
	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(divider + "\n")

	for i, s := range sum.Standings {
		line := fmt.Sprintf("%s <b>%s</b>", rankLabel(i+1), escapeHTML(s.RacerName))
		if s.TeamName != "" {
			line += " (" + escapeHTML(s.TeamName) + ")"
		}
		line += fmt.Sprintf(" — %d pts", s.Points)
		if s.Wins > 0 {
			line += fmt.Sprintf(" · %d wins", s.Wins)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (b *Bot) renderNextRace() string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("next race fetch failed: %v", err)
		return "⚠️ Could not fetch the next race."
	}
	return renderNextRace(sum.NextRace, seasonName(sum))
}

func renderNextRace(nr *apiNextRace, season string) string {
	if nr == nil {
		return "📅 No upcoming race scheduled yet."
	}
	var sb strings.Builder
	title := "📅 <b>Upcoming Race</b>"
	if season != "" {
		title += " · " + escapeHTML(season)
	}
	sb.WriteString(title + "\n")
	sb.WriteString(divider + "\n")

	var where []string
	if nr.Track != "" {
		where = append(where, escapeHTML(nr.Track))
	}
	if nr.Country != "" {
		where = append(where, escapeHTML(nr.Country))
	}
	if len(where) > 0 {
		fmt.Fprintf(&sb, "📍 %s\n", strings.Join(where, ", "))
	}
	fmt.Fprintf(&sb, "📅 %s — %s\n", escapeHTML(nr.RaceDate), countdownLabel(nr.DaysRemaining))
	if nr.TotalLaps > 0 {
		fmt.Fprintf(&sb, "🔁 %d laps\n", nr.TotalLaps)
	}
	return strings.TrimRight(sb.String(), "\n")
}

func countdownLabel(days int) string {
	switch {
	case days <= 0:
		return "race day! 🚦"
	case days == 1:
		return "in 1 day"
	default:
		return fmt.Sprintf("in %d days", days)
	}
}

func (b *Bot) renderHistory() string {
	var races []apiHistory
	if err := b.apiGet("/api/race-history", &races); err != nil {
		b.warnf("history fetch failed: %v", err)
		return "⚠️ Race history is unavailable right now."
	}
	if len(races) == 0 {
		return "📜 No races in the archive yet."
	}
	limit := 10
	if len(races) < limit {
		limit = len(races)
	}
	var sb strings.Builder
	sb.WriteString("📜 <b>Recent Races</b>\n")
	sb.WriteString(divider + "\n")
	for _, r := range races[:limit] {
		line := "• <b>" + escapeHTML(r.Name) + "</b>"
		if r.Track != "" {
			line += " — " + escapeHTML(r.Track)
		}
		if r.RaceDate != "" {
			line += " · " + escapeHTML(r.RaceDate)
		}
		if name := fastestLapRacer(r.Results); name != "" {
			line += " · ⚡ " + escapeHTML(name)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// fastestLapRacer returns the racer who set the fastest lap, if the payload
// records one.
func fastestLapRacer(results []apiHistoryResult) string {
	for _, res := range results {
		if res.FastestLap {
			return res.RacerName
		}
	}
	return ""
}

func (b *Bot) renderStats() string {
	var racers []apiRacer
	if err := b.apiGet("/api/racers", &racers); err != nil {
		b.warnf("racers fetch failed: %v", err)
		return "⚠️ Stats are unavailable right now."
	}
	nameByID := make(map[int]string, len(racers))
	for _, r := range racers {
		nameByID[r.ID] = r.Name
	}

	var stats []apiRacerStats
	if err := b.apiGet("/api/racer-stats?source=legacy", &stats); err != nil {
		b.warnf("stats fetch failed: %v", err)
		return "⚠️ Stats are unavailable right now."
	}
	sort.SliceStable(stats, func(i, j int) bool {
		if stats[i].Points != stats[j].Points {
			return stats[i].Points > stats[j].Points
		}
		return stats[i].Wins > stats[j].Wins
	})

	var sb strings.Builder
	sb.WriteString("📊 <b>Career Points Leaders</b>\n")
	sb.WriteString(divider + "\n")
	rank := 0
	for _, s := range stats {
		name := nameByID[s.RacerID]
		if name == "" || s.Races == 0 {
			continue
		}
		rank++
		if rank > 10 {
			break
		}
		line := fmt.Sprintf("%s <b>%s</b> — %d pts", rankLabel(rank), escapeHTML(name), s.Points)
		if s.Wins > 0 {
			line += fmt.Sprintf(" · %d wins", s.Wins)
		}
		if s.Spins > 0 {
			line += fmt.Sprintf(" · %d %s", s.Spins, pluralize(s.Spins, "spin"))
		}
		if s.Overheated > 0 {
			line += fmt.Sprintf(" · %d overheated", s.Overheated)
		}
		sb.WriteString(line + "\n")
	}
	if rank == 0 {
		return "📊 No career stats recorded yet."
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (b *Bot) renderQuote() string {
	var q apiQuote
	if err := b.apiGet("/api/quote/random", &q); err != nil || q.Text == "" {
		b.warnf("quote fetch failed: %v", err)
		return "💬 The paddock is silent."
	}
	out := "💬 <b>Paddock Quote</b>\n" + divider + "\n<i>" + escapeHTML(q.Text) + "</i>"
	if q.Author != "" {
		out += "\n— " + escapeHTML(q.Author)
	}
	return out
}

// renderQuotes lists the most recently added quotes with their IDs.
func (b *Bot) renderQuotes() string {
	rows, err := b.s.DB.Query("SELECT id, text, author FROM quotes ORDER BY id DESC LIMIT 5")
	if err != nil {
		b.warnf("quotes fetch failed: %v", err)
		return "⚠️ Quotes are unavailable right now."
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString("💬 <b>Latest Quotes</b>\n")
	sb.WriteString(divider + "\n")
	count := 0
	for rows.Next() {
		var id int
		var text, author string
		if err := rows.Scan(&id, &text, &author); err != nil {
			continue
		}
		count++
		fmt.Fprintf(&sb, "#%d — <i>%s</i>\n", id, escapeHTML(text))
		if author != "" {
			fmt.Fprintf(&sb, "— %s\n", escapeHTML(author))
		}
	}
	if count == 0 {
		return "💬 No quotes yet — add one with /addquote."
	}
	return strings.TrimRight(sb.String(), "\n")
}

func (b *Bot) renderSeason() string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("season fetch failed: %v", err)
		return "⚠️ Season info is unavailable right now."
	}
	if seasonName(sum) == "" {
		return "🏎️ No season data yet."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🏎️ <b>%s</b>\n", escapeHTML(sum.Season.Name))
	sb.WriteString(divider + "\n")

	standings := append([]apiStanding(nil), sum.Standings...)
	sort.SliceStable(standings, func(i, j int) bool {
		if standings[i].Points != standings[j].Points {
			return standings[i].Points > standings[j].Points
		}
		return standings[i].Wins > standings[j].Wins
	})
	if len(standings) == 0 {
		sb.WriteString("No championship standings yet.")
		return sb.String()
	}

	leader := standings[0]
	line := "🏆 Leader: <b>" + escapeHTML(leader.RacerName) + "</b>"
	if leader.TeamName != "" {
		line += " (" + escapeHTML(leader.TeamName) + ")"
	}
	line += fmt.Sprintf(" — %d pts", leader.Points)
	if leader.Wins > 0 {
		line += fmt.Sprintf(" · %d wins", leader.Wins)
	}
	sb.WriteString(line + "\n")
	fmt.Fprintf(&sb, "👥 %d racers in the championship", len(standings))
	return sb.String()
}

var goodBotReplies = []string{
	"🚗💨 Thanks! I try my best.",
	"🤖 Aww, you're too kind.",
	"🏁 Good human! (Don't tell the other bots.)",
	"⚡ Beep boop — appreciation logged.",
	"🥇 You're a legend.",
	"🏎️ Right back at you!",
}

func renderGoodBot() string {
	return goodBotReplies[rand.IntN(len(goodBotReplies))]
}

func (b *Bot) renderStatus() string {
	st, err := LoadSettings(b.s)
	state := "disabled"
	if err == nil && st.Enabled && st.BotToken != "" {
		state = "enabled"
	}
	var subs int
	b.s.DB.QueryRow("SELECT COUNT(*) FROM telegram_subscribers WHERE subscribed = 1").Scan(&subs)
	return fmt.Sprintf("🤖 <b>HEAT bot</b>\n%s\nStatus: %s\n👥 Subscribers: %d\n\nType /help to see what I can do.", divider, state, subs)
}

// renderArchivedRace builds a results message from the race archive, used by
// the push fired when a race is saved.
func (b *Bot) renderArchivedRace(raceID int) string {
	if raceID <= 0 {
		return ""
	}
	var races []apiHistory
	if err := b.apiGet(fmt.Sprintf("/api/race-history?id=%d", raceID), &races); err != nil || len(races) == 0 {
		b.warnf("archived race %d fetch failed: %v", raceID, err)
		return ""
	}
	r := races[0]
	if len(r.Results) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 <b>%s</b>\n", escapeHTML(r.Name))
	var meta []string
	if r.Track != "" {
		meta = append(meta, escapeHTML(r.Track))
	}
	if r.TotalLaps > 0 {
		meta = append(meta, fmt.Sprintf("%d laps", r.TotalLaps))
	}
	if r.RaceDate != "" {
		meta = append(meta, escapeHTML(r.RaceDate))
	}
	if len(meta) > 0 {
		fmt.Fprintf(&sb, "<i>%s</i>\n", strings.Join(meta, " · "))
	}
	sb.WriteString(divider + "\n")
	for _, res := range r.Results {
		line := fmt.Sprintf("%s <b>%s</b>", positionMedal(res.Position), escapeHTML(res.RacerName))
		if res.Points > 0 {
			line += fmt.Sprintf(" — %d pts", res.Points)
		}
		if res.FastestLap {
			line += " ⚡"
		}
		sb.WriteString(line + "\n")
	}
	if name := fastestLapRacer(r.Results); name != "" {
		fmt.Fprintf(&sb, "\n⚡ <b>Fastest lap:</b> %s", escapeHTML(name))
	}
	return strings.TrimRight(sb.String(), "\n")
}
