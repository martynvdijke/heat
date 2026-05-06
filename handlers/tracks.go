package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

// @Summary Get all tracks
// @Description Get the list of all race tracks
// @Tags Tracks
// @Produce json
// @Success 200 {array} models.Track
// @Router /api/tracks [get]
func GetTracks(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, name, country, geojson, length_km, lap_record, COALESCE(use_map_image, 0), COALESCE(map_image_url, ''), COALESCE(refresh_geojson, 1) FROM tracks ORDER BY name")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var tracks []models.Track
	for rows.Next() {
		var t models.Track
		var useMapImage, refreshGeoJSON int
		if err := rows.Scan(&t.ID, &t.Name, &t.Country, &t.GeoJSON, &t.Length, &t.LapRecord, &useMapImage, &t.MapImageURL, &refreshGeoJSON); err != nil {
			continue
		}
		t.UseMapImage = useMapImage == 1
		t.RefreshGeoJSON = refreshGeoJSON == 1
		tracks = append(tracks, t)
	}
	c.JSON(http.StatusOK, tracks)
}

// @Summary Create or update a track
// @Description Creates a new track or updates an existing one
// @Tags Tracks
// @Accept json
// @Produce json
// @Param track body models.Track true "Track data"
// @Success 200 {object} models.Track
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/tracks [post]
func SaveTrack(c *gin.Context) {
	var t models.Track
	if err := c.ShouldBindJSON(&t); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := app.DB.Exec(`INSERT OR REPLACE INTO tracks (id, name, country, geojson, length_km, lap_record, use_map_image, map_image_url, refresh_geojson) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Country, t.ID, t.Length, t.LapRecord, boolToInt(t.UseMapImage), t.MapImageURL, boolToInt(t.RefreshGeoJSON))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, t)
}

// @Summary Delete a track
// @Description Delete a track by ID
// @Tags Tracks
// @Produce json
// @Param id query string true "Track ID"
// @Success 200
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/tracks [delete]
func DeleteTrack(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}

	_, err := app.DB.Exec("DELETE FROM tracks WHERE id = ?", id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// @Summary AI Track Extraction
// @Description Analyzes a track image using AI to extract GeoJSON data
// @Tags Tracks
// @Accept json
// @Accept mpfd
// @Produce json
// @Param image_url body object false "JSON with image_url field"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/tracks/ai-extract [post]
func HandleAIExtract(c *gin.Context) {
	aiURL := os.Getenv("AI_TRACK_EXTRACT_URL")
	if aiURL == "" {
		var dbURL string
		var enabled bool
		err := app.DB.QueryRow("SELECT track_extract_url, enabled FROM ai_settings WHERE id = 1").Scan(&dbURL, &enabled)
		if err == nil && enabled && dbURL != "" {
			aiURL = dbURL
		}
	}
	if aiURL == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "AI endpoint not configured"})
		return
	}

	var imageData []byte
	var contentType string

	file, header, err := c.Request.FormFile("image")
	if err == nil {
		defer file.Close()
		imageData, err = io.ReadAll(file)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read image"})
			return
		}
		contentType = header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	} else {
		var input struct {
			ImageURL string `json:"image_url"`
		}
		if err := c.ShouldBindJSON(&input); err != nil || input.ImageURL == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "No image provided. Send multipart form with 'image' field or JSON with 'image_url'"})
			return
		}
		cleanPath := filepath.Clean(input.ImageURL)
		if strings.Contains(cleanPath, "..") || strings.HasPrefix(cleanPath, "/") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid image URL"})
			return
		}
		localPath := filepath.Join(app.MediaPath, cleanPath)
		if !strings.HasPrefix(localPath, filepath.Clean(app.MediaPath)+string(filepath.Separator)) && localPath != filepath.Clean(app.MediaPath) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid image URL"})
			return
		}
		imageData, err = os.ReadFile(localPath)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid image URL"})
			return
		}
		contentType = http.DetectContentType(imageData)
	}

	reqBody := &bytes.Buffer{}
	writer := multipart.NewWriter(reqBody)
	part, _ := writer.CreateFormFile("image", "track.png")
	part.Write(imageData)
	writer.Close()

	client := &http.Client{Timeout: 60 * time.Second}
	req, err := http.NewRequest("POST", aiURL, reqBody)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create AI request"})
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	var apiKey string
	app.DB.QueryRow("SELECT COALESCE(api_key, '') FROM ai_settings WHERE id = 1").Scan(&apiKey)
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "AI request failed: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read AI response"})
		return
	}

	if resp.StatusCode >= 400 {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "AI request failed"})
		return
	}

	var parsedResponse interface{}
	if err := json.Unmarshal(bodyBytes, &parsedResponse); err != nil {
		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}

	c.Data(http.StatusOK, "application/json", bodyBytes)
}
