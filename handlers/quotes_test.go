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
)

func quotesTestHandler(t *testing.T) *Handler {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE quotes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			text TEXT NOT NULL,
			author TEXT NOT NULL DEFAULT 'Commentator',
			created_at TEXT NOT NULL DEFAULT ''
		)`); err != nil {
		t.Fatalf("schema: %v", err)
	}
	return New(&app.Server{DB: db})
}

func TestGetRandomQuoteUsesRandomOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := quotesTestHandler(t)
	for _, q := range []string{"Alpha", "Beta", "Gamma"} {
		if _, err := h.S.DB.Exec("INSERT INTO quotes (text, author) VALUES (?, 'Tester')", q); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	r := gin.New()
	r.GET("/api/quote/random", h.GetRandomQuote)

	seen := map[string]bool{}
	for i := 0; i < 40; i++ {
		req, _ := http.NewRequest("GET", "/api/quote/random", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d", rr.Code)
		}
		var q models.Quote
		if err := json.Unmarshal(rr.Body.Bytes(), &q); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if q.Text == "" {
			t.Fatal("empty quote text")
		}
		seen[q.Text] = true
	}
	if len(seen) < 2 {
		t.Errorf("expected the random endpoint to surface multiple quotes, only saw %v", seen)
	}
}

func TestGetRandomQuoteFallbackWhenEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := quotesTestHandler(t)

	r := gin.New()
	r.GET("/api/quote/random", h.GetRandomQuote)

	req, _ := http.NewRequest("GET", "/api/quote/random", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	var q models.Quote
	if err := json.Unmarshal(rr.Body.Bytes(), &q); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if q.Text == "" || q.Author == "" {
		t.Errorf("fallback quote incomplete: %+v", q)
	}
}
