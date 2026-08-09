package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/middleware"
)

// tracksModulesRouter wires the endpoints exercised by the module/board game
// track tests, mirroring the admin group setup in main.go.
func tracksModulesRouter() *gin.Engine {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/tracks", testHandler.SaveTrack)
	admin.DELETE("/tracks", testHandler.DeleteTrack)
	admin.PUT("/board-game/tracks", testHandler.SetBoardGameTracks)
	admin.POST("/modules", testHandler.CreateModule)
	admin.DELETE("/modules", testHandler.DeleteModule)
	admin.GET("/extensions/detail", testHandler.GetExtensionDetail)
	r.GET("/api/tracks", testHandler.GetTracks)
	r.GET("/api/board-game/tracks", testHandler.GetBoardGameTracks)
	return r
}

func TestTrackModuleAttribution(t *testing.T) {
	r := tracksModulesRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	// A module owned by Heavy Rain (extension 2) to verify extension derivation.
	var moduleID int
	testServer.DB.QueryRow("SELECT id FROM modules WHERE extension_id = 2 ORDER BY id LIMIT 1").Scan(&moduleID)
	if moduleID == 0 {
		t.Skip("no module owned by extension 2 seeded")
	}

	trackID := "test-module-track"
	t.Cleanup(func() {
		testServer.Ent.Track.DeleteOneID(trackID).Exec(context.Background())
	})

	t.Run("save track with module_id derives extension", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"id": trackID, "name": "Test Module Track", "country": "Testland",
			"length_km": 6, "lap_record": "", "module_id": moduleID,
		})
		req := newAdminRequest("POST", "/api/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var extID, modID int
		testServer.DB.QueryRow("SELECT extension_id, module_id FROM tracks WHERE id = ?", trackID).Scan(&extID, &modID)
		if modID != moduleID {
			t.Errorf("expected module_id %d, got %d", moduleID, modID)
		}
		if extID != 2 {
			t.Errorf("expected extension derived to 2 (module owner), got %d", extID)
		}
	})

	t.Run("track list returns module_id and derived extension", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tracks", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", rr.Code)
		}
		var tracks []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &tracks)
		for _, tr := range tracks {
			if tr["id"] == trackID {
				if tr["module_id"] != float64(moduleID) {
					t.Errorf("expected module_id %d, got %v", moduleID, tr["module_id"])
				}
				if tr["extension_id"] != float64(2) {
					t.Errorf("expected extension_id 2, got %v", tr["extension_id"])
				}
				return
			}
		}
		t.Fatalf("expected created track %q in list", trackID)
	})

	t.Run("extension detail lists module tracks", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/extensions/detail?id=2", nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("detail: expected 200, got %d", rr.Code)
		}
		var d struct {
			Modules []struct {
				ID     int `json:"id"`
				Tracks []struct {
					ID string `json:"id"`
				} `json:"tracks"`
			} `json:"modules"`
		}
		json.Unmarshal(rr.Body.Bytes(), &d)
		for _, m := range d.Modules {
			if m.ID == moduleID {
				for _, tr := range m.Tracks {
					if tr.ID == trackID {
						return
					}
				}
				t.Fatalf("expected module %d to include track %q", moduleID, trackID)
			}
		}
		t.Fatalf("expected module %d in extension 2 detail", moduleID)
	})
}

