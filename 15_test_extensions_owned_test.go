package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/middleware"
)

// ownedRouter wires the routes exercised by the owned-extensions tests.
func ownedRouter() *gin.Engine {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.GET("/extensions", testHandler.GetExtensions)
	admin.GET("/extensions/detail", testHandler.GetExtensionDetail)
	admin.GET("/extensions/owned", testHandler.GetOwnedExtensions)
	admin.PUT("/extensions/owned", testHandler.SetOwnedExtensions)
	r.GET("/api/tracks", testHandler.GetTracks)
	r.GET("/api/available-upgrades", testHandler.GetAvailableUpgradesForRacer)
	r.GET("/api/legend-abilities", testHandler.GetLegendAbilities)
	return r
}

// resetOwned restores the owned set to just the Base Game (id 1).
func resetOwned() {
	testServer.DB.Exec("DELETE FROM owned_extensions")
	testServer.DB.Exec("INSERT INTO owned_extensions (extension_id) VALUES (1)")
}

func ownedIDs(t *testing.T, r *gin.Engine, sessionID string) []int {
	t.Helper()
	req := newAdminRequest("GET", "/api/extensions/owned", nil, sessionID)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("owned list: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var resp struct {
		OwnedIDs []int `json:"owned_ids"`
	}
	json.Unmarshal(rr.Body.Bytes(), &resp)
	return resp.OwnedIDs
}

func TestOwnedExtensionsAPI(t *testing.T) {
	r := ownedRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)
	resetOwned()

	t.Run("default owned list contains base game", func(t *testing.T) {
		ids := ownedIDs(t, r, sessionID)
		found := false
		for _, id := range ids {
			if id == 1 {
				found = true
			}
		}
		if !found {
			t.Errorf("expected Base Game (1) in owned_ids, got %v", ids)
		}
	})

	t.Run("base game flagged owned in extensions list", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/extensions", nil)
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("extensions list: expected 200, got %d", rr.Code)
		}
		var exts []map[string]any
		json.Unmarshal(rr.Body.Bytes(), &exts)
		for _, e := range exts {
			switch e["name"] {
			case "Base Game":
				if e["owned"] != true {
					t.Errorf("Base Game should be owned, got %v", e["owned"])
				}
			case "Heavy Rain":
				if e["owned"] == true {
					t.Errorf("Heavy Rain should not be owned by default, got %v", e["owned"])
				}
			}
		}
	})

	t.Run("full-replace adds heavy rain and ignores unknown ids", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"owned_ids": []int{2, 9999}})
		req := newAdminRequest("PUT", "/api/extensions/owned", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("set owned: expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		ids := ownedIDs(t, r, sessionID)
		set := map[int]bool{}
		for _, id := range ids {
			set[id] = true
		}
		if !set[1] {
			t.Errorf("Base Game missing after full-replace: %v", ids)
		}
		if !set[2] {
			t.Errorf("Heavy Rain missing after full-replace: %v", ids)
		}
		if set[9999] {
			t.Errorf("unknown id must be ignored, got %v", ids)
		}
	})

	t.Run("clearing owned leaves base game", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"owned_ids": []int{}})
		req := newAdminRequest("PUT", "/api/extensions/owned", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("set owned: expected 200, got %d", rr.Code)
		}
		ids := ownedIDs(t, r, sessionID)
		if len(ids) != 1 || ids[0] != 1 {
			t.Errorf("expected only Base Game after clearing, got %v", ids)
		}
	})

	resetOwned()
}

func TestExtensionDetailFullContentShapes(t *testing.T) {
	r := ownedRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	req := newAdminRequest("GET", "/api/extensions/detail?id=1", nil, sessionID)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("detail: expected 200, got %d", rr.Code)
	}
	var d struct {
		Tracks []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Country  string `json:"country"`
			Length   int    `json:"length_km"`
			GeoJSON  string `json:"geojson"`
			ModuleID int    `json:"module_id"`
			IsBoard  bool   `json:"is_board_game"`
		} `json:"tracks"`
		Upgrades []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			CardType    string `json:"card_type"`
			Cost        int    `json:"cost"`
			Effects     string `json:"effects"`
			ExtensionID int    `json:"extension_id"`
		} `json:"upgrades"`
		Legends []struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			AbilityType string `json:"ability_type"`
			RacerName   string `json:"racer_name"`
			ExtensionID int    `json:"extension_id"`
		} `json:"legends"`
	}
	json.Unmarshal(rr.Body.Bytes(), &d)

	if len(d.Tracks) == 0 {
		t.Fatal("expected tracks for Base Game extension")
	}
	t0 := d.Tracks[0]
	if t0.Name == "" || t0.Country == "" {
		t.Errorf("expected track name/country, got %+v", t0)
	}
	if t0.Length == 0 {
		t.Errorf("expected length_km populated, got %d", t0.Length)
	}
	if t0.GeoJSON == "" {
		t.Errorf("expected geojson populated")
	}
	anyBoard := false
	for _, tr := range d.Tracks {
		if tr.IsBoard {
			anyBoard = true
		}
	}
	if !anyBoard {
		t.Errorf("expected seeded base tracks flagged as board game")
	}

	if len(d.Upgrades) == 0 {
		t.Fatal("expected upgrade cards for Base Game extension")
	}
	u0 := d.Upgrades[0]
	if u0.Description == "" || u0.CardType == "" || u0.Effects == "" || u0.Cost == 0 {
		t.Errorf("expected full upgrade shape (description/card_type/effects/cost), got %+v", u0)
	}
	if u0.ExtensionID != 1 {
		t.Errorf("expected extension_id 1 on base upgrade, got %d", u0.ExtensionID)
	}

	if len(d.Legends) == 0 {
		t.Fatal("expected legend abilities for Base Game extension")
	}
	l0 := d.Legends[0]
	if l0.Description == "" || l0.AbilityType == "" || l0.RacerName == "" {
		t.Errorf("expected full legend shape (description/ability_type/racer_name), got %+v", l0)
	}
	if l0.ExtensionID != 1 {
		t.Errorf("expected extension_id 1 on base legend, got %d", l0.ExtensionID)
	}
}

