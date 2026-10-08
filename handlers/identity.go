package handlers

import (
	"crypto/subtle"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/middleware"
	"heat/models"
	"heat/racing"
)

// racerLinkTokenTTL is how long an emailed racer verification link stays valid.
const racerLinkTokenTTL = 15 * time.Minute

// racerSessionTTL is how long a verified racer's website session lasts.
const racerSessionTTL = 30 * 24 * time.Hour

// racerRecentResult is a single recent race result for a racer.
type racerRecentResult struct {
	RaceID     int    `json:"race_id"`
	Name       string `json:"name"`
	Date       string `json:"race_date"`
	Track      string `json:"track"`
	Country    string `json:"country"`
	Position   int    `json:"position"`
	Points     int    `json:"points"`
	FastestLap bool   `json:"fastest_lap"`
	RaceType   string `json:"race_type"`
}

// lookupRacerByEmail finds a racer whose email on file matches (case-insensitive).
func (h *Handler) lookupRacerByEmail(email string) (int, string, bool) {
	email = strings.TrimSpace(email)
	if email == "" {
		return 0, "", false
	}
	var id int
	var name string
	err := h.S.DB.QueryRow(
		`SELECT r.id, COALESCE(r.name, '') FROM racer_emails re JOIN racers r ON re.racer_id = r.id
		 WHERE LOWER(TRIM(re.email)) = LOWER(?)`, email).Scan(&id, &name)
	if err != nil {
		return 0, "", false
	}
	return id, name, true
}

// issueRacerLinkToken creates a single-use verification token for a racer.
func (h *Handler) issueRacerLinkToken(racerID int, chatID string) (string, error) {
	token := generateResetToken()
	if token == "" {
		return "", fmt.Errorf("failed to generate token")
	}
	expiresAt := time.Now().Add(racerLinkTokenTTL).Format(resetTimeLayout)
	if _, err := h.S.DB.Exec(
		"INSERT INTO telegram_link_tokens (token, racer_id, chat_id, expires_at) VALUES (?, ?, ?, ?)",
		token, racerID, chatID, expiresAt); err != nil {
		return "", err
	}
	return token, nil
}

// racerLinkURL builds the absolute verification URL for an emailed link.
// PUBLIC_BASE_URL takes precedence because the Telegram bot calls the server
// over localhost, so request-derived URLs would be wrong there.
func racerLinkURL(c *gin.Context, token string) string {
	path := "/verify.html?token=" + token
	if base := strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL")), "/"); base != "" {
		return base + path
	}
	return trmnlAbsoluteURL(c, path)
}

