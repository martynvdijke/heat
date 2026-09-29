// Package telegram implements a Telegram bot for the HEAT racing app using
// the github.com/go-telegram/bot client library. The bot runs in its own
// goroutine, answers commands by calling the app's own public HTTP API,
// pushes race results when asked to via Server.TelegramBroadcast, and sends
// upcoming-race reminders on a schedule.
package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"

	"heat/app"
	"heat/models"
)

const (
	idlePollInterval = 15 * time.Second
	sendTimeout      = 20 * time.Second
)

// Bot is the long-lived Telegram integration. One Bot is created at startup
// and its Run method occupies a single goroutine.
type Bot struct {
	s            *app.Server
	http         *http.Client
	baseURL      string // this app's own public API, e.g. http://127.0.0.1:6270
	apiServerURL string // optional Bot API override (tests / self-hosted)

	mu            sync.Mutex
	client        *tgbot.Bot
	cancel        context.CancelFunc
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
		s:             s,
		http:          &http.Client{Timeout: sendTimeout},
		baseURL:       "http://127.0.0.1:" + port,
		sentReminders: make(map[string]time.Time),
	}
}

// Run starts the settings supervisor and reminder loop, then services
// outbound push events until the broadcast channel is closed. Intended to be
// launched as a goroutine from main: `go telegram.New(server).Run()`.
func (b *Bot) Run() {
	go b.supervise()
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

// supervise keeps the running bot in sync with the stored settings: it starts
// the client when enabled, stops it when disabled, and restarts it when the
// token changes — all without an app restart.
func (b *Bot) supervise() {
	for {
		st, err := LoadSettings(b.s)
		switch {
		case err != nil || !st.Enabled || st.BotToken == "":
			b.stopClient()
		case b.currentToken() != st.BotToken:
			b.startClient(st.BotToken)
		}
		time.Sleep(idlePollInterval)
	}
}

func (b *Bot) currentToken() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.token
}

func (b *Bot) currentClient() *tgbot.Bot {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.client
}

func (b *Bot) startClient(token string) {
	b.stopClient()
	opts := []tgbot.Option{
		tgbot.WithDefaultHandler(b.onUpdate),
		tgbot.WithAllowedUpdates(tgbot.AllowedUpdates{tgmodels.AllowedUpdateMessage}),
	}
	if b.apiServerURL != "" {
		opts = append(opts, tgbot.WithServerURL(b.apiServerURL))
	}
	client, err := tgbot.New(token, opts...)
	if err != nil {
		b.warnf("failed to start Telegram bot: %v", err)
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	b.mu.Lock()
	b.client = client
	b.cancel = cancel
	b.token = token
	b.mu.Unlock()
	b.logf("Telegram bot started")
	go client.Start(ctx)
}

func (b *Bot) stopClient() {
	b.mu.Lock()
	cancel := b.cancel
	b.cancel = nil
	b.client = nil
	b.token = ""
	b.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// onUpdate is the library's default update handler. It only reacts to
// commands and replies privately to the sender.
func (b *Bot) onUpdate(_ context.Context, _ *tgbot.Bot, update *tgmodels.Update) {
	if update == nil || update.Message == nil {
		return
	}
	msg := update.Message

	raw := strings.TrimSpace(msg.Text)
	if !strings.HasPrefix(raw, "/") {
		return
	}
	body := strings.TrimPrefix(raw, "/")
	cmd, args := body, ""
	if i := strings.IndexAny(body, " \t\n"); i >= 0 {
		cmd = body[:i]
		args = strings.TrimSpace(body[i+1:])
	}
	if i := strings.IndexByte(cmd, '@'); i >= 0 {
		cmd = cmd[:i] // strip @BotName
	}
	cmd = "/" + strings.ToLower(cmd)

	username, firstName := "", ""
	if msg.From != nil {
		username = msg.From.Username
		firstName = msg.From.FirstName
	}

	if reply := b.execute(cmd, args, msg.Chat.ID, username, firstName); reply != "" {
		b.send(msg.Chat.ID, reply)
	}
}

func (b *Bot) CreateQuote(text, author string) error {
	if b.s == nil || b.s.Ent == nil {
		return fmt.Errorf("ent client unavailable")
	}
	_, err := b.s.Ent.Quote.Create().SetText(text).SetAuthor(author).Save(context.Background())
	return err
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
			b.send(evt.ChatID, evt.Text)
		}
	case "race_saved":
		if !st.NotifyResults {
			return
		}
		b.broadcast(b.renderArchivedRace(evt.RaceID))
	case "round_final":
		if !st.NotifyResults {
			return
		}
		b.broadcast(b.renderLatestRace())
	}
}

// send delivers one HTML message through the running client, if any.
func (b *Bot) send(chatID any, text string) {
	if strings.TrimSpace(text) == "" {
		return
	}
	client := b.currentClient()
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	if _, err := client.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: tgmodels.ParseModeHTML,
	}); err != nil {
		b.warnf("sendMessage to %v failed: %v", chatID, err)
	}
}

// broadcast sends text to the default chat and every subscribed chat, once each.
func (b *Bot) broadcast(text string) {
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
	if st, err := LoadSettings(b.s); err == nil {
		add(st.DefaultChatID)
	}
	for _, id := range b.subscriberChatIDs() {
		add(id)
	}
	for _, chatID := range targets {
		b.send(chatID, text)
	}
}

// SendTest delivers a one-off message, used by the admin "send test" button.
// It works even when the persistent client is not running.
func (b *Bot) SendTest(token, chatID, text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	defer cancel()
	var opts []tgbot.Option
	if b.apiServerURL != "" {
		opts = append(opts, tgbot.WithServerURL(b.apiServerURL))
	}
	client, err := tgbot.New(token, opts...)
	if err != nil {
		return err
	}
	_, err = client.SendMessage(ctx, &tgbot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: tgmodels.ParseModeHTML,
	})
	return err
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
func (b *Bot) setSubscription(chatID int64, username, firstName string, subscribed bool) error {
	id := strconv.FormatInt(chatID, 10)
	_, err := b.s.DB.Exec(`
		INSERT INTO telegram_subscribers (chat_id, username, first_name, subscribed, created_at)
		VALUES (?, ?, ?, ?, datetime('now'))
		ON CONFLICT(chat_id) DO UPDATE SET
			username = excluded.username,
			first_name = excluded.first_name,
			subscribed = excluded.subscribed`,
		id, username, firstName, boolToInt(subscribed))
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
