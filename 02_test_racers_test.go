package main

import (
	"bytes"
	"encoding/json"
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

func TestGetRacers(t *testing.T) {
	r := gin.New()
	r.GET("/api/racers", testHandler.GetRacers)

	req, err := http.NewRequest("GET", "/api/racers", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	var racers []models.Racer
	err = json.Unmarshal(rr.Body.Bytes(), &racers)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(racers) != 5 {
		t.Errorf("expected 5 racers, got %d", len(racers))
	}
}

func TestProfilePictureUpload(t *testing.T) {
	r := gin.New()
	r.MaxMultipartMemory = 32 << 20
	r.POST("/api/upload", middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer), testHandler.HandleUpload)
	r.POST("/api/racers", middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer), testHandler.UpdateRacer)
	r.GET("/api/racers", testHandler.GetRacers)
	r.Static("/media", testServer.MediaPath)

	sessionID := "profile-pic-test-session"
	testServer.SessionStoreMu.Lock()
	testServer.SessionStore[sessionID] = app.SessionInfo{Expiry: time.Now().Add(1 * time.Hour).Unix()}
	testServer.SessionStoreMu.Unlock()
	defer func() {
		testServer.SessionStoreMu.Lock()
		delete(testServer.SessionStore, sessionID)
		testServer.SessionStoreMu.Unlock()
	}()

	// Each subtest uses unique image data to avoid hash collisions
	t.Run("UploadAndVerifyHTTPAccess", func(t *testing.T) {
		pngData := makeUniquePNGData(0xAA)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("image", "profile_pic.png")
		if err != nil {
			t.Fatal(err)
		}
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
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
		if err := json.Unmarshal(rr.Body.Bytes(), &uploadResp); err != nil {
			t.Fatalf("failed to parse upload response: %v", err)
		}

		uploadURL, ok := uploadResp["url"].(string)
		if !ok {
			t.Fatalf("expected url in response, got %v", uploadResp)
		}
		if !strings.HasPrefix(uploadURL, "/media/") {
			t.Fatalf("expected url to start with /media/, got %q", uploadURL)
		}

		// Verify the uploaded file is HTTP-accessible
		getReq, _ := http.NewRequest("GET", uploadURL, nil)
		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, getReq)
		if status := rr2.Code; status != http.StatusOK {
			t.Errorf("uploaded file not accessible via HTTP: got status %v (url: %s)", status, uploadURL)
		}
		if rr2.Body.Len() == 0 {
			t.Error("uploaded file HTTP response body is empty")
		}

		// Clean up
		parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
		if len(parts) == 2 {
			defer os.RemoveAll(filepath.Join(testServer.MediaPath, parts[0]))
		}
	})

	t.Run("UploadAndVerifyRacerEndpoint", func(t *testing.T) {
		pngData := makeUniquePNGData(0xBB)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "racer_pic.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("upload failed: %v: %s", rr.Code, rr.Body.String())
		}

		var uploadResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &uploadResp)
		uploadURL := uploadResp["url"].(string)
		defer func() {
			parts := strings.Split(strings.TrimPrefix(uploadURL, "/media/"), "/")
			if len(parts) == 2 {
				os.RemoveAll(filepath.Join(testServer.MediaPath, parts[0]))
			}
		}()

		// Update racer with profile_picture
		racerData := map[string]interface{}{
			"id":              1,
			"name":            "Profile Pic Racer",
			"profile_picture": uploadURL,
			"car_color":       "blue",
			"car_name":        "Profile Pic Car",
			"points":          50,
			"rank":            2,
			"position":        0,
		}
		racerJSON, _ := json.Marshal(racerData)
		racerReq, _ := http.NewRequest("POST", "/api/racers", bytes.NewBuffer(racerJSON))
		racerReq.Header.Set("Content-Type", "application/json")
		racerReq.Header.Set("Origin", "http://127.0.0.1:6270")
		racerReq.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		racerReq.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, racerReq)
		if rr2.Code != http.StatusOK {
			t.Fatalf("update racer failed: %v: %s", rr2.Code, rr2.Body.String())
		}

		// Verify racers list includes the profile_picture
		getReq, _ := http.NewRequest("GET", "/api/racers", nil)
		rr3 := httptest.NewRecorder()
		r.ServeHTTP(rr3, getReq)
		if rr3.Code != http.StatusOK {
			t.Fatalf("get racers failed: %v", rr3.Code)
		}

		var racers []models.Racer
		if err := json.Unmarshal(rr3.Body.Bytes(), &racers); err != nil {
			t.Fatalf("failed to parse racers response: %v", err)
		}

		var found bool
		for _, racer := range racers {
			if racer.ID == 1 {
				found = true
				if racer.ProfilePicture != uploadURL {
					t.Errorf("expected profile_picture %q, got %q", uploadURL, racer.ProfilePicture)
				}
				break
			}
		}
		if !found {
			t.Error("racer with id=1 not found in racers list")
		}
	})

	t.Run("DuplicateUploadReturnsExistingURL", func(t *testing.T) {
		pngData := makeUniquePNGData(0xCC)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "dup.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("first upload failed: %v: %s", rr.Code, rr.Body.String())
		}

		var firstResp map[string]interface{}
		json.Unmarshal(rr.Body.Bytes(), &firstResp)
		firstURL := firstResp["url"].(string)
		defer func() {
			parts := strings.Split(strings.TrimPrefix(firstURL, "/media/"), "/")
			if len(parts) == 2 {
				os.RemoveAll(filepath.Join(testServer.MediaPath, parts[0]))
			}
		}()

		// Upload same image again
		body2 := new(bytes.Buffer)
		writer2 := multipart.NewWriter(body2)
		part2, _ := writer2.CreateFormFile("image", "dup2.png")
		part2.Write(pngData)
		writer2.Close()

		req2, _ := http.NewRequest("POST", "/api/upload", body2)
		req2.Header.Set("Content-Type", writer2.FormDataContentType())
		req2.Header.Set("Origin", "http://127.0.0.1:6270")
		req2.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req2.Host = "127.0.0.1:6270"

		rr2 := httptest.NewRecorder()
		r.ServeHTTP(rr2, req2)
		if rr2.Code != http.StatusOK {
			t.Fatalf("duplicate upload failed: %v: %s", rr2.Code, rr2.Body.String())
		}

		var secondResp map[string]interface{}
		json.Unmarshal(rr2.Body.Bytes(), &secondResp)

		secondURL, ok := secondResp["url"].(string)
		if !ok {
			t.Fatalf("expected url in duplicate response, got %v", secondResp)
		}
		if secondURL != firstURL {
			t.Errorf("duplicate upload returned different URL: got %q, want %q", secondURL, firstURL)
		}

		isDup, ok := secondResp["duplicate"].(bool)
		if !ok || !isDup {
			t.Errorf("expected duplicate=true in response, got %v", secondResp)
		}
	})

	t.Run("UploadWithoutAuthReturns401", func(t *testing.T) {
		pngData := makeUniquePNGData(0xDD)
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "noauth.png")
		part.Write(pngData)
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for unauthenticated upload, got %v", rr.Code)
		}
	})

	t.Run("UploadInvalidFileTypeReturns400", func(t *testing.T) {
		body := new(bytes.Buffer)
		writer := multipart.NewWriter(body)
		part, _ := writer.CreateFormFile("image", "test.txt")
		part.Write([]byte("not an image"))
		writer.Close()

		req, _ := http.NewRequest("POST", "/api/upload", body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Origin", "http://127.0.0.1:6270")
		req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
		req.Host = "127.0.0.1:6270"

		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for invalid file type, got %v", rr.Code)
		}
	})
}

func TestGetRacerEmails(t *testing.T) {
	r := gin.New()
	r.GET("/api/racer-emails", middleware.AuthMiddleware(testServer), testHandler.GetRacerEmails)

	t.Run("Unauthorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/racer-emails", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if status := rr.Code; status != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %v", status)
		}
	})
}
