// HEAT Racing API
//
// API for the HEAT: Pedal to the Metal racing championship management system.
//
//	Schemes: http
//	Host: localhost:6270
//	BasePath: /api
//	Version: 1.16.0
//
//	Consumes:
//	- application/json
//	- multipart/form-data
//
//	Produces:
//	- application/json
//
//	SecurityDefinitions:
//	cookieAuth:
//	  type: apiKey
//	  in: cookie
//	  name: session
//
// swagger:meta
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/net/webdav"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	swaggerFiles "github.com/swaggo/files/v2"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "heat/docs"

	"heat/app"
	"heat/db"
	"heat/ent"
	"heat/handlers"
	"heat/middleware"
	"heat/pkg/logger"
	"heat/racing"
	"heat/ws"
)

type PageData struct {
	Version string
}

type AdminData struct {
	Version      string
	TabID        string
	VendorCSS    string
	VendorJS     string
	VendorFA     string
	VendorNavCss string
}

var (
	templateCache sync.Map
	adminTemplate *template.Template
	vendorCSSPath string
	vendorJSPath  string
	vendorFAPath  string
	vendorNavCss  string
)

func loadVendorManifest(basePath string) {
	data, err := os.ReadFile(filepath.Join(basePath, "static/vendor/manifest.json"))
	if err != nil {
		log.Printf("Warning: vendor manifest not found (run 'node build.mjs'): %v", err)
		return
	}
	var m struct {
		BootstrapCss   string `json:"bootstrapCss"`
		BootstrapJs    string `json:"bootstrapJs"`
		FontawesomeCss string `json:"fontawesomeCss"`
		AdminNavCss    string `json:"adminNavCss"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		log.Printf("Warning: vendor manifest parse error: %v", err)
		return
	}
	vendorCSSPath = m.BootstrapCss
	vendorJSPath = m.BootstrapJs
	vendorFAPath = m.FontawesomeCss
	vendorNavCss = m.AdminNavCss
}

var swaggerHandler = func() *webdav.Handler {
	h := &webdav.Handler{
		FileSystem: webdav.NewMemFS(),
		LockSystem: webdav.NewMemLS(),
	}

	fs.WalkDir(swaggerFiles.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return h.FileSystem.Mkdir(context.Background(), path, 0755)
		}
		data, err := fs.ReadFile(swaggerFiles.FS, path)
		if err != nil {
			return err
		}
		f, err := h.FileSystem.OpenFile(context.Background(), path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(data)
		return err
	})

	return h
}()

func servePage(c *gin.Context, path string, s *app.Server) {
	content, err := os.ReadFile(path)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	html := strings.Replace(string(content), "{{VERSION}}", s.CurrentVersion, 1)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// loadCachedTemplate parses a template the first time and caches it forever.
func loadCachedTemplate(files ...string) *template.Template {
	key := strings.Join(files, "\x00")
	if cached, ok := templateCache.Load(key); ok {
		return cached.(*template.Template)
	}
	tmpl := template.Must(template.ParseFiles(files...))
	templateCache.Store(key, tmpl)
	return tmpl
}

func serveTemplate(c *gin.Context, name string, s *app.Server) {
	tmpl := loadCachedTemplate(
		filepath.Join(s.BasePath, "static/templates/base.html"),
		filepath.Join(s.BasePath, "static/templates", name),
	)
	data := PageData{Version: s.CurrentVersion}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", data); err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", buf.Bytes())
}

// initAdminTemplate parses the admin template at startup and caches it.
func initAdminTemplate(basePath string) {
	adminTemplate = template.Must(template.ParseFiles(
		filepath.Join(basePath, "static/templates/admin.html"),
		filepath.Join(basePath, "static/templates/admin-header.html"),
		filepath.Join(basePath, "static/templates/admin-footer.html"),
		filepath.Join(basePath, "static/templates/tab-race-day.html"),
		filepath.Join(basePath, "static/templates/tab-season.html"),
		filepath.Join(basePath, "static/templates/tab-drivers.html"),
		filepath.Join(basePath, "static/templates/tab-config.html"),
		filepath.Join(basePath, "static/templates/tab-extensions.html"),
	))
}

func isValidAdminTab(tab string) bool {
	switch tab {
	case "race-day", "season", "drivers", "config", "extensions":
		return true
	}
	return false
}

func main() {
	server := app.NewServer()

	if os.Getenv("DOCKER") != "true" {
		server.BasePath = "."
		server.DBPath = "./heat.db"
	}
	server.MediaPath = filepath.Join(server.BasePath, "media")

	if err := os.MkdirAll(server.MediaPath, 0755); err != nil {
		log.Printf("Warning: could not create media directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(server.DBPath), 0755); err != nil {
		log.Printf("Warning: could not create database directory: %v", err)
	}

	var err error
	// Most PRAGMAs are embedded in the DSN so they apply to EVERY connection
	// in the pool. Setting them via DB.Exec only affects the single connection
	// that runs the statement; with SetMaxOpenConns(8) new connections would
	// inherit SQLite defaults (busy_timeout=5000 and synchronous=NORMAL are
	// already the mattn/go-sqlite3 driver defaults, but cache_size defaults to
	// -2000 (2MB) which causes more disk I/O).
	//
	// temp_store is a connection-level PRAGMA not supported as a DSN parameter
	// in this driver version, so it's set via Exec as a best-effort measure.
	server.DB, err = sql.Open("sqlite3", server.DBPath+"?_fk=1&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_cache_size=-20000")
	if err != nil {
		log.Fatal(err)
	}
	server.DB.SetMaxOpenConns(8)
	server.DB.SetMaxIdleConns(8)
	server.DB.SetConnMaxLifetime(5 * time.Minute)
	server.DB.Exec("PRAGMA temp_store=MEMORY")

	drv := entsql.OpenDB(dialect.SQLite, server.DB)
	server.Ent = ent.NewClient(ent.Driver(drv))
	defer server.Ent.Close()

	// Initialize structured logger
	server.Log = logger.New(server.DB)
	defer server.Log.Stop()

	// Initialize stats cache with 5 minute TTL
	server.StatsCache = racing.NewCache(300 * time.Second)

	h := handlers.New(server)
	wsManager := ws.NewManager(server)

	// Pre-parse templates at startup
	loadVendorManifest(server.BasePath)
	initAdminTemplate(server.BasePath)
	// Warm up page templates in cache
	loadCachedTemplate(
		filepath.Join(server.BasePath, "static/templates/base.html"),
		filepath.Join(server.BasePath, "static/templates/index.html"),
	)
	loadCachedTemplate(
		filepath.Join(server.BasePath, "static/templates/base.html"),
		filepath.Join(server.BasePath, "static/templates/stats.html"),
	)
	loadCachedTemplate(
		filepath.Join(server.BasePath, "static/templates/base.html"),
		filepath.Join(server.BasePath, "static/templates/seasons.html"),
	)
	loadCachedTemplate(
		filepath.Join(server.BasePath, "static/templates/base.html"),
		filepath.Join(server.BasePath, "static/templates/trophies.html"),
	)
	server.BroadcastRacers = wsManager.BroadcastRacers
	server.BroadcastSelfService = wsManager.BroadcastSelfService

	db.Init(server)

	// Initialize OpenTelemetry after DB is ready (reads OTel settings from database)
	otelShutdown := initOTel(server)
	defer otelShutdown()
	// Create OTel metric instruments (requires global MeterProvider from initOTel)
	initOTelMetrics()
	go wsManager.BroadcastManager()
	go wsManager.BroadcastFlags()
	go wsManager.BroadcastGameMechanics()
	go wsManager.BroadcastWeather()
	go wsManager.BroadcastLapReplay()
	go wsManager.BroadcastSound()
	go wsManager.BroadcastRaceRadio()
	go func() {
		for {
			time.Sleep(15 * time.Minute)
			now := time.Now().Unix()
			server.SessionStoreMu.Lock()
			for k, v := range server.SessionStore {
				if now > v.Expiry {
					delete(server.SessionStore, k)
				}
			}
			server.SessionStoreMu.Unlock()
		}
	}()
	go func() {
		for {
			var enabled int
			var intervalHrs int
			server.DB.QueryRow("SELECT enabled, interval_hrs FROM backup_settings WHERE id = 1").Scan(&enabled, &intervalHrs)
			if enabled == 1 && intervalHrs > 0 {
				if err := db.CreateBackup(); err != nil {
					server.Log.Errorf("backup", "Periodic backup failed: %v", err)
				} else {
					server.Log.Infof("backup", "Periodic backup completed (interval: %dh)", intervalHrs)
				}
				if err := db.PruneBackups(); err != nil {
					server.Log.Warnf("backup", "Prune failed: %v", err)
				}
			}
			interval := 24
			if intervalHrs > 0 {
				interval = intervalHrs
			}
			time.Sleep(time.Duration(interval) * time.Hour)
		}
	}()

	r := gin.New()
	r.MaxMultipartMemory = 32 << 20
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(otelgin.Middleware("heat"))
	// Exclude /ws from gzip — the gzip middleware wraps c.Writer, which breaks
	// gorilla/websocket's Upgrader.Upgrade() (it needs the raw ResponseWriter
	// to hijack the connection for the WebSocket handshake).
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/ws"})))
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeaders())

	r.Use(metricsMiddleware())

	r.GET("/ws", wsManager.HandleWebSocket)

	r.POST("/api/login", middleware.RateLimitMiddleware(server), h.HandleLogin)
	r.POST("/api/logout", middleware.CSRFMiddleware(), h.HandleLogout)
	r.GET("/api/check-setup", h.HandleCheckSetup)

	// Forgot-password flow (public)
	r.POST("/api/forgot-password", middleware.RateLimitMiddleware(server), h.RequestPasswordReset)
	r.GET("/api/reset-password/validate", h.ValidateResetToken)
	r.POST("/api/reset-password", h.ResetPassword)

	r.GET("/api/racers", h.GetRacers)
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(server))
	{
		admin.POST("/racers", h.UpdateRacer)
		admin.PUT("/racers/ranks", h.UpdateRacerRanks)
		admin.DELETE("/racers", h.DeleteRacer)
		admin.POST("/race-info", h.UpdateRaceInfo)
		admin.POST("/upload", h.HandleUpload)
		admin.POST("/tracks", h.SaveTrack)
		admin.DELETE("/tracks", h.DeleteTrack)
		admin.PUT("/board-game/tracks", h.SetBoardGameTracks)
		admin.POST("/tracks/ai-extract", h.HandleAIExtract)
		admin.POST("/race-history", h.SaveRaceToHistory)
		admin.DELETE("/race-history", h.DeleteRaceHistory)
		admin.POST("/racer-stats", h.UpdateRacerStats)
		admin.GET("/notification-settings", h.GetNotificationSettings)
		admin.POST("/notification-settings", h.SaveNotificationSettings)
		admin.POST("/test-notification", h.TestNotification)
		admin.GET("/ai-settings", h.GetAISettings)
		admin.POST("/ai-settings", h.SaveAISettings)
		admin.GET("/email-settings", h.GetEmailSettings)
		admin.POST("/email-settings", h.SaveEmailSettings)
		admin.GET("/racer-emails", h.GetRacerEmails)
		admin.POST("/racer-emails", h.SaveRacerEmail)
		admin.POST("/send-race-email", h.SendRaceEmailManual)
		admin.DELETE("/oneoff-races", h.DeleteOneOffRace)
		admin.GET("/umami-settings", h.GetUmamiSettings)
		admin.POST("/umami-settings", h.SaveUmamiSettings)
		admin.GET("/otel-settings", h.GetOTelSettings)
		admin.POST("/otel-settings", h.SaveOTelSettings)
		admin.POST("/quotes", h.HandleQuotes)
		admin.PUT("/quotes", h.HandleQuotes)
		admin.DELETE("/quotes", h.HandleQuotes)
		admin.GET("/backup-settings", h.GetBackupSettings)
		admin.POST("/backup-settings", h.SaveBackupSettings)
		admin.POST("/backup/manual", h.TriggerManualBackup)
		admin.GET("/backup/list", h.ListBackups)
		admin.POST("/seasons", h.CreateSeason)
		admin.POST("/seasons/archive", h.ArchiveSeason)
		admin.DELETE("/seasons", h.DeleteSeason)

		// Admin: Extension & module catalog
		admin.GET("/extensions", h.GetExtensions)
		admin.POST("/extensions", h.CreateExtension)
		admin.PUT("/extensions", h.UpdateExtension)
		admin.DELETE("/extensions", h.DeleteExtension)
		admin.GET("/extensions/detail", h.GetExtensionDetail)
		admin.GET("/extensions/owned", h.GetOwnedExtensions)
		admin.PUT("/extensions/owned", h.SetOwnedExtensions)
		admin.GET("/modules", h.GetModules)
		admin.POST("/modules", h.CreateModule)
		admin.PUT("/modules", h.UpdateModule)
		admin.DELETE("/modules", h.DeleteModule)
		admin.PUT("/content/extension", h.AssignContentExtension)

		// Admin: Game Mechanics
		admin.POST("/heat-cards", h.AddHeatCard)
		admin.PUT("/heat-cards/move", h.MoveHeatCard)
		admin.DELETE("/heat-cards", h.DeleteHeatCard)
		admin.DELETE("/heat-cards/clear", h.ClearHeatCards)
		admin.POST("/heat-cards/init-decks", h.InitializeHeatDecks)
		admin.POST("/gear-shifts", h.AddGearShift)
		admin.DELETE("/gear-shifts", h.DeleteGearShift)
		admin.POST("/upgrade-cards", h.SaveUpgradeCard)
		admin.DELETE("/upgrade-cards", h.DeleteUpgradeCard)
		admin.POST("/player-upgrades/buy", h.BuyUpgrade)
		admin.PUT("/player-upgrades/toggle", h.ToggleUpgrade)
		admin.DELETE("/player-upgrades", h.DeletePlayerUpgrade)
		admin.POST("/legend-abilities/assign", h.AssignLegendAbility)
		admin.PUT("/legend-abilities/toggle", h.ToggleLegendAbility)

		// Admin: Multi-User
		admin.GET("/player-sessions", h.GetPlayerSessions)
		admin.DELETE("/player-sessions", h.DeletePlayerSession)

		// Admin: Race Enhancements
		admin.POST("/weather", h.SetWeather)
		admin.DELETE("/weather", h.DeleteWeather)
		admin.POST("/turbo-logs", h.AddTurboLog)
		admin.DELETE("/turbo-logs", h.DeleteTurboLog)
		admin.DELETE("/lap-records", h.DeleteLapRecords)
		admin.DELETE("/race-events", h.DeleteRaceEvent)
		admin.POST("/ai-difficulty", h.SetAIDifficulty)

		// Admin: Driver Share
		admin.POST("/driver-share", h.GenerateDriverShareToken)
		admin.GET("/driver-shares", h.GetDriverShareTokens)
		admin.DELETE("/driver-share", h.DeleteDriverShareToken)

		// Teams
		admin.POST("/teams", h.SaveTeam)
		admin.DELETE("/teams", h.DeleteTeam)
		admin.POST("/teams/assign", h.AssignTeam)

		// HTMX endpoints (under admin auth group)
		admin.GET("/html/racers", h.HtmxRacersTable)
		admin.POST("/html/racers", h.HtmxRacersSave)
		admin.GET("/html/racers/:id/edit", h.HtmxRacersEditForm)
		admin.DELETE("/html/racers/:id", h.HtmxRacersDelete)
		admin.POST("/html/racers/:id/share", h.HtmxRacersGenerateShare)

		admin.GET("/html/tracks", h.HtmxTracksTable)
		admin.POST("/html/tracks", h.HtmxTracksSave)
		admin.GET("/html/tracks/:id/edit", h.HtmxTracksEditForm)
		admin.DELETE("/html/tracks/:id", h.HtmxTracksDelete)

		admin.GET("/html/quotes", h.HtmxQuotesTable)
		admin.POST("/html/quotes", h.HtmxQuotesSave)
		admin.GET("/html/quotes/:id/edit", h.HtmxQuotesEditForm)
		admin.DELETE("/html/quotes/:id", h.HtmxQuotesDelete)

		admin.GET("/html/teams", h.HtmxTeamsTable)
		admin.POST("/html/teams", h.HtmxTeamsSave)
		admin.GET("/html/teams/:id/edit", h.HtmxTeamsEditForm)
		admin.DELETE("/html/teams/:id", h.HtmxTeamsDelete)

		admin.GET("/html/seasons", h.HtmxSeasonsTable)
		admin.GET("/html/seasons/new", h.HtmxSeasonsNewForm)
		admin.POST("/html/seasons", h.HtmxSeasonsCreate)
		admin.POST("/html/seasons/:id/archive", h.HtmxSeasonsArchive)
		admin.DELETE("/html/seasons/:id", h.HtmxSeasonsDelete)

		// Admin tab fragment endpoints
		admin.GET("/html/admin/:tab", func(c *gin.Context) {
			tabID := c.Param("tab")
			if !isValidAdminTab(tabID) {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}
			tmplName := "tabRaceDay"
			switch tabID {
			case "season":
				tmplName = "tabSeason"
			case "drivers":
				tmplName = "tabDrivers"
			case "config":
				tmplName = "tabConfig"
			case "extensions":
				tmplName = "tabExtensions"
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "no-store")
			adminData := AdminData{
				Version:      server.CurrentVersion,
				TabID:        tabID,
				VendorCSS:    vendorCSSPath,
				VendorJS:     vendorJSPath,
				VendorFA:     vendorFAPath,
				VendorNavCss: vendorNavCss,
			}
			if err := adminTemplate.ExecuteTemplate(c.Writer, tmplName, adminData); err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		})

		// Admin: Logs
		admin.GET("/admin/logs", h.GetLogs)
		admin.GET("/admin/logs/modules", h.GetLogModules)
		admin.GET("/admin/log-settings", h.GetLogSettings)
		admin.POST("/admin/log-settings", h.SaveLogSettings)
	}

	r.GET("/api/uploads", h.GetUploads)
	r.GET("/api/race-info", h.GetRaceInfo)
	r.GET("/api/tracks", h.GetTracks)
	r.GET("/api/tracks/geojson", h.GetTrackGeoJSON)
	r.GET("/api/board-game/tracks", h.GetBoardGameTracks)
	r.GET("/api/race-history", h.GetRaceHistory)
	r.GET("/api/racer-stats", h.GetRacerStats)
	r.GET("/api/oneoff-races", h.GetOneOffRaces)
	r.GET("/api/track-stats", h.GetTrackStats)
	r.GET("/api/quotes", h.GetQuotes)
	r.GET("/api/quote/random", h.GetRandomQuote)
	r.GET("/api/stats/head-to-head", h.GetHeadToHead)
	r.GET("/api/stats/points-progression", h.GetPointsProgression)
	r.GET("/api/stats/streaks", h.GetStreaks)
	r.GET("/api/stats/elo", h.GetELORatings)
	r.GET("/api/stats/export", h.ExportStatsCSV)
	r.GET("/api/stats/track-performance", h.GetTrackPerformance)
	r.GET("/api/stats/qualifying-delta", h.GetQualifyingRaceDelta)
	r.GET("/api/stats/consistency", h.GetConsistencyRatings)
	r.GET("/api/stats/incidents", h.GetRaceIncidentsReport)
	r.GET("/api/stats/pace-heatmap", h.GetPaceHeatmap)
	r.POST("/api/flags", h.HandleFlag)
	r.POST("/api/rounds", h.TakeRoundSnapshot)
	r.GET("/api/rounds", h.GetRoundSnapshots)
	r.GET("/api/rounds/batch", h.GetRoundSnapshotsBatch)
	r.PATCH("/api/rounds", middleware.CSRFMiddleware(), middleware.AuthMiddleware(server), h.UpdateRoundScores)
	r.PATCH("/api/rounds/finalize", middleware.CSRFMiddleware(), middleware.AuthMiddleware(server), h.FinalizeRound)
	r.DELETE("/api/rounds", h.DeleteRoundSnapshot)
	r.GET("/api/seasons", h.GetSeasons)
	r.GET("/api/trmnl/summary", h.GetTRMNLSummary)
	r.GET("/api/trmnl/next-race", h.GetTRMNLNextRace)

	// Game Mechanics routes
	r.GET("/api/heat-cards", h.GetHeatCards)
	r.GET("/api/gear-shifts", h.GetGearShifts)
	r.GET("/api/upgrade-cards", h.GetUpgradeCards)
	r.GET("/api/player-upgrades", h.GetPlayerUpgrades)
	r.GET("/api/legend-abilities", h.GetLegendAbilities)
	r.GET("/api/racer-legend-abilities", h.GetRacerLegendAbilities)
	r.GET("/api/available-upgrades", h.GetAvailableUpgradesForRacer)

	// Multi-User routes
	r.POST("/api/player/login", h.PlayerLogin)
	r.POST("/api/player/logout", h.PlayerLogout)
	r.GET("/api/player/validate", h.ValidatePlayerToken)
	r.GET("/api/player/status", h.PlayerGetStatus)
	r.POST("/api/player/gear", h.PlayerReportGear)
	r.POST("/api/player/heat", h.PlayerReportHeat)
	r.POST("/api/player/turbo", h.PlayerReportTurbo)
	r.GET("/api/spectator/state", h.GetSpectatorState)

	// Race Enhancement routes
	r.GET("/api/weather", h.GetWeather)
	r.GET("/api/turbo-logs", h.GetTurboLogs)
	r.GET("/api/lap-records", h.GetLapRecords)
	r.POST("/api/lap-records", h.RecordLap)
	r.POST("/api/lap-records/batch", h.RecordLapBatch)
	r.GET("/api/sectors", h.GetSectors)
	r.GET("/api/racer-sectors", h.GetRacerSectors)
	r.POST("/api/racer-sectors", h.RecordRacerSector)
	r.GET("/api/race-events", h.GetRaceEvents)
	r.POST("/api/race-events", h.AddRaceEvent)
	r.GET("/api/ai-difficulty", h.GetAIDifficulty)
	r.POST("/api/sound", h.PlaySound)
	r.GET("/api/race-radio", h.GetRaceRadio)
	r.POST("/api/race-radio", h.AddRaceRadio)

	// i18n
	r.GET("/api/translations", h.GetTranslations)
	r.POST("/api/language", h.SetLanguage)

	// Driver Share (public, token-based)
	r.GET("/api/shared/driver-stats", h.GetDriverStatsByToken)
	r.GET("/api/teams", h.GetTeams)
	r.GET("/api/teams/standings", h.GetConstructorStandings)

	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": server.CurrentVersion})
	})
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/metrics/prometheus", gin.WrapH(promhttp.Handler()))

	r.GET("/api/admin/backup", middleware.CSRFMiddleware(), middleware.AuthMiddleware(server), h.TriggerManualBackup)

	r.GET("/api-docs", func(c *gin.Context) {
		c.File(filepath.Join(server.BasePath, "static/swagger.json"))
	})

	r.GET("/docs", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!DOCTYPE html>
<html>
<head>
    <title>HEAT Racing API Documentation</title>
    <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
    <script>
        window.onload = function() {
            SwaggerUIBundle({
                url: "/api-docs",
                dom_id: '#swagger-ui',
                presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
                layout: "BaseLayout"
            });
        };
    </script>
</body>
</html>`))
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerHandler))

	cacheControl := func(value string) gin.HandlerFunc {
		return func(c *gin.Context) {
			c.Header("Cache-Control", value)
			c.Next()
		}
	}

	// Content-hashed vendor assets (see build.mjs) are immutable-safe; everything
	// else under /static uses stable URLs and must be revalidated — otherwise
	// returning visitors keep stale bundles (e.g. missing car colors) for a year.
	cacheControlStatic := func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/static/vendor/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			c.Header("Cache-Control", "no-cache")
		}
		c.Next()
	}
	r.Group("/static", cacheControlStatic).Static("", filepath.Join(server.BasePath, "static"))
	r.Group("/media", cacheControl("public, max-age=86400")).Static("", server.MediaPath)
	r.GET("/sw.js", func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache")
		c.File(filepath.Join(server.BasePath, "static/sw.js"))
	})

	pages := r.Group("")
	pages.Use(middleware.UmamiMiddleware(server))
	pages.Use(h.I18nMiddleware())
	{
		pages.GET("/admin.html", cacheControl("no-store"), func(c *gin.Context) {
			var validSession string
			for _, cookie := range c.Request.Cookies() {
				if cookie.Name == "session" {
					server.SessionStoreMu.RLock()
					_, ok := server.SessionStore[cookie.Value]
					server.SessionStoreMu.RUnlock()
					if ok {
						validSession = cookie.Value
						break
					}
				}
			}

			if validSession == "" {
				c.Redirect(http.StatusFound, "/login.html")
				return
			}
			c.Header("Content-Type", "text/html; charset=utf-8")
			adminData := AdminData{
				Version:      server.CurrentVersion,
				TabID:        "race-day",
				VendorCSS:    vendorCSSPath,
				VendorJS:     vendorJSPath,
				VendorFA:     vendorFAPath,
				VendorNavCss: vendorNavCss,
			}
			if err := adminTemplate.ExecuteTemplate(c.Writer, "admin.html", adminData); err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		})

		r.GET("/login.html", func(c *gin.Context) {
			cookie, err := c.Request.Cookie("session")
			if err == nil {
				server.SessionStoreMu.RLock()
				info, ok := server.SessionStore[cookie.Value]
				server.SessionStoreMu.RUnlock()
				if ok && time.Now().Unix() <= info.Expiry {
					c.Redirect(http.StatusFound, "/admin.html")
					return
				}
			}
			var count int
			server.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
			if count == 0 {
				c.Redirect(http.StatusFound, "/setup")
				return
			}
			c.File(filepath.Join(server.BasePath, "static/login.html"))
		})

		r.GET("/setup", func(c *gin.Context) {
			var count int
			server.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
			if count > 0 {
				c.Redirect(http.StatusFound, "/login.html")
				return
			}
			c.File(filepath.Join(server.BasePath, "static/setup.html"))
		})

		pages.GET("/forgot-password.html", func(c *gin.Context) {
			c.File(filepath.Join(server.BasePath, "static/forgot-password.html"))
		})

		pages.GET("/reset-password.html", func(c *gin.Context) {
			c.File(filepath.Join(server.BasePath, "static/reset-password.html"))
		})

		pages.GET("/controller.html", func(c *gin.Context) {
			var validSession string
			for _, cookie := range c.Request.Cookies() {
				if cookie.Name == "session" {
					server.SessionStoreMu.RLock()
					_, ok := server.SessionStore[cookie.Value]
					server.SessionStoreMu.RUnlock()
					if ok {
						validSession = cookie.Value
						break
					}
				}
			}
			if validSession == "" {
				c.Redirect(http.StatusFound, "/login.html")
				return
			}
			c.File(filepath.Join(server.BasePath, "static/controller.html"))
		})

		pages.GET("/stats.html", func(c *gin.Context) {
			serveTemplate(c, "stats.html", server)
		})

		pages.GET("/seasons.html", func(c *gin.Context) {
			serveTemplate(c, "seasons.html", server)
		})

		pages.GET("/trophies.html", func(c *gin.Context) {
			serveTemplate(c, "trophies.html", server)
		})

		pages.GET("/tv.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/tv.html"), server)
		})

		pages.GET("/pitboard.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/pitboard.html"), server)
		})

		pages.GET("/replay.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/replay.html"), server)
		})

		pages.GET("/player.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/player.html"), server)
		})

		pages.GET("/spectator.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/spectator.html"), server)
		})

		pages.GET("/driver.html", func(c *gin.Context) {
			c.File(filepath.Join(server.BasePath, "static/driver.html"))
		})

		pages.GET("/", func(c *gin.Context) {
			serveTemplate(c, "index.html", server)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "6270"
	}

	if server.Log != nil {
		server.Log.Infof("server", "Starting on port %s...", port)
	} else {
		log.Printf("Server starting on port %s...", port)
	}
	r.Run(":" + port)
}
