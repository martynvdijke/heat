package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/models"
)

// @Summary Get racer emails
// @Description Get the list of racer email addresses
// @Tags Email
// @Produce json
// @Success 200 {array} models.RacerEmail
// @Security cookieAuth
// @Router /api/racer-emails [get]
func (h *Handler) GetRacerEmails(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT id, racer_id, COALESCE(email, '') FROM racer_emails ORDER BY racer_id")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	emails := make([]models.RacerEmail, 0)
	for rows.Next() {
		var e models.RacerEmail
		if err := rows.Scan(&e.ID, &e.RacerID, &e.Email); err != nil {
			continue
		}
		emails = append(emails, e)
	}
	c.JSON(http.StatusOK, emails)
}

// @Summary Save racer email
// @Description Save or update a racer's email address
// @Tags Email
// @Accept json
// @Produce json
// @Param email body models.RacerEmail true "Racer email"
// @Success 200 {object} map[string]string
// @Security cookieAuth
// @Router /api/racer-emails [post]
func (h *Handler) SaveRacerEmail(c *gin.Context) {
	var e models.RacerEmail
	if err := c.ShouldBindJSON(&e); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := h.S.DB.Exec("INSERT OR REPLACE INTO racer_emails (id, racer_id, email) VALUES ((SELECT id FROM racer_emails WHERE racer_id = ?), ?, ?)",
		e.RacerID, e.RacerID, e.Email)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary Send race email
// @Description Manually send race result emails for a specific race
// @Tags Email
// @Produce json
// @Param race_id query int true "Race ID"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Security cookieAuth
// @Router /api/send-race-email [post]
func (h *Handler) SendRaceEmailManual(c *gin.Context) {
	raceIDStr := c.Query("race_id")
	raceID := 0
	if raceIDStr != "" {
		raceID, _ = strconv.Atoi(raceIDStr)
	}
	if raceID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "race_id required"})
		return
	}

	var raceName, raceDate, country, track, trackID, raceType string
	var totalLaps int
	err := h.S.DB.QueryRow("SELECT COALESCE(name,''), race_date, country, track, track_id, total_laps, COALESCE(race_type,'season') FROM race_history WHERE id = ?", raceID).
		Scan(&raceName, &raceDate, &country, &track, &trackID, &totalLaps, &raceType)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Race not found"})
		return
	}

	rRows, err := h.S.DB.Query("SELECT racer_id, racer_name, position, points, fastest_lap FROM race_results WHERE race_id = ? ORDER BY position", raceID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rRows.Close()

	results := make([]models.RaceResult, 0)
	for rRows.Next() {
		var res models.RaceResult
		var fl int
		if err := rRows.Scan(&res.RacerID, &res.RacerName, &res.Position, &res.Points, &fl); err != nil {
			continue
		}
		res.FastestLap = fl == 1
		results = append(results, res)
	}

	h.SendRaceEmail(raceName, country, track, totalLaps, results)
	c.JSON(http.StatusOK, gin.H{"status": "email sending initiated"})
}

