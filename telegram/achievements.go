package telegram

import (
	"fmt"
	"strconv"
	"strings"
)

// achievementDef describes one badge in the fixed catalog the bot awards.
type achievementDef struct {
	code  string
	emoji string
	label string
}

// achievementCatalog is the ordered set of badges the bot knows about. Order is
// used when /badges renders the full locked/unlocked list.
var achievementCatalog = []achievementDef{
	{"first_win", "🥇", "First Win"},
	{"first_podium", "🏅", "First Podium"},
	{"fastest_lap", "⚡", "Fastest Lap"},
	{"podium_streak", "🔥", "Podium Streak"},
	{"five_wins", "🏆", "Five-Time Winner"},
	{"podium_machine", "🥂", "Podium Machine"},
	{"clean_sweep", "🧹", "Clean Sweep"},
	{"centurion", "💯", "Century Club"},
	{"veteran", "🎖️", "Veteran"},
	{"elo_1600", "📈", "Rated 1600"},
	{"elo_1700", "🚀", "Elite Rating"},
	{"points_leader", "👑", "Championship Leader"},
}

var achievementByCode = func() map[string]achievementDef {
	m := make(map[string]achievementDef, len(achievementCatalog))
	for _, a := range achievementCatalog {
		m[a.code] = a
	}
	return m
}()

// achievementFacts captures everything needed to evaluate one racer's badges.
type achievementFacts struct {
	races       int
	wins        int
	podiums     int
	points      int
	fastestLaps int
	bestStreak  int
	elo         float64
	isLeader    bool
	cleanSweep  bool
}

// evaluateAchievements returns the badges a racer's facts satisfy.
func evaluateAchievements(f achievementFacts) []achievementDef {
	var out []achievementDef
	add := func(code string) {
		if a, ok := achievementByCode[code]; ok {
			out = append(out, a)
		}
	}
	if f.wins >= 1 {
		add("first_win")
	}
	if f.podiums >= 1 {
		add("first_podium")
	}
	if f.fastestLaps >= 1 {
		add("fastest_lap")
	}
	if f.bestStreak >= 3 {
		add("podium_streak")
	}
	if f.wins >= 5 {
		add("five_wins")
	}
	if f.podiums >= 10 {
		add("podium_machine")
	}
	if f.cleanSweep {
		add("clean_sweep")
	}
	if f.points >= 100 {
		add("centurion")
	}
	if f.races >= 25 {
		add("veteran")
	}
	if f.elo >= 1600 {
		add("elo_1600")
	}
	if f.elo >= 1700 {
		add("elo_1700")
	}
	if f.isLeader {
		add("points_leader")
	}
	return out
}

// checkAchievements evaluates every racer's badges against the current data and
// awards any newly unlocked ones, DMing opted-in linked chats. It is safe to
// call repeatedly: awards are de-duplicated in the database.
func (b *Bot) checkAchievements() {
	st, err := LoadSettings(b.s)
	if err != nil || !st.Enabled || st.BotToken == "" {
		return
	}

	var racers []apiRacer
	if err := b.apiGet("/api/racers", &racers); err != nil || len(racers) == 0 {
		return
	}

	eloByID := map[int]float64{}
	var elos []apiELORating
	if err := b.apiGet("/api/stats/elo", &elos); err == nil {
		for _, e := range elos {
			eloByID[e.RacerID] = e.Rating
		}
	}

	bestStreakByName := map[string]int{}
	var streaks []apiStreakInfo
	if err := b.apiGet("/api/stats/streaks", &streaks); err == nil {
		for _, s := range streaks {
			if s.BestValue > bestStreakByName[s.RacerName] {
				bestStreakByName[s.RacerName] = s.BestValue
			}
		}
	}

	leader := ""
	var sum apiSummary
	if err := b.apiGet("/api/telegram/summary", &sum); err == nil && len(sum.Standings) > 0 {
		leader = sum.Standings[0].RacerName
	}

	sweep := b.cleanSweepNames()

	for _, racer := range racers {
		var env apiPersonalEnvelope
		if err := b.apiGet(fmt.Sprintf("/api/racer-stats?id=%d", racer.ID), &env); err != nil {
			continue
		}
		s := env.Stats
		facts := achievementFacts{
			races:       s.Races,
			wins:        s.Wins,
			podiums:     s.Gold + s.Silver + s.Bronze,
			points:      s.Points,
			fastestLaps: s.FastestLaps,
			bestStreak:  bestStreakByName[racer.Name],
			elo:         eloByID[racer.ID],
			isLeader:    leader != "" && strings.EqualFold(leader, racer.Name),
			cleanSweep:  sweep[strings.ToLower(racer.Name)],
		}
		for _, a := range evaluateAchievements(facts) {
			if !b.awardAchievement(racer.ID, a) {
				continue
			}
			b.dmAchievement(racer, a)
		}
	}
}

