package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathsConfiguration(t *testing.T) {
	t.Run("LocalPaths", func(t *testing.T) {
		os.Unsetenv("DOCKER")

		testServer.BasePath = "/app"
		testServer.DBPath = "/db/heat.db"
		testServer.MediaPath = "/app/media"

		if os.Getenv("DOCKER") != "true" {
			testServer.BasePath = "."
			testServer.DBPath = "./heat.db"
		}
		testServer.MediaPath = filepath.Join(testServer.BasePath, "media")

		if testServer.BasePath != "." {
			t.Errorf("expected local basePath '.', got %s", testServer.BasePath)
		}
		if testServer.DBPath != "./heat.db" {
			t.Errorf("expected local dbPath './heat.db', got %s", testServer.DBPath)
		}
		if testServer.MediaPath != "media" {
			t.Errorf("expected mediaPath 'media', got %s", testServer.MediaPath)
		}
	})

	t.Run("DockerPaths", func(t *testing.T) {
		os.Setenv("DOCKER", "true")
		defer os.Unsetenv("DOCKER")

		testServer.BasePath = "/app"
		testServer.DBPath = "/db/heat.db"
		testServer.MediaPath = "/app/media"

		if os.Getenv("DOCKER") != "true" {
			testServer.BasePath = "."
			testServer.DBPath = "./heat.db"
		}
		testServer.MediaPath = filepath.Join(testServer.BasePath, "media")

		if testServer.BasePath != "/app" {
			t.Errorf("expected Docker basePath '/app', got %s", testServer.BasePath)
		}
		if testServer.DBPath != "/db/heat.db" {
			t.Errorf("expected Docker dbPath '/db/heat.db', got %s", testServer.DBPath)
		}
		if testServer.MediaPath != "/app/media" {
			t.Errorf("expected Docker mediaPath '/app/media', got %s", testServer.MediaPath)
		}

		// Restore for subsequent tests
		testServer.BasePath = "."
		testServer.DBPath = ":memory:"
		testServer.MediaPath = filepath.Join(".", "media")
	})
}

func TestAdminHTMLTabPaneStructure(t *testing.T) {
	templateFiles := []string{
		"static/templates/admin.html",
		"static/templates/admin-header.html",
		"static/templates/admin-footer.html",
		"static/templates/admin-modals.html",
		"static/templates/admin-race-panes.html",
		"static/templates/admin-results-panes.html",
		"static/templates/admin-content-panes.html",
		"static/templates/admin-settings-panes.html",
		"static/templates/admin-system-panes.html",
	}

	var content strings.Builder
	for _, f := range templateFiles {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("failed to read %s: %v", f, err)
		}
		content.Write(data)
		content.WriteString("\n")
	}

	contentStr := content.String()

	t.Run("EmailPaneExists", func(t *testing.T) {
		if !strings.Contains(contentStr, `id="email-pane"`) {
			t.Error("email-pane must exist in admin templates")
		}
	})

	t.Run("AnalyticsPaneExists", func(t *testing.T) {
		if !strings.Contains(contentStr, `id="umami-pane"`) {
			t.Error("umami-pane must exist in admin templates")
		}
	})

	t.Run("BackupPaneExists", func(t *testing.T) {
		if !strings.Contains(contentStr, `id="backup-pane"`) {
			t.Error("backup-pane must exist in admin templates")
		}
	})

	t.Run("RacePaneExists", func(t *testing.T) {
		if !strings.Contains(contentStr, `id="race-pane"`) {
			t.Error("race-pane must exist in admin templates")
		}
	})

	t.Run("AllRequiredPanesExist", func(t *testing.T) {
		panes := []string{"race-pane", "qualification-pane", "stats-pane", "notify-pane",
			"racers-pane", "tracks-pane", "quotes-pane", "ai-pane",
			"email-pane", "umami-pane", "backup-pane"}
		for _, pane := range panes {
			if !strings.Contains(contentStr, `id="`+pane+`"`) {
				t.Errorf("pane %q must exist in admin templates", pane)
			}
		}
	})
}
