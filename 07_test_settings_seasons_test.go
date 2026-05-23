package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/middleware"
	"heat/models"
)

func TestGetAISettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/ai-settings", middleware.AuthMiddleware(testServer), testHandler.GetAISettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/ai-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetNotificationSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/notification-settings", middleware.AuthMiddleware(testServer), testHandler.GetNotificationSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/notification-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetEmailSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/email-settings", middleware.AuthMiddleware(testServer), testHandler.GetEmailSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/email-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestGetUmamiSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/umami-settings", middleware.AuthMiddleware(testServer), testHandler.GetUmamiSettings)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/umami-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}

func TestBackupSettings(t *testing.T) {
	r := gin.New()
	r.GET("/api/backup-settings", testHandler.GetBackupSettings)
	r.POST("/api/backup-settings", testHandler.SaveBackupSettings)
	r.POST("/api/backup/manual", testHandler.TriggerManualBackup)
	r.GET("/api/backup/list", testHandler.ListBackups)

	t.Run("GetSettings", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/backup-settings", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var s models.BackupSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.ID != 1 {
			t.Errorf("expected id 1, got %d", s.ID)
		}
	})

	t.Run("SaveSettings", func(t *testing.T) {
		body, _ := json.Marshal(models.BackupSettings{Enabled: false, IntervalHrs: 12})
		req, _ := http.NewRequest("POST", "/api/backup-settings", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		req, _ = http.NewRequest("GET", "/api/backup-settings", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		var s models.BackupSettings
		json.Unmarshal(rr.Body.Bytes(), &s)
		if s.Enabled != false || s.IntervalHrs != 12 {
			t.Errorf("expected enabled=false interval=12, got enabled=%v interval=%d", s.Enabled, s.IntervalHrs)
		}
	})

	t.Run("ManualBackup", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/api/backup/manual", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v: %s", status, rr.Body.String())
		}
	})

	t.Run("ListBackups", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/backup/list", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("expected status 200, got %v", status)
		}

		var backups []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &backups)
		if backups == nil {
			t.Errorf("expected backups array, got nil")
		}
	})

	t.Run("PruneBackups", func(t *testing.T) {
		backupDir := filepath.Join(filepath.Dir(testServer.DBPath), "backups")
		os.MkdirAll(backupDir, 0755)

		for i := 1; i <= 10; i++ {
			name := fmt.Sprintf("heat_backup_20260101_%06d.db", i)
			os.WriteFile(filepath.Join(backupDir, name), []byte("test"), 0644)
		}

		db.PruneBackups()

		entries, _ := os.ReadDir(backupDir)
		var remaining int
		for _, e := range entries {
			if !e.IsDir() && strings.HasPrefix(e.Name(), "heat_backup_") {
				remaining++
			}
		}
		if remaining != 7 {
			t.Errorf("expected 7 backups after prune, got %d", remaining)
		}

		os.RemoveAll(backupDir)
	})

	// Reset settings back for other tests
	body, _ := json.Marshal(models.BackupSettings{Enabled: true, IntervalHrs: 24, RetentionCount: 7})
	req, _ := http.NewRequest("POST", "/api/backup-settings", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("reset: expected status 200, got %v", rr.Code)
	}
}

// session helper for authenticated admin tests

