package wled

import (
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/models"
	"heat/pkg/logger"
)

// newTestServer builds a minimal app.Server backed by an in-memory sqlite DB
// with the wled_settings table created.
func newTestServer(t *testing.T) *app.Server {
	t.Helper()
	s := app.NewServer()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	s.DB = db
	if _, err := db.Exec(`CREATE TABLE wled_settings (id INTEGER PRIMARY KEY, url TEXT, enabled INTEGER, presets TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}
	s.Log = logger.New(db)
	return s
}

func TestSync(t *testing.T) {
	var hits int
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := newTestServer(t)
	c := &Client{S: s, HTTP: srv.Client()}

	// Disabled → no request.
	s.DB.Exec(`INSERT INTO wled_settings (id, url, enabled, presets) VALUES (1, ?, 0, '{}')`, srv.URL)
	c.Sync(models.FlagCommand{Flag: "red", State: "on"})
	if hits != 0 {
		t.Fatalf("expected no request when disabled, got %d", hits)
	}

	// Enabled but not armed → no request.
	s.DB.Exec(`UPDATE wled_settings SET enabled = 1 WHERE id = 1`)
	c.Sync(models.FlagCommand{Flag: "red", State: "on"})
	if hits != 0 {
		t.Fatalf("expected no request when disarmed, got %d", hits)
	}

	// Enabled + armed → posts payload.
	s.WLEDArmed.Store(true)
	c.Sync(models.FlagCommand{Flag: "red", State: "on"})
	if hits != 1 {
		t.Fatalf("expected 1 request, got %d", hits)
	}
	if gotBody["on"] != true {
		t.Fatalf("expected on=true, got %v", gotBody)
	}

	// Preset configured → posts ps.
	s.DB.Exec(`UPDATE wled_settings SET presets = '{"red": 7}' WHERE id = 1`)
	c.Sync(models.FlagCommand{Flag: "red", State: "on"})
	if gotBody["ps"] != float64(7) {
		t.Fatalf("expected ps=7, got %v", gotBody)
	}
}

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
