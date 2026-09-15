package ws

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"heat/app"
	"heat/models"
)

// Timing and size defaults. Tests may override these fields on Manager.
const (
	defaultWriteWait      = 10 * time.Second
	defaultPongWait       = 60 * time.Second
	defaultPingPeriod     = 30 * time.Second
	defaultMaxMessageSize = 64 * 1024 // 64 KiB
	sendQueueSize         = 256
)

// Role classifies a connection for authorization and default topic delivery.
type Role string

const (
	RoleController Role = "controller"
	RolePlayer     Role = "player"
	RoleSpectator  Role = "spectator"
)

// WebSocket subprotocol carrying the player token: `heat.token.<token>`.
// Browsers cannot set custom headers on the WS handshake, so the token travels
// as a subprotocol; `clientProtocol` is echoed back on success.
const (
	playerTokenPrefix = "heat.token."
	clientProtocol    = "heat"
)

// publicTopics are subscribed for every role on connect, preserving the
// pre-auth broadcast behaviour that TV/spectator/pitboard pages rely on.
var publicTopics = []string{
	"flags", "racers", "commentary", "weather", "race_state",
	"game_mechanics", "sound", "lap_replay",
}

func defaultTopics(role Role) map[string]bool {
	topics := make(map[string]bool, len(publicTopics)+4)
	for _, t := range publicTopics {
		topics[t] = true
	}
	switch role {
	case RoleController:
		topics["telemetry"] = true
		topics["presence"] = true
		topics["race_radio"] = true
	case RolePlayer:
		topics["telemetry"] = true
	}
	return topics
}

// topicAllowed reports whether a role may subscribe to a topic. Public topics
// are open to all; sensitive topics are role-restricted.
func topicAllowed(role Role, topic string) bool {
	for _, t := range publicTopics {
		if t == topic {
			return true
		}
	}
	switch topic {
	case "telemetry":
		return role == RoleController || role == RolePlayer
	case "presence":
		return role == RoleController
	case "race_radio":
		return role == RoleController || role == RolePlayer
	}
	return false
}

// ConnMeta is the identity and subscription state for a connection.
type ConnMeta struct {
	mu       sync.RWMutex
	ID       string
	Role     Role
	RacerID  int
	topics   map[string]bool
	LastSeen time.Time
}

func (mt *ConnMeta) hasTopic(topic string) bool {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.topics[topic]
}

func (mt *ConnMeta) setTopics(topics map[string]bool) {
	mt.mu.Lock()
	mt.topics = topics
	mt.mu.Unlock()
}

// client is a single WebSocket connection. Only the per-connection writer
// goroutine writes to conn; everyone else enqueues on send. A lagging client
// whose send queue is full is evicted instead of blocking the broadcast path.
type client struct {
	conn *websocket.Conn
	send chan []byte

	mu     sync.Mutex
	closed bool
}

func (c *client) trySend(msg []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

// shutdown closes the send queue (stopping the writer) and the connection.
// Safe to call multiple times.
func (c *client) shutdown() {
	c.mu.Lock()
	if !c.closed {
		c.closed = true
		close(c.send)
	}
	c.mu.Unlock()
	if c.conn != nil {
		c.conn.Close()
	}
}

func (c *client) writePump(pingPeriod, writeWait time.Duration) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

type Manager struct {
	S *app.Server

	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64

	mu      sync.RWMutex
	clients map[*client]*ConnMeta
}

func NewManager(s *app.Server) *Manager {
	return &Manager{
		S:              s,
		WriteWait:      defaultWriteWait,
		PongWait:       defaultPongWait,
		PingPeriod:     defaultPingPeriod,
		MaxMessageSize: defaultMaxMessageSize,
		clients:        make(map[*client]*ConnMeta),
	}
}

// ClientCount reports the number of live connections (used by tests/metrics).
func (m *Manager) ClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func newConnID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "conn"
	}
	return hex.EncodeToString(b)
}

