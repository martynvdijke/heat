package wled

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"heat/models"
)

func TestBuildPayload(t *testing.T) {
	// Preset wins when configured.
	p := buildPayload(models.FlagCommand{Flag: "safety", State: "on"}, map[string]int{"safety": 3})
	if p["ps"] != 3 {
		t.Fatalf("expected preset ps=3, got %v", p)
	}

	// Falls back to default color when no preset.
	p = buildPayload(models.FlagCommand{Flag: "blue", State: "on"}, map[string]int{})
	seg := p["seg"].([]map[string]any)
	col := seg[0]["col"].([][3]int)[0]
	if col != [3]int{0, 0, 255} {
		t.Fatalf("expected blue color, got %v", col)
	}

	// Off state maps to clear.
	p = buildPayload(models.FlagCommand{Flag: "red", State: "off"}, map[string]int{})
	seg = p["seg"].([]map[string]any)
	col = seg[0]["col"].([][3]int)[0]
	if col != [3]int{0, 255, 0} {
		t.Fatalf("expected clear color, got %v", col)
	}

	// Unknown flag with no preset yields nil.
	if buildPayload(models.FlagCommand{Flag: "bogus", State: "on"}, map[string]int{}) != nil {
		t.Fatal("expected nil for unknown flag")
	}
}

func TestSendTestPosts(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := &Client{HTTP: srv.Client()}
	if err := c.SendTest(srv.URL); err != nil {
		t.Fatalf("SendTest: %v", err)
	}
	if gotPath != "/json/state" {
		t.Fatalf("expected /json/state, got %s", gotPath)
	}
	if gotBody["on"] != true {
		t.Fatalf("expected on=true, got %v", gotBody)
	}
}
