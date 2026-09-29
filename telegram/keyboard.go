package telegram

import (
	"context"
	"strconv"
	"strings"

	tgbot "github.com/go-telegram/bot"
	tgmodels "github.com/go-telegram/bot/models"
)

const (
	callbackSubscribe = "sub:toggle"
	callbackNavPrefix = "nav:"
)

// navKeyboard is the standard navigation keyboard attached to the main views.
func (b *Bot) navKeyboard(chatID int64) *tgmodels.InlineKeyboardMarkup {
	subLabel := "🔔 Subscribe"
	if b.isSubscribed(chatID) {
		subLabel = "🔕 Unsubscribe"
	}
	return &tgmodels.InlineKeyboardMarkup{InlineKeyboard: [][]tgmodels.InlineKeyboardButton{
		{
			{Text: "🏁 Results", CallbackData: callbackNavPrefix + "/results"},
			{Text: "🏆 Standings", CallbackData: callbackNavPrefix + "/standings"},
			{Text: "📅 Next", CallbackData: callbackNavPrefix + "/next"},
		},
		{
			{Text: "💬 Quote", CallbackData: callbackNavPrefix + "/quote"},
			{Text: subLabel, CallbackData: callbackSubscribe},
		},
	}}
}

func (b *Bot) isSubscribed(chatID int64) bool {
	var n int
	if err := b.s.DB.QueryRow(
		"SELECT COUNT(*) FROM telegram_subscribers WHERE chat_id = ? AND subscribed = 1",
		strconv.FormatInt(chatID, 10)).Scan(&n); err != nil {
		return false
	}
	return n > 0
}

// handleCallback answers an inline-button tap and sends the resulting view.
func (b *Bot) handleCallback(cq *tgmodels.CallbackQuery) {
	client := b.currentClient()
	if client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), sendTimeout)
	if _, err := client.AnswerCallbackQuery(ctx, &tgbot.AnswerCallbackQueryParams{CallbackQueryID: cq.ID}); err != nil {
		b.warnf("answerCallbackQuery failed: %v", err)
	}
	cancel()

	if cq.Message.Message == nil {
		return
	}
	chatID := cq.Message.Message.Chat.ID
	c := cmdContext{chatID: chatID, username: cq.From.Username, firstName: cq.From.FirstName}
	reply, ok := b.handleCallbackData(c, cq.Data)
	if !ok || reply == "" {
		return
	}
	b.sendWithKeyboard(chatID, reply, b.navKeyboard(chatID))
}

// handleCallbackData resolves a callback payload into a reply. Split out from
// handleCallback so it can be tested without a live Telegram client.
func (b *Bot) handleCallbackData(c cmdContext, data string) (string, bool) {
	switch {
	case data == callbackSubscribe:
		subscribed := !b.isSubscribed(c.chatID)
		if err := b.setSubscription(c.chatID, c.username, c.firstName, subscribed); err != nil {
			b.warnf("subscription toggle failed: %v", err)
			return "⚠️ Could not update your subscription right now.", true
		}
		if subscribed {
			return "✅ <b>Subscribed!</b> You'll now get race results and upcoming-race reminders.", true
		}
		return "🔕 <b>Unsubscribed.</b> You won't receive any more pushes.", true
	case strings.HasPrefix(data, callbackNavPrefix):
		name := strings.TrimPrefix(data, callbackNavPrefix)
		if _, ok := findCommand(name); !ok {
			return "", false
		}
		c.name = name
		return b.execute(c), true
	}
	return "", false
}