// @Summary Start a Telegram chat link (bot only)
// @Description Called by the Telegram bot to email a racer a verification link. Requires the bot token.
// @Tags Racers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/telegram/link/start [post]
func (h *Handler) StartTelegramLink(c *gin.Context) {
	var botToken string
	h.S.DB.QueryRow("SELECT COALESCE(bot_token, '') FROM telegram_settings WHERE id = 1").Scan(&botToken)
	provided := strings.TrimSpace(c.GetHeader("X-Bot-Token"))
	if botToken == "" || provided == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(botToken)) != 1 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var input struct {
		Email  string `json:"email"`
		ChatID string `json:"chat_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.ChatID = strings.TrimSpace(input.ChatID)
	if input.ChatID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "chat_id is required"})
		return
	}

	racerID, _, found := h.lookupRacerByEmail(input.Email)
	if !found {
		// The bot endpoint is trusted, so it is safe to tell the bot the email
		// is not on file; it must not reveal anything to the public.
		c.JSON(http.StatusOK, gin.H{"status": "ok", "found": false, "sent": false})
		return
	}

	token, err := h.issueRacerLinkToken(racerID, input.ChatID)
	if err != nil {
		h.S.Log.Errorf("racer-identity", "StartTelegramLink: failed to store token: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "found": true, "sent": false})
		return
	}
	if err := h.sendRacerLinkEmail(strings.TrimSpace(input.Email), racerLinkURL(c, token)); err != nil {
		h.S.Log.Errorf("racer-identity", "StartTelegramLink: failed to send email: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok", "found": true, "sent": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "found": true, "sent": true})
}

// @Summary Request a racer sign-in link
// @Description Email a racer a passwordless sign-in link. Always returns 200 to avoid user enumeration.
// @Tags Racers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/me/request-link [post]
func (h *Handler) RequestRacerLink(c *gin.Context) {
	var input struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	racerID, _, found := h.lookupRacerByEmail(input.Email)
	if !found {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	token, err := h.issueRacerLinkToken(racerID, "")
	if err != nil {
		h.S.Log.Errorf("racer-identity", "RequestRacerLink: failed to store token: %v", err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}
	if err := h.sendRacerLinkEmail(strings.TrimSpace(input.Email), racerLinkURL(c, token)); err != nil {
		h.S.Log.Errorf("racer-identity", "RequestRacerLink: failed to send email: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Validate a racer verification token
// @Description Check whether a racer verification token is valid without consuming it
// @Tags Racers
// @Produce json
// @Param token query string true "Verification token"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]bool
// @Router /api/telegram/verify/validate [get]
func (h *Handler) ValidateRacerLink(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	racerID, _, ok := h.validRacerLinkToken(token)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"valid": false})
		return
	}
	var name string
	h.S.DB.QueryRow("SELECT COALESCE(name, '') FROM racers WHERE id = ?", racerID).Scan(&name)
	c.JSON(http.StatusOK, gin.H{"valid": true, "racer_name": name})
}

// @Summary Confirm a racer verification token
// @Description Consume a verification token: link the originating Telegram chat (if any) and start a racer website session
// @Tags Racers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Router /api/telegram/verify [post]
func (h *Handler) VerifyRacerLink(c *gin.Context) {
	var input struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token := strings.TrimSpace(input.Token)

	racerID, chatID, ok := h.validRacerLinkToken(token)
	if !ok {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired link"})
		return
	}
	if _, err := h.S.DB.Exec("UPDATE telegram_link_tokens SET used = 1 WHERE token = ?", token); err != nil {
		h.S.Log.Errorf("racer-identity", "VerifyRacerLink: failed to consume token: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify link"})
		return
	}

	if chatID != "" {
		if _, err := h.S.DB.Exec(
			`INSERT INTO telegram_links (chat_id, racer_id) VALUES (?, ?)
			 ON CONFLICT(chat_id) DO UPDATE SET racer_id = excluded.racer_id, linked_at = datetime('now')`,
			chatID, racerID); err != nil {
			h.S.Log.Errorf("racer-identity", "VerifyRacerLink: failed to link chat: %v", err)
		}
	}

	var name string
	h.S.DB.QueryRow("SELECT COALESCE(name, '') FROM racers WHERE id = ?", racerID).Scan(&name)

	h.createRacerSession(c, racerID)

	if chatID != "" {
		h.enqueueTelegram(models.TelegramEvent{Kind: "identity_linked", ChatID: chatID, Text: name})
	}

	h.S.Log.Infof("racer-identity", "Racer %d verified via link (chat linked: %t)", racerID, chatID != "")
	c.JSON(http.StatusOK, gin.H{"status": "ok", "racer_id": racerID, "racer_name": name})
}

// validRacerLinkToken returns the racer and chat a live token belongs to.
func (h *Handler) validRacerLinkToken(token string) (int, string, bool) {
	if token == "" {
		return 0, "", false
	}
	var racerID int
	var chatID string
	var used int
	var expiresAt string
	err := h.S.DB.QueryRow(
		"SELECT racer_id, COALESCE(chat_id, ''), used, expires_at FROM telegram_link_tokens WHERE token = ?",
		token).Scan(&racerID, &chatID, &used, &expiresAt)
	if err != nil || used != 0 {
		return 0, "", false
	}
	exp, err := time.Parse(resetTimeLayout, expiresAt)
	if err != nil || !time.Now().Before(exp) {
		return 0, "", false
	}
	return racerID, chatID, true
}

// createRacerSession stores a racer session and sets the racer_session cookie.
func (h *Handler) createRacerSession(c *gin.Context, racerID int) {
	sessionID := generateSessionID()
	if sessionID == "" {
		return
	}
	h.S.RacerSessionsMu.Lock()
	h.S.RacerSessions[sessionID] = app.RacerSession{
		RacerID: racerID,
		Expiry:  time.Now().Add(racerSessionTTL).Unix(),
		IP:      c.ClientIP(),
	}
	h.S.RacerSessionsMu.Unlock()
	setRacerSessionCookie(c, sessionID, h.S.SecureCookies)
}

func setRacerSessionCookie(c *gin.Context, sessionID string, secureCookies bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(app.RacerSessionCookie, sessionID, int(racerSessionTTL.Seconds()), "/", "", secureCookies, true)
}

func clearRacerSessionCookie(c *gin.Context, secureCookies bool) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(app.RacerSessionCookie, "", -1, "/", "", secureCookies, true)
}

// @Summary Current racer's personal summary
// @Description Career stats, current-season standing, and recent results for the signed-in racer
// @Tags Racers
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/me [get]
func (h *Handler) MeRacer(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	racer := racing.RacerInfo(h.S.DB, racerID)
	stats, ok := racing.SingleRacerStatsBySeasons(h.S.DB, racerID, nil)
	if !ok {
		stats, _ = racing.SingleRacerStatsFallback(h.S.DB, racerID)
	}

	resp := gin.H{
		"racer":  racer,
		"stats":  stats,
		"recent": h.recentResults(racerID, 5),
	}

	if season, seasonID := h.trmnlSeason(); season != nil && seasonID > 0 {
		standings := racing.SeasonStandings(h.S.DB, seasonID, 1000)
		rank := 0
		var standing *models.SeasonStanding
		for i := range standings {
			if standings[i].RacerID == racerID {
				rank = i + 1
				standing = &standings[i]
				break
			}
		}
		resp["season"] = gin.H{"id": season.ID, "name": season.Name}
		if standing != nil {
			resp["standing"] = gin.H{
				"rank":   rank,
				"points": standing.Points,
				"races":  standing.Races,
				"wins":   standing.Wins,
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}

// @Summary Current racer's upgrades
// @Description Upgrades owned by the signed-in racer plus upgrades available to buy
// @Tags Racers
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/me/upgrades [get]
func (h *Handler) MeRacerUpgrades(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"owned":     h.ownedUpgrades(racerID),
		"available": h.availableUpgrades(racerID),
	})
}

func (h *Handler) ownedUpgrades(racerID int) []models.PlayerUpgrade {
	rows, err := h.S.DB.Query(`SELECT pu.id, pu.racer_id, pu.upgrade_id, pu.season_id, pu.equipped, pu.round_bought,
		uc.id, uc.name, uc.description, uc.card_type, uc.cost, uc.effects, uc.extension_id
		FROM player_upgrades pu JOIN upgrade_cards uc ON pu.upgrade_id = uc.id
		WHERE pu.racer_id = ? ORDER BY pu.round_bought`, racerID)
	if err != nil {
		return []models.PlayerUpgrade{}
	}
	defer rows.Close()
	out := make([]models.PlayerUpgrade, 0)
	for rows.Next() {
		var pu models.PlayerUpgrade
		var uc models.UpgradeCard
		if err := rows.Scan(&pu.ID, &pu.RacerID, &pu.UpgradeID, &pu.SeasonID, &pu.Equipped, &pu.RoundBought,
			&uc.ID, &uc.Name, &uc.Description, &uc.CardType, &uc.Cost, &uc.Effects, &uc.ExtensionID); err != nil {
			continue
		}
		pu.Upgrade = &uc
		out = append(out, pu)
	}
	return out
}

func (h *Handler) availableUpgrades(racerID int) []models.UpgradeCard {
	rows, err := h.S.DB.Query(`SELECT uc.id, uc.name, uc.description, uc.card_type, uc.cost, uc.effects
		FROM upgrade_cards uc WHERE uc.card_type = 'upgrade'
		AND (uc.extension_id = 0 OR uc.extension_id IN (SELECT extension_id FROM owned_extensions))
		AND uc.id NOT IN (SELECT upgrade_id FROM player_upgrades WHERE racer_id = ?)
		ORDER BY uc.cost`, racerID)
	if err != nil {
		return []models.UpgradeCard{}
	}
	defer rows.Close()
	out := make([]models.UpgradeCard, 0)
	for rows.Next() {
		var u models.UpgradeCard
		if err := rows.Scan(&u.ID, &u.Name, &u.Description, &u.CardType, &u.Cost, &u.Effects); err != nil {
			continue
		}
		out = append(out, u)
	}
	return out
}

// @Summary Buy an upgrade as the signed-in racer
// @Description Buy an eligible upgrade for the current racer
// @Tags Racers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /api/me/upgrades/buy [post]
func (h *Handler) MeBuyUpgrade(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		UpgradeID int `json:"upgrade_id"`
		SeasonID  int `json:"season_id"`
		Round     int `json:"round"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !h.racerUpgradeEligible(racerID, req.UpgradeID) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Upgrade not available to this racer"})
		return
	}
	res, err := h.S.DB.Exec("INSERT INTO player_upgrades (racer_id, upgrade_id, season_id, equipped, round_bought) VALUES (?, ?, ?, 1, ?)",
		racerID, req.UpgradeID, req.SeasonID, req.Round)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := res.LastInsertId()
	app.TrySend(h.S, h.S.GameMechanicsBroadcast, models.GameMechanicsUpdate{
		Type: "upgrades", RacerID: racerID, Action: "bought",
	})
	c.JSON(http.StatusOK, gin.H{"status": "ok", "id": id})
}

