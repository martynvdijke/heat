package app

import (
	"database/sql"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"heat/models"
)

var (
	DB             *sql.DB
	SessionStore   = make(map[string]int64)
	StaticCache    = make(map[string][]byte)
	Clients        = make(map[*websocket.Conn]bool)
	Broadcast      = make(chan []models.Racer)
	BasePath       = "/app"
	DBPath         = "/db/heat.db"
	MediaPath      = "/app/media"
	CurrentVersion = "1.11.1"
	LoginLimiter   = rate.NewLimiter(rate.Limit(5), 10)

	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Per-IP rate limiters for login
	LoginLimiters   = make(map[string]*rate.Limiter)
	LoginLimitersMu sync.Mutex
)
