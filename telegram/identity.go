package telegram

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// loginFlowTimeout bounds how long the bot waits for an email after /login.
const loginFlowTimeout = 5 * time.Minute

// pendingLogin tracks a chat that has started the /login flow.
type pendingLogin struct {
	expiresAt time.Time
}

func (b *Bot) loginFlow(chatID int64) (pendingLogin, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	p, ok := b.pendingLogins[chatID]
	if !ok {
		return pendingLogin{}, false
	}
	if time.Now().After(p.expiresAt) {
		delete(b.pendingLogins, chatID)
		return pendingLogin{}, false
	}
	return p, true
}

func (b *Bot) setLoginFlow(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pendingLogins == nil {
		b.pendingLogins = make(map[int64]pendingLogin)
	}
	b.pendingLogins[chatID] = pendingLogin{expiresAt: time.Now().Add(loginFlowTimeout)}
}

func (b *Bot) clearLoginFlow(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.pendingLogins, chatID)
}

// apiPersonalStats mirrors the career stat fields returned by /api/racer-stats.
type apiPersonalStats struct {
	Races       int `json:"races"`
	Wins        int `json:"wins"`
	Gold        int `json:"gold"`
	Silver      int `json:"silver"`
	Bronze      int `json:"bronze"`
	FastestLaps int `json:"fastest_laps"`
	Points      int `json:"points"`
	DNF         int `json:"dnf"`
	DNS         int `json:"dns"`
	Spins       int `json:"spins"`
	Overheated  int `json:"overheated"`
}

type apiPersonalEnvelope struct {
	Stats apiPersonalStats `json:"stats"`
	Racer apiRacer         `json:"racer"`
}