func TestOwnedContentFiltering(t *testing.T) {
	r := ownedRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)
	resetOwned()

	// Fixtures on the Heavy Rain extension (id 2), which is not owned by default.
	testServer.DB.Exec("INSERT OR IGNORE INTO tracks (id, name, country, geojson, length_km, lap_record, extension_id) VALUES ('testext_track', 'Test Track', 'Testland', '', 9, '1:00.000', 2)")
	testServer.DB.Exec("INSERT OR IGNORE INTO upgrade_cards (name, description, card_type, cost, effects, extension_id) VALUES ('Test Upgrade', 'desc', 'upgrade', 2, '{}', 2)")
	testServer.DB.Exec("INSERT OR IGNORE INTO legend_abilities (name, description, ability_type, racer_name, extension_id) VALUES ('Test Legend', 'desc', 'start', 'T. DRIVER', 2)")
	defer func() {
		testServer.DB.Exec("DELETE FROM tracks WHERE id = 'testext_track'")
		testServer.DB.Exec("DELETE FROM upgrade_cards WHERE name = 'Test Upgrade'")
		testServer.DB.Exec("DELETE FROM legend_abilities WHERE name = 'Test Legend'")
		resetOwned()
	}()

	var upgradeID, legendID int
	testServer.DB.QueryRow("SELECT id FROM upgrade_cards WHERE name = 'Test Upgrade'").Scan(&upgradeID)
	testServer.DB.QueryRow("SELECT id FROM legend_abilities WHERE name = 'Test Legend'").Scan(&legendID)
	if upgradeID == 0 || legendID == 0 {
		t.Fatal("failed to create fixtures")
	}

	t.Run("tracks owned=1 excludes unowned extension content", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/tracks?owned=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var tracks []struct {
			ID string `json:"id"`
		}
		json.Unmarshal(rr.Body.Bytes(), &tracks)
		for _, tr := range tracks {
			if tr.ID == "testext_track" {
				t.Error("unowned track leaked into owned=1 list")
			}
		}
		// The management surface still shows everything.
		req2, _ := http.NewRequest("GET", "/api/tracks", nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		var all []struct {
			ID string `json:"id"`
		}
		json.Unmarshal(rr2.Body.Bytes(), &all)
		found := false
		for _, tr := range all {
			if tr.ID == "testext_track" {
				found = true
			}
		}
		if !found {
			t.Error("full track list should still include unowned content (management surface)")
		}
	})

	t.Run("available-upgrades excludes unowned cards", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/available-upgrades?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var ups []struct {
			ID int `json:"id"`
		}
		json.Unmarshal(rr.Body.Bytes(), &ups)
		for _, u := range ups {
			if u.ID == upgradeID {
				t.Error("unowned upgrade leaked into deck builder")
			}
		}
	})

	t.Run("legend catalog excludes unowned legends", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/legend-abilities", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var ls []struct {
			ID int `json:"id"`
		}
		json.Unmarshal(rr.Body.Bytes(), &ls)
		for _, l := range ls {
			if l.ID == legendID {
				t.Error("unowned legend leaked into catalog")
			}
		}
	})

	t.Run("owning the extension restores content in all selection lists", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{"owned_ids": []int{1, 2}})
		req := newAdminRequest("PUT", "/api/extensions/owned", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("set owned: expected 200, got %d", rr.Code)
		}

		reqT, _ := http.NewRequest("GET", "/api/tracks?owned=1", nil)
		rrT := httptest.NewRecorder()
		r.ServeHTTP(rrT, reqT)
		var tracks []struct {
			ID string `json:"id"`
		}
		json.Unmarshal(rrT.Body.Bytes(), &tracks)
		found := false
		for _, tr := range tracks {
			if tr.ID == "testext_track" {
				found = true
			}
		}
		if !found {
			t.Error("owned track missing from owned=1 list")
		}

		reqU, _ := http.NewRequest("GET", "/api/available-upgrades?racer_id=1", nil)
		rrU := httptest.NewRecorder()
		r.ServeHTTP(rrU, reqU)
		var ups []struct {
			ID int `json:"id"`
		}
		json.Unmarshal(rrU.Body.Bytes(), &ups)
		found = false
		for _, u := range ups {
			if u.ID == upgradeID {
				found = true
			}
		}
		if !found {
			t.Error("owned upgrade missing from deck builder")
		}

		reqL, _ := http.NewRequest("GET", "/api/legend-abilities", nil)
		rrL := httptest.NewRecorder()
		r.ServeHTTP(rrL, reqL)
		var ls []struct {
			ID int `json:"id"`
		}
		json.Unmarshal(rrL.Body.Bytes(), &ls)
		found = false
		for _, l := range ls {
			if l.ID == legendID {
				found = true
			}
		}
		if !found {
			t.Error("owned legend missing from catalog")
		}
	})
}
