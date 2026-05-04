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
		app.ImagesPath = "/app/images"

		if os.Getenv("DOCKER") != "true" {
			app.BasePath = "."
			app.DBPath = "./heat.db"
		}
		app.ImagesPath = filepath.Join(app.BasePath, "static/images")

		if app.BasePath != "." {
			t.Errorf("expected local basePath '.', got %s", app.BasePath)
		}
		if app.DBPath != "./heat.db" {
			t.Errorf("expected local dbPath './heat.db', got %s", app.DBPath)
		}
		if app.ImagesPath != "static/images" {
			t.Errorf("expected imagesPath 'static/images', got %s", app.ImagesPath)
		}
	})

	t.Run("DockerPaths", func(t *testing.T) {
		os.Setenv("DOCKER", "true")
		defer os.Unsetenv("DOCKER")

		app.BasePath = "/app"
		app.DBPath = "/db/heat.db"
		app.ImagesPath = "/app/images"

		if os.Getenv("DOCKER") != "true" {
			app.BasePath = "."
			app.DBPath = "./heat.db"
		}
		app.ImagesPath = filepath.Join(app.BasePath, "static/images")

		if app.BasePath != "/app" {
			t.Errorf("expected Docker basePath '/app', got %s", app.BasePath)
		}
		if app.DBPath != "/db/heat.db" {
			t.Errorf("expected Docker dbPath '/db/heat.db', got %s", app.DBPath)
		}
		if app.ImagesPath != "/app/static/images" {
			t.Errorf("expected Docker imagesPath '/app/static/images', got %s", app.ImagesPath)
		}
	})
}
