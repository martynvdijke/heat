package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tgmodels "github.com/go-telegram/bot/models"
)

func flattenKeyboard(kb *tgmodels.InlineKeyboardMarkup) []tgmodels.InlineKeyboardButton {
	var out []tgmodels.InlineKeyboardButton
	for _, row := range kb.InlineKeyboard {
		out = append(out, row...)
	}
	return out
}

func TestNavKeyboard(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}

	wantLabels := map[string]string{
		callbackNavPrefix + "/results":   "🏁 Results",
		callbackNavPrefix + "/standings": "🏆 Standings",
		callbackNavPrefix + "/next":      "📅 Next",
		callbackNavPrefix + "/quote":     "💬 Quote",
		callbackSubscribe:                "🔔 Subscribe",
	}
	flat := flattenKeyboard(b.navKeyboard(10))
	if len(flat) != len(wantLabels) {
		t.Fatalf("keyboard has %d buttons, want %d", len(flat), len(wantLabels))
	}
	for _, btn := range flat {
		want, ok := wantLabels[btn.CallbackData]
		if !ok {
			t.Errorf("unexpected callback data %q", btn.CallbackData)
			continue
		}
		if btn.Text != want {
			t.Errorf("button %q text = %q, want %q", btn.CallbackData, btn.Text, want)
		}
	}

	if err := b.setSubscription(10, "u", "U", true); err != nil {
		t.Fatalf("setSubscription: %v", err)
	}
	for _, btn := range flattenKeyboard(b.navKeyboard(10)) {
		if btn.CallbackData == callbackSubscribe && btn.Text != "🔕 Unsubscribe" {
			t.Errorf("subscribed button text = %q, want Unsubscribe", btn.Text)
		}
	}
}

func TestHandleCallbackData(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(apiSummary{
			LatestRace: &apiRace{Name: "Spa", Results: []apiRaceResult{{RacerName: "Carol", Position: 1, Points: 25}}},
		})
	}))
	defer api.Close()

	b := &Bot{s: testServer(t), http: api.Client(), baseURL: api.URL, sentReminders: map[string]time.Time{}}
	c := cmdContext{chatID: 5, username: "dave", firstName: "Dave"}

	reply, ok := b.handleCallbackData(c, callbackNavPrefix+"/results")
	if !ok || !strings.Contains(reply, "Carol") {
		t.Errorf("nav callback = (%q, %v)", reply, ok)
	}
	if _, ok := b.handleCallbackData(c, callbackNavPrefix+"/nope"); ok {
		t.Error("unknown nav target should not resolve")
	}
	if _, ok := b.handleCallbackData(c, "nonsense"); ok {
		t.Error("unknown callback data should not resolve")
	}

	reply, ok = b.handleCallbackData(c, callbackSubscribe)
	if !ok || !strings.Contains(reply, "Subscribed") || !b.isSubscribed(5) {
		t.Errorf("subscribe toggle = (%q, %v), subscribed=%v", reply, ok, b.isSubscribed(5))
	}
	reply, ok = b.handleCallbackData(c, callbackSubscribe)
	if !ok || !strings.Contains(reply, "Unsubscribed") || b.isSubscribed(5) {
		t.Errorf("unsubscribe toggle = (%q, %v), subscribed=%v", reply, ok, b.isSubscribed(5))
	}
}

func TestHandleCallbackWithoutClient(t *testing.T) {
	b := &Bot{s: testServer(t), sentReminders: map[string]time.Time{}}
	b.handleCallback(&tgmodels.CallbackQuery{ID: "1"}) // must not panic without a client
}
