package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"heat/middleware"
	"heat/models"
)

const testPDF = "%PDF-1.4\n1 0 obj\n<< /Type /Catalog >>\nendobj\ntrailer\n<<>>\n%%EOF"

func assetsRouter() *gin.Engine {
	r := gin.New()
	admin := r.Group("/api")
	admin.Use(middleware.CSRFMiddleware(), middleware.AuthMiddleware(testServer))
	admin.POST("/assets", testHandler.UploadAsset)
	admin.PUT("/assets", testHandler.UpdateAsset)
	admin.DELETE("/assets", testHandler.DeleteAsset)
	admin.GET("/extensions/detail", testHandler.GetExtensionDetail)
	r.GET("/api/assets", testHandler.GetAssets)
	r.GET("/api/assets/:id/file", testHandler.GetAssetFile)
	r.GET("/api/assets/:id/thumbnail", testHandler.GetAssetThumbnail)
	r.GET("/api/assets/:id/download", testHandler.DownloadAsset)
	r.Group("/media", middleware.BlockStaticSubdir("assets")).Static("", testServer.MediaPath)
	return r
}

// uniquePNGAsset returns a decodable 2x2 PNG whose pixel data and trailing byte
// vary with seed, so different seeds hash differently while the file stays a
// valid image.
func uniquePNGAsset(seed byte) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: seed, A: 255})
	img.Set(1, 1, color.RGBA{B: seed, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return append(buf.Bytes(), seed)
}

type assetResponse struct {
	models.Asset
	Duplicate bool `json:"duplicate"`
}

func newAssetUpload(t *testing.T, fields map[string]string, filename string, data []byte, sessionID string) *http.Request {
	t.Helper()
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write form file: %v", err)
	}
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	req, err := http.NewRequest("POST", "/api/assets", body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://127.0.0.1:6270")
	req.AddCookie(&http.Cookie{Name: "session", Value: sessionID})
	req.Host = "127.0.0.1:6270"
	return req
}

func baseExtensionID(t *testing.T) int {
	t.Helper()
	var id int
	if err := testServer.DB.QueryRow("SELECT id FROM extensions WHERE is_base = 1 LIMIT 1").Scan(&id); err != nil {
		t.Fatalf("base extension: %v", err)
	}
	return id
}

func TestAssetLibrary(t *testing.T) {
	r := assetsRouter()
	sessionID := createAdminSession(t)
	defer removeAdminSession(sessionID)

	baseExt := baseExtensionID(t)

	var pngID int

	t.Run("upload image map", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{
			"title":        "Test Map",
			"description":  "An unofficial map",
			"kind":         "map",
			"extension_id": strconv.Itoa(baseExt),
		}, "test-map.png", uniquePNGAsset(0x21), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp assetResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		pngID = resp.ID
		if pngID == 0 {
			t.Fatal("expected asset id")
		}
		if resp.URL != fmt.Sprintf("/api/assets/%d/file", pngID) {
			t.Errorf("expected the controlled file URL, got %q", resp.URL)
		}
		if resp.ThumbnailURL != fmt.Sprintf("/api/assets/%d/thumbnail", pngID) {
			t.Errorf("expected the controlled thumbnail URL, got %q", resp.ThumbnailURL)
		}
		if resp.MIMEType != "image/png" {
			t.Errorf("expected image/png, got %q", resp.MIMEType)
		}
		if resp.SizeBytes <= 0 {
			t.Error("expected a positive size")
		}
		if resp.ExtensionID != baseExt {
			t.Errorf("expected extension %d, got %d", baseExt, resp.ExtensionID)
		}
	})

	t.Run("duplicate upload reuses the asset", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{"kind": "map"}, "test-map-copy.png", uniquePNGAsset(0x21), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp assetResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		if !resp.Duplicate {
			t.Error("expected duplicate flag")
		}
		if resp.ID != pngID {
			t.Errorf("expected existing id %d, got %d", pngID, resp.ID)
		}
	})

	var pdfID int
	t.Run("upload extension PDF", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{
			"title":        "Rule Sheet",
			"kind":         "extension",
			"extension_id": strconv.Itoa(baseExt),
		}, "rule-sheet.pdf", []byte(testPDF), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp assetResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		pdfID = resp.ID
		if resp.MIMEType != "application/pdf" {
			t.Errorf("expected application/pdf, got %q", resp.MIMEType)
		}
		if resp.ThumbnailURL != "" {
			t.Errorf("expected no thumbnail for a PDF, got %q", resp.ThumbnailURL)
		}
		if resp.URL != fmt.Sprintf("/api/assets/%d/file", pdfID) {
			t.Errorf("expected the controlled file URL, got %q", resp.URL)
		}
	})

	t.Run("reject unsupported extension", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{"kind": "other"}, "malware.exe", []byte("MZ"), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("reject mismatched content", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{"kind": "map"}, "fake.png", []byte(testPDF), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("extension kind requires an extension", func(t *testing.T) {
		req := newAssetUpload(t, map[string]string{"kind": "extension"}, "orphan.pdf", []byte(testPDF), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("list and filter assets", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/assets?kind=extension", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		var assets []models.Asset
		json.Unmarshal(rr.Body.Bytes(), &assets)
		found := false
		for _, a := range assets {
			if a.ID == pdfID {
				found = true
			}
			if a.Kind != "extension" {
				t.Errorf("expected only extension assets, got %q", a.Kind)
			}
		}
		if !found {
			t.Error("expected the PDF in the kind filter")
		}

		req, _ = http.NewRequest("GET", "/api/assets?extension_id="+strconv.Itoa(baseExt), nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		var byExt []models.Asset
		json.Unmarshal(rr.Body.Bytes(), &byExt)
		if len(byExt) < 2 {
			t.Errorf("expected at least 2 assets for extension %d, got %d", baseExt, len(byExt))
		}
	})

	t.Run("download asset", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/assets/"+strconv.Itoa(pdfID)+"/download", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		if !strings.Contains(rr.Header().Get("Content-Disposition"), "attachment") {
			t.Errorf("expected attachment disposition, got %q", rr.Header().Get("Content-Disposition"))
		}
		if rr.Body.String() != testPDF {
			t.Error("expected the original PDF bytes")
		}
	})

	t.Run("update metadata", func(t *testing.T) {
		body, _ := json.Marshal(map[string]any{
			"id":           pngID,
			"title":        "Updated Map",
			"description":  "Now with a new title",
			"kind":         "other",
			"extension_id": 0,
			"track_id":     "",
		})
		req := newAdminRequest("PUT", "/api/assets", body, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var updated models.Asset
		json.Unmarshal(rr.Body.Bytes(), &updated)
		if updated.Title != "Updated Map" || updated.Kind != "other" || updated.ExtensionID != 0 {
			t.Errorf("unexpected updated asset: %+v", updated)
		}
	})

	t.Run("set uploaded map as track background", func(t *testing.T) {
		var trackID string
		if err := testServer.DB.QueryRow("SELECT id FROM tracks ORDER BY id LIMIT 1").Scan(&trackID); err != nil {
			t.Fatalf("pick track: %v", err)
		}
		req := newAssetUpload(t, map[string]string{
			"title":                 "Background Map",
			"kind":                  "map",
			"track_id":              trackID,
			"set_as_map_background": "1",
		}, "background.png", uniquePNGAsset(0x33), sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var resp assetResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)

		var mapURL string
		var useMap int
		if err := testServer.DB.QueryRow("SELECT map_image_url, use_map_image FROM tracks WHERE id = ?", trackID).Scan(&mapURL, &useMap); err != nil {
			t.Fatalf("read track: %v", err)
		}
		if mapURL != resp.URL || useMap != 1 {
			t.Errorf("expected track background %q and use_map_image=1, got %q/%d", resp.URL, mapURL, useMap)
		}
	})

	t.Run("extension detail lists files", func(t *testing.T) {
		req := newAdminRequest("GET", "/api/extensions/detail?id="+strconv.Itoa(baseExt), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		var detail struct {
			Assets []models.Asset `json:"assets"`
		}
		json.Unmarshal(rr.Body.Bytes(), &detail)
		found := false
		for _, a := range detail.Assets {
			if a.ID == pdfID {
				found = true
			}
		}
		if !found {
			t.Errorf("expected the extension PDF in the detail files, got %+v", detail.Assets)
		}
	})

	t.Run("preview and thumbnail endpoints", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/assets/"+strconv.Itoa(pngID)+"/file", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		if ct := rr.Header().Get("Content-Type"); !strings.HasPrefix(ct, "image/png") {
			t.Errorf("expected image/png, got %q", ct)
		}
		if !strings.Contains(rr.Header().Get("Content-Disposition"), "inline") {
			t.Errorf("expected inline disposition, got %q", rr.Header().Get("Content-Disposition"))
		}

		req, _ = http.NewRequest("GET", "/api/assets/"+strconv.Itoa(pngID)+"/thumbnail", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 for thumbnail, got %d: %s", rr.Code, rr.Body.String())
		}

		req, _ = http.NewRequest("GET", "/api/assets/"+strconv.Itoa(pdfID)+"/thumbnail", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for a PDF thumbnail, got %d", rr.Code)
		}
	})

	t.Run("raw media path is not served", func(t *testing.T) {
		var storedURL string
		if err := testServer.DB.QueryRow("SELECT url FROM assets WHERE id = ?", pngID).Scan(&storedURL); err != nil {
			t.Fatalf("read stored path: %v", err)
		}
		if !strings.HasPrefix(storedURL, "/media/assets/") {
			t.Fatalf("expected a stored media path, got %q", storedURL)
		}
		req, _ := http.NewRequest("GET", storedURL, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 for direct media access, got %d", rr.Code)
		}
	})

	t.Run("delete asset removes the file", func(t *testing.T) {
		var storedURL, storedThumb string
		if err := testServer.DB.QueryRow("SELECT url, thumbnail_url FROM assets WHERE id = ?", pngID).Scan(&storedURL, &storedThumb); err != nil {
			t.Fatalf("read stored paths: %v", err)
		}
		localPath := filepath.Join(testServer.MediaPath, filepath.FromSlash(strings.TrimPrefix(storedURL, "/media/")))
		localThumb := filepath.Join(testServer.MediaPath, filepath.FromSlash(strings.TrimPrefix(storedThumb, "/media/")))
		if _, err := os.Stat(localPath); err != nil {
			t.Fatalf("expected stored file: %v", err)
		}

		req := newAdminRequest("DELETE", "/api/assets?id="+strconv.Itoa(pngID), nil, sessionID)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
		}
		if _, err := os.Stat(localPath); !os.IsNotExist(err) {
			t.Errorf("expected the file to be removed, err=%v", err)
		}
		if _, err := os.Stat(localThumb); !os.IsNotExist(err) {
			t.Errorf("expected the thumbnail to be removed, err=%v", err)
		}

		req, _ = http.NewRequest("GET", "/api/assets/"+strconv.Itoa(pngID)+"/download", nil)
		rr = httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", rr.Code)
		}
	})
}
