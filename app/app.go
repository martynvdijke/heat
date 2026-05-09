package app

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"heat/models"
)

var FlagBroadcast = make(chan models.FlagCommand)

type SessionInfo struct {
	Expiry int64
	IP     string
}

var (
	DB             *sql.DB
	SessionStore   = make(map[string]SessionInfo)
	SessionStoreMu sync.RWMutex
	StaticCache    = make(map[string][]byte)
	Clients        = make(map[*websocket.Conn]bool)
	Broadcast      = make(chan []models.Racer)
	BasePath       = "/app"
	DBPath         = "/db/heat.db"
	MediaPath      = "/app/media"
	CurrentVersion = "1.19.1"
	LoginLimiter   = rate.NewLimiter(rate.Limit(5), 10)
	SecureCookies  = os.Getenv("DOCKER") == "true"

	Upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			u, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return u.Host == r.Host
		},
	}

	LoginLimiters   = make(map[string]*rate.Limiter)
	LoginLimitersMu sync.Mutex
)
