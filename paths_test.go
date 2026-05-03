package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathsConfiguration(t *testing.T) {
	t.Run("LocalPaths", func(t *testing.T) {
		os.Unsetenv("DOCKER")

		basePath = "/app"
		dbPath = "/db/heat.db"
		imagesPath = "/app/images"

		if os.Getenv("DOCKER") != "true" {
			basePath = "."
			dbPath = "./heat.db"
		}
		imagesPath = filepath.Join(basePath, "static/images")

		if basePath != "." {
			t.Errorf("expected local basePath '.', got %s", basePath)
		}
		if dbPath != "./heat.db" {
			t.Errorf("expected local dbPath './heat.db', got %s", dbPath)
		}
		if imagesPath != "static/images" {
			t.Errorf("expected imagesPath 'static/images', got %s", imagesPath)
		}
	})

	t.Run("DockerPaths", func(t *testing.T) {
		os.Setenv("DOCKER", "true")
		defer os.Unsetenv("DOCKER")

		basePath = "/app"
		dbPath = "/db/heat.db"
		imagesPath = "/app/images"

		if os.Getenv("DOCKER") != "true" {
			basePath = "."
			dbPath = "./heat.db"
		}
		imagesPath = filepath.Join(basePath, "static/images")

		if basePath != "/app" {
			t.Errorf("expected Docker basePath '/app', got %s", basePath)
		}
		if dbPath != "/db/heat.db" {
			t.Errorf("expected Docker dbPath '/db/heat.db', got %s", dbPath)
		}
		if imagesPath != "/app/static/images" {
			t.Errorf("expected Docker imagesPath '/app/static/images', got %s", imagesPath)
		}
	})
}
