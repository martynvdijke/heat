package ws

import (
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
		_, _, err := ws.ReadMessage()
		if err != nil {
			log.Printf("[WS] Client disconnected: %v", err)
			delete(app.Clients, ws)
			break
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
	app.Broadcast <- racers
}
