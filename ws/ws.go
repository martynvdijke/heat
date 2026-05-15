package ws

import (
	"encoding/json"
	"log"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

func HandleWebSocket(c *gin.Context) {
	ws, err := app.Upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("error upgrading: %v", err)
		return
	}
	defer ws.Close()

	app.ClientsMu.Lock()
	app.Clients[ws] = true
	app.ClientsMu.Unlock()
	log.Printf("[WS] New client connected. Total clients: %d", len(app.Clients))

	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client disconnected: %v", err)
			app.ClientsMu.Lock()
			delete(app.Clients, ws)
			app.ClientsMu.Unlock()
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			continue
		}

		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "flag":
				var cmd models.FlagCommand
				if err := json.Unmarshal(msgBytes, &cmd); err == nil {
					app.FlagBroadcast <- cmd
				}
			case "self_service":
				var action models.SelfServiceAction
				if err := json.Unmarshal(msgBytes, &action); err == nil {
					BroadcastSelfService(action)
				}
			case "lap_update":
				var frame models.LapReplayFrame
				if err := json.Unmarshal(msgBytes, &frame); err == nil {
					app.LapReplayBroadcast <- frame
				}
			case "weather_update":
				var wc models.WeatherCondition
				if err := json.Unmarshal(msgBytes, &wc); err == nil {
					app.WeatherBroadcast <- wc
				}
			}
		}
	}
}

func broadcastToClients(msg interface{}) {
	app.ClientsMu.RLock()
	defer app.ClientsMu.RUnlock()
	for client := range app.Clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Printf("[WS] error broadcasting to client: %v", err)
			client.Close()
			app.ClientsMu.RUnlock()
			app.ClientsMu.Lock()
			delete(app.Clients, client)
			app.ClientsMu.Unlock()
			app.ClientsMu.RLock()
		}
	}
}

func BroadcastManager() {
	for {
		racers := <-app.Broadcast
		broadcastToClients(racers)
	}
}

func BroadcastFlags() {
	for {
		cmd := <-app.FlagBroadcast
		broadcastToClients(cmd)
	}
}

func BroadcastGameMechanics() {
	for {
		update := <-app.GameMechanicsBroadcast
		broadcastToClients(update)
	}
}

func BroadcastWeather() {
	for {
		wc := <-app.WeatherBroadcast
		broadcastToClients(wc)
	}
}

func BroadcastLapReplay() {
	for {
		frame := <-app.LapReplayBroadcast
		broadcastToClients(frame)
	}
}

func BroadcastSound() {
	for {
		cmd := <-app.SoundBroadcast
		broadcastToClients(cmd)
	}
}

func BroadcastSelfService(action models.SelfServiceAction) {
	msg := map[string]interface{}{
		"type":     "self_service",
		"action":   action.Type,
		"racer_id": action.RacerID,
		"lap":      action.Lap,
		"gear":     action.Gear,
		"stress":   action.Stress,
		"turbo":    action.TurboUsed,
	}
	broadcastToClients(msg)
}

func BroadcastRacers() {
	rows, err := app.DB.Query("SELECT id, name, profile_picture, car_color, car_name, points, rank, position FROM racers ORDER BY rank ASC")
	if err != nil {
		log.Printf("error fetching racers for broadcast: %v", err)
		return
	}
	defer rows.Close()

	var racers []models.Racer
	for rows.Next() {
		var r models.Racer
		err := rows.Scan(&r.ID, &r.Name, &r.ProfilePicture, &r.CarColor, &r.CarName, &r.Points, &r.Rank, &r.Position)
		if err != nil {
			log.Printf("error scanning racer for broadcast: %v", err)
			return
		}
		racers = append(racers, r)
	}
	select {
	case app.Broadcast <- racers:
	default:
	}
}
