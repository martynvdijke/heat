package ws

import (
	"encoding/json"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"heat/app"
	"heat/models"
)

type Manager struct {
	S *app.Server
}

func NewManager(s *app.Server) *Manager {
	return &Manager{S: s}
}

func (m *Manager) HandleWebSocket(c *gin.Context) {
	ws, err := m.S.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.S.Log.Errorf("ws", "Error upgrading WebSocket: %v", err)
		return
	}
	defer ws.Close()

	m.S.ClientsMu.Lock()
	m.S.Clients[ws] = true
	m.S.ClientsMu.Unlock()
	m.S.Log.Infof("ws", "New client connected. Total clients: %d", len(m.S.Clients))

	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			m.S.Log.Infof("ws", "Client disconnected: %v", err)
			m.S.ClientsMu.Lock()
			delete(m.S.Clients, ws)
			m.S.ClientsMu.Unlock()
			break
		}

		var msg map[string]any
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "flag":
				var cmd models.FlagCommand
				if err := json.Unmarshal(msgBytes, &cmd); err == nil {
					m.S.FlagBroadcast <- cmd
				}
			case "self_service":
				var action models.SelfServiceAction
				if err := json.Unmarshal(msgBytes, &action); err == nil {
					m.BroadcastSelfService(action)
				}
			case "lap_update":
				var frame models.LapReplayFrame
				if err := json.Unmarshal(msgBytes, &frame); err == nil {
					m.S.LapReplayBroadcast <- frame
				}
			case "weather_update":
				var wc models.WeatherCondition
				if err := json.Unmarshal(msgBytes, &wc); err == nil {
					m.S.WeatherBroadcast <- wc
				}
			}
		}
	}
}

func (m *Manager) broadcastToClients(msg any) {
	var failed []*websocket.Conn
	m.S.ClientsMu.RLock()
	for client := range m.S.Clients {
		err := client.WriteJSON(msg)
		if err != nil {
			m.S.Log.Warnf("ws", "Error broadcasting to client: %v", err)
			client.Close()
			failed = append(failed, client)
		}
	}
	m.S.ClientsMu.RUnlock()
	if len(failed) > 0 {
		m.S.ClientsMu.Lock()
		for _, client := range failed {
			delete(m.S.Clients, client)
		}
		m.S.ClientsMu.Unlock()
	}
}

func (m *Manager) BroadcastManager() {
	for {
		racers := <-m.S.Broadcast
		m.broadcastToClients(racers)
	}
}

func (m *Manager) BroadcastFlags() {
	for {
		cmd := <-m.S.FlagBroadcast
		m.broadcastToClients(cmd)
	}
}

func (m *Manager) BroadcastGameMechanics() {
	for {
		update := <-m.S.GameMechanicsBroadcast
		m.broadcastToClients(update)
	}
}

func (m *Manager) BroadcastWeather() {
	for {
		wc := <-m.S.WeatherBroadcast
		m.broadcastToClients(wc)
	}
}

func (m *Manager) BroadcastLapReplay() {
	for {
		frame := <-m.S.LapReplayBroadcast
		m.broadcastToClients(frame)
	}
}

func (m *Manager) BroadcastSound() {
	for {
		cmd := <-m.S.SoundBroadcast
		m.broadcastToClients(cmd)
	}
}

func (m *Manager) BroadcastRaceRadio() {
	for {
		msg := <-m.S.RaceRadioBroadcast
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
	rows, err := m.S.DB.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position, COALESCE(team_id, 0) FROM racers ORDER BY rank ASC")
	if err != nil {
		m.S.Log.Errorf("ws", "Error fetching racers for broadcast: %v", err)
		return
	}
	defer rows.Close()

	var racers []models.Racer
	for rows.Next() {
		var r models.Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position, &r.TeamID)
		if err != nil {
			m.S.Log.Errorf("ws", "Error scanning racer for broadcast: %v", err)
			return
		}
		racers = append(racers, r)
	}
	select {
	case m.S.Broadcast <- racers:
	default:
	}
}
