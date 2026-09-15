package ws

import (
	"bytes"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/pkg/logger"
)

// newTestManager builds a Manager backed by a throwaway app.Server with a real
// (in-memory) logger so broadcast error/eviction paths can be exercised.
func newTestManager(t *testing.T) *Manager {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	l := logger.New(db)
	t.Cleanup(l.Stop)

	srv := app.NewServer()
	srv.Log = l
	return NewManager(srv)
}

// TestBroadcastEvictsOnlyLaggingClient covers requirement 3.3/3.4: a client
// with a full send queue must not block delivery, and only that client is
// evicted while healthy clients still receive the message.
func TestBroadcastEvictsOnlyLaggingClient(t *testing.T) {
	m := newTestManager(t)

	healthy := &client{send: make(chan []byte, 4)}
	lagging := &client{send: make(chan []byte, 1)}
	lagging.send <- []byte("already queued") // fill to capacity

	m.clients[healthy] = true
	m.clients[lagging] = true

	done := make(chan struct{})
	go func() {
		m.broadcastToClients(map[string]string{"type": "test"})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcastToClients blocked on a lagging client")
	}

	if !m.clients[healthy] {
		t.Error("healthy client was evicted")
	}
	if m.clients[lagging] {
		t.Error("lagging client was not evicted")
	}

	select {
	case msg := <-healthy.send:
		if !bytes.Contains(msg, []byte(`"type":"test"`)) {
			t.Errorf("healthy client got unexpected payload: %s", msg)
		}
	default:
		t.Fatal("healthy client did not receive the broadcast")
	}
}

// TestTrySendRejectsAfterShutdown verifies an evicted/closed client can no
// longer enqueue messages (writer goroutine has been stopped).
func TestTrySendRejectsAfterShutdown(t *testing.T) {
	c := &client{send: make(chan []byte, 4)}
	if !c.trySend([]byte("ok")) {
		t.Fatal("expected trySend to succeed on open client")
	}
	c.shutdown()
	if c.trySend([]byte("nope")) {
		t.Error("expected trySend to fail on shut-down client")
	}
}
