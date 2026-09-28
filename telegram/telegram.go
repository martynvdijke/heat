// Package telegram implements a self-contained Telegram bot for the HEAT
// racing app. It long-polls the Telegram Bot API in its own goroutine,
// answers commands by calling the app's own public HTTP API, pushes race
// results when asked to via Server.TelegramBroadcast, and sends upcoming-race
// reminders on a schedule.
//
// It deliberately has no third-party dependencies: the handful of Bot API
// methods it needs (getUpdates, sendMessage) are called with net/http, the
// same way package wled talks to WLED devices.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"heat/app"
	"heat/models"
)

const (
	tgAPIBase        = "https://api.telegram.org"
	pollTimeoutSecs  = 25
	idlePollInterval = 15 * time.Second
	errorBackoff     = 5 * time.Second
)

// Bot is the long-lived Telegram integration. One Bot is created at startup
// and its Run method occupies a single goroutine.
type Bot struct {
	s       *app.Server
	http    *http.Client
	baseURL string // this app's own public API, e.g. http://127.0.0.1:6270
	apiBase string // Telegram Bot API base

	mu            sync.Mutex
	offset        int64
	token         string
	sentReminders map[string]time.Time
}

// New builds a Bot bound to the given server.
func New(s *app.Server) *Bot {
	port := os.Getenv("PORT")
	if port == "" {
		port = "6270"
	}
	return &Bot{
		s: s,
		http: &http.Client{
			// Must exceed the getUpdates long-poll timeout.
			Timeout: (pollTimeoutSecs + 15) * time.Second,
		},
		baseURL:       "http://127.0.0.1:" + port,
		apiBase:       tgAPIBase,
		sentReminders: make(map[string]time.Time),
	}
}

// Run starts the poll and reminder loops and then services outbound push
// events until the broadcast channel is closed. Intended to be launched as a
// goroutine from main: `go telegram.New(server).Run()`.
func (b *Bot) Run() {
	go b.pollLoop()
	go b.reminderLoop()
	for evt := range b.s.TelegramBroadcast {
		b.handleEvent(evt)
	}
}

// LoadSettings reads the single-row telegram_settings configuration.
func LoadSettings(s *app.Server) (models.TelegramSettings, error) {
	var st models.TelegramSettings
	var enabled, notifyResults, notifyNext int
	err := s.DB.QueryRow(`
		SELECT id, COALESCE(bot_token, ''), COALESCE(enabled, 0),
			COALESCE(default_chat_id, ''), COALESCE(notify_results, 1),
			COALESCE(notify_next_race, 1), COALESCE(reminder_days, '7,1'),
			COALESCE(reminder_hour, 18)
		FROM telegram_settings WHERE id = 1`).
		Scan(&st.ID, &st.BotToken, &enabled, &st.DefaultChatID,
			&notifyResults, &notifyNext, &st.ReminderDays, &st.ReminderHour)
	if err != nil {
		return st, err
	}
	st.Enabled = enabled == 1
	st.NotifyResults = notifyResults == 1
	st.NotifyNextRace = notifyNext == 1
	return st, nil
}

func (b *Bot) logf(format string, args ...any) {
	if b.s != nil && b.s.Log != nil {
		b.s.Log.Infof("telegram", format, args...)
	}
}

func (b *Bot) warnf(format string, args ...any) {
	if b.s != nil && b.s.Log != nil {
		b.s.Log.Warnf("telegram", format, args...)
	}
}

// pollLoop long-polls Telegram and dispatches incoming commands. It re-reads
// settings every cycle so that enabling/disabling or changing the token takes
// effect without a restart.
func (b *Bot) pollLoop() {
	for {
		st, err := LoadSettings(b.s)
		if err != nil || !st.Enabled || st.BotToken == "" {
			time.Sleep(idlePollInterval)
			continue
		}

		b.mu.Lock()
		if b.token != st.BotToken {
			b.token = st.BotToken
			b.offset = 0 // a new token starts a new update stream
		}
		offset := b.offset
		b.mu.Unlock()

		updates, err := b.getUpdates(st.BotToken, offset)
		if err != nil {
			b.warnf("getUpdates failed: %v", err)
			time.Sleep(errorBackoff)
			continue
		}
		for _, u := range updates {
			b.mu.Lock()
			if u.UpdateID+1 > b.offset {
				b.offset = u.UpdateID + 1
			}
			b.mu.Unlock()
			if u.Message == nil {
				continue
			}
			b.handleMessage(st, u.Message)
		}
	}
}

