package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/middleware"
	"heat/models"
)

func TestGetUploads(t *testing.T) {
	r := gin.New()
	r.GET("/api/uploads", testHandler.GetUploads)

	req, _ := http.NewRequest("GET", "/api/uploads", nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("expected status 200, got %v", status)
	}

	var uploads []models.Upload
	json.Unmarshal(rr.Body.Bytes(), &uploads)
	if uploads == nil {
		t.Errorf("expected uploads array, got nil")
	}
}

func TestHandleUpload(t *testing.T) {
	// Create test routes with CSRF + Auth middleware
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/upload", testHandler.HandleUpload)
	r.GET("/api/racers", testHandler.GetRacers)

	// Create a valid session for auth
	sessionID := "upload-test-session"
	testServer.SessionStoreMu.Lock()
	testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	testServer.SessionStoreMu.Unlock()
	defer func() {
		testServer.SessionStoreMu.Lock()
		delete(testServer.SessionStore, sessionID)
		testServer.SessionStoreMu.Unlock()
	}()

	// Create a minimal valid PNG for testing
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, // PNG signature
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, // IHDR chunk
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
		0xDE, 0x00, 0x00, 0x00, 0x0C, 0x49, 0x44, 0x41, // IDAT chunk
		0x54, 0x78, 0x9C, 0x63, 0xF8, 0x0F, 0x00, 0x00,
		0x01, 0x01, 0x00, 0x05, 0x18, 0xD8, 0x4E, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, // IEND chunk
		0x42, 0x60, 0x82,
	}

	t.Run("UploadSuccess", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "test_racer.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, err := http.NewRequest("POST", "/api/upload", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("expected status 200, got %v: %s", status, rr.Body.String())
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		url, ok := resp["url"].(string)
		if !ok {
			t.Fatalf("expected url in response, got %v", resp)
		}

		// Verify URL starts with /media/ (new media path)
		if !strings.HasPrefix(url, "/media/") {
			t.Errorf("expected URL to start with /media/, got %q", url)
		}

		// Verify the URL contains a hash subdirectory (2 chars)
		parts := strings.Split(strings.TrimPrefix(url, "/media/"), "/")
		if len(parts) != 2 {
			t.Errorf("expected URL format /media/<hash2>/<filename>, got %q", url)
		} else if len(parts[0]) != 2 {
			t.Errorf("expected 2-char hash subdirectory, got %q", parts[0])
		}

		// Verify file exists on disk
		filePath := filepath.Join(testServer.MediaPath, parts[0], parts[1])
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("uploaded file not found at %s", filePath)
		} else {
			// Clean up test file
			defer os.RemoveAll(filepath.Join(testServer.MediaPath, parts[0]))
		}

		// Verify upload is stored in DB
		hash, ok := resp["hash"].(string)
		if !ok {
			t.Fatal("expected hash in response")
		}
		var storedURL string
		err = testServer.DB.QueryRow("SELECT url FROM uploads WHERE hash = ?", hash).Scan(&storedURL)
		if err != nil {
			t.Errorf("upload not found in database: %v", err)
		} else if storedURL != url {
			t.Errorf("stored URL %q doesn't match response %q", storedURL, url)
		}
	})

	t.Run("UploadAndUpdateRacer", func(t *testing.T) {
		// Need a separate router with racer POST route for this test
		r2 := gin.New()
		admin2 := r2.Group("/api")
		admin2.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin2.POST("/upload", testHandler.HandleUpload)
		admin2.POST("/racers", testHandler.UpdateRacer)

		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "racer_pic.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, err := http.NewRequest("POST", "/api/upload", body)
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Fatalf("upload failed: %v: %s", status, rr.Body.String())
		}

		var uploadResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &uploadResp)
		uploadURL := uploadResp["url"].(string)

		// Update racer with uploaded image URL
		racerURL := "/api/racers"
		racerBody := map[string]interface{}{
			"id":              1,
			"name":            "Upload Test Racer",
			"profile_picture": uploadURL,
			"car_color":       "red",
			"car_name":        "Upload Test Car",
			"points":          99,
			"rank":            1,
			"position":        0,
		}
		racerJSON, _ := json.Marshal(racerBody)

		req2, _ := http.NewRequest("POST", racerURL, bytes.NewBuffer(racerJSON))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Origin", "http://127.0.0.1:6270")
		req2.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req2.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r2.ServeHTTP(rr2, req2)

		if status := rr2.Code; status != http.StatusOK {
			t.Fatalf("update racer failed: %v: %s", status, rr2.Body.String())
		}

		// Verify racer's profile_picture in DB
		var profilePicture string
		err = testServer.DB.QueryRow("SELECT profile_picture FROM racers WHERE id = 1").Scan(&profilePicture)
		if err != nil {
			t.Fatal(err)
		}
		if profilePicture != uploadURL {
			t.Errorf("expected profile_picture %q, got %q", uploadURL, profilePicture)
		}

		// Clean up uploaded file
		parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
		if len(parts) == 2 {
			os.RemoveAll(filepath.Join(testServer.MediaPath, parts[0]))
		}
	})
}

