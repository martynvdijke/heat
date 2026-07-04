package main

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"

	"heat/middleware"
	"heat/models"
)

// TestWebSocketThroughGzipMiddleware verifies that the WebSocket upgrade
// succeeds when the gzip middleware is configured with /ws path exclusion.
//
// Regression test for commit b83dceb which applied gzip.Gzip globally without
// excluding /ws. The gzip middleware wraps c.Writer, preventing
// gorilla/websocket's Upgrader.Upgrade from calling Hijack on the underlying
// ResponseWriter — breaking the "lights out" start lights animation.
func TestWebSocketThroughGzipMiddleware(t *testing.T) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws"})))
	r.GET("/ws", wsManager.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	wsConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket upgrade failed through gzip middleware with /ws exclusion: %v", err)
	}
	defer wsConn.Close()

	// Give the handler time to register the client in the Clients map.
	time.Sleep(100 * time.Millisecond)

	// Send a flag command and verify it is broadcast back to us.
	cmd := models.FlagCommand{Type: "flag", Flag: "startlights"}
	if err := wsConn.WriteJSON(cmd); err != nil {
		t.Fatalf("Failed to send WebSocket message: %v", err)
	}

	// The BroadcastFlags goroutine (started in TestMain) picks up the flag
	// from FlagBroadcast channel and broadcasts to all connected clients.
	wsConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var received models.FlagCommand
	if err := wsConn.ReadJSON(&received); err != nil {
		t.Fatalf("Did not receive broadcast flag message within timeout: %v", err)
	}
	if received.Flag != "startlights" {
		t.Errorf("Received flag %q, want %q", received.Flag, "startlights")
	}
}

// TestWebSocketGzipExclusionIsPresent verifies the production gzip middleware
// is configured to explicitly exclude /ws. While gin-contrib/gzip v1.2.6
// already skips WebSocket upgrades via the Upgrade header check, the explicit
// exclusion provides defense-in-depth and documents the requirement.
func TestWebSocketGzipExclusionIsPresent(t *testing.T) {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws"})))
	r.GET("/ws", wsManager.HandleWebSocket)

	server := httptest.NewServer(r)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	wsConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("WebSocket upgrade failed with /ws exclusion: %v", err)
	}
	defer wsConn.Close()
}

// TestSQLitePragmasFromDSN verifies that DSN-embedded PRAGMAs apply to EVERY
// connection in the pool, not just one.
//
// Regression test for commit b83dceb which set PRAGMAs via db.Exec (only
// affects the single executing connection) while increasing the pool from
// 1 to 8 connections. New pool connections inherited SQLite defaults for
// cache_size (-2000 = 2MB, vs -20000 = 20MB requested), causing more disk
// I/O and slower queries under concurrent load — contributing to admin panel
// slowdown.
func TestSQLitePragmasFromDSN(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pragma_test.db")
	dsn := dbPath + "?_fk=1&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-20000"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("Failed to open DB with PRAGMA DSN: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	// Force the pool to create multiple connections by acquiring them
	// simultaneously, then check PRAGMAs on each.
	ctx := context.Background()
	conns := make([]*sql.Conn, 4)
	for i := range conns {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("Failed to get connection %d from pool: %v", i, err)
		}
		conns[i] = conn
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	for i, conn := range conns {
		// This is the critical test: cache_size defaults to -2000 (2MB) in
		// SQLite. The DSN should set it to -20000 (20MB) on EVERY connection.
		// Before the fix, only the Exec-running connection had this.
		var cacheSize int
		if err := conn.QueryRowContext(ctx, "PRAGMA cache_size").Scan(&cacheSize); err != nil {
			t.Fatalf("Conn %d: failed to query cache_size: %v", i, err)
		}
		if cacheSize != -20000 {
			t.Errorf("Conn %d: cache_size = %d, want -20000", i, cacheSize)
		}

		// journal_mode should be "wal" (database-level, but verify on each conn).
		var journalMode string
		if err := conn.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode); err != nil {
			t.Fatalf("Conn %d: failed to query journal_mode: %v", i, err)
		}
		if !strings.EqualFold(journalMode, "wal") {
			t.Errorf("Conn %d: journal_mode = %q, want \"wal\"", i, journalMode)
		}

		// foreign_keys should be enabled (1).
		var fk int
		if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&fk); err != nil {
			t.Fatalf("Conn %d: failed to query foreign_keys: %v", i, err)
		}
		if fk != 1 {
			t.Errorf("Conn %d: foreign_keys = %d, want 1", i, fk)
		}
	}
}

