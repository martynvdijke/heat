package handlers

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/middleware"
)

// deleteConfirmation is the exact word a user must send in the request body.
// A typed word is not a security control, but it stops a stray tap or a
// replay of a generic form from destroying an identity.
const deleteConfirmation = "DELETE"

// Deleting a racer anonymises historical results instead of removing them.
// The record of who finished where stays intact for the championship, but it
// is detached from the person and their name is replaced.
const anonymisedRacerName = "Deleted racer"

type deleteRequest struct {
	Confirm string `json:"confirm"`
}

func confirmed(body deleteRequest) bool {
	return strings.EqualFold(strings.TrimSpace(body.Confirm), deleteConfirmation)
}

// accountSession resolves the admin session cookie to its owning admin id.
func (h *Handler) accountSession(c *gin.Context) (string, app.SessionInfo, bool) {
	cookie, err := c.Request.Cookie("session")
	if err != nil || cookie.Value == "" {
		return "", app.SessionInfo{}, false
	}
	info, ok := h.S.LookupSession(cookie.Value)
	return cookie.Value, info, ok
}

// @Summary Export the signed-in racer's data
// @Description JSON export of the racer profile, identity, results and upgrades. No secrets or session tokens.
// @Tags Account
// @Produce json
// @Security RacerSession
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/me/export [get]
func (h *Handler) ExportMe(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	out := gin.H{
		"exported_at":  time.Now().UTC().Format(time.RFC3339),
		"account_type": "racer",
		"base_url":     h.S.Legal.BaseURL,
		"racer":        h.racerProfile(racerID),
		"emails":       h.racerEmails(racerID),
		"stats":        h.racerStats(racerID),
		"upgrades":     h.racerUpgrades(racerID),
		"results":      h.racerResults(racerID),
		"round_scores": h.racerRoundScores(racerID),
		"achievements": h.racerAchievements(racerID),
		"telegram":     h.racerTelegram(racerID),
		"devices":      h.racerDevices(racerID),
	}
	c.Header("Content-Disposition", `attachment; filename="heat-racer-export.json"`)
	c.JSON(http.StatusOK, out)
}

// @Summary Delete the signed-in racer identity
// @Description Permanently deletes the racer row and every identity link (email, Telegram, player sessions, upgrades, achievements). Historical race results are kept but detached and the name is replaced with "Deleted racer".
// @Tags Account
// @Accept json
// @Produce json
// @Security RacerSession
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/me [delete]
func (h *Handler) DeleteMe(c *gin.Context) {
	racerID := middleware.RacerID(c)
	if racerID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var body deleteRequest
	if err := c.ShouldBindJSON(&body); err != nil || !confirmed(body) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": `confirmation required: send {"confirm":"DELETE"}`})
		return
	}

	if err := h.deleteRacer(c.Request.Context(), racerID); err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Racer not found"})
			return
		}
		internalError(c, err)
		return
	}

	// Revoke every website session for this racer and clear this one's cookie.
	h.S.RacerSessionsMu.Lock()
	for token, sess := range h.S.RacerSessions {
		if sess.RacerID == racerID {
			delete(h.S.RacerSessions, token)
		}
	}
	h.S.RacerSessionsMu.Unlock()
	clearRacerSessionCookie(c, h.S.SecureCookies)
	h.S.BroadcastRacers()
	c.JSON(http.StatusOK, gin.H{"status": "ok", "deleted": "racer", "results": "anonymised"})
}

// deleteRacer performs the whole racer cascade in one transaction. Tables are
// removed with DELETE except historical result tables, whose rows are detached
// and anonymised so the championship record survives without the person.
func (h *Handler) deleteRacer(ctx context.Context, racerID int) error {
	tx, err := h.S.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var exists int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM racers WHERE id = ?", racerID).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		return sql.ErrNoRows
	}

	// Telegram prefs are keyed by chat id, not racer id.
	chatIDs := make([]string, 0)
	rows, err := tx.QueryContext(ctx, "SELECT chat_id FROM telegram_links WHERE racer_id = ?", racerID)
	if err == nil {
		for rows.Next() {
			var chatID string
			if rows.Scan(&chatID) == nil && chatID != "" {
				chatIDs = append(chatIDs, chatID)
			}
		}
		rows.Close()
	}
	for _, chatID := range chatIDs {
		execIfTable(tx, "DELETE FROM telegram_notify_prefs WHERE chat_id = ?", chatID)
		execIfTable(tx, "DELETE FROM telegram_rsvp WHERE chat_id = ?", chatID)
	}

	execIfTable(tx, "DELETE FROM telegram_links WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM telegram_link_tokens WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM telegram_rsvp WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM racer_emails WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM player_sessions WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM player_upgrades WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM racer_legend_abilities WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM racer_stats WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM racer_achievements WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM driver_share WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM heat_cards WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM gear_shifts WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM lap_records WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM turbo_logs WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM racer_sectors WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM race_radio WHERE racer_id = ?", racerID)
	execIfTable(tx, "DELETE FROM race_events WHERE racer_id = ? OR racer_id2 = ?", racerID, racerID)

	// Historical records: keep the competitive fact, remove the identity.
	if err := execIfTable(tx, "UPDATE race_results SET racer_id = 0, racer_name = ? WHERE racer_id = ?", anonymisedRacerName, racerID); err != nil {
		return err
	}
	if err := execIfTable(tx, "UPDATE round_snapshot_scores SET racer_id = 0, racer_name = ? WHERE racer_id = ?", anonymisedRacerName, racerID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, "DELETE FROM racers WHERE id = ?", racerID); err != nil {
		return err
	}
	return tx.Commit()
}