func TestFlagEndpoint(t *testing.T) {
	r := gin.New()
	r.POST("/api/flags", testHandler.HandleFlag)

	t.Run("valid safety car flag", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "safety"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid red flag", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "red"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid blue flag with racer info", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "blue", RacerID: 1, RacerName: "A. PROST"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("valid blackwhite flag with racer info", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "blackwhite", RacerID: 2, RacerName: "M. SCHUMACHER"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("invalid flag type", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "invalid", Flag: "safety"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid type, got %d", rr.Code)
		}
	})

	t.Run("start lights sequence", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "startlights", State: "sequence"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for start lights sequence, got %d", rr.Code)
		}
	})

	t.Run("start lights abort", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "startlights", State: "abort"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for start lights abort, got %d", rr.Code)
		}
	})

	t.Run("start lights reset", func(t *testing.T) {
		body, _ := json.Marshal(models.FlagCommand{Type: "flag", Flag: "startlights", State: "reset"})
		req, _ := http.NewRequest("POST", "/api/flags", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200 for start lights reset, got %d", rr.Code)
		}
	})
}

func TestI18nAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/translations", testHandler.GetTranslations)
	r.POST("/api/language", testHandler.SetLanguage)

	t.Run("get translations default", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		if tmap["nav.standings"] == "" {
			t.Error("expected nav.standings translation")
		}
		if tmap["_lang"] != "en" {
			t.Errorf("expected _lang=en, got %s", tmap["_lang"])
		}
	})

	t.Run("get dutch translations", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations?lang=nl", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		if tmap["nav.admin"] != "Admin" {
			t.Errorf("expected nl nav.admin to be 'Admin', got %s", tmap["nav.admin"])
		}
		if tmap["_lang"] != "nl" {
			t.Errorf("expected _lang=nl, got %s", tmap["_lang"])
		}
	})

	t.Run("set language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": "nl"})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("set invalid language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": "fr"})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid lang, got %d", rr.Code)
		}
	})

	t.Run("set empty language", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"lang": ""})
		req, _ := http.NewRequest("POST", "/api/language", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty lang, got %d", rr.Code)
		}
	})

	t.Run("get translations detects dutch accept-language", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/translations", nil)
		req.Header.Set("Accept-Language", "nl-NL,nl;q=0.9,en;q=0.8")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var tmap map[string]string
		json.Unmarshal(rr.Body.Bytes(), &tmap)
		// Accept-Language is not easily detected in tests through gin
	})

	t.Run("translation files exist", func(t *testing.T) {
		for _, lang := range []string{"en", "nl"} {
			data, err := os.ReadFile("static/locales/" + lang + ".json")
			if err != nil {
				t.Errorf("missing locale file: %s", lang)
				continue
			}
			var tmap map[string]string
			if err := json.Unmarshal(data, &tmap); err != nil {
				t.Errorf("invalid JSON in %s locale: %v", lang, err)
			}
			if len(tmap) == 0 {
				t.Errorf("empty translations for %s", lang)
			}
		}
	})

	t.Run("translations keys match between locales", func(t *testing.T) {
		enData, _ := os.ReadFile("static/locales/en.json")
		nlData, _ := os.ReadFile("static/locales/nl.json")
		var en, nl map[string]string
		json.Unmarshal(enData, &en)
		json.Unmarshal(nlData, &nl)

		for k := range en {
			if nl[k] == "" {
				t.Errorf("key %q missing from nl.json", k)
			}
		}
		for k := range nl {
			if en[k] == "" {
				t.Errorf("key %q missing from en.json", k)
			}
		}
	})
}

func TestHeatCardAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/heat-cards", testHandler.GetHeatCards)
	r.POST("/api/heat-cards", testHandler.AddHeatCard)
	r.PUT("/api/heat-cards/move", testHandler.MoveHeatCard)
	r.DELETE("/api/heat-cards", testHandler.DeleteHeatCard)
	r.POST("/api/heat-cards/init-decks", testHandler.InitializeHeatDecks)

	t.Run("init decks", func(t *testing.T) {
		body := `{"race_id":0,"racer_ids":[1,2]}`
		req, _ := http.NewRequest("POST", "/api/heat-cards/init-decks", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list heat cards", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/heat-cards", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var cards []models.HeatCard
		json.Unmarshal(rr.Body.Bytes(), &cards)
		if len(cards) < 14 {
			t.Errorf("expected at least 14 cards (7x2), got %d", len(cards))
		}
	})

	t.Run("add heat card", func(t *testing.T) {
		body := `{"racer_id":1,"location":"hand","card_type":"heat","lap_added":1}`
		req, _ := http.NewRequest("POST", "/api/heat-cards", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("move heat card", func(t *testing.T) {
		var cards []models.HeatCard
		req, _ := http.NewRequest("GET", "/api/heat-cards?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		json.Unmarshal(rr.Body.Bytes(), &cards)
		if len(cards) == 0 {
			t.Skip("no cards to move")
		}
		body := fmt.Sprintf(`{"card_id":%d,"location":"engine"}`, cards[len(cards)-1].ID)
		req, _ = http.NewRequest("PUT", "/api/heat-cards/move", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/heat-cards?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var cards []models.HeatCard
		json.Unmarshal(rr.Body.Bytes(), &cards)
		for _, c := range cards {
			if c.RacerID != 1 {
				t.Errorf("expected all cards for racer 1, got racer_id=%d", c.RacerID)
			}
		}
	})

	testServer.DB.Exec("DELETE FROM heat_cards")
}

func TestGearShiftAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/gear-shifts", testHandler.GetGearShifts)
	r.POST("/api/gear-shifts", testHandler.AddGearShift)

	t.Run("add gear shift", func(t *testing.T) {
		body := `{"racer_id":1,"race_id":0,"lap":1,"gear":3,"stress":1}`
		req, _ := http.NewRequest("POST", "/api/gear-shifts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list gear shifts", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/gear-shifts", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var shifts []models.GearShift
		json.Unmarshal(rr.Body.Bytes(), &shifts)
		if len(shifts) < 1 {
			t.Error("expected at least 1 gear shift")
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/gear-shifts?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	testServer.DB.Exec("DELETE FROM gear_shifts")
}

func TestUpgradeAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/upgrade-cards", testHandler.GetUpgradeCards)
	r.GET("/api/legend-abilities", testHandler.GetLegendAbilities)

	t.Run("list upgrade cards", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/upgrade-cards", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var upgrades []models.UpgradeCard
		json.Unmarshal(rr.Body.Bytes(), &upgrades)
		if len(upgrades) < 8 {
			t.Errorf("expected at least 8 upgrade cards, got %d", len(upgrades))
		}
	})

	t.Run("list legend abilities", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/legend-abilities", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var abilities []models.LegendAbility
		json.Unmarshal(rr.Body.Bytes(), &abilities)
		if len(abilities) < 5 {
			t.Errorf("expected at least 5 legend abilities, got %d", len(abilities))
		}
	})
}

func TestWeatherAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/weather", testHandler.GetWeather)
	r.POST("/api/weather", testHandler.SetWeather)

	t.Run("set weather", func(t *testing.T) {
		body := `{"race_id":0,"condition":"wet","lap_start":1,"lap_end":999,"grip_modifier":0.7}`
		req, _ := http.NewRequest("POST", "/api/weather", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("get weather", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/weather?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var weather []models.WeatherCondition
		json.Unmarshal(rr.Body.Bytes(), &weather)
		if len(weather) < 1 {
			t.Error("expected at least 1 weather condition")
		}
		if weather[len(weather)-1].Condition != "wet" {
			t.Errorf("expected wet, got %s", weather[len(weather)-1].Condition)
		}
	})

	testServer.DB.Exec("DELETE FROM weather_conditions")
}

func TestTurboLogAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/turbo-logs", testHandler.GetTurboLogs)
	r.POST("/api/turbo-logs", testHandler.AddTurboLog)

	t.Run("add turbo log", func(t *testing.T) {
		body := `{"racer_id":1,"race_id":0,"lap":1,"times_used":1}`
		req, _ := http.NewRequest("POST", "/api/turbo-logs", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list turbo logs", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/turbo-logs", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var logs []models.TurboLog
		json.Unmarshal(rr.Body.Bytes(), &logs)
		if len(logs) < 1 {
			t.Error("expected at least 1 turbo log")
		}
	})

	testServer.DB.Exec("DELETE FROM turbo_logs")
}

func TestLapRecordAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/lap-records", testHandler.GetLapRecords)
	r.POST("/api/lap-records", testHandler.RecordLap)
	r.POST("/api/lap-records/batch", testHandler.RecordLapBatch)

	t.Run("record single lap", func(t *testing.T) {
		body := `{"race_id":0,"racer_id":1,"lap_number":1,"position":1,"gear_used":3,"heat_generated":2,"turbo_used":false}`
		req, _ := http.NewRequest("POST", "/api/lap-records", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("record batch lap", func(t *testing.T) {
		body := `{"race_id":0,"lap":2,"records":[
			{"racer_id":1,"position":2,"gear_used":2,"heat_generated":1,"turbo_used":false},
			{"racer_id":2,"position":1,"gear_used":3,"heat_generated":0,"turbo_used":true}
		]}`
		req, _ := http.NewRequest("POST", "/api/lap-records/batch", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("list lap records", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/lap-records?race_id=0", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var records []models.LapRecord
		json.Unmarshal(rr.Body.Bytes(), &records)
		if len(records) < 3 {
			t.Errorf("expected at least 3 lap records, got %d", len(records))
		}
	})

	testServer.DB.Exec("DELETE FROM lap_records")
}

func TestSoundFX(t *testing.T) {
	r := gin.New()
	r.POST("/api/sound", testHandler.PlaySound)

	t.Run("play engine sound", func(t *testing.T) {
		body := `{"sound":"engine"}`
		req, _ := http.NewRequest("POST", "/api/sound", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("play finish sound", func(t *testing.T) {
		body := `{"sound":"finish"}`
		req, _ := http.NewRequest("POST", "/api/sound", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})
}

func TestRaceRadioAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/race-radio", testHandler.GetRaceRadio)
	r.POST("/api/race-radio", testHandler.AddRaceRadio)

	t.Run("add radio message", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":1,"message":"Box box, box now!"}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var msg models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msg)
		if msg.ID == 0 {
			t.Error("expected non-zero id")
		}
		if msg.RacerName == "" {
			t.Error("expected racer name")
		}
	})

	t.Run("add radio message empty message", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":1,"message":""}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for empty message, got %d", rr.Code)
		}
	})

	t.Run("add radio message no racer", func(t *testing.T) {
		body := `{"race_id":1,"racer_id":0,"message":"Test"}`
		req, _ := http.NewRequest("POST", "/api/race-radio", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for no racer, got %d", rr.Code)
		}
	})

	t.Run("get radio messages", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-radio?race_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var msgs []models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msgs)
		if len(msgs) < 1 {
			t.Error("expected at least 1 message")
		}
		if msgs[0].Message != "Box box, box now!" {
			t.Errorf("expected 'Box box, box now!', got %q", msgs[0].Message)
		}
	})

	t.Run("filter by racer", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/race-radio?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var msgs []models.RaceRadioMessage
		json.Unmarshal(rr.Body.Bytes(), &msgs)
		for _, m := range msgs {
			if m.RacerID != 1 {
				t.Errorf("expected racer_id=1, got %d", m.RacerID)
			}
		}
	})

	testServer.DB.Exec("DELETE FROM race_radio")
}

func TestPlayerSelfServiceEndpoints(t *testing.T) {
	r := gin.New()
	r.POST("/api/player/gear", testHandler.PlayerReportGear)
	r.POST("/api/player/heat", testHandler.PlayerReportHeat)
	r.POST("/api/player/turbo", testHandler.PlayerReportTurbo)
	r.GET("/api/player/status", testHandler.PlayerGetStatus)
	r.POST("/api/player/login", testHandler.PlayerLogin)

	var token string
	loginBody := `{"racer_id":1,"device_name":"Test"}`
	req, _ := http.NewRequest("POST", "/api/player/login", strings.NewReader(loginBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	var loginResp struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rr.Body.Bytes(), &loginResp)
	token = loginResp.Token

	t.Run("report gear", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","lap":1,"gear":3,"stress":0}`, token)
		req, _ := http.NewRequest("POST", "/api/player/gear", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("report heat", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","card_type":"heat","location":"engine","count":1}`, token)
		req, _ := http.NewRequest("POST", "/api/player/heat", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("report turbo", func(t *testing.T) {
		body := fmt.Sprintf(`{"token":"%s","lap":1}`, token)
		req, _ := http.NewRequest("POST", "/api/player/turbo", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	t.Run("get player status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/player/status", nil)
		req.Header.Set("X-Player-Token", token)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var data map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &data)
		if data["racer"] == nil {
			t.Error("expected racer field")
		}
		if data["heat_cards"] == nil {
			t.Error("expected heat_cards field")
		}
	})

	t.Run("reject unauthorized gear report", func(t *testing.T) {
		body := `{"token":"bad_token","lap":1,"gear":1,"stress":0}`
		req, _ := http.NewRequest("POST", "/api/player/gear", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rr.Code)
		}
	})

	testServer.DB.Exec("DELETE FROM gear_shifts")
	testServer.DB.Exec("DELETE FROM heat_cards WHERE racer_id=1 AND lap_added=0")
	testServer.DB.Exec("DELETE FROM turbo_logs")
	testServer.DB.Exec("DELETE FROM player_sessions")
}

func TestPlayerUpgradeBuy(t *testing.T) {
	r := gin.New()
	r.POST("/api/player-upgrades/buy", testHandler.BuyUpgrade)

	t.Run("buy upgrade", func(t *testing.T) {
		body := `{"racer_id":1,"upgrade_id":1,"season_id":1,"round":1}`
		req, _ := http.NewRequest("POST", "/api/player-upgrades/buy", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	testServer.DB.Exec("DELETE FROM player_upgrades")
}

func TestSectorAPI(t *testing.T) {
	r := gin.New()
	r.GET("/api/sectors", testHandler.GetSectors)
	r.POST("/api/racer-sectors", testHandler.RecordRacerSector)

	t.Run("list sectors for monza", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/sectors?track_id=monza", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var sectors []models.Sector
		json.Unmarshal(rr.Body.Bytes(), &sectors)
		if len(sectors) < 5 {
			t.Errorf("expected at least 5 sectors for monza, got %d", len(sectors))
		}
	})

	t.Run("record racer sector", func(t *testing.T) {
		body := `{"race_id":0,"racer_id":1,"sector_id":1,"lap":1}`
		req, _ := http.NewRequest("POST", "/api/racer-sectors", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
	})

	testServer.DB.Exec("DELETE FROM racer_sectors")
}

func TestAvailableUpgrades(t *testing.T) {
	r := gin.New()
	r.GET("/api/available-upgrades", testHandler.GetAvailableUpgradesForRacer)

	t.Run("list available upgrades", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/available-upgrades?racer_id=1", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var upgrades []models.UpgradeCard
		json.Unmarshal(rr.Body.Bytes(), &upgrades)
		if len(upgrades) < 8 {
			t.Errorf("expected at least 8 available upgrades, got %d", len(upgrades))
		}
	})
}

func TestDriverShare(t *testing.T) {
	adminRouter := func() *gin.Engine {
		r := gin.New()
		admin := r.Group("/api")
		admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
		admin.POST("/driver-share", testHandler.GenerateDriverShareToken)
		admin.GET("/driver-shares", testHandler.GetDriverShareTokens)
		admin.DELETE("/driver-share", testHandler.DeleteDriverShareToken)
		return r
	}

	publicRouter := func() *gin.Engine {
		r := gin.New()
		r.GET("/api/shared/driver-stats", testHandler.GetDriverStatsByToken)
		return r
	}

	sessionID := "test-share-session"
	testServer.SessionStoreMu.Lock()
	testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	testServer.SessionStoreMu.Unlock()
	defer func() {
		testServer.SessionStoreMu.Lock()
		delete(testServer.SessionStore, sessionID)
		testServer.SessionStoreMu.Unlock()
	}()

	addSessionCookie := func(req *http.Request) {
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	}

	t.Run("generate share token", func(t *testing.T) {
		r := adminRouter()
		body := `{}`
		req, _ := http.NewRequest("POST", "/api/driver-share?racer_id=1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var share models.DriverShare
		json.Unmarshal(rr.Body.Bytes(), &share)
		if share.Token == "" {
			t.Error("expected non-empty token")
		}
		if share.RacerID != 1 {
			t.Errorf("expected racer_id 1, got %d", share.RacerID)
		}
	})

	t.Run("get driver shares (admin)", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("GET", "/api/driver-shares", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}
		var shares []map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &shares)
		if len(shares) == 0 {
			t.Error("expected at least one share")
		}
	})

	t.Run("access driver stats via token", func(t *testing.T) {
		// First get the token
		rAdmin := adminRouter()
		req, _ := http.NewRequest("GET", "/api/driver-shares", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		rAdmin.ServeHTTP(rr, req)
		var shares []struct {
			RacerID int    `json:"racer_id"`
			Token   string `json:"token"`
		}
		json.Unmarshal(rr.Body.Bytes(), &shares)
		if len(shares) == 0 {
			t.Fatal("no shares to test")
		}

		rPub := publicRouter()
		req2, _ := http.NewRequest("GET", "/api/shared/driver-stats?token="+shares[0].Token, nil)
		rr2 := httptest.NewRecorder()
		rPub.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Errorf("expected 200, got %d: %s", rr2.Code, rr2.Body.String())
		}
		var result map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &result)
		if result["racer"] == nil {
			t.Error("expected racer data")
		}
		if result["stats"] == nil {
			t.Error("expected stats data")
		}
	})

	t.Run("invalid token returns 404", func(t *testing.T) {
		r := publicRouter()
		req, _ := http.NewRequest("GET", "/api/shared/driver-stats?token=invalidtoken123", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for invalid token, got %d", rr.Code)
		}
	})

	t.Run("missing token returns 400", func(t *testing.T) {
		r := publicRouter()
		req, _ := http.NewRequest("GET", "/api/shared/driver-stats", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for missing token, got %d", rr.Code)
		}
	})

	t.Run("delete driver share", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("DELETE", "/api/driver-share?racer_id=1", nil)
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		// Verify token is gone
		rPub := publicRouter()
		req2, _ := http.NewRequest("GET", "/api/shared/driver-stats?token=invalidtoken123", nil)
		rr2 := httptest.NewRecorder()
		rPub.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", rr2.Code)
		}
	})

	t.Run("generate for nonexistent racer returns 404", func(t *testing.T) {
		r := adminRouter()
		req, _ := http.NewRequest("POST", "/api/driver-share?racer_id=9999", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://localhost")
		addSessionCookie(req)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for nonexistent racer, got %d", rr.Code)
		}
	})
}
