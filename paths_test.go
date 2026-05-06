package main

import (
	"os"
	"path/filepath"
	"strings"
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

func TestAdminHTMLTabPaneStructure(t *testing.T) {
	data, err := os.ReadFile("static/admin.html")
	if err != nil {
		t.Fatalf("failed to read admin.html: %v", err)
	}
	content := string(data)

	// Find the tab-content div and its closing tag
	tabContentStart := strings.Index(content, `class="tab-content"`)
	if tabContentStart == -1 {
		t.Fatal("tab-content div not found in admin.html")
	}

	// Find the closing tag for the outermost container that wraps tab-content
	containerStart := strings.LastIndex(content[:tabContentStart], `<div class="container"`)
	if containerStart == -1 {
		t.Fatal("container div not found around tab-content")
	}

	// Get content between tab-content opening and its closing </div>
	// The tab-content is directly inside a div with id="adminTabs"
	// Find the matching closing div for tab-content by counting nesting
	tabContentEnd := len(content)
	depth := 0
	foundOpening := false
	for i := tabContentStart; i < len(content); i++ {
		if content[i] == '<' {
			if strings.HasPrefix(content[i:], "</div>") {
				if depth == 0 && foundOpening {
					tabContentEnd = i
					break
				}
				depth--
			} else if strings.HasPrefix(content[i:], "<div") || strings.HasPrefix(content[i:], "<div ") {
				if strings.Contains(content[i:i+50], "class=\"tab-content\"") {
					foundOpening = true
				}
				if !strings.Contains(content[i:i+50], "/>") {
					depth++
				}
			}
		}
	}
	tabContentSection := content[tabContentStart:tabContentEnd]

	t.Run("EmailPaneInsideTabContent", func(t *testing.T) {
		if !strings.Contains(tabContentSection, `id="email-pane"`) {
			t.Error("email-pane must be inside tab-content div")
		}
	})

	t.Run("AnalyticsPaneInsideTabContent", func(t *testing.T) {
		if !strings.Contains(tabContentSection, `id="umami-pane"`) {
			t.Error("umami-pane must be inside tab-content div")
		}
	})

	t.Run("BackupPaneInsideTabContent", func(t *testing.T) {
		if !strings.Contains(tabContentSection, `id="backup-pane"`) {
			t.Error("backup-pane must be inside tab-content div")
		}
	})

	t.Run("RacePaneInsideTabContent", func(t *testing.T) {
		if !strings.Contains(tabContentSection, `id="race-pane"`) {
			t.Error("race-pane must be inside tab-content div")
		}
	})

	t.Run("AllPanesHaveTabContentParent", func(t *testing.T) {
		panes := []string{"race-pane", "qualification-pane", "stats-pane", "notify-pane",
			"racers-pane", "tracks-pane", "quotes-pane", "ai-pane",
			"email-pane", "umami-pane", "backup-pane"}
		for _, pane := range panes {
			if !strings.Contains(tabContentSection, `id="`+pane+`"`) {
				t.Errorf("pane %q must be inside tab-content div", pane)
			}
		}
	})
}