// execIfTable runs a cascade statement but treats a missing optional table as a
// no-op. All tables exist in a current database; this makes upgrades from older
// layouts safe without masking a real SQL error.
func execIfTable(tx *sql.Tx, query string, args ...any) error {
	_, err := tx.Exec(query, args...)
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "no such table") {
		return nil
	}
	return err
}

// @Summary Export the signed-in admin account
// @Description JSON export of the admin profile. Password hashes and session tokens are never included.
// @Tags Account
// @Produce json
// @Security cookieAuth
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]string
// @Router /api/account/export [get]
func (h *Handler) ExportAccount(c *gin.Context) {
	_, info, ok := h.accountSession(c)
	if !ok || info.AdminUserID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sign in again to export your account"})
		return
	}
	var id int
	var username, email, oidcSub, authMethod string
	if err := h.S.DB.QueryRow("SELECT id, username, COALESCE(email, ''), COALESCE(oidc_sub, ''), COALESCE(auth_method, '') FROM admin_users WHERE id = ?", info.AdminUserID).
		Scan(&id, &username, &email, &oidcSub, &authMethod); err != nil {
		if err == sql.ErrNoRows {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Account not found"})
			return
		}
		internalError(c, err)
		return
	}
	c.Header("Content-Disposition", `attachment; filename="heat-account-export.json"`)
	c.JSON(http.StatusOK, gin.H{
		"exported_at":  time.Now().UTC().Format(time.RFC3339),
		"account_type": "admin",
		"base_url":     h.S.Legal.BaseURL,
		"account": gin.H{
			"id":          id,
			"username":    username,
			"email":       email,
			"oidc_sub":    oidcSub,
			"auth_method": authMethod,
		},
	})
}

// @Summary Delete the signed-in admin account
// @Description Deletes the admin account, its password-reset tokens and revokes every admin web session. Race, racer and result data are not touched. The last remaining admin cannot be deleted.
// @Tags Account
// @Accept json
// @Produce json
// @Security cookieAuth
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 409 {object} map[string]string
// @Router /api/account [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	_, info, ok := h.accountSession(c)
	if !ok || info.AdminUserID <= 0 {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Sign in again before deleting your account"})
		return
	}
	var body deleteRequest
	if err := c.ShouldBindJSON(&body); err != nil || !confirmed(body) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": `confirmation required: send {"confirm":"DELETE"}`})
		return
	}

	var admins int
	if err := h.S.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&admins); err != nil {
		internalError(c, err)
		return
	}
	if admins <= 1 {
		c.AbortWithStatusJSON(http.StatusConflict, gin.H{"error": "The last admin account cannot be deleted. Create another admin first."})
		return
	}

	tx, err := h.S.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		internalError(c, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	execIfTable(tx, "DELETE FROM password_reset_tokens WHERE admin_user_id = ?", info.AdminUserID)
	if _, err := tx.Exec("DELETE FROM admin_users WHERE id = ?", info.AdminUserID); err != nil {
		internalError(c, err)
		return
	}
	if err := tx.Commit(); err != nil {
		internalError(c, err)
		return
	}

	// A session that outlived its account would still pass AuthMiddleware, so
	// revoke every admin session rather than only the caller's.
	h.S.DeleteAllSessions()
	c.SetCookie("session", "", -1, "/", "", h.S.SecureCookies, true)
	h.S.Log.Infof("account", "Admin account %d deleted; all sessions revoked", info.AdminUserID)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "deleted": "admin"})
}

// ---- export helpers -------------------------------------------------------

func (h *Handler) racerProfile(racerID int) gin.H {
	var id int
	var name, picture, carColor, carName string
	var points, rank, position, teamID int
	var teamName, teamColor string
	err := h.S.DB.QueryRow(`SELECT r.id, r.name, r.profile_picture, r.car_color, r.car_name, r.points, r.rank, r.position,
		COALESCE(r.team_id, 0), COALESCE(t.name, ''), COALESCE(t.color, '')
		FROM racers r LEFT JOIN teams t ON r.team_id = t.id WHERE r.id = ?`, racerID).
		Scan(&id, &name, &picture, &carColor, &carName, &points, &rank, &position, &teamID, &teamName, &teamColor)
	if err != nil {
		return gin.H{"id": racerID}
	}
	return gin.H{
		"id": id, "name": name, "profile_picture": picture,
		"car_color": carColor, "car_name": carName,
		"points": points, "rank": rank, "position": position,
		"team_id": teamID, "team_name": teamName, "team_color": teamColor,
	}
}

