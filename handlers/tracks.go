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

// boardGameTrackSet returns the set of track ids currently in the board game list.
func (h *Handler) boardGameTrackSet() map[string]bool {
	set := make(map[string]bool)
	rows, err := h.S.DB.Query("SELECT track_id FROM board_game_tracks")
	if err != nil {
		return set
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			set[id] = true
		}
	}
	return set
}

// trackToModel maps an Ent track row to the shared models.Track shape.
func trackToModel(t *ent.Track, boardGame map[string]bool) models.Track {
	return models.Track{
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
		ModuleID:       t.ModuleID,
		IsBoardGame:    boardGame[t.ID],
	}
}

func (h *Handler) GetTracks(c *gin.Context) {
	query := h.S.Ent.Track.Query().Order(track.ByName())
	// owned=1 restricts the list to content the group owns (selection UIs).
	// Without the param the full catalog is returned (management/catalog use).
	if c.Query("owned") == "1" {
		query = query.Where(track.ExtensionIDIn(h.ownedExtensionIDs()...))
	}
	tracks, err := query.All(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	boardGame := h.boardGameTrackSet()
	result := make([]models.Track, len(tracks))
	for i, t := range tracks {
		result[i] = trackToModel(t, boardGame)
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
	// When a module is set, derive the extension from the module's owning extension
	// so the extension catalog stays consistent.
	if t.ModuleID > 0 {
		var extID int
		if err := h.S.DB.QueryRow("SELECT extension_id FROM modules WHERE id = ?", t.ModuleID).Scan(&extID); err == nil && extID > 0 {
			t.ExtensionID = extID
		}
	}
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
			SetModuleID(t.ModuleID).
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
			SetModuleID(t.ModuleID).
			Save(ctx)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, t)
}

// GetBoardGameTracks returns the ids of tracks currently in the board game list.
func (h *Handler) GetBoardGameTracks(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT track_id FROM board_game_tracks")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			ids = append(ids, id)
		}
	}
	c.JSON(http.StatusOK, gin.H{"track_ids": ids})
}

// SetBoardGameTracks replaces the entire board game track list with the
// submitted track ids. Unknown ids are ignored so a stale editor can't
// reference deleted tracks.
func (h *Handler) SetBoardGameTracks(c *gin.Context) {
	var input struct {
		TrackIDs []string `json:"track_ids"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	tx, err := h.S.DB.Begin()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec("DELETE FROM board_game_tracks"); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, id := range input.TrackIDs {
		if id == "" {
			continue
		}
		// Ignore unknown ids so a stale editor can't reference deleted tracks.
		var exists int
		if err := tx.QueryRow("SELECT COUNT(*) FROM tracks WHERE id = ?", id).Scan(&exists); err != nil || exists == 0 {
			continue
		}
		if _, err := tx.Exec("INSERT OR IGNORE INTO board_game_tracks (track_id) VALUES (?)", id); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
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
	h.S.DB.Exec("DELETE FROM board_game_tracks WHERE track_id = ?", id)

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
