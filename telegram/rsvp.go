package telegram

import (
	"fmt"
	"strconv"
	"strings"

	tgmodels "github.com/go-telegram/bot/models"
)

const callbackRsvpPrefix = "rsvp:"

// rsvpStatuses are the three answers the race-day poll accepts, in display order.
var rsvpStatuses = []struct {
	key   string
	emoji string
	label string
}{
	{"in", "✅", "In"},
	{"out", "❌", "Can't"},
	{"maybe", "🤔", "Maybe"},
}

func rsvpStatusLabel(key string) string {
	for _, s := range rsvpStatuses {
		if s.key == key {
			return s.emoji + " " + s.label
		}
	}
	return key
}

// rsvpKeyboard builds the ✅/❌/🤔 poll for a race date.
func (b *Bot) rsvpKeyboard(raceDate string) *tgmodels.InlineKeyboardMarkup {
	row := make([]tgmodels.InlineKeyboardButton, 0, len(rsvpStatuses))
	for _, s := range rsvpStatuses {
		row = append(row, tgmodels.InlineKeyboardButton{
			Text:         s.emoji + " " + s.label,
			CallbackData: callbackRsvpPrefix + raceDate + ":" + s.key,
		})
	}
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{row}}
}

// startRsvpCommand opens the race-day check-in for the next race. On success it
// sends the poll itself and returns "" so the dispatcher adds nothing.
func (b *Bot) startRsvpCommand(c cmdContext) string {
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("rsvp summary fetch failed: %v", err)
		return "⚠️ Could not fetch the next race right now."
	}
	nr := sum.NextRace
	if nr == nil || nr.RaceDate == "" {
		return "📅 No upcoming race to check in for yet."
	}
	b.sendWithKeyboard(c.chatID, rsvpMessage(nr, b.rsvpTally(nr.RaceDate)), b.rsvpKeyboard(nr.RaceDate))
	return ""
}

// rsvpMessage renders the poll header plus the current tally.
func rsvpMessage(nr *apiNextRace, tally string) string {
	var sb strings.Builder
	sb.WriteString("🏁 <b>Race-day check-in</b>")
	if nr.Track != "" {
		sb.WriteString(" — " + escapeHTML(nr.Track))
	}
	sb.WriteString("\n")
	sb.WriteString(divider + "\n")
	fmt.Fprintf(&sb, "📅 %s — %s\n\n", escapeHTML(nr.RaceDate), countdownLabel(nr.DaysRemaining))
	sb.WriteString(tally)
	return sb.String()
}

// handleRsvpCallback records a tap and returns the refreshed poll text along
// with the race date to re-attach the keyboard.
func (b *Bot) handleRsvpCallback(c cmdContext, data string) (string, string, bool) {
	rest := strings.TrimPrefix(data, callbackRsvpPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	raceDate, status := parts[0], parts[1]
	if !isRsvpStatus(status) {
		return "", "", false
	}

	racerID := 0
	if id, _, ok := b.racerForChat(c.chatID); ok {
		racerID = id
	}
	if err := b.setRsvp(raceDate, c.chatID, racerID, status); err != nil {
		b.warnf("rsvp save failed: %v", err)
		return "⚠️ Could not save your answer right now.", raceDate, true
	}

	tally := b.rsvpTally(raceDate)
	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 <b>Race-day check-in</b> — %s\n", escapeHTML(raceDate))
	sb.WriteString(divider + "\n\n")
	sb.WriteString(tally)
	return sb.String(), raceDate, true
}

func isRsvpStatus(status string) bool {
	for _, s := range rsvpStatuses {
		if s.key == status {
			return true
		}
	}
	return false
}

// setRsvp upserts a chat's answer for a race date.
func (b *Bot) setRsvp(raceDate string, chatID int64, racerID int, status string) error {
	_, err := b.s.DB.Exec(`
		INSERT INTO telegram_rsvp (race_date, chat_id, racer_id, status, updated_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(race_date, chat_id) DO UPDATE SET
			racer_id = excluded.racer_id,
			status = excluded.status,
			updated_at = excluded.updated_at`,
		raceDate, strconv.FormatInt(chatID, 10), racerID, status)
	return err
}

// rsvpTally lists who is in, out, and maybe for a race date.
func (b *Bot) rsvpTally(raceDate string) string {
	rows, err := b.s.DB.Query(`
		SELECT t.status, COALESCE(r.name, ''), t.chat_id
		FROM telegram_rsvp t
		LEFT JOIN racers r ON r.id = t.racer_id
		WHERE t.race_date = ?
		ORDER BY t.updated_at`, raceDate)
	if err != nil {
		b.warnf("rsvp tally failed: %v", err)
		return "No answers yet — be the first to reply!"
	}
	defer rows.Close()

	names := map[string][]string{}
	count := 0
	for rows.Next() {
		var status, name, chatID string
		if rows.Scan(&status, &name, &chatID) != nil {
			continue
		}
		count++
		if name == "" {
			name = "Chat " + chatID
		}
		names[status] = append(names[status], name)
	}
	if count == 0 {
		return "No answers yet — be the first to reply!"
	}

	var sb strings.Builder
	for _, s := range rsvpStatuses {
		list := names[s.key]
		fmt.Fprintf(&sb, "%s %s: %d", s.emoji, s.label, len(list))
		if len(list) > 0 {
			fmt.Fprintf(&sb, " — %s", escapeHTML(strings.Join(list, ", ")))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}