// handleMessage parses and answers a single incoming Telegram message.
func (b *Bot) handleMessage(st models.TelegramSettings, m *tgMessage) {
	if m.Chat == nil {
		return
	}
	text := strings.TrimSpace(m.Text)
	if !strings.HasPrefix(text, "/") {
		return
	}
	cmd := text
	if i := strings.IndexByte(cmd, ' '); i >= 0 {
		cmd = cmd[:i]
	}
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i] // strip the @BotName suffix used in group chats
	}
	cmd = strings.ToLower(cmd)

	if reply := b.execute(cmd, m); reply != "" {
		b.sendMessage(st.BotToken, strconv.FormatInt(m.Chat.ID, 10), reply)
	}
}

// handleEvent services an outbound push request.
func (b *Bot) handleEvent(evt models.TelegramEvent) {
	st, err := LoadSettings(b.s)
	if err != nil || !st.Enabled || st.BotToken == "" {
		return
	}
	switch evt.Kind {
	case "test":
		if evt.ChatID != "" {
			b.sendMessage(st.BotToken, evt.ChatID, evt.Text)
		}
	case "race_saved":
		if !st.NotifyResults {
			return
		}
		b.broadcast(st, b.renderArchivedRace(evt.RaceID))
	case "round_final":
		if !st.NotifyResults {
			return
		}
		b.broadcast(st, b.renderLatestRace())
	}
}

// broadcast sends text to the default chat and every subscribed chat, once each.
func (b *Bot) broadcast(st models.TelegramSettings, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	seen := map[string]bool{}
	var targets []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		targets = append(targets, id)
	}
	add(st.DefaultChatID)
	for _, id := range b.subscriberChatIDs() {
		add(id)
	}
	for _, chatID := range targets {
		b.sendMessage(st.BotToken, chatID, text)
	}
}

// SendTest delivers a one-off message, used by the admin "send test" button.
func (b *Bot) SendTest(token, chatID, text string) error {
	var res tgResult
	if err := b.call(token, "sendMessage", map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}, &res); err != nil {
		return err
	}
	if !res.Ok {
		return fmt.Errorf("telegram: %s", res.Description)
	}
	return nil
}

func (b *Bot) sendMessage(token, chatID, text string) {
	if text == "" {
		return
	}
	var res tgResult
	err := b.call(token, "sendMessage", map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}, &res)
	if err != nil {
		b.warnf("sendMessage to %s failed: %v", chatID, err)
		return
	}
	if !res.Ok {
		b.warnf("sendMessage to %s rejected: %s", chatID, res.Description)
	}
}

func (b *Bot) getUpdates(token string, offset int64) ([]tgUpdate, error) {
	var res tgUpdates
	err := b.call(token, "getUpdates", map[string]any{
		"offset":          offset,
		"timeout":         pollTimeoutSecs,
		"allowed_updates": []string{"message"},
	}, &res)
	if err != nil {
		return nil, err
	}
	if !res.Ok {
		return nil, fmt.Errorf("telegram: %s", res.Description)
	}
	return res.Result, nil
}

// call performs a single Bot API request. It honours 429 retry_after by
// sleeping before returning so callers can simply retry.
func (b *Bot) call(token, method string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := b.apiBase + "/bot" + token + "/" + method
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusTooManyRequests {
		var rl struct {
			Parameters struct {
				RetryAfter int `json:"retry_after"`
			} `json:"parameters"`
		}
		_ = json.Unmarshal(data, &rl)
		wait := rl.Parameters.RetryAfter
		if wait <= 0 {
			wait = 5
		}
		if wait > 60 {
			wait = 60
		}
		time.Sleep(time.Duration(wait) * time.Second)
		return fmt.Errorf("telegram: rate limited, retry after %ds", wait)
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("telegram: decode %s response: %w", method, err)
		}
	}
	return nil
}

// apiGet fetches JSON from this app's own public API.
func (b *Bot) apiGet(path string, out any) error {
	resp, err := b.http.Get(b.baseURL + path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("api %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(out)
}

// subscriberChatIDs returns every chat that opted in via /subscribe.
func (b *Bot) subscriberChatIDs() []string {
	rows, err := b.s.DB.Query("SELECT chat_id FROM telegram_subscribers WHERE subscribed = 1")
	if err != nil {
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// setSubscription opts a chat in or out of pushes/reminders.
func (b *Bot) setSubscription(m *tgMessage, subscribed bool) error {
	username, firstName := "", ""
	if m.From != nil {
		username = m.From.Username
		firstName = m.From.FirstName
	}
	chatID := strconv.FormatInt(m.Chat.ID, 10)
	_, err := b.s.DB.Exec(`
		INSERT INTO telegram_subscribers (chat_id, username, first_name, subscribed, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(chat_id) DO UPDATE SET
			username = excluded.username,
			first_name = excluded.first_name,
			subscribed = excluded.subscribed`,
		chatID, username, firstName, boolToInt(subscribed))
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