func (h *Handler) racerUpgradeEligible(racerID, upgradeID int) bool {
	var count int
	err := h.S.DB.QueryRow(`SELECT COUNT(*) FROM upgrade_cards uc WHERE uc.id = ? AND uc.card_type = 'upgrade'
		AND (uc.extension_id = 0 OR uc.extension_id IN (SELECT extension_id FROM owned_extensions))
		AND uc.id NOT IN (SELECT upgrade_id FROM player_upgrades WHERE racer_id = ?)`, upgradeID, racerID).Scan(&count)
	return err == nil && count > 0
}

// @Summary Equip or unequip an owned upgrade
// @Description Toggle the equipped state of a player-upgrade owned by the signed-in racer
// @Tags Racers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/me/upgrades/toggle [put]
func (h *Handler) MeToggleUpgrade(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		ID       int  `json:"id"`
		Equipped bool `json:"equipped"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	equipped := 0
	if req.Equipped {
		equipped = 1
	}
	res, err := h.S.DB.Exec("UPDATE player_upgrades SET equipped = ? WHERE id = ? AND racer_id = ?", equipped, req.ID, racerID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Upgrade not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Sign out the racer session
// @Description Clear the racer website session
// @Tags Racers
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/me/logout [post]
func (h *Handler) MeLogout(c *gin.Context) {
	var token string
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name == app.RacerSessionCookie {
			token = cookie.Value
			break
		}
	}
	if token != "" {
		h.S.RacerSessionsMu.Lock()
		delete(h.S.RacerSessions, token)
		h.S.RacerSessionsMu.Unlock()
	}
	clearRacerSessionCookie(c, h.S.SecureCookies)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Recent results for a racer
// @Description The most recent race results for a racer (public)
// @Tags Racers
// @Produce json
// @Param racer_id query int true "Racer ID"
// @Param limit query int false "Max results (default 5)"
// @Success 200 {array} handlers.racerRecentResult
// @Router /api/racer-recent-results [get]
func (h *Handler) RacerRecentResults(c *gin.Context) {
	racerID, _ := strconv.Atoi(c.Query("racer_id"))
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "racer_id is required"})
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	c.JSON(http.StatusOK, h.recentResults(racerID, limit))
}

func (h *Handler) recentResults(racerID, limit int) []racerRecentResult {
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}
	rows, err := h.S.DB.Query(`SELECT rh.id, COALESCE(rh.name, ''), rh.race_date, COALESCE(rh.track, ''), COALESCE(rh.country, ''),
		rr.position, rr.points, rr.fastest_lap, COALESCE(rh.race_type, 'season')
		FROM race_results rr JOIN race_history rh ON rr.race_id = rh.id
		WHERE rr.racer_id = ? ORDER BY rh.race_date DESC, rh.id DESC LIMIT ?`, racerID, limit)
	if err != nil {
		return []racerRecentResult{}
	}
	defer rows.Close()
	out := make([]racerRecentResult, 0)
	for rows.Next() {
		var r racerRecentResult
		if err := rows.Scan(&r.RaceID, &r.Name, &r.Date, &r.Track, &r.Country,
			&r.Position, &r.Points, &r.FastestLap, &r.RaceType); err != nil {
			continue
		}
		out = append(out, r)
	}
	return out
}

func (h *Handler) sendRacerLinkEmail(to, linkURL string) error {
	var s models.EmailSettings
	var enabled int
	err := h.S.DB.QueryRow("SELECT smtp_host, COALESCE(smtp_port, 587), username, password, from_addr, COALESCE(enabled, 0) FROM email_settings WHERE id = 1").
		Scan(&s.SMTPHost, &s.SMTPPort, &s.Username, &s.Password, &s.FromAddr, &enabled)
	if err != nil || enabled == 0 || s.SMTPHost == "" || s.FromAddr == "" {
		return fmt.Errorf("SMTP not configured")
	}
	return sendSMTP(s, to, buildRacerLinkEmailContent(linkURL))
}

func buildRacerLinkEmailContent(linkURL string) string {
	var b strings.Builder
	b.WriteString("Subject: HEAT - Sign in to your racer profile\n")
	b.WriteString("MIME-Version: 1.0\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\n\n")
	b.WriteString("<!DOCTYPE html><html><head><style>")
	b.WriteString("body{font-family:Arial,sans-serif;background:#111;color:#eee;padding:20px}")
	b.WriteString("h1{color:#d40000;border-bottom:3px solid #d40000;padding-bottom:10px}")
	b.WriteString(".btn{display:inline-block;background:#d40000;color:#fff;padding:12px 24px;text-decoration:none;border-radius:5px;font-weight:bold}")
	b.WriteString("</style></head><body>")
	b.WriteString("<h1>HEAT - Sign in</h1>")
	b.WriteString("<p>Click the button below to sign in to your racer profile and view your personal stats and upgrades. This link expires in 15 minutes and can only be used once.</p>")
	b.WriteString(fmt.Sprintf("<p><a class=\"btn\" href=\"%s\">Sign in</a></p>", html.EscapeString(linkURL)))
	b.WriteString("<p style=\"color:#666;font-size:12px;margin-top:30px\">If you didn't request this, you can safely ignore this email.</p>")
	b.WriteString("</body></html>")
	return b.String()
}
