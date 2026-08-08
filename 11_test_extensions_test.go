package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/middleware"
)

func extensionsRouter() *gin.Engine {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/extensions", testHandler.GetExtensions)
	admin.POST("/extensions", testHandler.CreateExtension)
	admin.PUT("/extensions", testHandler.UpdateExtension)
	admin.DELETE("/extensions", testHandler.DeleteExtension)
	admin.GET("/extensions/detail", testHandler.GetExtensionDetail)
	admin.GET("/modules", testHandler.GetModules)
	admin.POST("/modules", testHandler.CreateModule)
	admin.PUT("/modules", testHandler.UpdateModule)
	admin.DELETE("/modules", testHandler.DeleteModule)
	admin.PUT("/content/extension", testHandler.AssignContentExtension)
	return r
}

func TestExtensionsCRUD(t *testing.T) {
	r := extensionsRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("list seeded extensions", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/extensions", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", rr.Code)
		}
		var exts []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &exts)
		if len(exts) < 2 {
			t.Fatalf("expected at least Base Game + Heavy Rain, got %d", len(exts))
		}
		var base, heavy bool
		for _, e := range exts {
			if e["name"] == "Base Game" && e["is_base"] == float64(1) {
				base = true
			}
			if e["name"] == "Heavy Rain" {
				heavy = true
			}
		}
		if !base || !heavy {
			t.Errorf("expected seeded Base Game (is_base=1) and Heavy Rain, got base=%v heavy=%v", base, heavy)
		}
	})

	var newID int
	t.Run("create extension", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Test Pack", "description": "A test expansion", "is_base": 0, "sort_order": 9})
		req := newAdminRequest("POST", "/api/extensions", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		newID = int(resp["id"].(float64))
		if newID == 0 {
			t.Fatal("expected created id")
		}
	})

	t.Run("update extension", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"id": newID, "name": "Test Pack v2", "description": "updated", "is_base": 0, "sort_order": 10})
		req := newAdminRequest("PUT", "/api/extensions", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("detail shows counts and empty content", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/extensions/detail?id="+strconv.Itoa(newID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("detail: expected 200, got %d", rr.Code)
		}
		var d map[string]any
		json.Unmarshal(rr.Body.Bytes(), &d)
		if d["name"] != "Test Pack v2" {
			t.Errorf("expected name Test Pack v2, got %v", d["name"])
		}
	})

	t.Run("delete extension", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/extensions?id="+strconv.Itoa(newID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("extension gone after delete", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/extensions/detail?id="+strconv.Itoa(newID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", rr.Code)
		}
	})
}

func TestModulesCRUD(t *testing.T) {
	r := extensionsRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("list seeded modules with extension names", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/modules", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", rr.Code)
		}
		var mods []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &mods)
		if len(mods) < 5 {
			t.Fatalf("expected seeded modules, got %d", len(mods))
		}
		if mods[0]["extension"] == "" {
			t.Errorf("expected extension name on module, got %v", mods[0]["extension"])
		}
	})

	var newID int
	t.Run("create module", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Test Module", "description": "d", "extension_id": 2, "sort_order": 50})
		req := newAdminRequest("POST", "/api/modules", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		newID = int(resp["id"].(float64))
	})

	t.Run("update module", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"id": newID, "name": "Test Module v2", "description": "d2", "extension_id": 1, "sort_order": 51})
		req := newAdminRequest("PUT", "/api/modules", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var extName string
		testServer.DB.QueryRow("SELECT e.name FROM modules m JOIN extensions e ON m.extension_id = e.id WHERE m.id = ?", newID).Scan(&extName)
		if extName != "Base Game" {
			t.Errorf("expected module moved to Base Game, got %v", extName)
		}
	})

	t.Run("delete module", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/modules?id="+strconv.Itoa(newID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM modules WHERE id = ?", newID).Scan(&count)
		if count != 0 {
			t.Errorf("expected module deleted, found %d rows", count)
		}
	})
}