type apiRecentResult struct {
	Name       string `json:"name"`
	Date       string `json:"race_date"`
	Track      string `json:"track"`
	Country    string `json:"country"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
	FastestLap bool   `json:"fastest_lap"`
	RaceType   string `json:"race_type"`
}

type apiUpgradeCard struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CardType    string `json:"card_type"`
	Cost        int    `json:"cost"`
	Effects     string `json:"effects"`
}

type apiPlayerUpgrade struct {
	ID          int             `json:"id"`
	RacerID     int             `json:"racer_id"`
	UpgradeID   int             `json:"upgrade_id"`
	SeasonID    int             `json:"season_id"`
	Equipped    bool            `json:"equipped"`
	RoundBought int             `json:"round_bought"`
	Upgrade     *apiUpgradeCard `json:"upgrade,omitempty"`
}

// loginCommand starts or completes the email sign-in flow.
func (b *Bot) loginCommand(c cmdContext) string {
	if email := strings.TrimSpace(c.args); email != "" {
		return b.startLogin(c, email)
	}
	b.setLoginFlow(c.chatID)
	return "🔐 <b>Sign in</b>\n\nSend the email address registered for your racer profile and I'll email you a single-use sign-in link.\n\nSend /cancel to abort."
}

// handleLoginText consumes a plain message while the login flow is pending.
func (b *Bot) handleLoginText(c cmdContext, text string) (string, bool) {
	if _, ok := b.loginFlow(c.chatID); !ok {
		return "", false
	}
	return b.startLogin(c, text), true
}

// startLogin asks the server to email a verification link for this chat.
func (b *Bot) startLogin(c cmdContext, email string) string {
	b.clearLoginFlow(c.chatID)
	email = strings.TrimSpace(email)
	if email == "" || !strings.Contains(email, "@") {
		return "That doesn't look like an email address. Send /login to try again."
	}

	var resp struct {
		Status string `json:"status"`
		Found  bool   `json:"found"`
		Sent   bool   `json:"sent"`
	}
	payload := map[string]string{
		"email":   email,
		"chat_id": strconv.FormatInt(c.chatID, 10),
	}
	if err := b.apiPost("/api/telegram/link/start", payload, &resp); err != nil {
		b.warnf("link start failed: %v", err)
		return "⚠️ Could not start sign-in right now. Please try again later."
	}
	if !resp.Found {
		return "❌ No racer is registered with that email address. Ask your race admin to add it, then try /login again."
	}
	if !resp.Sent {
		return "⚠️ Found your racer, but the sign-in email could not be sent. Ask your race admin to check the email settings."
	}
	return "📧 Check your inbox — I've emailed you a sign-in link. Open it to finish signing in, then come back here."
}

func (b *Bot) cancelLogin(c cmdContext) string {
	b.clearLoginFlow(c.chatID)
	return "🤷 Sign-in cancelled. Send /login to try again."
}

// logoutCommand unlinks this chat from its racer.
func (b *Bot) logoutCommand(c cmdContext) string {
	res, err := b.s.DB.Exec("DELETE FROM telegram_links WHERE chat_id = ?", strconv.FormatInt(c.chatID, 10))
	if err != nil {
		return "⚠️ Could not sign you out right now. Please try again later."
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "You're not signed in on this chat. Send /login to sign in."
	}
	return "👋 Signed out. This chat is no longer linked to your racer profile."
}

// racerForChat resolves the racer linked to a Telegram chat.
func (b *Bot) racerForChat(chatID int64) (int, string, bool) {
	var racerID int
	var name string
	err := b.s.DB.QueryRow(
		`SELECT l.racer_id, COALESCE(r.name, '') FROM telegram_links l
		 JOIN racers r ON l.racer_id = r.id WHERE l.chat_id = ?`,
		strconv.FormatInt(chatID, 10)).Scan(&racerID, &name)
	if err != nil {
		return 0, "", false
	}
	return racerID, name, true
}

// renderMyStats renders the linked racer's career, season, and recent form.
func (b *Bot) renderMyStats(c cmdContext) string {
	racerID, name, ok := b.racerForChat(c.chatID)
	if !ok {
		return "🔐 You're not signed in yet. Send /login to link this chat to your racer profile."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🏁 <b>My Stats</b> — %s\n", escapeHTML(name))

	var env apiPersonalEnvelope
	if err := b.apiGet(fmt.Sprintf("/api/racer-stats?id=%d", racerID), &env); err != nil {
		b.warnf("renderMyStats stats failed: %v", err)
	}
	s := env.Stats
	sb.WriteString(divider + "\n")
	sb.WriteString("<b>Career</b>\n")
	fmt.Fprintf(&sb, "Races: %d · Wins: %d · Podiums: %d\n", s.Races, s.Wins, s.Gold+s.Silver+s.Bronze)
	fmt.Fprintf(&sb, "Points: %d · Fastest laps: %d\n", s.Points, s.FastestLaps)
	fmt.Fprintf(&sb, "DNF: %d · DNS: %d\n", s.DNF, s.DNS)

	var summary apiSummary
	if err := b.apiGet("/api/telegram/summary", &summary); err == nil {
		rank := 0
		var points, wins int
		for i, st := range summary.Standings {
			if st.RacerName == name {
				rank = i + 1
				points = st.Points
				wins = st.Wins
				break
			}
		}
		if rank > 0 {
			sb.WriteString(divider + "\n")
			if summary.Season.Name != "" {
				fmt.Fprintf(&sb, "<b>Season</b> — %s\n", escapeHTML(summary.Season.Name))
			} else {
				sb.WriteString("<b>Season</b>\n")
			}
			fmt.Fprintf(&sb, "%s · %d pts · %d %s\n", rankLabel(rank), points, wins, pluralize(wins, "win"))
		}
	}

	var recent []apiRecentResult
	if err := b.apiGet(fmt.Sprintf("/api/racer-recent-results?racer_id=%d&limit=5", racerID), &recent); err == nil && len(recent) > 0 {
		sb.WriteString(divider + "\n")
		sb.WriteString("<b>Recent form</b>\n")
		for _, r := range recent {
			track := r.Track
			if track == "" {
				track = r.Name
			}
			line := fmt.Sprintf("%s %s — %s\n", positionMedal(r.Position), escapeHTML(track), pluralize(r.Points, "pt"))
			sb.WriteString(line)
		}
	}

	sb.WriteString("\n🌐 Manage your upgrades on the HEAT website.")
	return sb.String()
}

// renderMyUpgrades renders the linked racer's owned upgrades, read-only.
func (b *Bot) renderMyUpgrades(c cmdContext) string {
	racerID, _, ok := b.racerForChat(c.chatID)
	if !ok {
		return "🔐 You're not signed in yet. Send /login to link this chat to your racer profile."
	}

	var owned []apiPlayerUpgrade
	if err := b.apiGet(fmt.Sprintf("/api/player-upgrades?racer_id=%d", racerID), &owned); err != nil {
		b.warnf("renderMyUpgrades failed: %v", err)
		return "⚠️ Could not load your upgrades right now. Please try again later."
	}

	var sb strings.Builder
	sb.WriteString("🧰 <b>My Upgrades</b>\n")
	if len(owned) == 0 {
		sb.WriteString(divider + "\n")
		sb.WriteString("You don't own any upgrades yet.")
		return sb.String()
	}
	sb.WriteString(divider + "\n")
	for _, pu := range owned {
		cardName := "Upgrade"
		cost := 0
		if pu.Upgrade != nil {
			cardName = pu.Upgrade.Name
			cost = pu.Upgrade.Cost
		}
		state := "▫️"
		if pu.Equipped {
			state = "✅ equipped"
		}
		fmt.Fprintf(&sb, "• <b>%s</b> — %d cost · Round %d %s\n", escapeHTML(cardName), cost, pu.RoundBought, state)
	}
	return sb.String()
}
