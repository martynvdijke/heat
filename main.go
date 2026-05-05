package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"

	"heat/app"
	"heat/db"
	"heat/handlers"
	"heat/middleware"
	"heat/ws"
)

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

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.SecurityHeaders())

	r.GET("/ws", ws.HandleWebSocket)

	r.POST("/api/login", middleware.RateLimitMiddleware(), handlers.HandleLogin)
	r.POST("/api/logout", handlers.HandleLogout)
	r.GET("/api/check-setup", handlers.HandleCheckSetup)

	r.GET("/api/racers", handlers.GetRacers)
	r.POST("/api/racers", middleware.AuthMiddleware(), handlers.UpdateRacer)
	r.DELETE("/api/racers", middleware.AuthMiddleware(), handlers.DeleteRacer)

	r.GET("/api/race-info", handlers.GetRaceInfo)
	r.POST("/api/race-info", middleware.AuthMiddleware(), handlers.UpdateRaceInfo)

	r.GET("/api/uploads", handlers.GetUploads)
	r.POST("/api/upload", middleware.AuthMiddleware(), handlers.HandleUpload)

	r.GET("/api/tracks", handlers.GetTracks)
	r.POST("/api/tracks", middleware.AuthMiddleware(), handlers.SaveTrack)
	r.DELETE("/api/tracks", middleware.AuthMiddleware(), handlers.DeleteTrack)

	r.POST("/api/tracks/ai-extract", middleware.AuthMiddleware(), handlers.HandleAIExtract)

	r.GET("/api/race-history", handlers.GetRaceHistory)
	r.POST("/api/race-history", middleware.AuthMiddleware(), handlers.SaveRaceToHistory)
	r.DELETE("/api/race-history", middleware.AuthMiddleware(), handlers.DeleteRaceHistory)

	r.GET("/api/racer-stats", handlers.GetRacerStats)
	r.POST("/api/racer-stats", middleware.AuthMiddleware(), handlers.UpdateRacerStats)

	r.GET("/api/notification-settings", middleware.AuthMiddleware(), handlers.GetNotificationSettings)
	r.POST("/api/notification-settings", middleware.AuthMiddleware(), handlers.SaveNotificationSettings)

	r.POST("/api/test-notification", middleware.AuthMiddleware(), handlers.TestNotification)

	r.GET("/api/ai-settings", middleware.AuthMiddleware(), handlers.GetAISettings)
	r.POST("/api/ai-settings", middleware.AuthMiddleware(), handlers.SaveAISettings)

	r.GET("/api/email-settings", middleware.AuthMiddleware(), handlers.GetEmailSettings)
	r.POST("/api/email-settings", middleware.AuthMiddleware(), handlers.SaveEmailSettings)
	r.GET("/api/racer-emails", middleware.AuthMiddleware(), handlers.GetRacerEmails)
	r.POST("/api/racer-emails", middleware.AuthMiddleware(), handlers.SaveRacerEmail)
	r.POST("/api/send-race-email", middleware.AuthMiddleware(), handlers.SendRaceEmailManual)

	r.GET("/api/oneoff-races", handlers.GetOneOffRaces)
	r.DELETE("/api/oneoff-races", middleware.AuthMiddleware(), handlers.DeleteOneOffRace)

	r.GET("/api/track-stats", handlers.GetTrackStats)

	r.GET("/api/umami-settings", middleware.AuthMiddleware(), handlers.GetUmamiSettings)
	r.POST("/api/umami-settings", middleware.AuthMiddleware(), handlers.SaveUmamiSettings)

	r.GET("/api/quotes", handlers.GetQuotes)
	r.POST("/api/quotes", middleware.AuthMiddleware(), handlers.HandleQuotes)
	r.PUT("/api/quotes", middleware.AuthMiddleware(), handlers.HandleQuotes)
	r.DELETE("/api/quotes", middleware.AuthMiddleware(), handlers.HandleQuotes)

	r.GET("/api/quote/random", handlers.GetRandomQuote)

	// New Statistics & Analytics endpoints
	r.GET("/api/stats/head-to-head", handlers.GetHeadToHead)
	r.GET("/api/stats/points-progression", handlers.GetPointsProgression)
	r.GET("/api/stats/streaks", handlers.GetStreaks)
	r.GET("/api/stats/elo", handlers.GetELORatings)
	r.GET("/api/stats/export", handlers.ExportStatsCSV)
	r.GET("/api/stats/track-performance", handlers.GetTrackPerformance)

	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": app.CurrentVersion})
	})

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
					if _, ok := app.SessionStore[cookie.Value]; ok {
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
				expiry, ok := app.SessionStore[cookie.Value]
				if ok && time.Now().Unix() <= expiry {
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
					if _, ok := app.SessionStore[cookie.Value]; ok {
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
			c.File(filepath.Join(app.BasePath, "static/stats.html"))
		})

		pages.GET("/trophies.html", func(c *gin.Context) {
			c.File(filepath.Join(app.BasePath, "static/trophies.html"))
		})

		pages.GET("/chat.html", func(c *gin.Context) {
			c.File(filepath.Join(app.BasePath, "static/chat.html"))
		})

		pages.GET("/", func(c *gin.Context) {
			c.File(filepath.Join(app.BasePath, "static/index.html"))
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