func buildRaceEmailContent(raceName, country, track string, totalLaps int, results []models.RaceResult) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Subject: HEAT Race Results - %s\n", raceName))
	b.WriteString("MIME-Version: 1.0\n")
	b.WriteString("Content-Type: text/html; charset=\"UTF-8\"\n\n")
	b.WriteString("<!DOCTYPE html><html><head><style>")
	b.WriteString("body{font-family:Arial,sans-serif;background:#111;color:#eee;padding:20px}")
	b.WriteString("h1{color:#d40000;border-bottom:3px solid #d40000;padding-bottom:10px}")
	b.WriteString("table{width:100%%;border-collapse:collapse;margin:20px 0}")
	b.WriteString("th{background:#d40000;color:#fff;padding:10px;text-align:left}")
	b.WriteString("td{padding:10px;border-bottom:1px solid #333}")
	b.WriteString("tr:nth-child(even){background:#1a1a1a}")
	b.WriteString(".gold{color:#ffd700}.silver{color:#c0c0c0}.bronze{color:#cd7f32}")
	b.WriteString("</style></head><body>")
	b.WriteString(fmt.Sprintf("<h1>Race %s</h1>", html.EscapeString(raceName)))
	b.WriteString(fmt.Sprintf("<p><strong>Location:</strong> %s - %s</p>", html.EscapeString(country), html.EscapeString(track)))
	b.WriteString(fmt.Sprintf("<p><strong>Laps:</strong> %d</p>", totalLaps))
	b.WriteString("<h2>Final Standings</h2><table><thead><tr><th>Pos</th><th>Driver</th><th>Points</th><th>Fastest Lap</th></tr></thead><tbody>")
	for _, r := range results {
		cls := ""
		if r.Position == 1 {
			cls = "gold"
		} else if r.Position == 2 {
			cls = "silver"
		} else if r.Position == 3 {
			cls = "bronze"
		}
		fl := ""
		if r.FastestLap {
			fl = "Yes"
		}
		b.WriteString(fmt.Sprintf("<tr><td class=\"%s\">#%d</td><td class=\"%s\">%s</td><td>%d pts</td><td>%s</td></tr>",
			cls, r.Position, cls, html.EscapeString(r.RacerName), r.Points, fl))
	}
	b.WriteString("</tbody></table>")
	b.WriteString("<p style=\"color:#666;font-size:12px;margin-top:30px\">HEAT: Pedal to the Metal Board Game Companion</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func (h *Handler) SendRaceEmail(raceName, country, track string, totalLaps int, results []models.RaceResult) {
	var s models.EmailSettings
	var enabled int
	err := h.S.DB.QueryRow("SELECT smtp_host, COALESCE(smtp_port, 587), username, password, from_addr, COALESCE(enabled, 0) FROM email_settings WHERE id = 1").
		Scan(&s.SMTPHost, &s.SMTPPort, &s.Username, &s.Password, &s.FromAddr, &enabled)
	if err != nil || enabled == 0 || s.SMTPHost == "" || s.FromAddr == "" {
		return
	}

	content := buildRaceEmailContent(raceName, country, track, totalLaps, results)

	rows, err := h.S.DB.Query(`SELECT re.email, r.name FROM racer_emails re JOIN racers r ON r.id = re.racer_id WHERE re.email != ''`)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var email, name string
		if err := rows.Scan(&email, &name); err != nil {
			continue
		}
		go func(to, racerName, content string) {
			if err := sendSMTP(s, to, content); err != nil {
				h.S.Log.Errorf("email", "Failed to send to %s: %v", to, err)
			}
		}(email, name, content)
	}
}

func sendSMTP(s models.EmailSettings, to, content string) error {
	addr := fmt.Sprintf("%s:%d", s.SMTPHost, s.SMTPPort)
	auth := smtp.PlainAuth("", s.Username, s.Password, s.SMTPHost)
	msg := []byte(content)
	return smtp.SendMail(addr, auth, s.FromAddr, []string{to}, msg)
}

// @Summary Test notification
// @Description Send a test notification via Gotify
// @Tags Settings
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/test-notification [post]
func (h *Handler) TestNotification(c *gin.Context) {
	var s models.NotificationSettings
	h.S.DB.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, '') FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken)

	if s.GotiFyURL == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Gotify URL not configured"})
		return
	}

	h.NotifyRaceWinner("Test Driver", "Test Track")
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *Handler) NotifyRaceWinner(winnerName, trackName string) {
	var s models.NotificationSettings
	h.S.DB.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), notify_winner FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken, &s.NotifyWinner)

	if s.NotifyWinner && s.GotiFyURL != "" {
		go sendGotifyNotification("Race Winner!", winnerName+" wins at "+trackName, s.GotiFyURL, s.GotiFyToken)
	}
}

func (h *Handler) NotifyRacePodium(first, second, third, trackName string) {
	var s models.NotificationSettings
	h.S.DB.QueryRow("SELECT COALESCE(gotify_url, ''), COALESCE(gotify_token, ''), notify_podium FROM notification_settings WHERE id = 1").
		Scan(&s.GotiFyURL, &s.GotiFyToken, &s.NotifyPodium)

	if s.NotifyPodium && s.GotiFyURL != "" {
		go sendGotifyNotification("Podium Result", "Podium at "+trackName+": 1. "+first+" 2. "+second+" 3. "+third, s.GotiFyURL, s.GotiFyToken)
	}
}

func sendGotifyNotification(title, message, gotifyURL, token string) error {
	if gotifyURL == "" || token == "" {
		return nil
	}

	client := &http.Client{Timeout: 10 * time.Second}
	payload, _ := json.Marshal(map[string]interface{}{
		"title":    title,
		"message":  message,
		"priority": 5,
	})
	req, err := http.NewRequest("POST", gotifyURL+"/message", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Token", token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("gotify returned status %d", resp.StatusCode)
	}
	return nil
}
