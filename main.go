package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/db"
	"heat/handlers"
	"heat/middleware"
	"heat/ws"
)

func servePage(c *gin.Context, path string) {
	content, err := os.ReadFile(path)
	if err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	html := strings.Replace(string(content), "{{VERSION}}", app.CurrentVersion, 1)
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

func main() {
	if os.Getenv("DOCKER") != "true" {
		app.BasePath = "."
		app.DBPath = "./heat.db"
	}
	app.MediaPath = filepath.Join(app.BasePath, "media")

	if err := os.MkdirAll(app.MediaPath, 0755); err != nil {
		log.Printf("Warning: could not create media directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(app.DBPath), 0755); err != nil {
		log.Printf("Warning: could not create database directory: %v", err)
	}

	var err error
	app.DB, err = sql.Open("sqlite3", app.DBPath)
	if err != nil {
		log.Fatal(err)
	}
	app.DB.SetMaxOpenConns(1)
	defer app.DB.Close()

	db.Init()
	go ws.BroadcastManager()
	go func() {
		for {
			time.Sleep(15 * time.Minute)
			now := time.Now().Unix()
			app.SessionStoreMu.Lock()
			for k, v := range app.SessionStore {
				if now > v.Expiry {
					delete(app.SessionStore, k)
				}
			}
			app.SessionStoreMu.Unlock()
		}
	}()
	go func() {
		for {
			var enabled int
			var intervalHrs int
			app.DB.QueryRow("SELECT enabled, interval_hrs FROM backup_settings WHERE id = 1").Scan(&enabled, &intervalHrs)
			if enabled == 1 && intervalHrs > 0 {
				if err := db.CreateBackup(); err != nil {
					log.Printf("[BACKUP] Periodic backup failed: %v", err)
				} else {
					log.Printf("[BACKUP] Periodic backup completed (interval: %dh)", intervalHrs)
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

	r.GET("/ws", ws.HandleWebSocket)

	r.POST("/api/login", middleware.RateLimitMiddleware(), handlers.HandleLogin)
	r.POST("/api/logout", middleware.CSRFMiddleware(), handlers.HandleLogout)
	r.GET("/api/check-setup", handlers.HandleCheckSetup)

	r.GET("/api/racers", handlers.GetRacers)
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware())
	{
		admin.POST("/racers", handlers.UpdateRacer)
		admin.DELETE("/racers", handlers.DeleteRacer)
		admin.POST("/race-info", handlers.UpdateRaceInfo)
		admin.POST("/upload", handlers.HandleUpload)
		admin.POST("/tracks", handlers.SaveTrack)
		admin.DELETE("/tracks", handlers.DeleteTrack)
		admin.POST("/tracks/ai-extract", handlers.HandleAIExtract)
		admin.POST("/race-history", handlers.SaveRaceToHistory)
		admin.DELETE("/race-history", handlers.DeleteRaceHistory)
		admin.POST("/racer-stats", handlers.UpdateRacerStats)
		admin.GET("/notification-settings", handlers.GetNotificationSettings)
		admin.POST("/notification-settings", handlers.SaveNotificationSettings)
		admin.POST("/test-notification", handlers.TestNotification)
		admin.GET("/ai-settings", handlers.GetAISettings)
		admin.POST("/ai-settings", handlers.SaveAISettings)
		admin.GET("/email-settings", handlers.GetEmailSettings)
		admin.POST("/email-settings", handlers.SaveEmailSettings)
		admin.GET("/racer-emails", handlers.GetRacerEmails)
		admin.POST("/racer-emails", handlers.SaveRacerEmail)
		admin.POST("/send-race-email", handlers.SendRaceEmailManual)
		admin.DELETE("/oneoff-races", handlers.DeleteOneOffRace)
		admin.GET("/umami-settings", handlers.GetUmamiSettings)
		admin.POST("/umami-settings", handlers.SaveUmamiSettings)
		admin.POST("/quotes", handlers.HandleQuotes)
		admin.PUT("/quotes", handlers.HandleQuotes)
		admin.DELETE("/quotes", handlers.HandleQuotes)
		admin.GET("/backup-settings", handlers.GetBackupSettings)
		admin.POST("/backup-settings", handlers.SaveBackupSettings)
		admin.POST("/backup/manual", handlers.TriggerManualBackup)
		admin.GET("/backup/list", handlers.ListBackups)
	}

	r.GET("/api/uploads", handlers.GetUploads)
	r.GET("/api/race-info", handlers.GetRaceInfo)
	r.GET("/api/tracks", handlers.GetTracks)
	r.GET("/api/race-history", handlers.GetRaceHistory)
	r.GET("/api/racer-stats", handlers.GetRacerStats)
	r.GET("/api/oneoff-races", handlers.GetOneOffRaces)
	r.GET("/api/track-stats", handlers.GetTrackStats)
	r.GET("/api/quotes", handlers.GetQuotes)
	r.GET("/api/quote/random", handlers.GetRandomQuote)
	r.GET("/api/stats/head-to-head", handlers.GetHeadToHead)
	r.GET("/api/stats/points-progression", handlers.GetPointsProgression)
	r.GET("/api/stats/streaks", handlers.GetStreaks)
	r.GET("/api/stats/elo", handlers.GetELORatings)
	r.GET("/api/stats/export", handlers.ExportStatsCSV)
	r.GET("/api/stats/track-performance", handlers.GetTrackPerformance)

	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": app.CurrentVersion})
	})

	r.GET("/api/admin/backup", middleware.CSRFMiddleware(), middleware.AuthMiddleware(), handlers.TriggerManualBackup)

	r.GET("/api-docs", func(c *gin.Context) {
		c.File(filepath.Join(app.BasePath, "static/swagger.json"))
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

	r.Static("/media", app.MediaPath)
	r.Static("/static", filepath.Join(app.BasePath, "static"))

	pages := r.Group("")
	pages.Use(middleware.UmamiMiddleware())
	{
		pages.GET("/admin.html", func(c *gin.Context) {
			var validSession string
			for _, cookie := range c.Request.Cookies() {
				if cookie.Name == "session" {
					app.SessionStoreMu.RLock()
					_, ok := app.SessionStore[cookie.Value]
					app.SessionStoreMu.RUnlock()
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
			c.File(filepath.Join(app.BasePath, "static/admin.html"))
		})

		r.GET("/login.html", func(c *gin.Context) {
			cookie, err := c.Request.Cookie("session")
			if err == nil {
				app.SessionStoreMu.RLock()
				info, ok := app.SessionStore[cookie.Value]
				app.SessionStoreMu.RUnlock()
				if ok && time.Now().Unix() <= info.Expiry {
					c.Redirect(http.StatusFound, "/admin.html")
					return
				}
			}
			var count int
			app.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
			if count == 0 {
				c.Redirect(http.StatusFound, "/setup")
				return
			}
			c.File(filepath.Join(app.BasePath, "static/login.html"))
		})

		r.GET("/setup", func(c *gin.Context) {
			var count int
			app.DB.QueryRow("SELECT COUNT(*) FROM admin_users").Scan(&count)
			if count > 0 {
				c.Redirect(http.StatusFound, "/login.html")
				return
			}
			c.File(filepath.Join(app.BasePath, "static/setup.html"))
		})

		pages.GET("/controller.html", func(c *gin.Context) {
			var validSession string
			for _, cookie := range c.Request.Cookies() {
				if cookie.Name == "session" {
					app.SessionStoreMu.RLock()
					_, ok := app.SessionStore[cookie.Value]
					app.SessionStoreMu.RUnlock()
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
			c.File(filepath.Join(app.BasePath, "static/controller.html"))
		})

		pages.GET("/stats.html", func(c *gin.Context) {
			servePage(c, filepath.Join(app.BasePath, "static/stats.html"))
		})

		pages.GET("/trophies.html", func(c *gin.Context) {
			servePage(c, filepath.Join(app.BasePath, "static/trophies.html"))
		})

		pages.GET("/", func(c *gin.Context) {
			servePage(c, filepath.Join(app.BasePath, "static/index.html"))
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "6270"
	}

	log.Printf("Server starting on port %s...", port)
	r.Run(":" + port)
}

func init() {
	if os.Getenv("DOCKER") != "true" {
		app.BasePath = "."
		app.DBPath = "./heat.db"
	}
}
