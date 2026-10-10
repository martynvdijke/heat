package telegram

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// Types mirroring the read-only stats endpoints the bot consumes.

type apiELORating struct {
	RacerID   int     `json:"racer_id"`
	RacerName string  `json:"racer_name"`
	Rating    float64 `json:"rating"`
	Races     int     `json:"races"`
}

type apiStreakInfo struct {
	RacerName    string `json:"racer_name"`
	StreakType   string `json:"streak_type"`
	CurrentValue int    `json:"current_value"`
	BestValue    int    `json:"best_value"`
}

type apiHeadToHead struct {
	Racer1     string  `json:"racer1"`
	Racer2     string  `json:"racer2"`
	Races      int     `json:"races"`
	Racer1Wins int     `json:"racer1_wins"`
	Racer2Wins int     `json:"racer2_wins"`
	Racer1Avg  float64 `json:"racer1_avg_position"`
	Racer2Avg  float64 `json:"racer2_avg_position"`
}

type apiTrackPerf struct {
	TrackName    string  `json:"track_name"`
	Country      string  `json:"country"`
	Races        int     `json:"races"`
	Wins         int     `json:"wins"`
	Podiums      int     `json:"podiums"`
	AvgPosition  float64 `json:"avg_position"`
	BestPosition int     `json:"best_position"`
	TotalPoints  float64 `json:"total_points"`
	DNFRate      float64 `json:"dnf_rate"`
}

// racerNameByID fetches the roster and returns an id -> name lookup.
func (b *Bot) racerNameByID() (map[int]string, error) {
	var racers []apiRacer
	if err := b.apiGet("/api/racers", &racers); err != nil {
		return nil, err
	}
	out := make(map[int]string, len(racers))
	for _, r := range racers {
		out[r.ID] = r.Name
	}
	return out, nil
}

// resolveRacer turns a user-typed name into a roster entry, preferring an exact
// case-insensitive match, then a unique prefix/substring match.
func (b *Bot) resolveRacer(query string) (apiRacer, bool) {
	query = strings.TrimSpace(query)
	if query == "" {
		return apiRacer{}, false
	}
	var racers []apiRacer
	if err := b.apiGet("/api/racers", &racers); err != nil {
		b.warnf("resolveRacer fetch failed: %v", err)
		return apiRacer{}, false
	}
	lower := strings.ToLower(query)

	for _, r := range racers {
		if strings.ToLower(r.Name) == lower {
			return r, true
		}
	}
	var matches []apiRacer
	for _, r := range racers {
		if strings.Contains(strings.ToLower(r.Name), lower) {
			matches = append(matches, r)
		}
	}
	if len(matches) == 1 {
		return matches[0], true
	}
	return apiRacer{}, false
}

