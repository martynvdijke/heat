package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
)

func umamiTestServer(t *testing.T) *app.Server {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE umami_settings (
			id INTEGER PRIMARY KEY,
			url TEXT NOT NULL DEFAULT '',
			website_id TEXT NOT NULL DEFAULT '',
			enabled INTEGER NOT NULL DEFAULT 0
		);
		INSERT INTO umami_settings (id, url, website_id, enabled)
		VALUES (1, 'https://umami.example.com', 'test-site', 1);
	`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return &app.Server{DB: db}
}

func TestUmamiMiddlewareUpdatesContentLengthAfterInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := umamiTestServer(t)

	r := gin.New()
	r.Use(UmamiMiddleware(s))
	r.GET("/", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte("<html><head></head><body>hello</body></html>"))
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	body := rr.Body.Bytes()
	if !strings.Contains(string(body), "umami.example.com/script.js") {
		t.Fatalf("umami script not injected: %s", body)
	}
	if got, want := rr.Header().Get("Content-Length"), strconv.Itoa(len(body)); got != want {
		t.Fatalf("Content-Length = %q, want %q (body length %d)", got, want, len(body))
	}
}

func TestUmamiMiddlewareRemovesContentLengthForEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := umamiTestServer(t)

	r := gin.New()
	r.Use(UmamiMiddleware(s))
	r.GET("/empty", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/empty", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rr.Code)
	}
	if got := rr.Header().Get("Content-Length"); got != "" {
		t.Fatalf("Content-Length = %q, want empty", got)
	}
}