// cleanSweepNames returns the lower-cased names of racers who won and set the
// fastest lap in the most recent finalized race.
func (b *Bot) cleanSweepNames() map[string]bool {
	out := map[string]bool{}
	var histories []apiHistory
	if err := b.apiGet("/api/race-history", &histories); err != nil || len(histories) == 0 {
		return out
	}
	race := histories[0]
	var winner string
	fastest := fastestLapRacer(race.Results)
	if fastest == "" {
		return out
	}
	for _, r := range race.Results {
		if r.Position == 1 {
			winner = r.RacerName
			break
		}
	}
	if winner != "" && strings.EqualFold(winner, fastest) {
		out[strings.ToLower(winner)] = true
	}
	return out
}

// awardAchievement inserts a badge if it is new, reporting whether it was
// actually awarded.
func (b *Bot) awardAchievement(racerID int, a achievementDef) bool {
	res, err := b.s.DB.Exec(
		"INSERT OR IGNORE INTO racer_achievements (racer_id, code, label) VALUES (?, ?, ?)",
		racerID, a.code, a.label)
	if err != nil {
		b.warnf("award achievement %s failed: %v", a.code, err)
		return false
	}
	n, _ := res.RowsAffected()
	return n > 0
}

// dmAchievement messages every opted-in chat linked to the racer.
func (b *Bot) dmAchievement(racer apiRacer, a achievementDef) {
	rows, err := b.s.DB.Query(`
		SELECT l.chat_id FROM telegram_links l
		JOIN telegram_notify_prefs p ON p.chat_id = l.chat_id AND p.personal = 1
		WHERE l.racer_id = ?`, racer.ID)
	if err != nil {
		return
	}
	defer rows.Close()
	text := fmt.Sprintf("🏆 <b>New achievement!</b>\n%s\n<b>%s</b> — %s unlocked %s %s",
		divider, escapeHTML(racer.Name), a.emoji, a.emoji, escapeHTML(a.label))
	for rows.Next() {
		var chatID string
		if rows.Scan(&chatID) != nil {
			continue
		}
		if id, err := strconv.ParseInt(chatID, 10, 64); err == nil {
			b.send(id, text)
		}
	}
}

// badgesCommand renders a racer's unlocked and locked badges.
func (b *Bot) badgesCommand(c cmdContext) string {
	racer, name, ok := b.careerTarget(c)
	if !ok {
		return "🔐 Link your racer with /login, or try <code>/badges Alice</code>."
	}

	unlocked := map[string]bool{}
	rows, err := b.s.DB.Query("SELECT code FROM racer_achievements WHERE racer_id = ?", racer.ID)
	if err != nil {
		b.warnf("badges fetch failed: %v", err)
		return "⚠️ Could not load achievements right now."
	}
	for rows.Next() {
		var code string
		if rows.Scan(&code) == nil {
			unlocked[code] = true
		}
	}
	rows.Close()

	var sb strings.Builder
	fmt.Fprintf(&sb, "🎖️ <b>Badges</b> — %s\n", escapeHTML(name))
	sb.WriteString(divider + "\n")
	earned := 0
	for _, a := range achievementCatalog {
		if unlocked[a.code] {
			earned++
			fmt.Fprintf(&sb, "✅ %s <b>%s</b>\n", a.emoji, escapeHTML(a.label))
		} else {
			fmt.Fprintf(&sb, "▫️ %s\n", escapeHTML(a.label))
		}
	}
	fmt.Fprintf(&sb, "\n%d/%d unlocked", earned, len(achievementCatalog))
	return sb.String()
}