// authenticate classifies an upgrade request. ok=false means the request
// carried invalid/expired credentials and must be rejected with 401.
func (m *Manager) authenticate(c *gin.Context) (*ConnMeta, bool) {
	meta := &ConnMeta{ID: newConnID(), LastSeen: time.Now()}

	sessionID := ""
	for _, cookie := range c.Request.Cookies() {
		if cookie.Name == "session" {
			sessionID = cookie.Value
			break
		}
	}

	token := ""
	for _, proto := range websocket.Subprotocols(c.Request) {
		if strings.HasPrefix(proto, playerTokenPrefix) {
			token = strings.TrimPrefix(proto, playerTokenPrefix)
			break
		}
	}

	switch {
	case sessionID != "":
		if !m.S.ValidateSession(sessionID, c.ClientIP()) {
			return nil, false
		}
		meta.Role = RoleController
	case token != "":
		racerID, ok := m.S.RacerIDForToken(token)
		if !ok {
			return nil, false
		}
		meta.Role = RolePlayer
		meta.RacerID = racerID
	default:
		meta.Role = RoleSpectator
	}

	meta.topics = defaultTopics(meta.Role)
	return meta, true
}

func selectSubprotocol(r *http.Request) string {
	for _, proto := range websocket.Subprotocols(r) {
		if proto == clientProtocol {
			return clientProtocol
		}
	}
	return ""
}

func (m *Manager) registerClient(c *client, meta *ConnMeta) {
	m.mu.Lock()
	m.clients[c] = meta
	n := len(m.clients)
	m.mu.Unlock()
	m.S.Log.Infof("ws", "New %s client connected. Total clients: %d", meta.Role, n)
	m.deliver("presence", presenceMessage("join", meta), nil)
}

func (m *Manager) removeClient(c *client) {
	m.mu.Lock()
	meta, ok := m.clients[c]
	delete(m.clients, c)
	m.mu.Unlock()
	if ok {
		m.deliver("presence", presenceMessage("leave", meta), nil)
	}
	c.shutdown()
}

func presenceMessage(event string, meta *ConnMeta) map[string]any {
	msg := map[string]any{
		"type":          "presence",
		"topic":         "presence",
		"event":         event,
		"role":          meta.Role,
		"connection_id": meta.ID,
	}
	if meta.RacerID != 0 {
		msg["racer_id"] = meta.RacerID
	}
	return msg
}

func (m *Manager) HandleWebSocket(c *gin.Context) {
	meta, ok := m.authenticate(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	hdr := http.Header{}
	if proto := selectSubprotocol(c.Request); proto != "" {
		hdr.Set("Sec-WebSocket-Protocol", proto)
	}

	conn, err := m.S.Upgrader.Upgrade(c.Writer, c.Request, hdr)
	if err != nil {
		m.S.Log.Errorf("ws", "Error upgrading WebSocket: %v", err)
		return
	}

	cl := &client{conn: conn, send: make(chan []byte, sendQueueSize)}
	m.registerClient(cl, meta)
	go cl.writePump(m.PingPeriod, m.WriteWait)

	defer m.removeClient(cl)
	m.readPump(cl, meta)
}

func (m *Manager) readPump(cl *client, meta *ConnMeta) {
	cl.conn.SetReadLimit(m.MaxMessageSize)
	cl.conn.SetReadDeadline(time.Now().Add(m.PongWait))
	cl.conn.SetPongHandler(func(string) error {
		cl.conn.SetReadDeadline(time.Now().Add(m.PongWait))
		return nil
	})

	for {
		_, msgBytes, err := cl.conn.ReadMessage()
		if err != nil {
			m.S.Log.Infof("ws", "Client disconnected: %v", err)
			return
		}

		meta.mu.Lock()
		meta.LastSeen = time.Now()
		meta.mu.Unlock()

		var msg map[string]any
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}
		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}
		m.handleMessage(cl, meta, msgType, msgBytes)
	}
}