// renderElo lists the top-rated racers by ELO.
func (b *Bot) renderElo() string {
	var ratings []apiELORating
	if err := b.apiGet("/api/stats/elo", &ratings); err != nil {
		b.warnf("elo fetch failed: %v", err)
		return "⚠️ ELO ratings are unavailable right now."
	}
	if len(ratings) == 0 {
		return "📈 No ELO ratings yet."
	}
	sort.SliceStable(ratings, func(i, j int) bool { return ratings[i].Rating > ratings[j].Rating })

	var sb strings.Builder
	sb.WriteString("📈 <b>ELO Ratings</b>\n")
	sb.WriteString(divider + "\n")
	for i, r := range ratings {
		if i >= 10 {
			break
		}
		fmt.Fprintf(&sb, "%s <b>%s</b> — %.0f", rankLabel(i+1), escapeHTML(r.RacerName), r.Rating)
		if r.Races > 0 {
			fmt.Fprintf(&sb, " <i>(%d %s)</i>", r.Races, pluralize(r.Races, "race"))
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderStreaks lists the longest current podium streaks.
func (b *Bot) renderStreaks() string {
	var streaks []apiStreakInfo
	if err := b.apiGet("/api/stats/streaks", &streaks); err != nil {
		b.warnf("streaks fetch failed: %v", err)
		return "⚠️ Streaks are unavailable right now."
	}
	sort.SliceStable(streaks, func(i, j int) bool {
		if streaks[i].CurrentValue != streaks[j].CurrentValue {
			return streaks[i].CurrentValue > streaks[j].CurrentValue
		}
		return streaks[i].BestValue > streaks[j].BestValue
	})

	var sb strings.Builder
	sb.WriteString("🔥 <b>Podium Streaks</b>\n")
	sb.WriteString(divider + "\n")
	shown := 0
	for _, s := range streaks {
		if s.RacerName == "" || s.BestValue < 2 {
			continue
		}
		shown++
		if shown > 10 {
			break
		}
		fmt.Fprintf(&sb, "• <b>%s</b> — 🔥 %d in a row", escapeHTML(s.RacerName), s.CurrentValue)
		if s.BestValue > s.CurrentValue {
			fmt.Fprintf(&sb, " <i>(best %d)</i>", s.BestValue)
		}
		sb.WriteString("\n")
	}
	if shown == 0 {
		return "🔥 No podium streaks of 2 or more yet."
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderH2H compares two racers passed as "/h2h Alice vs Bob".
func (b *Bot) renderH2H(c cmdContext) string {
	parts := splitH2HArgs(c.args)
	if len(parts) != 2 {
		return "⚔️ Usage: <code>/h2h Alice vs Bob</code> — compare two racers."
	}
	r1, ok1 := b.resolveRacer(parts[0])
	r2, ok2 := b.resolveRacer(parts[1])
	if !ok1 {
		return fmt.Sprintf("🤷 I couldn't find a racer matching %q.", escapeHTML(parts[0]))
	}
	if !ok2 {
		return fmt.Sprintf("🤷 I couldn't find a racer matching %q.", escapeHTML(parts[1]))
	}
	if r1.ID == r2.ID {
		return "🤷 Pick two different racers."
	}

	var h2h apiHeadToHead
	if err := b.apiGet(fmt.Sprintf("/api/stats/head-to-head?racer1=%d&racer2=%d", r1.ID, r2.ID), &h2h); err != nil {
		b.warnf("head-to-head fetch failed: %v", err)
		return "⚠️ Head-to-head is unavailable right now."
	}
	if h2h.Races == 0 {
		return fmt.Sprintf("⚔️ <b>%s</b> and <b>%s</b> haven't raced each other yet.", escapeHTML(r1.Name), escapeHTML(r2.Name))
	}

	var sb strings.Builder
	sb.WriteString("⚔️ <b>Head to Head</b>\n")
	sb.WriteString(divider + "\n")
	fmt.Fprintf(&sb, "<b>%s</b> %d — %d <b>%s</b>\n", escapeHTML(r1.Name), h2h.Racer1Wins, h2h.Racer2Wins, escapeHTML(r2.Name))
	fmt.Fprintf(&sb, "<i>%d head-to-head %s</i>\n", h2h.Races, pluralize(h2h.Races, "race"))
	fmt.Fprintf(&sb, "Avg finish: %.1f vs %.1f", h2h.Racer1Avg, h2h.Racer2Avg)
	return sb.String()
}

// splitH2HArgs splits "/h2h A vs B" on common separators.
func splitH2HArgs(args string) []string {
	for _, sep := range []string{" vs ", " VS ", " v ", " — ", " -- ", ",", "|"} {
		if i := strings.Index(args, sep); i >= 0 {
			a := strings.TrimSpace(args[:i])
			c := strings.TrimSpace(args[i+len(sep):])
			if a != "" && c != "" {
				return []string{a, c}
			}
		}
	}
	return nil
}

// renderCareer shows a full career line for a named racer, or the linked
// racer when no name is given.
func (b *Bot) renderCareer(c cmdContext) string {
	racer, name, ok := b.careerTarget(c)
	if !ok {
		return "🔎 Usage: <code>/career Alice</code> — or /login to link your racer."
	}

	var env apiPersonalEnvelope
	if err := b.apiGet(fmt.Sprintf("/api/racer-stats?id=%d", racer.ID), &env); err != nil {
		b.warnf("career fetch failed: %v", err)
		return "⚠️ Career stats are unavailable right now."
	}
	s := env.Stats
	var sb strings.Builder
	fmt.Fprintf(&sb, "🎓 <b>%s</b>\n", escapeHTML(name))
	sb.WriteString(divider + "\n")
	fmt.Fprintf(&sb, "Races: %d · Wins: %d · Podiums: %d\n", s.Races, s.Wins, s.Gold+s.Silver+s.Bronze)
	fmt.Fprintf(&sb, "Points: %d · Fastest laps: %d\n", s.Points, s.FastestLaps)
	fmt.Fprintf(&sb, "🥇 %d · 🥈 %d · 🥉 %d\n", s.Gold, s.Silver, s.Bronze)
	fmt.Fprintf(&sb, "Spins: %d · Overheated: %d\n", s.Spins, s.Overheated)
	fmt.Fprintf(&sb, "DNF: %d · DNS: %d", s.DNF, s.DNS)
	return sb.String()
}

// careerTarget resolves the racer a /career (or /trackperf) request refers to:
// an explicit name, otherwise the linked racer.
func (b *Bot) careerTarget(c cmdContext) (apiRacer, string, bool) {
	if arg := strings.TrimSpace(c.args); arg != "" {
		r, ok := b.resolveRacer(arg)
		return r, r.Name, ok
	}
	racerID, name, ok := b.racerForChat(c.chatID)
	if !ok {
		return apiRacer{}, "", false
	}
	return apiRacer{ID: racerID, Name: name}, name, true
}

// renderTrackPerf shows a racer's best/worst tracks.
func (b *Bot) renderTrackPerf(c cmdContext) string {
	target, name, ok := b.careerTarget(c)
	if !ok {
		return "🔎 Usage: <code>/trackperf Alice</code> — or /login to link your racer."
	}
	var perf []apiTrackPerf
	if err := b.apiGet(fmt.Sprintf("/api/stats/track-performance?racer_id=%d", target.ID), &perf); err != nil {
		b.warnf("track performance fetch failed: %v", err)
		return "⚠️ Track performance is unavailable right now."
	}
	if len(perf) == 0 {
		return fmt.Sprintf("🛣️ No track history yet for <b>%s</b>.", escapeHTML(name))
	}
	sort.SliceStable(perf, func(i, j int) bool {
		if perf[i].Wins != perf[j].Wins {
			return perf[i].Wins > perf[j].Wins
		}
		return perf[i].Podiums > perf[j].Podiums
	})

	var sb strings.Builder
	fmt.Fprintf(&sb, "🛣️ <b>Track Form</b> — %s\n", escapeHTML(name))
	sb.WriteString(divider + "\n")
	limit := 10
	if len(perf) < limit {
		limit = len(perf)
	}
	for _, p := range perf[:limit] {
		track := p.TrackName
		if track == "" {
			track = "Unknown track"
		}
		fmt.Fprintf(&sb, "• <b>%s</b> — %d %s · %dW/%dP · avg %.1f\n",
			escapeHTML(track), p.Races, pluralize(p.Races, "race"), p.Wins, p.Podiums, p.AvgPosition)
	}
	return strings.TrimRight(sb.String(), "\n")
}

// renderCalendar returns a subscribe link for the season iCal feed.
func (b *Bot) renderCalendar() string {
	base := strings.TrimRight(os.Getenv("PUBLIC_BASE_URL"), "/")
	if base == "" {
		return "📆 The season calendar isn't published yet — ask your race admin to set the public base URL."
	}
	url := base + "/api/races/export.ics"
	var sb strings.Builder
	sb.WriteString("📆 <b>Season Calendar</b>\n")
	sb.WriteString(divider + "\n")
	sb.WriteString("Subscribe to every race in your calendar app:\n")
	fmt.Fprintf(&sb, "🔗 %s\n", url)
	sb.WriteString("\n<i>Paste the link into Google/Apple Calendar → Add by URL.</i>")
	return sb.String()
}