func TestSettingsSave(t *testing.T) {
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("save notification settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin.POST("/notification-settings", testHandler.SaveNotificationSettings)
		admin.GET("/notification-settings", testHandler.GetNotificationSettings)

		s := models.NotificationSettings{GotiFyURL: "https://gotify.example.com", GotiFyToken: "token123", NotifyWinner: true, NotifyPodium: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/notification-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify saved (token should be hidden but settings should stick)
		req, _ = http.NewRequest("GET", "/api/notification-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["gotify_url"] != "https://gotify.example.com" {
			t.Errorf("expected gotify_url saved, got %v", resp["gotify_url"])
		}
	})

	t.Run("save AI settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin.POST("/ai-settings", testHandler.SaveAISettings)
		admin.GET("/ai-settings", testHandler.GetAISettings)

		s := models.AISettings{TrackExtractURL: "https://ai.example.com/extract", APIKey: "key123", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/ai-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/ai-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["track_extract_url"] != "https://ai.example.com/extract" {
			t.Errorf("expected track_extract_url saved, got %v", resp["track_extract_url"])
		}
	})

	t.Run("save email settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin.POST("/email-settings", testHandler.SaveEmailSettings)
		admin.GET("/email-settings", testHandler.GetEmailSettings)

		s := models.EmailSettings{SMTPHost: "smtp.example.com", SMTPPort: 587, Username: "user", Password: "pass123", FromAddr: "from@example.com", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/email-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/email-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["smtp_host"] != "smtp.example.com" {
			t.Errorf("expected smtp_host saved, got %v", resp["smtp_host"])
		}
	})

	t.Run("save umami settings", func(t *testing.T) {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin.POST("/umami-settings", testHandler.SaveUmamiSettings)
		admin.GET("/umami-settings", testHandler.GetUmamiSettings)

		s := models.UmamiSettings{URL: "https://analytics.example.com", WebsiteID: "abc-123", Enabled: true}
		body, _ := json.Marshal(s)
		req := newAdminRequest("POST", "/api/umami-settings", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		// verify
		req, _ = http.NewRequest("GET", "/api/umami-settings", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if resp["url"] != "https://analytics.example.com" {
			t.Errorf("expected url saved, got %v", resp["url"])
		}
		if resp["website_id"] != "abc-123" {
			t.Errorf("expected website_id saved, got %v", resp["website_id"])
		}
	})
}

func TestDeleteSeason(t *testing.T) {
	r := gin.New()
	r.GET("/api/seasons", testHandler.GetSeasons)
	r.POST("/api/seasons", testHandler.CreateSeason)
	r.DELETE("/api/seasons", testHandler.DeleteSeason)

	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	// Create a season to delete
	body, _ := json.Marshal(map[string]string{"name": "Season To Delete"})
	req := newAdminRequest("POST", "/api/seasons", body, sessionID)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("create: expected 200, got %d", rr.Code)
	}

	// Find the season we just created
	req, _ = http.NewRequest("GET", "/api/seasons", nil)
	rr = httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var seasons []models.Season
	json.Unmarshal(rr.Body.Bytes(), &seasons)
	var targetID int
	for _, s := range seasons {
		if s.Name == "Season To Delete" {
			targetID = s.ID
			break
		}
	}
	if targetID == 0 {
		t.Skip("could not find created season")
	}

	t.Run("delete season", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/seasons?id="+strconv.Itoa(targetID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("verify season deleted", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		for _, s := range seasons {
			if s.ID == targetID {
				t.Error("season should be deleted but still found")
			}
		}
	})
}

// Helper functions kept for tests

func TestCreateSeason(t *testing.T) {
	r := gin.New()
	r.GET("/api/seasons", testHandler.GetSeasons)
	r.POST("/api/seasons", testHandler.CreateSeason)
	r.POST("/api/seasons/archive", testHandler.ArchiveSeason)
	r.DELETE("/api/seasons", testHandler.DeleteSeason)

	t.Run("create season", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": "Test Season 2025"})
		req, _ := http.NewRequest("POST", "/api/seasons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("create season empty name", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": ""})
		req, _ := http.NewRequest("POST", "/api/seasons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty name, got %d", rr.Code)
		}
	})

	t.Run("list seasons", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("list: expected 200, got %d", rr.Code)
		}
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		if len(seasons) < 1 {
			t.Error("expected at least 1 season")
		}
		if seasons[0].Name == "" {
			t.Error("expected season to have a name")
		}
	})

	t.Run("archive season", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var seasons []models.Season
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		if len(seasons) == 0 {
			t.Skip("no seasons to archive")
		}
		req, _ = http.NewRequest("POST", "/api/seasons/archive?id="+strconv.Itoa(seasons[0].ID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("archive: expected 200, got %d", rr.Code)
		}
	})
}

func TestCreateRoundSnapshot(t *testing.T) {
	testServer.DB.Exec("DELETE FROM round_snapshot_scores")
	testServer.DB.Exec("DELETE FROM round_snapshots")

	r := gin.New()
	r.POST("/api/rounds", testHandler.TakeRoundSnapshot)
	r.GET("/api/rounds", testHandler.GetRoundSnapshots)
	r.DELETE("/api/rounds", testHandler.DeleteRoundSnapshot)

	t.Run("take round snapshot", func(t *testing.T) {
		body, _ := json.Marshal(map[string]interface{}{
			"race_name": "Round 1",
			"season_id": 1,
		})
		req, _ := http.NewRequest("POST", "/api/rounds", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("snapshot: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var result map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &result)
		if result["id"] == nil {
			t.Error("expected snapshot id in response")
		}
	})

	t.Run("list round snapshots", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("list: expected 200, got %d", rr.Code)
		}
		var snapshots []models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &snapshots)
		if len(snapshots) < 1 {
			t.Error("expected at least 1 snapshot")
		}
	})

	t.Run("get snapshot with scores", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var snapshots []models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &snapshots)
		if len(snapshots) == 0 {
			t.Skip("no snapshots")
		}
		req, _ = http.NewRequest("GET", "/api/rounds?id="+strconv.Itoa(snapshots[0].ID), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("get by id: expected 200, got %d", rr.Code)
		}
		var details models.RoundSnapshot
		json.Unmarshal(rr.Body.Bytes(), &details)
		if len(details.Scores) == 0 {
			t.Error("expected snapshot to have scores")
		}
	})

	t.Run("filter by season", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/rounds?season_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("filter: expected 200, got %d", rr.Code)
		}
	})

	testServer.DB.Exec("DELETE FROM round_snapshot_scores")
	testServer.DB.Exec("DELETE FROM round_snapshots")
}
