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

	app.Clients[ws] = true
	log.Printf("[WS] New client connected. Total clients: %d", len(app.Clients))

	for {
		_, msgBytes, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client disconnected: %v", err)
			delete(app.Clients, ws)
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

func BroadcastManager() {
	for {
		racers := <-app.Broadcast
		for client := range app.Clients {
			err := client.WriteJSON(racers)
			if err != nil {
				log.Printf("[WS] error broadcasting to client: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
	}
}

func BroadcastFlags() {
	for {
		cmd := <-app.FlagBroadcast
		for client := range app.Clients {
			err := client.WriteJSON(cmd)
			if err != nil {
				log.Printf("[WS] error broadcasting flag: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
	}
}

func BroadcastGameMechanics() {
	for {
		update := <-app.GameMechanicsBroadcast
		for client := range app.Clients {
			err := client.WriteJSON(update)
			if err != nil {
				log.Printf("[WS] error broadcasting game mechanics: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
	}
}

func BroadcastWeather() {
	for {
		wc := <-app.WeatherBroadcast
		for client := range app.Clients {
			err := client.WriteJSON(wc)
			if err != nil {
				log.Printf("[WS] error broadcasting weather: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
	}
}

func BroadcastLapReplay() {
	for {
		frame := <-app.LapReplayBroadcast
		for client := range app.Clients {
			err := client.WriteJSON(frame)
			if err != nil {
				log.Printf("[WS] error broadcasting lap replay: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
	}
}

func BroadcastSound() {
	for {
		cmd := <-app.SoundBroadcast
		for client := range app.Clients {
			err := client.WriteJSON(cmd)
			if err != nil {
				log.Printf("[WS] error broadcasting sound: %v", err)
				client.Close()
				delete(app.Clients, client)
			}
		}
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
	for client := range app.Clients {
		err := client.WriteJSON(msg)
		if err != nil {
			log.Printf("[WS] error broadcasting self-service: %v", err)
			client.Close()
			delete(app.Clients, client)
		}
	}
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
