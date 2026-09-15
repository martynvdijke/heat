package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"heat/app"
	"heat/ws"
)

// newAuthTestServer builds a fresh Server/Manager on the shared test DB so its
// broadcast channels are not contended by the globally-started test manager.
func newAuthTestServer(t *testing.T) (string, *ws.Manager, *app.Server) {
	t.Helper()

	srv := app.NewServer()
	srv.DB = testServer.DB
	srv.Log = testServer.Log
	srv.Upgrader = testServer.Upgrader

	m := ws.NewManager(srv)
	go m.BroadcastFlags()
	go m.BroadcastManager()

	r := gin.New()
	r.GET("/ws", m.HandleWebSocket)
	ts := httptest.NewServer(r)
	t.Cleanup(ts.Close)

	return "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws", m, srv
}

// Regression test for the CI failure where Firefox opened the controller
// WebSocket over ::1 while the session had been created over 127.0.0.1.
func TestWSAcceptsSessionAcrossLoopbackFamilies(t *testing.T) {
	ln, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("ipv6 loopback unavailable: %v", err)
	}

	srv := app.NewServer()
	srv.DB = testServer.DB
	srv.Log = testServer.Log
	srv.Upgrader = testServer.Upgrader
	m := ws.NewManager(srv)

	r := gin.New()
	r.GET("/ws", m.HandleWebSocket)
	ts := httptest.NewUnstartedServer(r)
	ts.Listener.Close()
	ts.Listener = ln
	ts.Start()
	t.Cleanup(ts.Close)

	id := fmt.Sprintf("sess-loop-%d", time.Now().UnixNano())
	srv.SessionStoreMu.Lock()
	srv.SessionStore[id] = app.SessionInfo{Expiry: time.Now().Add(time.Hour).Unix(), IP: "127.0.0.1"}
	srv.SessionStoreMu.Unlock()

	wsDial(t, "ws://"+ln.Addr().String()+"/ws", id)
}

func addTestSession(t *testing.T, srv *app.Server) string {
	t.Helper()
	id := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	srv.SessionStoreMu.Lock()
	srv.SessionStore[id] = app.SessionInfo{Expiry: time.Now().Add(time.Hour).Unix()}
	srv.SessionStoreMu.Unlock()
	return id
}

func createTestRacer(t *testing.T, name string) int {
	t.Helper()
	r, err := testServer.Ent.Racer.Create().SetName(name).Save(context.Background())
	if err != nil {
		t.Fatalf("create racer: %v", err)
	}
	return r.ID
}

func createTestPlayerToken(t *testing.T, racerID int) string {
	t.Helper()
	token := fmt.Sprintf("tok-%d-%d", racerID, time.Now().UnixNano())
	if _, err := testServer.DB.Exec(
		"INSERT INTO player_sessions (racer_id, token, device_name, last_seen, created_at) VALUES (?, ?, '', '', '')",
		racerID, token,
	); err != nil {
		t.Fatalf("insert player session: %v", err)
	}
	t.Cleanup(func() { testServer.DB.Exec("DELETE FROM player_sessions WHERE token = ?", token) })
	return token
}

func wsDial(t *testing.T, url string, cookie string, subprotocols ...string) *websocket.Conn {
	t.Helper()
	header := http.Header{}
	if cookie != "" {
		header.Add("Cookie", "session="+cookie)
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second, Subprotocols: subprotocols}
	conn, resp, err := dialer.Dial(url, header)
	if err != nil {
		status := 0
		if resp != nil {
			status = resp.StatusCode
		}
		t.Fatalf("dial: %v (status %d)", err, status)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func wsRead(t *testing.T, c *websocket.Conn, timeout time.Duration) map[string]any {
	t.Helper()
	c.SetReadDeadline(time.Now().Add(timeout))
	_, data, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal %s: %v", data, err)
	}
	return m
}

func wsWaitFor(t *testing.T, c *websocket.Conn, typ string, timeout time.Duration) map[string]any {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %q", typ)
		}
		if m := wsRead(t, c, time.Until(deadline)); m["type"] == typ {
			return m
		}
	}
}

// wsExpectNone fails if a message of the given type arrives within d.
func wsExpectNone(t *testing.T, c *websocket.Conn, typ string, d time.Duration) {
	t.Helper()
	c.SetReadDeadline(time.Now().Add(d))
	for {
		_, data, err := c.ReadMessage()
		if err != nil {
			return // timeout/close: the forbidden message never arrived
		}
		var m map[string]any
		if json.Unmarshal(data, &m) == nil && m["type"] == typ {
			t.Fatalf("unexpected %q message: %s", typ, data)
		}
	}
}

