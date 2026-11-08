package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/models"
	"heat/pkg/logger"
)

func otelSettingsTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE otel_settings (
			id INTEGER PRIMARY KEY,
			endpoint TEXT NOT NULL DEFAULT '',
			traces_enabled INTEGER NOT NULL DEFAULT 0,
			metrics_enabled INTEGER NOT NULL DEFAULT 0,
			logs_enabled INTEGER NOT NULL DEFAULT 0
		)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	log := logger.New(db)
	t.Cleanup(log.Stop)
	return New(&app.Server{DB: db, Log: log})
}

// The mobile app reads this route without a session, so it is registered on
// the public router (no CSRF/auth middleware) in main.go. This exercises the
// same wiring: a bare GET must succeed and expose the documented keys.
func TestGetOTelSettingsPublicNoAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := otelSettingsTestHandler(t)
	if _, err := h.S.DB.Exec(
		`INSERT INTO otel_settings (id, endpoint, traces_enabled, metrics_enabled, logs_enabled) VALUES (1, 'http://collector:4318', 1, 0, 1)`,
	); err != nil {
		t.Fatalf("insert: %v", err)
	}

	r := gin.New()
	r.GET("/api/otel-settings", h.GetOTelSettings)

	req, _ := http.NewRequest("GET", "/api/otel-settings", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"id", "endpoint", "traces_enabled", "metrics_enabled", "logs_enabled"} {
		if _, ok := body[key]; !ok {
			t.Errorf("missing key %q in response %v", key, body)
		}
	}

	var got models.OTelSettings
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode model: %v", err)
	}
	if got.Endpoint != "http://collector:4318" || !got.TracesEnabled || got.MetricsEnabled || !got.LogsEnabled {
		t.Errorf("unexpected settings: %+v", got)
	}
}

func TestGetOTelSettingsDefaultWhenUnseeded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := otelSettingsTestHandler(t)

	r := gin.New()
	r.GET("/api/otel-settings", h.GetOTelSettings)

	req, _ := http.NewRequest("GET", "/api/otel-settings", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var got models.OTelSettings
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.ID != 1 || got.Endpoint != "" || got.TracesEnabled {
		t.Errorf("unexpected default settings: %+v", got)
	}
}
