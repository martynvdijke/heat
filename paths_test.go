package main

import (
	"os"
	"path/filepath"
	"testing"

	"heat/app"
)

func TestPathsConfiguration(t *testing.T) {
	t.Run("LocalPaths", func(t *testing.T) {
		os.Unsetenv("DOCKER")

		app.BasePath = "/app"
		app.DBPath = "/db/heat.db"
		app.MediaPath = "/app/media"

		if os.Getenv("DOCKER") != "true" {
			app.BasePath = "."
			app.DBPath = "./heat.db"
		}
		app.MediaPath = filepath.Join(app.BasePath, "media")

		if app.BasePath != "." {
			t.Errorf("expected local basePath '.', got %s", app.BasePath)
		}
		if app.DBPath != "./heat.db" {
			t.Errorf("expected local dbPath './heat.db', got %s", app.DBPath)
		}
		if app.MediaPath != "media" {
			t.Errorf("expected mediaPath 'media', got %s", app.MediaPath)
		}
	})

	t.Run("DockerPaths", func(t *testing.T) {
		os.Setenv("DOCKER", "true")
		defer os.Unsetenv("DOCKER")

		app.BasePath = "/app"
		app.DBPath = "/db/heat.db"
		app.MediaPath = "/app/media"

		if os.Getenv("DOCKER") != "true" {
			app.BasePath = "."
			app.DBPath = "./heat.db"
		}
		app.MediaPath = filepath.Join(app.BasePath, "media")

		if app.BasePath != "/app" {
			t.Errorf("expected Docker basePath '/app', got %s", app.BasePath)
		}
		if app.DBPath != "/db/heat.db" {
			t.Errorf("expected Docker dbPath '/db/heat.db', got %s", app.DBPath)
		}
		if app.MediaPath != "/app/media" {
			t.Errorf("expected Docker mediaPath '/app/media', got %s", app.MediaPath)
		}
	})
}