func TestAssignContentExtension(t *testing.T) {
	r := extensionsRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	var upgradeID int
	testServer.DB.QueryRow("SELECT id FROM upgrade_cards ORDER BY id LIMIT 1").Scan(&upgradeID)
	if upgradeID == 0 {
		t.Skip("no upgrade cards seeded")
	}

	t.Run("assign upgrade to Heavy Rain", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"content_type": "upgrade", "content_id": strconv.Itoa(upgradeID), "extension_id": 2})
		req := newAdminRequest("PUT", "/api/content/extension", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("assign: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var extID int
		testServer.DB.QueryRow("SELECT extension_id FROM upgrade_cards WHERE id = ?", upgradeID).Scan(&extID)
		if extID != 2 {
			t.Errorf("expected upgrade on extension 2, got %d", extID)
		}
	})

	t.Run("assign track to Heavy Rain", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"content_type": "track", "content_id": "monza", "extension_id": 2})
		req := newAdminRequest("PUT", "/api/content/extension", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("assign: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var extID int
		testServer.DB.QueryRow("SELECT extension_id FROM tracks WHERE id = 'monza'").Scan(&extID)
		if extID != 2 {
			t.Errorf("expected track on extension 2, got %d", extID)
		}
	})

	t.Run("invalid content type rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"content_type": "spaceship", "content_id": "1", "extension_id": 2})
		req := newAdminRequest("PUT", "/api/content/extension", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid content type, got %d", rr.Code)
		}
	})

	t.Run("unknown extension rejected", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"content_type": "track", "content_id": "monza", "extension_id": 9999})
		req := newAdminRequest("PUT", "/api/content/extension", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for unknown extension, got %d", rr.Code)
		}
	})

	// Restore seeded state for other tests
	testServer.DB.Exec("UPDATE upgrade_cards SET extension_id = 1 WHERE id = ?", upgradeID)
	testServer.DB.Exec("UPDATE tracks SET extension_id = 1 WHERE id = 'monza'")
}

func TestSeasonModules(t *testing.T) {
	r2 := gin.New()
	r2.POST("/api/seasons", testHandler.CreateSeason)
	r2.GET("/api/seasons", testHandler.GetSeasons)
	r2.DELETE("/api/seasons", testHandler.DeleteSeason)
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	var moduleID int
	testServer.DB.QueryRow("SELECT id FROM modules ORDER BY id LIMIT 1").Scan(&moduleID)
	if moduleID == 0 {
		t.Skip("no modules seeded")
	}

	var seasonID int
	t.Run("create season with modules", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Ext Season", "modules": []int{moduleID}})
		req := newAdminRequest("POST", "/api/seasons", body, sessionID)
		rr := httptest.NewRecorder()
		r2.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		testServer.DB.QueryRow("SELECT id FROM seasons WHERE name = 'Ext Season'").Scan(&seasonID)
		if seasonID == 0 {
			t.Fatal("expected created season id")
		}
	})

	t.Run("season list returns module_ids", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/seasons", nil)
		rr := httptest.NewRecorder()
		r2.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", rr.Code)
		}
		var seasons []struct {
			ID        int    `json:"id"`
			Name      string `json:"name"`
			ModuleIDs []int  `json:"module_ids"`
		}
		json.Unmarshal(rr.Body.Bytes(), &seasons)
		found := false
		for _, s := range seasons {
			if s.Name == "Ext Season" {
				found = true
				if len(s.ModuleIDs) != 1 || s.ModuleIDs[0] != moduleID {
					t.Errorf("expected module_ids [%d], got %v", moduleID, s.ModuleIDs)
				}
			}
		}
		if !found {
			t.Error("expected created season in list")
		}
	})

	t.Run("create season without modules stays backward compatible", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"name": "Ext Season Plain"})
		req, _ := http.NewRequest("POST", "/api/seasons", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r2.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("create plain: expected 200, got %d", rr.Code)
		}
	})

	// Cleanup
	testServer.DB.Exec("DELETE FROM season_modules WHERE season_id = ?", seasonID)
	testServer.DB.Exec("DELETE FROM seasons WHERE name IN ('Ext Season', 'Ext Season Plain')")
}

func TestExtensionSeedingIdempotent(t *testing.T) {
	// Re-running the seeders must not duplicate rows (they early-return when data exists)
	db.SeedExtensions()
	db.SeedModules()

	var extCount, modCount int
	testServer.DB.QueryRow("SELECT COUNT(*) FROM extensions").Scan(&extCount)
	testServer.DB.QueryRow("SELECT COUNT(*) FROM modules").Scan(&modCount)
	if extCount != 2 {
		t.Errorf("expected 2 extensions (Base Game, Heavy Rain), got %d", extCount)
	}
	if modCount != 6 {
		t.Errorf("expected 6 seeded modules, got %d", modCount)
	}

	// Base game id must be 1 and flagged is_base
	var baseName string
	var isBase int
	testServer.DB.QueryRow("SELECT name, is_base FROM extensions WHERE id = 1").Scan(&baseName, &isBase)
	if baseName != "Base Game" || isBase != 1 {
		t.Errorf("expected id 1 to be Base Game with is_base=1, got %q is_base=%d", baseName, isBase)
	}
}