func TestBoardGameTracks(t *testing.T) {
	r := tracksModulesRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	t.Run("seeded on startup with base tracks", func(t *testing.T) {
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks").Scan(&count)
		if count == 0 {
			t.Fatal("expected board game tracks seeded at startup")
		}
	})

	t.Run("GET returns track_ids", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/board-game/tracks", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d", rr.Code)
		}
		var resp struct {
			TrackIDs []string `json:"track_ids"`
		}
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if len(resp.TrackIDs) == 0 {
			t.Fatal("expected non-empty track_ids")
		}
		found := false
		for _, id := range resp.TrackIDs {
			if id == "monza" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected monza among board game tracks, got %v", resp.TrackIDs)
		}
	})

	t.Run("track list flags is_board_game", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tracks", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d", rr.Code)
		}
		var tracks []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &tracks)
		monzaFlagged := false
		for _, tr := range tracks {
			if tr["id"] == "monza" {
				if isBG, _ := tr["is_board_game"].(bool); isBG {
					monzaFlagged = true
				}
			}
		}
		if !monzaFlagged {
			t.Error("expected monza flagged is_board_game=true")
		}
	})

	t.Run("PUT replaces the list", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"track_ids": []string{"spa"}})
		req := newAdminRequest("PUT", "/api/board-game/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("put: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		req2, _ := http.NewRequest("GET", "/api/board-game/tracks", nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		var resp struct {
			TrackIDs []string `json:"track_ids"`
		}
		json.Unmarshal(rr2.Body.Bytes(), &resp)
		if len(resp.TrackIDs) != 1 || resp.TrackIDs[0] != "spa" {
			t.Errorf("expected exactly [spa], got %v", resp.TrackIDs)
		}
		// monza must no longer be a board game track
		var monzaBG int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks WHERE track_id = 'monza'").Scan(&monzaBG)
		if monzaBG != 0 {
			t.Errorf("expected monza removed from board game list, found %d rows", monzaBG)
		}
	})

	t.Run("PUT ignores unknown track ids", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"track_ids": []string{"does-not-exist"}})
		req := newAdminRequest("PUT", "/api/board-game/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("put: expected 200, got %d", rr.Code)
		}
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks").Scan(&count)
		if count != 0 {
			t.Errorf("expected unknown id dropped (count 0), got %d", count)
		}
	})

	t.Run("seed fills an empty list without duplicating rows", func(t *testing.T) {
		// Self-contained: create a fresh base game track, empty the list, then
		// verify the seeder picks it up exactly once (idempotent, like the
		// other seeders). Kept independent of other tests' DB state.
		seedTrack := "seed-test-track"
		body, _ := json.Marshal(map[string]any{
			"id": seedTrack, "name": "Seed Test", "country": "X",
			"length_km": 3, "extension_id": 1,
		})
		req := newAdminRequest("POST", "/api/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create seed track: expected 200, got %d", rr.Code)
		}
		t.Cleanup(func() { testServer.Ent.Track.DeleteOneID(seedTrack).Exec(context.Background()) })

		var before int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks").Scan(&before)
		if before != 0 {
			t.Fatalf("expected empty board game list before re-seed, got %d", before)
		}
		// 12_test_trmnl_test.go re-points the db package's global server to its
		// own instance; ensure the seeder runs against the shared test DB.
		db.SetServer(testServer)
		db.SeedBoardGameTracks()
		var count int
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks WHERE track_id = ?", seedTrack).Scan(&count)
		if count != 1 {
			t.Errorf("expected re-seed to add the base game track, got %d", count)
		}
		db.SeedBoardGameTracks()
		testServer.DB.QueryRow("SELECT COUNT(*) FROM board_game_tracks WHERE track_id = ?", seedTrack).Scan(&count)
		if count != 1 {
			t.Errorf("expected second re-seed to not duplicate, got %d", count)
		}
	})

	// Restore the seeded state for other tests: all base game tracks again.
	restoreBoardGameTracks()
}

// restoreBoardGameTracks resets board_game_tracks to the seeded base set.
func restoreBoardGameTracks() {
	testServer.DB.Exec("DELETE FROM board_game_tracks")
	rows, err := testServer.DB.Query("SELECT id FROM tracks WHERE extension_id = 1 ORDER BY name")
	if err != nil {
		return
	}
	var ids []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	rows.Close()
	for _, id := range ids {
		testServer.DB.Exec("INSERT OR IGNORE INTO board_game_tracks (track_id) VALUES (?)", id)
	}
}

func TestDeleteModuleResetsTracks(t *testing.T) {
	r := tracksModulesRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	var moduleID int
	t.Run("create module under Heavy Rain", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"name": "Track Reset Module", "description": "d", "extension_id": 2, "sort_order": 60})
		req := newAdminRequest("POST", "/api/modules", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("create module: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp map[string]any
		json.Unmarshal(rr.Body.Bytes(), &resp)
		moduleID = int(resp["id"].(float64))
		if moduleID == 0 {
			t.Fatal("expected created module id")
		}
	})

	trackID := "test-reset-track"
	t.Cleanup(func() {
		testServer.Ent.Track.DeleteOneID(trackID).Exec(context.Background())
		if moduleID != 0 {
			testServer.DB.Exec("DELETE FROM modules WHERE id = ?", moduleID)
		}
	})

	t.Run("assign track to the module", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"id": trackID, "name": "Reset Me", "country": "X",
			"length_km": 4, "module_id": moduleID,
		})
		req := newAdminRequest("POST", "/api/tracks", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("save track: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var modID, extID int
		testServer.DB.QueryRow("SELECT module_id, extension_id FROM tracks WHERE id = ?", trackID).Scan(&modID, &extID)
		if modID != moduleID || extID != 2 {
			t.Fatalf("expected module_id=%d extension_id=2, got module_id=%d extension_id=%d", moduleID, modID, extID)
		}
	})

	t.Run("deleting the module resets its tracks to base game", func(t *testing.T) {
		req := newAdminRequest("DELETE", "/api/modules?id="+strconv.Itoa(moduleID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("delete module: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var modID, extID int
		testServer.DB.QueryRow("SELECT module_id, extension_id FROM tracks WHERE id = ?", trackID).Scan(&modID, &extID)
		if modID != 0 {
			t.Errorf("expected module_id reset to 0, got %d", modID)
		}
		if extID != 1 {
			t.Errorf("expected extension_id reset to 1 (Base Game), got %d", extID)
		}
		moduleID = 0 // deleted; skip cleanup re-delete
	})
}
