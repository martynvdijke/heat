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
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"heat/ent"
	"heat/ent/aisetting"
	"heat/ent/track"
	"heat/models"
)

type geoJSONCacheEntry struct {
	data   []byte
	expiry int64
}

var (
	geoJSONCache     map[string]*geoJSONCacheEntry
	geoJSONCacheMu   sync.RWMutex
	geoJSONCacheInit sync.Once
)

func (h *Handler) GetTrackGeoJSON(c *gin.Context) {
	geoJSONCacheInit.Do(func() {
		geoJSONCache = make(map[string]*geoJSONCacheEntry)
	})

	geoJSONCacheMu.RLock()
	entry, ok := geoJSONCache["all"]
	geoJSONCacheMu.RUnlock()
	if ok && time.Now().Unix() < entry.expiry {
		c.Data(http.StatusOK, "application/json", entry.data)
		return
	}

	tracks, err := h.S.Ent.Track.Query().All(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make(map[string]json.RawMessage)
	for _, t := range tracks {
		if t.Geojson != "" {
			result[t.ID] = json.RawMessage(strings.TrimSpace(t.Geojson))
		}
	}

	data, err := json.Marshal(result)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal GeoJSON"})
		return
	}

	geoJSONCacheMu.Lock()
	geoJSONCache["all"] = &geoJSONCacheEntry{data: data, expiry: time.Now().Unix() + 300}
	geoJSONCacheMu.Unlock()

	c.Data(http.StatusOK, "application/json", data)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (h *Handler) GetTracks(c *gin.Context) {
	tracks, err := h.S.Ent.Track.Query().Order(track.ByName()).All(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	result := make([]models.Track, len(tracks))
	for i, t := range tracks {
		result[i] = models.Track{
			ID:             t.ID,
			Name:           t.Name,
			Country:        t.Country,
			GeoJSON:        t.Geojson,
			Length:         t.LengthKm,
			LapRecord:      t.LapRecord,
			UseMapImage:    t.UseMapImage == 1,
			MapImageURL:    t.MapImageURL,
			RefreshGeoJSON: t.RefreshGeojson == 1,
			ExtensionID:    t.ExtensionID,
		}
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) SaveTrack(c *gin.Context) {
	var t models.Track
	if err := c.ShouldBindJSON(&t); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	_, err := h.S.Ent.Track.Get(ctx, t.ID)
	if ent.IsNotFound(err) {
		_, err = h.S.Ent.Track.Create().
			SetID(t.ID).
			SetName(t.Name).
			SetCountry(t.Country).
			SetGeojson(t.GeoJSON).
			SetLengthKm(t.Length).
			SetLapRecord(t.LapRecord).
			SetUseMapImage(boolToInt(t.UseMapImage)).
			SetMapImageURL(t.MapImageURL).
			SetRefreshGeojson(boolToInt(t.RefreshGeoJSON)).
			SetExtensionID(t.ExtensionID).
			Save(ctx)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else {
		_, err = h.S.Ent.Track.UpdateOneID(t.ID).
			SetName(t.Name).
			SetCountry(t.Country).
			SetGeojson(t.GeoJSON).
			SetLengthKm(t.Length).
			SetLapRecord(t.LapRecord).
			SetUseMapImage(boolToInt(t.UseMapImage)).
			SetMapImageURL(t.MapImageURL).
			SetRefreshGeojson(boolToInt(t.RefreshGeoJSON)).
			SetExtensionID(t.ExtensionID).
			Save(ctx)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, t)
}

func (h *Handler) DeleteTrack(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "ID required"})
		return
	}

	err := h.S.Ent.Track.DeleteOneID(id).Exec(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (h *Handler) HandleAIExtract(c *gin.Context) {
	aiURL := os.Getenv("AI_TRACK_EXTRACT_URL")
	if aiURL == "" {
		setting, err := h.S.Ent.AISetting.Query().Where(aisetting.ID(1)).First(c.Request.Context())
		if err == nil && setting.Enabled == 1 && setting.TrackExtractURL != "" {
			aiURL = setting.TrackExtractURL
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
		localPath := filepath.Join(h.S.MediaPath, cleanPath)
		if !strings.HasPrefix(localPath, filepath.Clean(h.S.MediaPath)+string(filepath.Separator)) && localPath != filepath.Clean(h.S.MediaPath) {
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

	setting, err := h.S.Ent.AISetting.Query().Where(aisetting.ID(1)).First(c.Request.Context())
	if err == nil && setting.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+setting.APIKey)
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

	var parsedResponse json.RawMessage
	if err := json.Unmarshal(bodyBytes, &parsedResponse); err != nil {
		c.Data(http.StatusOK, "application/json", bodyBytes)
		return
	}

	c.Data(http.StatusOK, "application/json", bodyBytes)
}
