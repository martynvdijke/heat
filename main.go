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
	"context"
	"database/sql"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
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
	"heat/ws"
)

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
	server.DB, err = sql.Open("sqlite3", server.DBPath+"?_fk=1")
	if err != nil {
		log.Fatal(err)
	}
	server.DB.SetMaxOpenConns(1)
	server.DB.Exec("PRAGMA journal_mode=WAL")

	drv := entsql.OpenDB(dialect.SQLite, server.DB)
	server.Ent = ent.NewClient(ent.Driver(drv))
	defer server.Ent.Close()

	// Initialize structured logger
	server.Log = logger.New(server.DB)
	defer server.Log.Stop()

	h := handlers.New(server)
	wsManager := ws.NewManager(server)
	server.BroadcastRacers = wsManager.BroadcastRacers
	server.BroadcastSelfService = wsManager.BroadcastSelfService

	db.Init(server)

	// Initialize OpenTelemetry after DB is ready (reads OTel settings from database)
	otelShutdown := initOTel(server)
	defer otelShutdown()
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
	r.Use(middleware.RequestIDMiddleware())
	r.Use(middleware.SecurityHeaders())

	r.Use(metricsMiddleware())

	r.GET("/ws", wsManager.HandleWebSocket)

	r.POST("/api/login", middleware.RateLimitMiddleware(server), h.HandleLogin)
	r.POST("/api/logout", middleware.CSRFMiddleware(), h.HandleLogout)
	r.GET("/api/check-setup", h.HandleCheckSetup)

	r.GET("/api/racers", h.GetRacers)
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(server))
	{
		admin.POST("/racers", h.UpdateRacer)
		admin.DELETE("/racers", h.DeleteRacer)
		admin.POST("/race-info", h.UpdateRaceInfo)
		admin.POST("/upload", h.HandleUpload)
		admin.POST("/tracks", h.SaveTrack)
		admin.DELETE("/tracks", h.DeleteTrack)
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

		// Admin: Logs
		admin.GET("/admin/logs", h.GetLogs)
		admin.GET("/admin/logs/modules", h.GetLogModules)
		admin.GET("/admin/log-settings", h.GetLogSettings)
		admin.POST("/admin/log-settings", h.SaveLogSettings)
	}

	r.GET("/api/uploads", h.GetUploads)
	r.GET("/api/race-info", h.GetRaceInfo)
	r.GET("/api/tracks", h.GetTracks)
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
	r.GET("/api/race-report", h.GetRaceReport)
	r.POST("/api/flags", h.HandleFlag)
	r.POST("/api/rounds", h.TakeRoundSnapshot)
	r.GET("/api/rounds", h.GetRoundSnapshots)
	r.DELETE("/api/rounds", h.DeleteRoundSnapshot)
	r.GET("/api/seasons", h.GetSeasons)

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

	r.Static("/media", server.MediaPath)
	r.Static("/static", filepath.Join(server.BasePath, "static"))
	r.StaticFile("/sw.js", filepath.Join(server.BasePath, "static/sw.js"))

	pages := r.Group("")
	pages.Use(middleware.UmamiMiddleware(server))
	pages.Use(h.I18nMiddleware())
	{
		pages.GET("/admin.html", func(c *gin.Context) {
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
			c.File(filepath.Join(server.BasePath, "static/admin.html"))
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
			servePage(c, filepath.Join(server.BasePath, "static/stats.html"), server)
		})

		pages.GET("/seasons.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/seasons.html"), server)
		})

		pages.GET("/trophies.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/trophies.html"), server)
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

		pages.GET("/race-report.html", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/race-report.html"), server)
		})

		pages.GET("/", func(c *gin.Context) {
			servePage(c, filepath.Join(server.BasePath, "static/index.html"), server)
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
