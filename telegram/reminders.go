package telegram

import (
	"strconv"
	"strings"
	"time"
)

const reminderCheckInterval = 10 * time.Minute

// reminderLoop periodically checks the upcoming race and fires a reminder on
// the configured lead days (e.g. 7 and 1 day before, plus race day).
func (b *Bot) reminderLoop() {
	ticker := time.NewTicker(reminderCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		b.checkReminders()
	}
}

func (b *Bot) checkReminders() {
	st, err := LoadSettings(b.s)
	if err != nil || !st.Enabled || st.BotToken == "" || !st.NotifyNextRace {
		return
	}

	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err != nil {
		b.warnf("reminder summary fetch failed: %v", err)
		return
	}
	nr := sum.NextRace
	if nr == nil {
		return
	}
	if !reminderDueAt(nr.RaceDate, nr.DaysRemaining, st.ReminderHour, st.ReminderDays, time.Now()) {
		return
	}

	key := nr.RaceDate + ":" + strconv.Itoa(nr.DaysRemaining)
	b.mu.Lock()
	if _, done := b.sentReminders[key]; done {
		b.mu.Unlock()
		return
	}
	b.sentReminders[key] = time.Now()
	b.mu.Unlock()

	b.logf("sending upcoming-race reminder for %s (T-%d)", nr.RaceDate, nr.DaysRemaining)
	b.broadcast(renderNextRace(nr, seasonName(sum)))
}

// reminderDueAt reports whether a reminder should fire right now: the race is
// today or in the future, its day count is in the configured lead set, and the
// local clock has reached reminder_hour.
func reminderDueAt(raceDate string, days, hour int, daysCSV string, now time.Time) bool {
	if !allowsDay(daysCSV, days) {
		return false
	}
	if now.Hour() < hour {
		return false
	}
	if t, err := time.Parse("2006-01-02", raceDate); err == nil {
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
		if t.Before(today) {
			return false
		}
	}
	return true
}

func allowsDay(daysCSV string, day int) bool {
	for _, part := range strings.Split(daysCSV, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if v, err := strconv.Atoi(part); err == nil && v == day {
			return true
		}
	}
	return false
}
