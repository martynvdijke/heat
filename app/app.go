package app

import (
	"database/sql"
	"net/http"
	"net/url"
	"os"
	"strings"
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

// OIDCConfig holds Authelia OIDC relying-party settings (see
// openspec/changes/add-authelia-oidc). Secrets come from
// OIDC_CLIENT_SECRET_FILE or OIDC_CLIENT_SECRET env, never git.
type OIDCConfig struct {
	Enabled      bool
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
	LogoutURL    string
}

// LoadOIDCConfig reads OIDC_* env vars. Secret file takes precedence.
func LoadOIDCConfig() OIDCConfig {
	scopes := []string{"openid", "email", "profile", "groups"}
	if s := os.Getenv("OIDC_SCOPES"); s != "" {
		scopes = strings.Split(s, " ")
	}
	secret := os.Getenv("OIDC_CLIENT_SECRET")
	if f := os.Getenv("OIDC_CLIENT_SECRET_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			secret = strings.TrimSpace(string(b))
		}
	}
	return OIDCConfig{
		Enabled:      os.Getenv("OIDC_ENABLED") == "true",
		IssuerURL:    strings.TrimSuffix(os.Getenv("OIDC_ISSUER_URL"), "/"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: secret,
		RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
		Scopes:       scopes,
		LogoutURL:    os.Getenv("OIDC_LOGOUT_URL"),
	}
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
	CommentaryBroadcast    chan models.Commentary

	BasePath       string
	DBPath         string
	MediaPath      string
	CurrentVersion string

	LoginLimiter         *rate.Limiter
	SecureCookies        bool
	OIDC                 OIDCConfig
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
		CommentaryBroadcast:    make(chan models.Commentary),
		LoginLimiter:           rate.NewLimiter(rate.Limit(5), 10),
		LoginLimiters:          make(map[string]*rate.Limiter),
		CurrentVersion:         "1.58.9",
		BasePath:               "/app",
		DBPath:                 "/db/heat.db",
		MediaPath:              "/app/media",
		SecureCookies:          os.Getenv("DOCKER") == "true",
		OIDC:                   LoadOIDCConfig(),
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
