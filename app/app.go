package app

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/gorilla/websocket"
	"golang.org/x/time/rate"

	"heat/ent"
	"heat/models"
	"heat/pkg/logger"
	"heat/racing"
)

type SessionInfo struct {
	Expiry int64
	IP     string
}

type Server struct {
	DB             *sql.DB
	Ent            *ent.Client
	SessionStore   map[string]SessionInfo
	SessionStoreMu sync.RWMutex
	Clients        map[*websocket.Conn]bool
	ClientsMu      sync.RWMutex
	Broadcast      chan []models.Racer

	FlagBroadcast          chan models.FlagCommand
	GameMechanicsBroadcast chan models.GameMechanicsUpdate
	WeatherBroadcast       chan models.WeatherCondition
	LapReplayBroadcast     chan models.LapReplayFrame
	SoundBroadcast         chan models.SoundCommand
	RaceRadioBroadcast     chan models.RaceRadioMessage

	BasePath       string
	DBPath         string
	MediaPath      string
	CurrentVersion string

	LoginLimiter         *rate.Limiter
	SecureCookies        bool
	Upgrader             websocket.Upgrader
	LoginLimiters        map[string]*rate.Limiter
	LoginLimitersMu      sync.Mutex
	BroadcastRacers      func()
	BroadcastSelfService func(action models.SelfServiceAction)

	Log        *logger.Logger
	StatsCache *racing.Cache
}

func NewServer() *Server {
	return &Server{
		SessionStore:           make(map[string]SessionInfo),
		Clients:                make(map[*websocket.Conn]bool),
		Broadcast:              make(chan []models.Racer),
		FlagBroadcast:          make(chan models.FlagCommand),
		GameMechanicsBroadcast: make(chan models.GameMechanicsUpdate),
		WeatherBroadcast:       make(chan models.WeatherCondition),
		LapReplayBroadcast:     make(chan models.LapReplayFrame),
		SoundBroadcast:         make(chan models.SoundCommand),
		RaceRadioBroadcast:     make(chan models.RaceRadioMessage),
		LoginLimiter:           rate.NewLimiter(rate.Limit(5), 10),
		LoginLimiters:          make(map[string]*rate.Limiter),
		CurrentVersion:         "1.55.1",
		BasePath:               "/app",
		DBPath:                 "/db/heat.db",
		MediaPath:              "/app/media",
		SecureCookies:          os.Getenv("DOCKER") == "true",
		Upgrader: websocket.Upgrader{
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
		},
	}
}
