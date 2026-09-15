package main

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"heat/ws"
)

// newWSTestServer starts an httptest server serving HandleWebSocket for a
// freshly built manager (caller may tune timings/limits) and returns the
// dialable ws:// URL and the manager.
func newWSTestServer(t *testing.T, tune func(*ws.Manager)) (string, *ws.Manager) {
	t.Helper()
	m := ws.NewManager(testServer)
	if tune != nil {
		tune(m)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.GET("/ws", m.HandleWebSocket)
	server := httptest.NewServer(r)
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http") + "/ws", m
}

// TestWebSocketHeartbeatReapsDeadClient covers requirement 3.1: a peer that
// never answers pings is dropped once the read deadline expires.
func TestWebSocketHeartbeatReapsDeadClient(t *testing.T) {
	url, m := newWSTestServer(t, func(m *ws.Manager) {
		m.PongWait = 200 * time.Millisecond
		m.PingPeriod = 50 * time.Millisecond
	})

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	// Deliberately never call ReadMessage: gorilla only answers pings while a
	// read is in flight, so the server receives no pong and must reap us.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if m.ClientCount() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("dead client was not reaped by heartbeat (ClientCount=%d)", m.ClientCount())
}

// TestWebSocketRejectsOversizedMessage covers requirement 3.2: a frame larger
// than MaxMessageSize tears the connection down instead of being buffered.
func TestWebSocketRejectsOversizedMessage(t *testing.T) {
	url, _ := newWSTestServer(t, func(m *ws.Manager) {
		m.MaxMessageSize = 1024
	})

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, bytes.Repeat([]byte("a"), 2048)); err != nil {
		t.Fatalf("write oversized frame: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Fatal("expected connection to close after oversized message, read succeeded")
	}
}