func (h *Handler) racerEmails(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query("SELECT id, COALESCE(email, '') FROM racer_emails WHERE racer_id = ?", racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var email string
		if rows.Scan(&id, &email) == nil {
			out = append(out, gin.H{"id": id, "email": email})
		}
	}
	return out
}

func (h *Handler) racerStats(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query("SELECT races, wins, gold, silver, bronze, fastest_laps, points, dnf, dns FROM racer_stats WHERE racer_id = ?", racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var races, wins, gold, silver, bronze, fastestLaps, points, dnf, dns int
		if rows.Scan(&races, &wins, &gold, &silver, &bronze, &fastestLaps, &points, &dnf, &dns) == nil {
			out = append(out, gin.H{
				"races": races, "wins": wins, "gold": gold, "silver": silver, "bronze": bronze,
				"fastest_laps": fastestLaps, "points": points, "dnf": dnf, "dns": dns,
			})
		}
	}
	return out
}

func (h *Handler) racerUpgrades(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query(`SELECT pu.id, pu.upgrade_id, pu.season_id, pu.equipped, pu.round_bought,
		COALESCE(uc.name, ''), COALESCE(uc.description, ''), COALESCE(uc.card_type, ''), COALESCE(uc.cost, 0)
		FROM player_upgrades pu LEFT JOIN upgrade_cards uc ON pu.upgrade_id = uc.id
		WHERE pu.racer_id = ? ORDER BY pu.id`, racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, upgradeID, seasonID, equipped, roundBought, cost int
		var name, description, cardType string
		if rows.Scan(&id, &upgradeID, &seasonID, &equipped, &roundBought, &name, &description, &cardType, &cost) == nil {
			out = append(out, gin.H{
				"id": id, "upgrade_id": upgradeID, "season_id": seasonID, "equipped": equipped == 1,
				"round_bought": roundBought, "name": name, "description": description,
				"card_type": cardType, "cost": cost,
			})
		}
	}
	return out
}

func (h *Handler) racerResults(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query(`SELECT id, race_id, racer_name, position, points, fastest_lap, finished
		FROM race_results WHERE racer_id = ? ORDER BY race_id, id`, racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, raceID, position, points, fastestLap, finished int
		var name string
		if rows.Scan(&id, &raceID, &name, &position, &points, &fastestLap, &finished) == nil {
			out = append(out, gin.H{
				"id": id, "race_id": raceID, "racer_name": name, "position": position,
				"points": points, "fastest_lap": fastestLap == 1, "finished": finished == 1,
			})
		}
	}
	return out
}

func (h *Handler) racerRoundScores(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query(`SELECT id, snapshot_id, racer_name, points, position, dnf, dns
		FROM round_snapshot_scores WHERE racer_id = ? ORDER BY snapshot_id, id`, racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var id, snapshotID, points, position, dnf, dns int
		var name string
		if rows.Scan(&id, &snapshotID, &name, &points, &position, &dnf, &dns) == nil {
			out = append(out, gin.H{
				"id": id, "snapshot_id": snapshotID, "racer_name": name,
				"points": points, "position": position, "dnf": dnf == 1, "dns": dns == 1,
			})
		}
	}
	return out
}

func (h *Handler) racerAchievements(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query(`SELECT code, label, detail, race_id, achieved_at
		FROM racer_achievements WHERE racer_id = ? ORDER BY achieved_at`, racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var code, label, detail, achievedAt string
		var raceID int
		if rows.Scan(&code, &label, &detail, &raceID, &achievedAt) == nil {
			out = append(out, gin.H{"code": code, "label": label, "detail": detail, "race_id": raceID, "achieved_at": achievedAt})
		}
	}
	return out
}

func (h *Handler) racerTelegram(racerID int) gin.H {
	out := gin.H{"links": []gin.H{}, "notify_personal": false}
	rows, err := h.S.DB.Query("SELECT chat_id, linked_at FROM telegram_links WHERE racer_id = ?", racerID)
	if err == nil {
		links := make([]gin.H, 0)
		for rows.Next() {
			var chatID, linkedAt string
			if rows.Scan(&chatID, &linkedAt) == nil {
				links = append(links, gin.H{"chat_id": chatID, "linked_at": linkedAt})
			}
		}
		rows.Close()
		out["links"] = links
	}
	var personal int
	if err := h.S.DB.QueryRow(`SELECT COALESCE(personal, 0) FROM telegram_notify_prefs
		WHERE chat_id IN (SELECT chat_id FROM telegram_links WHERE racer_id = ?) LIMIT 1`, racerID).Scan(&personal); err == nil {
		out["notify_personal"] = personal == 1
	}
	return out
}

// racerDevices lists non-secret player-session metadata. The token itself is a
// credential and is intentionally excluded from an export.
func (h *Handler) racerDevices(racerID int) []gin.H {
	out := make([]gin.H, 0)
	rows, err := h.S.DB.Query("SELECT device_name, last_seen, created_at FROM player_sessions WHERE racer_id = ?", racerID)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var device, lastSeen, createdAt string
		if rows.Scan(&device, &lastSeen, &createdAt) == nil {
			out = append(out, gin.H{"device_name": device, "last_seen": lastSeen, "created_at": createdAt})
		}
	}
	return out
}
