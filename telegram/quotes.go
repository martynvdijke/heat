package telegram

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	quoteTextMax     = 500
	quoteAuthorMax   = 100
	quoteFlowTimeout = 5 * time.Minute
)

var (
	errQuoteTextRequired  = errors.New("quote text is required")
	errQuoteTextTooLong   = errors.New("quote text is too long")
	errQuoteAuthorTooLong = errors.New("quote author is too long")
)

type quoteStep int

const (
	quoteStepText quoteStep = iota
	quoteStepAuthor
)

// pendingQuote tracks a guided /addquote conversation for one chat.
type pendingQuote struct {
	step      quoteStep
	text      string
	expiresAt time.Time
}

// quoteFlow returns the pending guided submission for a chat, if any, dropping
// entries that have exceeded quoteFlowTimeout.
func (b *Bot) quoteFlow(chatID int64) (pendingQuote, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	pq, ok := b.pendingQuotes[chatID]
	if !ok {
		return pendingQuote{}, false
	}
	if time.Now().After(pq.expiresAt) {
		delete(b.pendingQuotes, chatID)
		return pendingQuote{}, false
	}
	return pq, true
}

func (b *Bot) setQuoteFlow(chatID int64, pq pendingQuote) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.pendingQuotes == nil {
		b.pendingQuotes = map[int64]pendingQuote{}
	}
	b.pendingQuotes[chatID] = pq
}

func (b *Bot) clearQuoteFlow(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.pendingQuotes, chatID)
}

// insertQuote validates and stores a quote directly in the web app's quotes
// table, returning the new quote's ID. Text is stored raw and escaped when it
// is rendered back into a message.
func (b *Bot) insertQuote(text, author string) (int64, error) {
	text = strings.TrimSpace(text)
	author = strings.TrimSpace(author)
	if text == "" {
		return 0, errQuoteTextRequired
	}
	if utf8.RuneCountInString(text) > quoteTextMax {
		return 0, errQuoteTextTooLong
	}
	if utf8.RuneCountInString(author) > quoteAuthorMax {
		return 0, errQuoteAuthorTooLong
	}
	res, err := b.s.DB.Exec("INSERT INTO quotes (text, author) VALUES (?, ?)", text, author)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// saveQuote inserts a quote and renders the confirmation preview.
func (b *Bot) saveQuote(text, author string) string {
	id, err := b.insertQuote(text, author)
	if err != nil {
		b.warnf("quote insert failed: %v", err)
		return "⚠️ " + quoteValidationMessage(err)
	}
	return quotePreview(id, text, author)
}

func quoteValidationMessage(err error) string {
	switch {
	case errors.Is(err, errQuoteTextRequired):
		return "A quote needs some text — try again."
	case errors.Is(err, errQuoteTextTooLong):
		return fmt.Sprintf("That quote is too long (max %d characters).", quoteTextMax)
	case errors.Is(err, errQuoteAuthorTooLong):
		return fmt.Sprintf("That author name is too long (max %d characters).", quoteAuthorMax)
	default:
		return "Could not save the quote right now."
	}
}

func quotePreview(id int64, text, author string) string {
	var sb strings.Builder
	sb.WriteString("✅ <b>Quote added!</b>\n")
	sb.WriteString(divider + "\n")
	sb.WriteString("💬 <i>" + escapeHTML(strings.TrimSpace(text)) + "</i>\n")
	if author = strings.TrimSpace(author); author != "" {
		sb.WriteString("— " + escapeHTML(author) + "\n")
	}
	fmt.Fprintf(&sb, "🆔 #%d · live in the web app", id)
	return sb.String()
}

// addQuoteCommand handles /addquote: inline when arguments are present,
// otherwise it starts the guided flow.
func (b *Bot) addQuoteCommand(c cmdContext) string {
	args := strings.TrimSpace(c.args)
	if args == "" {
		b.setQuoteFlow(c.chatID, pendingQuote{step: quoteStepText, expiresAt: time.Now().Add(quoteFlowTimeout)})
		return "💬 <b>New quote</b>\n" + divider + "\nSend me the quote text (or /cancel)."
	}
	text, author := parseInlineQuote(args)
	if author == "" {
		author = displayName(c.username, c.firstName)
	}
	return b.saveQuote(text, author)
}

// parseInlineQuote splits "/addquote <text> — <author>" on the first em dash,
// double hyphen or pipe separator. With no separator the whole argument is text.
func parseInlineQuote(args string) (string, string) {
	for _, sep := range []string{" — ", " -- ", " | ", "|"} {
		if i := strings.Index(args, sep); i >= 0 {
			return strings.TrimSpace(args[:i]), strings.TrimSpace(args[i+len(sep):])
		}
	}
	return strings.TrimSpace(args), ""
}

// displayName is the author credited when the sender does not give one.
func displayName(username, firstName string) string {
	if firstName != "" {
		return firstName
	}
	if username != "" {
		return username
	}
	return "Commentator"
}

// handleQuoteText feeds a plain message into a pending guided submission. It
// returns the reply and whether the message belonged to the flow.
func (b *Bot) handleQuoteText(c cmdContext, text string) (string, bool) {
	pq, ok := b.quoteFlow(c.chatID)
	if !ok {
		return "", false
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return "⚠️ That looks empty — send the quote text or /cancel.", true
	}

	switch pq.step {
	case quoteStepText:
		if utf8.RuneCountInString(text) > quoteTextMax {
			return fmt.Sprintf("⚠️ That's too long (max %d characters). Try a shorter one or /cancel.", quoteTextMax), true
		}
		pq.step = quoteStepAuthor
		pq.text = text
		pq.expiresAt = time.Now().Add(quoteFlowTimeout)
		b.setQuoteFlow(c.chatID, pq)
		name := displayName(c.username, c.firstName)
		return fmt.Sprintf("✍️ Got it. Now send the author, or /skip to credit <b>%s</b>.", escapeHTML(name)), true
	case quoteStepAuthor:
		if utf8.RuneCountInString(text) > quoteAuthorMax {
			return fmt.Sprintf("⚠️ That author name is too long (max %d characters). Try again or /cancel.", quoteAuthorMax), true
		}
		b.clearQuoteFlow(c.chatID)
		return b.saveQuote(pq.text, text), true
	}
	return "", false
}

// skipQuoteAuthor credits the sender's Telegram name as the author.
func (b *Bot) skipQuoteAuthor(c cmdContext) string {
	pq, ok := b.quoteFlow(c.chatID)
	if !ok {
		return "🤷 No quote in progress. Start one with /addquote."
	}
	if pq.step != quoteStepAuthor {
		return "✍️ I still need the quote text first — or /cancel."
	}
	b.clearQuoteFlow(c.chatID)
	return b.saveQuote(pq.text, displayName(c.username, c.firstName))
}

// cancelQuoteCommand aborts a pending guided submission.
func (b *Bot) cancelQuoteCommand(c cmdContext) string {
	if _, ok := b.quoteFlow(c.chatID); !ok {
		return "🤷 Nothing to cancel. Start a quote with /addquote."
	}
	b.clearQuoteFlow(c.chatID)
	return "❌ Quote cancelled."
}