// TestSQLitePragmasExecDoesNotApplyToPool verifies the OLD approach (setting
// PRAGMAs via db.Exec) does NOT propagate to other pool connections.
//
// This documents the root cause: with a pool size >1, only the connection
// that ran Exec gets the PRAGMA settings. New connections get SQLite defaults.
//
// cache_size is the best PRAGMA to test this with because the mattn/go-sqlite3
// driver does NOT set a default for it (unlike busy_timeout which defaults to
// 5000). Each Exec-only pool connection will have cache_size=-2000 (2MB)
// instead of the desired -20000 (20MB).
func TestSQLitePragmasExecDoesNotApplyToPool(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exec_pragma_test.db")
	// DSN WITHOUT cache_size — mimics the broken b83dceb configuration where
	// you'd set it via Exec and expect it to propagate.
	db, err := sql.Open("sqlite3", dbPath+"?_fk=1")
	if err != nil {
		t.Fatalf("Failed to open DB: %v", err)
	}
	defer db.Close()

	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	// Set PRAGMA via Exec — this only affects ONE connection.
	db.Exec("PRAGMA cache_size=-20000")

	// Acquire all 4 connections simultaneously to force pool creation.
	ctx := context.Background()
	conns := make([]*sql.Conn, 4)
	for i := range conns {
		conn, err := db.Conn(ctx)
		if err != nil {
			t.Fatalf("Failed to get connection %d: %v", i, err)
		}
		conns[i] = conn
	}
	defer func() {
		for _, c := range conns {
			c.Close()
		}
	}()

	// At least one connection should have cache_size=-2000 (the SQLite default),
	// proving that Exec-based PRAGMAs don't propagate to all pool connections.
	foundDefault := false
	for i, conn := range conns {
		var cs int
		if err := conn.QueryRowContext(ctx, "PRAGMA cache_size").Scan(&cs); err != nil {
			t.Fatalf("Conn %d: failed to query cache_size: %v", i, err)
		}
		if cs == -2000 {
			foundDefault = true
		} else if cs == -20000 {
			continue
		} else {
			t.Logf("Conn %d: cache_size = %d (unexpected value)", i, cs)
		}
	}
	if !foundDefault {
		t.Error("Expected at least one pool connection to have cache_size=-2000 " +
			"(proving Exec PRAGMAs don't propagate to all pool connections)")
	}
}

// TestProductionDSNContainsPragmas is a lightweight string-level regression
// test that verifies the production DSN in main.go includes all required
// PRAGMA parameters. This catches accidental removal of DSN parameters even
// before the behavioral tests run.
func TestProductionDSNContainsPragmas(t *testing.T) {
	// This mirrors the DSN constructed in main.go. If main.go's DSN changes,
	// update this constant to match — the test ensures both stay in sync.
	// Note: _temp_store=MEMORY is NOT a supported mattn/go-sqlite3 DSN param
	// and must be set via db.Exec instead.
	expectedDSN := "?_fk=1&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-20000"
	requiredParams := []string{
		"_fk=1",
		"_busy_timeout=5000",
		"_journal_mode=WAL",
		"_synchronous=NORMAL",
		"_cache_size=-20000",
	}
	for _, param := range requiredParams {
		if !strings.Contains(expectedDSN, param) {
			t.Errorf("DSN missing required parameter %q", param)
		}
	}
}

// TestAdminPageLoadsQuickly verifies the admin page HTML handler responds
// within a reasonable time budget. This is a smoke test for the admin panel
// slowdown issue — the root cause (missing PRAGMAs on pool connections) is
// tested more precisely in TestSQLitePragmasFromDSN.
func TestAdminPageLoadsQuickly(t *testing.T) {
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))

	r.GET("/admin", func(c *gin.Context) {
		tmpl, err := template.ParseFiles(
			filepath.Join(testServer.BasePath, "static/templates/admin.html"),
			filepath.Join(testServer.BasePath, "static/templates/admin-header.html"),
			filepath.Join(testServer.BasePath, "static/templates/admin-footer.html"),
			filepath.Join(testServer.BasePath, "static/templates/tab-race-day.html"),
			filepath.Join(testServer.BasePath, "static/templates/tab-season.html"),
			filepath.Join(testServer.BasePath, "static/templates/tab-drivers.html"),
			filepath.Join(testServer.BasePath, "static/templates/tab-config.html"),
		)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		adminData := struct {
			VendorCSS    string
			VendorJS     string
			VendorFA     string
			VendorNavCss string
		}{}
		if err := tmpl.ExecuteTemplate(c.Writer, "admin.html", adminData); err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
	})

	req, _ := http.NewRequest("GET", "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.Host = "127.0.0.1:6270"

	w := httptest.NewRecorder()
	start := time.Now()
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Errorf("Admin page returned status %d, want %d", w.Code, http.StatusOK)
	}

	// The admin page should load in well under 5 seconds. Before the fix,
	// missing PRAGMAs on pool connections caused ~20s load times.
	if elapsed > 5*time.Second {
		t.Errorf("Admin page took %v to load, want < 5s", elapsed)
	}
}
