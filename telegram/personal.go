package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

// personalNotifyEnabled reports whether a chat has opted in to personal result
// DMs. Opt-in is off by default, so a missing row means disabled.
func (b *Bot) personalNotifyEnabled(chatID int64) bool {
	var n int
	err := b.s.DB.QueryRow(
		"SELECT COUNT(*) FROM telegram_notify_prefs WHERE chat_id = ? AND personal = 1",
		strconv.FormatInt(chatID, 10)).Scan(&n)
	return err == nil && n > 0
}

// setPersonalNotify records a chat's personal-DM opt-in.
func (b *Bot) setPersonalNotify(chatID int64, enabled bool) error {
	_, err := b.s.DB.Exec(`
		INSERT INTO telegram_notify_prefs (chat_id, personal, updated_at)
		VALUES (?, ?, datetime('now'))
		ON CONFLICT(chat_id) DO UPDATE SET
			personal = excluded.personal,
			updated_at = excluded.updated_at`,
		strconv.FormatInt(chatID, 10), boolToInt(enabled))
	return err
}

// notifyCommand handles /notify on|off, the opt-in switch for personal DMs.
func (b *Bot) notifyCommand(c cmdContext) string {
	if _, _, ok := b.racerForChat(c.chatID); !ok {
		return "🔐 Link your racer first with /login, then /notify on to get your results DMed after each race."
	}
	switch strings.ToLower(strings.TrimSpace(c.args)) {
	case "on":
		if err := b.setPersonalNotify(c.chatID, true); err != nil {
			b.warnf("notify on failed: %v", err)
			return "⚠️ Could not update your notification settings right now."
		}
		return "✅ <b>Personal DMs on.</b> I'll message you your result after each finalized race."
	case "off":
		if err := b.setPersonalNotify(c.chatID, false); err != nil {
			b.warnf("notify off failed: %v", err)
			return "⚠️ Could not update your notification settings right now."
		}
		return "🔕 <b>Personal DMs off.</b> You won't get individual result messages."
	default:
		state := "off"
		if b.personalNotifyEnabled(c.chatID) {
			state = "on"
		}
		return fmt.Sprintf("🔔 Personal result DMs are <b>%s</b>.\n\nUse <code>/notify on</code> or <code>/notify off</code>.", state)
	}
}

// sendPersonalResults fans out each linked racer's result from the most recent
// race to the chats that opted in via /notify on.
func (b *Bot) sendPersonalResults() {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil || sum.LatestRace == nil || len(sum.LatestRace.Results) == 0 {
		return
	}
	race := sum.LatestRace

	rows, err := b.s.DB.Query(`
		SELECT l.chat_id, l.racer_id, COALESCE(r.name, '')
		FROM telegram_links l
		JOIN racers r ON r.id = l.racer_id
		JOIN telegram_notify_prefs p ON p.chat_id = l.chat_id AND p.personal = 1`)
	if err != nil {
		b.warnf("personal results lookup failed: %v", err)
		return
	}
	type target struct {
		chatID int64
		name   string
	}
	var targets []target
	for rows.Next() {
		var chatID, name string
		var racerID int
		if err := rows.Scan(&chatID, &racerID, &name); err != nil {
			continue
		}
		id, err := strconv.ParseInt(chatID, 10, 64)
		if err != nil {
			continue
		}
		targets = append(targets, target{chatID: id, name: name})
	}
	rows.Close()

	for _, t := range targets {
		msg := personalResultMessage(race, t.name, sum.Standings, seasonName(sum))
		if msg != "" {
			b.send(t.chatID, msg)
		}
	}
}

// personalResultMessage renders one racer's result line for a race. It returns
// "" when the racer did not take part.
func personalResultMessage(race *apiRace, racerName string, standings []apiStanding, season string) string {
	var res *apiRaceResult
	for i := range race.Results {
		if strings.EqualFold(race.Results[i].RacerName, racerName) {
			res = &race.Results[i]
			break
		}
	}
	if res == nil {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("🏁 <b>Your result</b>")
	if race.Name != "" {
		sb.WriteString(" — " + escapeHTML(race.Name))
	}
	sb.WriteString("\n")
	sb.WriteString(divider + "\n")
	fmt.Fprintf(&sb, "%s <b>%s</b> — %d %s\n", positionMedal(res.Position), escapeHTML(racerName), res.Points, pluralize(res.Points, "pt"))

	if rank := standingsRank(standings, racerName); rank > 0 {
		if season != "" {
			sb.WriteString("<i>" + escapeHTML(season) + "</i>\n")
		}
		fmt.Fprintf(&sb, "📊 Season: %s", rankLabel(rank))
		if rank <= len(standings) {
			fmt.Fprintf(&sb, " · %d pts", standings[rank-1].Points)
		}
	}
	return strings.TrimRight(sb.String(), "\n")
}

// standingsRank returns the 1-based championship position for a racer name.
func standingsRank(standings []apiStanding, racerName string) int {
	for i, s := range standings {
		if strings.EqualFold(s.RacerName, racerName) {
			return i + 1
		}
	}
	return 0
}