func (m *Manager) handleMessage(cl *client, meta *ConnMeta, msgType string, msgBytes []byte) {
	switch msgType {
	case "subscribe":
		var req struct {
			Topics []string `json:"topics"`
		}
		if err := json.Unmarshal(msgBytes, &req); err != nil {
			return
		}
		topics := make(map[string]bool, len(req.Topics))
		for _, t := range req.Topics {
			if topicAllowed(meta.Role, t) {
				topics[t] = true
			}
		}
		meta.setTopics(topics)
	case "flag":
		if meta.Role != RoleController {
			m.sendError(cl, 403, "forbidden: flag")
			return
		}
		var cmd models.FlagCommand
		if err := json.Unmarshal(msgBytes, &cmd); err == nil {
			app.TrySend(m.S, m.S.FlagBroadcast, cmd)
		}
	case "lap_update":
		if meta.Role != RoleController {
			m.sendError(cl, 403, "forbidden: lap_update")
			return
		}
		var frame models.LapReplayFrame
		if err := json.Unmarshal(msgBytes, &frame); err == nil {
			app.TrySend(m.S, m.S.LapReplayBroadcast, frame)
		}
	case "weather_update":
		if meta.Role != RoleController {
			m.sendError(cl, 403, "forbidden: weather_update")
			return
		}
		var wc models.WeatherCondition
		if err := json.Unmarshal(msgBytes, &wc); err == nil {
			app.TrySend(m.S, m.S.WeatherBroadcast, wc)
		}
	case "self_service":
		if meta.Role != RolePlayer {
			m.sendError(cl, 403, "forbidden: self_service")
			return
		}
		var action models.SelfServiceAction
		if err := json.Unmarshal(msgBytes, &action); err != nil {
			return
		}
		if action.RacerID != meta.RacerID {
			m.sendError(cl, 403, "forbidden: racer mismatch")
			return
		}
		m.BroadcastSelfService(action)
	case "notify":
		if meta.Role != RoleController {
			m.sendError(cl, 403, "forbidden: notify")
			return
		}
		var n struct {
			RacerID int    `json:"racer_id"`
			ID      string `json:"id"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(msgBytes, &n); err != nil {
			return
		}
		m.notifyRacer(n.RacerID, n.ID, n.Message)
	case "notify_ack":
		var a struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(msgBytes, &a); err != nil {
			return
		}
		m.deliverToRole(RoleController, map[string]any{
			"type":     "notify_ack",
			"id":       a.ID,
			"racer_id": meta.RacerID,
		})
	}
}

func (m *Manager) sendError(cl *client, code int, message string) {
	data, err := json.Marshal(map[string]any{"type": "error", "code": code, "message": message})
	if err != nil {
		return
	}
	cl.trySend(data)
}

// withTopic marshals a protocol struct and stamps a topic onto the envelope.
func withTopic(topic string, msg any) map[string]any {
	b, err := json.Marshal(msg)
	if err != nil {
		return map[string]any{"type": topic, "topic": topic}
	}
	out := map[string]any{}
	if err := json.Unmarshal(b, &out); err != nil {
		return map[string]any{"type": topic, "topic": topic}
	}
	out["topic"] = topic
	return out
}

// deliver enqueues msg to every client subscribed to topic. allow, when
// non-nil, further restricts delivery (e.g. telemetry to the owning player).
// Clients whose queue is full are evicted so they cannot stall the broadcast.
func (m *Manager) deliver(topic string, msg any, allow func(*ConnMeta) bool) {
	data, err := json.Marshal(msg)
	if err != nil {
		m.S.Log.Errorf("ws", "Error marshalling broadcast: %v", err)
		return
	}

	m.mu.RLock()
	var lagging []*client
	for c, meta := range m.clients {
		if !meta.hasTopic(topic) {
			continue
		}
		if allow != nil && !allow(meta) {
			continue
		}
		if !c.trySend(data) {
			lagging = append(lagging, c)
		}
	}
	m.mu.RUnlock()

	for _, c := range lagging {
		m.S.Log.Warnf("ws", "Client send queue full; evicting lagging client")
		m.removeClient(c)
	}
}

func (m *Manager) deliverToRole(role Role, msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	m.mu.RLock()
	var lagging []*client
	for c, meta := range m.clients {
		if meta.Role != role {
			continue
		}
		if !c.trySend(data) {
			lagging = append(lagging, c)
		}
	}
	m.mu.RUnlock()
	for _, c := range lagging {
		m.removeClient(c)
	}
}

// notifyRacer delivers a notification only to connections bound to racerID.
func (m *Manager) notifyRacer(racerID int, id, message string) {
	data, err := json.Marshal(map[string]any{
		"type":     "notify",
		"id":       id,
		"racer_id": racerID,
		"message":  message,
	})
	if err != nil {
		return
	}
	m.mu.RLock()
	var lagging []*client
	for c, meta := range m.clients {
		if meta.Role != RolePlayer || meta.RacerID != racerID {
			continue
		}
		if !c.trySend(data) {
			lagging = append(lagging, c)
		}
	}
	m.mu.RUnlock()
	for _, c := range lagging {
		m.removeClient(c)
	}
}

func (m *Manager) BroadcastManager() {
	for racers := range m.S.Broadcast {
		m.deliver("racers", racers, nil)
	}
}

func (m *Manager) BroadcastFlags() {
	for cmd := range m.S.FlagBroadcast {
		m.deliver("flags", withTopic("flags", cmd), nil)
	}
}

func (m *Manager) BroadcastGameMechanics() {
	for update := range m.S.GameMechanicsBroadcast {
		m.deliver("game_mechanics", withTopic("game_mechanics", update), nil)
	}
}

func (m *Manager) BroadcastWeather() {
	for wc := range m.S.WeatherBroadcast {
		m.deliver("weather", map[string]any{
			"type":          "weather_update",
			"topic":         "weather",
			"id":            wc.ID,
			"race_id":       wc.RaceID,
			"condition":     wc.Condition,
			"lap_start":     wc.LapStart,
			"lap_end":       wc.LapEnd,
			"grip_modifier": wc.GripModifier,
		}, nil)
	}
}

func (m *Manager) BroadcastLapReplay() {
	for frame := range m.S.LapReplayBroadcast {
		m.deliver("lap_replay", withTopic("lap_replay", frame), nil)
	}
}

func (m *Manager) BroadcastSound() {
	for cmd := range m.S.SoundBroadcast {
		m.deliver("sound", withTopic("sound", cmd), nil)
	}
}

func (m *Manager) BroadcastRaceRadio() {
	for msg := range m.S.RaceRadioBroadcast {
		racerID := msg.RacerID
		m.deliver("race_radio", map[string]any{
			"type":       "race_radio",
			"topic":      "race_radio",
			"id":         msg.ID,
			"racer_id":   racerID,
			"racer_name": msg.RacerName,
			"message":    msg.Message,
			"timestamp":  msg.Timestamp,
		}, func(meta *ConnMeta) bool {
			return meta.Role != RolePlayer || racerID == 0 || meta.RacerID == racerID
		})
	}
}

func (m *Manager) BroadcastCommentary() {
	for entry := range m.S.CommentaryBroadcast {
		m.deliver("commentary", map[string]any{
			"type":         "commentary",
			"topic":        "commentary",
			"id":           entry.ID,
			"race_id":      entry.RaceID,
			"lap":          entry.Lap,
			"racer_id":     entry.RacerID,
			"racer_name":   entry.RacerName,
			"message":      entry.Message,
			"template_key": entry.TemplateKey,
			"created_at":   entry.CreatedAt,
		}, nil)
	}
}

func (m *Manager) BroadcastSelfService(action models.SelfServiceAction) {
	msg := map[string]any{
		"type":     "self_service",
		"topic":    "telemetry",
		"action":   action.Type,
		"racer_id": action.RacerID,
		"lap":      action.Lap,
		"gear":     action.Gear,
		"stress":   action.Stress,
		"turbo":    action.TurboUsed,
	}
	m.deliver("telemetry", msg, func(meta *ConnMeta) bool {
		return meta.Role != RolePlayer || meta.RacerID == action.RacerID
	})
}

func (m *Manager) BroadcastRacers() {
	rows, err := m.S.DB.Query("SELECT r.id, r.name, r.profile_picture, r.car_color, r.car_name, r.points, r.rank, r.position, COALESCE(r.team_id, 0), COALESCE(t.name, ''), COALESCE(t.color, '') FROM racers r LEFT JOIN teams t ON r.team_id = t.id ORDER BY r.rank ASC")
	if err != nil {
		m.S.Log.Errorf("ws", "Error fetching racers for broadcast: %v", err)
		return
	}
	defer rows.Close()

	var racers []models.Racer
	for rows.Next() {
		var r models.Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position, &r.TeamID, &r.TeamName, &r.TeamColor)
		if err != nil {
			m.S.Log.Errorf("ws", "Error scanning racer for broadcast: %v", err)
			return
		}
		racers = append(racers, r)
	}
	app.TrySend(m.S, m.S.Broadcast, racers)
}
