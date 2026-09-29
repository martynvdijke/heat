package telegram

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseInlineQuote(t *testing.T) {
	cases := []struct {
		in         string
		wantText   string
		wantAuthor string
	}{
		{"Hello — Alice", "Hello", "Alice"},
		{"Hello -- Bob", "Hello", "Bob"},
		{"Hello | Carol", "Hello", "Carol"},
		{"Hello|Dave", "Hello", "Dave"},
		{"Hello", "Hello", ""},
		{"  Hello   —  Alice  ", "Hello", "Alice"},
		{"A — B — C", "A", "B — C"},
		{"", "", ""},
	}
	for _, tc := range cases {
		text, author := parseInlineQuote(tc.in)
		if text != tc.wantText || author != tc.wantAuthor {
			t.Errorf("parseInlineQuote(%q) = (%q, %q), want (%q, %q)", tc.in, text, author, tc.wantText, tc.wantAuthor)
		}
	}
}

func TestInsertQuoteValidation(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}

	if _, err := b.insertQuote("   ", "Alice"); !errors.Is(err, errQuoteTextRequired) {
		t.Errorf("empty text error = %v, want errQuoteTextRequired", err)
	}
	if _, err := b.insertQuote(strings.Repeat("x", quoteTextMax+1), "Alice"); !errors.Is(err, errQuoteTextTooLong) {
		t.Errorf("long text error = %v, want errQuoteTextTooLong", err)
	}
	if _, err := b.insertQuote("ok", strings.Repeat("a", quoteAuthorMax+1)); !errors.Is(err, errQuoteAuthorTooLong) {
		t.Errorf("long author error = %v, want errQuoteAuthorTooLong", err)
	}

	id, err := b.insertQuote("  Nice race  ", " Alice ")
	if err != nil || id == 0 {
		t.Fatalf("insertQuote = (%d, %v)", id, err)
	}
	var text, author string
	if err := b.s.DB.QueryRow("SELECT text, author FROM quotes WHERE id = ?", id).Scan(&text, &author); err != nil {
		t.Fatalf("query: %v", err)
	}
	if text != "Nice race" || author != "Alice" {
		t.Errorf("stored (%q, %q), want trimmed values", text, author)
	}
}

func TestAddQuoteInline(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}

	out := b.addQuoteCommand(cmdContext{name: "/addquote", args: "What a lap — Alice", chatID: 7, firstName: "Dave"})
	if !strings.Contains(out, "Quote added") || !strings.Contains(out, "What a lap") || !strings.Contains(out, "Alice") {
		t.Errorf("inline quote reply: %q", out)
	}

	out = b.addQuoteCommand(cmdContext{name: "/addquote", args: "No author given", chatID: 8, firstName: "Dave"})
	if !strings.Contains(out, "Quote added") || !strings.Contains(out, "Dave") {
		t.Errorf("default author reply: %q", out)
	}
}

func TestGuidedQuoteFlow(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	c := cmdContext{name: "/addquote", chatID: 99, username: "dave", firstName: "Dave"}

	out := b.addQuoteCommand(c)
	if !strings.Contains(out, "quote text") {
		t.Fatalf("guided prompt: %q", out)
	}
	if pq, ok := b.quoteFlow(99); !ok || pq.step != quoteStepText {
		t.Fatalf("expected pending text step, got %+v, %v", pq, ok)
	}

	reply, consumed := b.handleQuoteText(c, "Team radio gold")
	if !consumed || !strings.Contains(reply, "author") || !strings.Contains(reply, "Dave") {
		t.Fatalf("author prompt: %q, consumed=%v", reply, consumed)
	}
	pq, ok := b.quoteFlow(99)
	if !ok || pq.step != quoteStepAuthor || pq.text != "Team radio gold" {
		t.Fatalf("expected pending author step, got %+v, %v", pq, ok)
	}

	reply, consumed = b.handleQuoteText(c, "Alice")
	if !consumed || !strings.Contains(reply, "Quote added") || !strings.Contains(reply, "Alice") {
		t.Fatalf("save reply: %q, consumed=%v", reply, consumed)
	}
	if _, ok := b.quoteFlow(99); ok {
		t.Error("flow should be cleared after saving")
	}
	var count int
	if err := b.s.DB.QueryRow("SELECT COUNT(*) FROM quotes WHERE text = 'Team radio gold' AND author = 'Alice'").Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("stored quotes = %d, want 1", count)
	}
}

func TestGuidedQuoteFlowSkip(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	c := cmdContext{name: "/addquote", chatID: 5, firstName: "Dave"}

	b.addQuoteCommand(c)
	if _, consumed := b.handleQuoteText(c, "Skip author test"); !consumed {
		t.Fatal("text step not consumed")
	}
	if out := b.skipQuoteAuthor(c); !strings.Contains(out, "Quote added") {
		t.Fatalf("skip reply: %q", out)
	}
	var author string
	if err := b.s.DB.QueryRow("SELECT author FROM quotes WHERE text = 'Skip author test'").Scan(&author); err != nil {
		t.Fatalf("query: %v", err)
	}
	if author != "Dave" {
		t.Errorf("author = %q, want Dave", author)
	}

	if out := b.skipQuoteAuthor(c); !strings.Contains(out, "No quote in progress") {
		t.Errorf("skip without flow: %q", out)
	}
}

func TestQuoteFlowCancelAndTimeout(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	c := cmdContext{name: "/addquote", chatID: 6}

	b.addQuoteCommand(c)
	if out := b.cancelQuoteCommand(c); !strings.Contains(out, "cancelled") {
		t.Fatalf("cancel reply: %q", out)
	}
	if _, ok := b.quoteFlow(6); ok {
		t.Error("cancel should clear the flow")
	}
	if out := b.cancelQuoteCommand(c); !strings.Contains(out, "Nothing to cancel") {
		t.Errorf("second cancel: %q", out)
	}

	b.addQuoteCommand(c)
	b.setQuoteFlow(6, pendingQuote{step: quoteStepText, expiresAt: time.Now().Add(-time.Second)})
	if _, ok := b.quoteFlow(6); ok {
		t.Error("expired flow should be dropped")
	}
	if _, consumed := b.handleQuoteText(c, "late text"); consumed {
		t.Error("expired flow should not consume messages")
	}
}

func TestHandleCommandAbortsPendingFlow(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	c := cmdContext{name: "/addquote", chatID: 3}

	b.addQuoteCommand(c)
	if _, ok := b.quoteFlow(3); !ok {
		t.Fatal("flow should be pending")
	}
	b.handleCommand(cmdContext{name: "/status", chatID: 3})
	if _, ok := b.quoteFlow(3); ok {
		t.Error("another command should abort the pending flow")
	}
}
