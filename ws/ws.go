package ws

import (
	"encoding/json"
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
	clients map[*client]bool
}

func NewManager(s *app.Server) *Manager {
	return &Manager{
		S:              s,
		WriteWait:      defaultWriteWait,
		PongWait:       defaultPongWait,
		PingPeriod:     defaultPingPeriod,
		MaxMessageSize: defaultMaxMessageSize,
		clients:        make(map[*client]bool),
	}
}

// ClientCount reports the number of live connections (used by tests/metrics).
func (m *Manager) ClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

func (m *Manager) registerClient(c *client) {
	m.mu.Lock()
	m.clients[c] = true
	n := len(m.clients)
	m.mu.Unlock()
	m.S.Log.Infof("ws", "New client connected. Total clients: %d", n)
}

func (m *Manager) removeClient(c *client) {
	m.mu.Lock()
	delete(m.clients, c)
	m.mu.Unlock()
	c.shutdown()
}

func (m *Manager) HandleWebSocket(c *gin.Context) {
	conn, err := m.S.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.S.Log.Errorf("ws", "Error upgrading WebSocket: %v", err)
		return
	}

	cl := &client{conn: conn, send: make(chan []byte, sendQueueSize)}
	m.registerClient(cl)
	go cl.writePump(m.PingPeriod, m.WriteWait)

	defer m.removeClient(cl)
	m.readPump(cl)
}

func (m *Manager) readPump(cl *client) {
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

		var msg map[string]any
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "flag":
			var cmd models.FlagCommand
			if err := json.Unmarshal(msgBytes, &cmd); err == nil {
				app.TrySend(m.S, m.S.FlagBroadcast, cmd)
			}
		case "self_service":
			var action models.SelfServiceAction
			if err := json.Unmarshal(msgBytes, &action); err == nil {
				m.BroadcastSelfService(action)
			}
		case "lap_update":
			var frame models.LapReplayFrame
			if err := json.Unmarshal(msgBytes, &frame); err == nil {
				app.TrySend(m.S, m.S.LapReplayBroadcast, frame)
			}
		case "weather_update":
			var wc models.WeatherCondition
			if err := json.Unmarshal(msgBytes, &wc); err == nil {
				app.TrySend(m.S, m.S.WeatherBroadcast, wc)
			}
		}
	}
}

// broadcastToClients marshals msg once and enqueues it to every client without
// blocking. Clients whose queue is full are evicted so a single slow consumer
// cannot stall delivery to healthy ones.
func (m *Manager) broadcastToClients(msg any) {
	data, err := json.Marshal(msg)
	if err != nil {
		m.S.Log.Errorf("ws", "Error marshalling broadcast: %v", err)
		return
	}

	m.mu.RLock()
	var lagging []*client
	for c := range m.clients {
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

func (m *Manager) BroadcastManager() {
	for racers := range m.S.Broadcast {
		m.broadcastToClients(racers)
	}
}

func (m *Manager) BroadcastFlags() {
	for cmd := range m.S.FlagBroadcast {
		m.broadcastToClients(cmd)
	}
}

func (m *Manager) BroadcastGameMechanics() {
	for update := range m.S.GameMechanicsBroadcast {
		m.broadcastToClients(update)
	}
}

func (m *Manager) BroadcastWeather() {
	for wc := range m.S.WeatherBroadcast {
		m.broadcastToClients(map[string]any{
			"type":          "weather_update",
			"id":            wc.ID,
			"race_id":       wc.RaceID,
			"condition":     wc.Condition,
			"lap_start":     wc.LapStart,
			"lap_end":       wc.LapEnd,
			"grip_modifier": wc.GripModifier,
		})
	}
}

func (m *Manager) BroadcastLapReplay() {
	for frame := range m.S.LapReplayBroadcast {
		m.broadcastToClients(frame)
	}
}

func (m *Manager) BroadcastSound() {
	for cmd := range m.S.SoundBroadcast {
		m.broadcastToClients(cmd)
	}
}

func (m *Manager) BroadcastRaceRadio() {
	for msg := range m.S.RaceRadioBroadcast {
		m.broadcastToClients(map[string]any{
			"type":       "race_radio",
			"id":         msg.ID,
			"racer_id":   msg.RacerID,
			"racer_name": msg.RacerName,
			"message":    msg.Message,
			"timestamp":  msg.Timestamp,
		})
	}
}

func (m *Manager) BroadcastCommentary() {
	for entry := range m.S.CommentaryBroadcast {
		m.broadcastToClients(map[string]any{
			"type":         "commentary",
			"id":           entry.ID,
			"race_id":      entry.RaceID,
			"lap":          entry.Lap,
			"racer_id":     entry.RacerID,
			"racer_name":   entry.RacerName,
			"message":      entry.Message,
			"template_key": entry.TemplateKey,
			"created_at":   entry.CreatedAt,
		})
	}
}

func (m *Manager) BroadcastSelfService(action models.SelfServiceAction) {
	msg := map[string]any{
		"type":     "self_service",
		"action":   action.Type,
		"racer_id": action.RacerID,
		"lap":      action.Lap,
		"gear":     action.Gear,
		"stress":   action.Stress,
		"turbo":    action.TurboUsed,
	}
	m.broadcastToClients(msg)
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
