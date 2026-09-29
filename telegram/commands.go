package telegram

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

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

// execute answers a single command and returns the reply text ("" = no reply).
func (b *Bot) execute(cmd, args string, chatID int64, username, firstName string) string {
	switch cmd {
	case "/start", "/help":
		return helpText()
	case "/results":
		return b.renderLatestRace()
	case "/standings":
		return b.renderStandings()
	case "/next":
		return b.renderNextRace()
	case "/history":
		return b.renderHistory()
	case "/stats":
		return b.renderStats()
	case "/quote":
		return b.renderQuote()
	case "/addquote":
		st, err := LoadSettings(b.s)
		if err != nil || st.DefaultChatID == "" || strconv.FormatInt(chatID, 10) != st.DefaultChatID {
			return "🔒 Only the configured admin chat can add quotes."
		}
		parts := strings.SplitN(args, "|", 2)
		text := strings.TrimSpace(parts[0])
		if text == "" {
			return "✍️ Usage: /addquote <text> [| author]"
		}
		var author string
		if len(parts) == 2 {
			author = strings.TrimSpace(parts[1])
		}
		if author == "" {
			author = strings.TrimSpace(firstName)
			if author == "" && username != "" {
				author = "@" + username
			}
			if author == "" {
				author = "Telegram"
			}
		}
		if err := b.CreateQuote(text, author); err != nil {
			return "⚠️ Could not save the quote right now."
		}
		return fmt.Sprintf("✅ <b>Quote added</b>\n\n💬 <i>%s</i>\n— %s", escapeHTML(text), escapeHTML(author))
	case "/subscribe":
		if err := b.setSubscription(chatID, username, firstName, true); err != nil {
			b.warnf("subscribe failed: %v", err)
			return "⚠️ Could not subscribe right now."
		}
		return "✅ <b>Subscribed!</b> You'll now get race results and upcoming-race reminders."
	case "/unsubscribe":
		if err := b.setSubscription(chatID, username, firstName, false); err != nil {
			b.warnf("unsubscribe failed: %v", err)
			return "⚠️ Could not unsubscribe right now."
		}
		return "🔕 <b>Unsubscribed.</b> You won't receive any more pushes."
	case "/status":
		return b.renderStatus()
	default:
		return "🤖 Unknown command. Try /help"
	}
}

func helpText() string {
	return strings.Join([]string{
		"🏎️ <b>HEAT Racing Bot</b>",
		"",
		"<b>Commands</b>",
		"/results — latest race finishing order",
		"/standings — season championship table",
		"/next — upcoming race countdown",
		"/history — recent races",
		"/stats — career points leaders",
		"/quote — a paddock quote",
		"/addquote <text> [| author] — add a paddock quote (admin chat only)",
		"/subscribe — get results &amp; race reminders pushed to you",
		"/unsubscribe — stop pushes",
		"",
		"Powered by the HEAT public API.",
	}, "\n")
}

func (b *Bot) renderLatestRace() string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("summary fetch failed: %v", err)
		return "⚠️ Results are unavailable right now."
	}
	return renderRace(sum.LatestRace)
}

func renderRace(r *apiRace) string {
	if r == nil || len(r.Results) == 0 {
		return "🏁 No finalized race results yet."
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 <b>%s</b>\n", escapeHTML(r.Name))

	var meta []string
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
	sb.WriteString("\n")

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
	if sum.Season != nil && sum.Season.Name != "" {
		title = "🏆 <b>" + escapeHTML(sum.Season.Name) + "</b>"
	}
	var sb strings.Builder
	sb.WriteString(title + "\n\n")

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
	return renderNextRace(sum.NextRace)
}

func renderNextRace(nr *apiNextRace) string {
	if nr == nil {
		return "📅 No upcoming race scheduled yet."
	}
	var sb strings.Builder
	sb.WriteString("🏁 <b>Upcoming Race</b>\n")
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
	sb.WriteString("📜 <b>Recent Races</b>\n\n")
	for _, r := range races[:limit] {
		line := "• <b>" + escapeHTML(r.Name) + "</b>"
		if r.Track != "" {
			line += " — " + escapeHTML(r.Track)
		}
		if r.RaceDate != "" {
			line += " · " + escapeHTML(r.RaceDate)
		}
		sb.WriteString(line + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
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
	sb.WriteString("📊 <b>Career Points Leaders</b>\n\n")
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
	out := "💬 <i>" + escapeHTML(q.Text) + "</i>"
	if q.Author != "" {
		out += "\n— " + escapeHTML(q.Author)
	}
	return out
}

func (b *Bot) renderStatus() string {
	st, err := LoadSettings(b.s)
	state := "disabled"
	if err == nil && st.Enabled && st.BotToken != "" {
		state = "enabled"
	}
	var subs int
	b.s.DB.QueryRow("SELECT COUNT(*) FROM telegram_subscribers WHERE subscribed = 1").Scan(&subs)
	return fmt.Sprintf("🤖 <b>HEAT bot</b> is %s.\n👥 Subscribers: %d\n\nType /help to see what I can do.", state, subs)
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
	sb.WriteString("\n")
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
	return strings.TrimRight(sb.String(), "\n")
}
