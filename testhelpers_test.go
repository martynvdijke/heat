package main

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"

	"heat/app"
	"heat/db"
	"heat/ent"
	"heat/handlers"
	"heat/middleware"
	"heat/ws"
)

var testServer *app.Server
var testHandler *handlers.Handler
var wsManager *ws.Manager

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	testServer = app.NewServer()
	os.Unsetenv("DOCKER")
	testServer.BasePath = "."
	testServer.DBPath = ":memory:"
	testServer.MediaPath = filepath.Join(testServer.BasePath, "media")

	var err error
	testServer.DB, err = sql.Open("sqlite3", testServer.DBPath+"?_fk=1")
	if err != nil {
		log.Fatalf("failed to open in-memory db: %v", err)
	}
	testServer.DB.SetMaxOpenConns(1)

	drv := entsql.OpenDB(dialect.SQLite, testServer.DB)
	testServer.Ent = ent.NewClient(ent.Driver(drv))
	testServer.BroadcastRacers = func() {}

	db.Init(testServer)

	wsManager = ws.NewManager(testServer)
	testServer.BroadcastSelfService = wsManager.BroadcastSelfService
	go wsManager.BroadcastManager()
	go wsManager.BroadcastFlags()
	go wsManager.BroadcastGameMechanics()
	go wsManager.BroadcastWeather()
	go wsManager.BroadcastLapReplay()
	go wsManager.BroadcastSound()
	go wsManager.BroadcastRaceRadio()

	if err := os.MkdirAll(testServer.MediaPath, 0755); err != nil {
		log.Fatalf("failed to create media directory: %v", err)
	}

	testHandler = handlers.New(testServer)

	os.Exit(m.Run())
}

func makeUniquePNGData(seed byte) []byte {
	return []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		seed, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x01, 0x00, 0x05, 0x18, 0xD8, 0x4E, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}
}

func createAdminSession(t *testing.T) string {
	t.Helper()
	sessionID := "admin-session-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	testServer.SessionStoreMu.Lock()
	testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	testServer.SessionStoreMu.Unlock()
	return sessionID
}

func removeAdminSession(sessionID string) {
	testServer.SessionStoreMu.Lock()
	delete(testServer.SessionStore, sessionID)
	testServer.SessionStoreMu.Unlock()
}

func newAdminRequest(method, path string, body []byte, sessionID string) *http.Request {
	req, _ := http.NewRequest(method, path, bytes.NewBuffer(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Host = "127.0.0.1:6270"
	return req
}

func setupAdminRouter() (*gin.Engine, string) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	r.GET("/api/racers", testHandler.GetRacers)
	return r, ""
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func escapeHTML(s string) string {
	var result string
	for _, c := range s {
		switch c {
		case '&':
			result += "&amp;"
		case '<':
			result += "&lt;"
		case '>':
			result += "&gt;"
		case '"':
			result += "&quot;"
		default:
			result += string(c)
		}
	}
	return result
}

func shorten(s string) string {
	if len(s) > 16 {
		return s[:16] + "..."
	}
	return s
}

func setupHtmxRouter() (*gin.Engine, string) {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/html/racers", testHandler.HtmxRacersTable)
	admin.POST("/html/racers", testHandler.HtmxRacersSave)
	admin.GET("/html/racers/:id/edit", testHandler.HtmxRacersEditForm)
	admin.DELETE("/html/racers/:id", testHandler.HtmxRacersDelete)
	admin.POST("/html/racers/:id/share", testHandler.HtmxRacersGenerateShare)
	admin.GET("/html/tracks", testHandler.HtmxTracksTable)
	admin.POST("/html/tracks", testHandler.HtmxTracksSave)
	admin.GET("/html/tracks/:id/edit", testHandler.HtmxTracksEditForm)
	admin.DELETE("/html/tracks/:id", testHandler.HtmxTracksDelete)
	admin.GET("/html/quotes", testHandler.HtmxQuotesTable)
	admin.POST("/html/quotes", testHandler.HtmxQuotesSave)
	admin.GET("/html/quotes/:id/edit", testHandler.HtmxQuotesEditForm)
	admin.DELETE("/html/quotes/:id", testHandler.HtmxQuotesDelete)
	admin.GET("/html/teams", testHandler.HtmxTeamsTable)
	admin.POST("/html/teams", testHandler.HtmxTeamsSave)
	admin.GET("/html/teams/:id/edit", testHandler.HtmxTeamsEditForm)
	admin.DELETE("/html/teams/:id", testHandler.HtmxTeamsDelete)
	admin.GET("/html/seasons", testHandler.HtmxSeasonsTable)
	admin.GET("/html/seasons/new", testHandler.HtmxSeasonsNewForm)
	admin.POST("/html/seasons", testHandler.HtmxSeasonsCreate)
	admin.POST("/html/seasons/:id/archive", testHandler.HtmxSeasonsArchive)
	admin.DELETE("/html/seasons/:id", testHandler.HtmxSeasonsDelete)
	sessionID := "htmx-test-session-" + strconv.FormatInt(time.Now().UnixNano(), 36)
	testServer.SessionStoreMu.Lock()
	testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	testServer.SessionStoreMu.Unlock()
	return r, sessionID
}

func newHtmxAdminFormRequest(path string, formData map[string]string, sessionID string) *http.Request {
	body := ""
	for k, v := range formData {
		if body != "" {
			body += "&"
		}
		body += url.Values{k: {v}}.Encode()
	}
	req, _ := http.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Host = "127.0.0.1:6270"
	return req
}