func wsSend(t *testing.T, c *websocket.Conn, payload string) {
	t.Helper()
	if err := c.WriteMessage(websocket.TextMessage, []byte(payload)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// wsPayload returns the sequenced envelope's payload object.
func wsPayload(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	p, ok := m["payload"].(map[string]any)
	if !ok {
		t.Fatalf("message has no payload object: %v", m)
	}
	return p
}

// TestWSAuthHandshake covers handshake classification: no credentials is
// spectator, valid session is controller, valid token is player, and invalid
// credentials are rejected with 401.
func TestWSAuthHandshake(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	// No credentials -> spectator (accepted).
	spectator := wsDial(t, url, "")
	wsSend(t, spectator, `{"type":"flag","flag":"safety","state":"on"}`)
	if m := wsWaitFor(t, spectator, "error", 2*time.Second); m["code"] != float64(403) {
		t.Fatalf("spectator should be denied flags, got %v", m)
	}

	// Invalid session -> 401.
	header := http.Header{}
	header.Add("Cookie", "session=does-not-exist")
	if _, resp, err := (&websocket.Dialer{}).Dial(url, header); err == nil {
		t.Fatal("invalid session was accepted")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid session, got %v", resp)
	}

	// Valid session -> controller.
	session := addTestSession(t, srv)
	controller := wsDial(t, url, session)
	wsSend(t, controller, `{"type":"flag","flag":"safety","state":"on"}`)
	if m := wsWaitFor(t, controller, "flag", 2*time.Second); wsPayload(t, m)["flag"] != "safety" {
		t.Fatalf("controller flag not echoed: %v", m)
	}

	// Valid player token -> player.
	racerID := createTestRacer(t, "Auth Racer")
	token := createTestPlayerToken(t, racerID)
	player := wsDial(t, url, "", "heat", "heat.token."+token)
	wsSend(t, player, `{"type":"self_service","racer_id":0,"lap":1}`)
	if m := wsWaitFor(t, player, "error", 2*time.Second); m["code"] != float64(403) {
		t.Fatalf("player with wrong racer_id should be denied, got %v", m)
	}

	// Invalid player token -> 401.
	header = http.Header{}
	if _, resp, err := (&websocket.Dialer{Subprotocols: []string{"heat", "heat.token.bogus"}}).Dial(url, header); err == nil {
		t.Fatal("invalid player token was accepted")
	} else if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid player token, got %v", resp)
	}
}

// TestWSInboundAuthorization covers spoof rejection: spectators cannot emit
// flags and players cannot emit self_service for another racer.
func TestWSInboundAuthorization(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	controller := wsDial(t, url, addTestSession(t, srv))

	racer7 := createTestRacer(t, "Racer Seven")
	token7 := createTestPlayerToken(t, racer7)
	player7 := wsDial(t, url, "", "heat", "heat.token."+token7)

	// Spectator flag is rejected.
	spectator := wsDial(t, url, "")
	wsSend(t, spectator, `{"type":"flag","flag":"red","state":"on"}`)
	if m := wsWaitFor(t, spectator, "error", 2*time.Second); m["code"] != float64(403) {
		t.Fatalf("expected 403 for spectator flag, got %v", m)
	}

	// Player cannot spoof another racer.
	wsSend(t, player7, `{"type":"self_service","racer_id":999,"lap":1}`)
	if m := wsWaitFor(t, player7, "error", 2*time.Second); m["code"] != float64(403) {
		t.Fatalf("expected 403 for spoofed self_service, got %v", m)
	}

	// Player's own self_service is accepted (controller sees telemetry).
	wsSend(t, player7, `{"type":"self_service","action":"gear","racer_id":`+fmt.Sprint(racer7)+`,"gear":3}`)
	if m := wsWaitFor(t, controller, "self_service", 2*time.Second); m["topic"] != "telemetry" {
		t.Fatalf("expected telemetry self_service, got %v", m)
	}

	// The rejected spectator flag was never re-broadcast (spectator's last read).
	wsExpectNone(t, spectator, "flag", 300*time.Millisecond)
}

// TestWSTopicFiltering covers topic subscription: defaults preserve public TV
// behaviour, and a client subscribed only to flags does not get telemetry.
func TestWSTopicFiltering(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	// Default spectator must keep receiving public topics (defaults preserve TV).
	spectator := wsDial(t, url, "")

	racer7 := createTestRacer(t, "Topic Racer 7")
	token7 := createTestPlayerToken(t, racer7)
	player7 := wsDial(t, url, "", "heat", "heat.token."+token7)

	racer9 := createTestRacer(t, "Topic Racer 9")
	token9 := createTestPlayerToken(t, racer9)
	player9 := wsDial(t, url, "", "heat", "heat.token."+token9)

	// player9 opts out of telemetry.
	wsSend(t, player9, `{"type":"subscribe","topics":["flags"]}`)

	wsSend(t, player7, `{"type":"self_service","action":"turbo","racer_id":`+fmt.Sprint(racer7)+`}`)

	// player7 gets its own telemetry; player9 (unsubscribed) must not.
	if m := wsWaitFor(t, player7, "self_service", 2*time.Second); wsPayload(t, m)["racer_id"] != float64(racer7) {
		t.Fatalf("unexpected self_service for player7: %v", m)
	}
	wsExpectNone(t, player9, "self_service", 300*time.Millisecond)

	// A controller subscribed to flags still gets flags; the default spectator
	// also receives it, proving public defaults are intact.
	controller := wsDial(t, url, addTestSession(t, srv))
	wsSend(t, controller, `{"type":"flag","flag":"yellow","state":"on"}`)
	if m := wsWaitFor(t, controller, "flag", 2*time.Second); wsPayload(t, m)["flag"] != "yellow" {
		t.Fatalf("controller did not receive flags: %v", m)
	}
	if m := wsWaitFor(t, spectator, "flag", 2*time.Second); wsPayload(t, m)["flag"] != "yellow" {
		t.Fatalf("spectator did not receive default public flags: %v", m)
	}
}

// TestWSTargetedNotify covers targeted delivery: only the bound racer's
// connections receive a notify, and notify_ack is relayed to controllers.
func TestWSTargetedNotify(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	controller := wsDial(t, url, addTestSession(t, srv))

	racer7 := createTestRacer(t, "Notify Racer 7")
	player7 := wsDial(t, url, "", "heat", "heat.token."+createTestPlayerToken(t, racer7))

	racer9 := createTestRacer(t, "Notify Racer 9")
	player9 := wsDial(t, url, "", "heat", "heat.token."+createTestPlayerToken(t, racer9))

	wsSend(t, controller, `{"type":"notify","racer_id":`+fmt.Sprint(racer7)+`,"id":"n1","message":"Box this lap"}`)

	if m := wsWaitFor(t, player7, "notify", 2*time.Second); wsPayload(t, m)["id"] != "n1" || wsPayload(t, m)["message"] != "Box this lap" {
		t.Fatalf("player7 got unexpected notify: %v", m)
	}
	wsExpectNone(t, player9, "notify", 300*time.Millisecond)

	wsSend(t, player7, `{"type":"notify_ack","id":"n1"}`)
	if m := wsWaitFor(t, controller, "notify_ack", 2*time.Second); wsPayload(t, m)["id"] != "n1" {
		t.Fatalf("controller did not receive notify_ack: %v", m)
	}
}

// TestWSPresence covers presence broadcast to subscribers only.
func TestWSPresence(t *testing.T) {
	url, _, srv := newAuthTestServer(t)

	controllerA := wsDial(t, url, addTestSession(t, srv))
	if m := wsWaitFor(t, controllerA, "presence", 2*time.Second); wsPayload(t, m)["event"] != "join" {
		t.Fatalf("controller did not receive its own presence join: %v", m)
	}

	// A controller that opts out of presence must not see later changes.
	controllerC := wsDial(t, url, addTestSession(t, srv))
	if m := wsWaitFor(t, controllerC, "presence", 2*time.Second); wsPayload(t, m)["event"] != "join" {
		t.Fatalf("controllerC did not receive its own join: %v", m)
	}
	if m := wsWaitFor(t, controllerA, "presence", 2*time.Second); wsPayload(t, m)["event"] != "join" || wsPayload(t, m)["role"] != "controller" {
		t.Fatalf("controllerA did not receive controllerC's join: %v", m)
	}
	wsSend(t, controllerC, `{"type":"subscribe","topics":["flags"]}`)

	racerID := createTestRacer(t, "Presence Racer")
	player := wsDial(t, url, "", "heat", "heat.token."+createTestPlayerToken(t, racerID))

	if m := wsWaitFor(t, controllerA, "presence", 2*time.Second); wsPayload(t, m)["event"] != "join" || wsPayload(t, m)["role"] != "player" {
		t.Fatalf("controllerA did not receive player presence join: %v", m)
	}
	wsExpectNone(t, controllerC, "presence", 300*time.Millisecond)

	player.Close()
	if m := wsWaitFor(t, controllerA, "presence", 2*time.Second); wsPayload(t, m)["event"] != "leave" {
		t.Fatalf("controllerA did not receive presence leave: %v", m)
	}
}
