package handlers

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	_ "golang.org/x/image/webp"

	"heat/models"
)

const maxAssetSize int64 = 25 << 20 // 25 MB

// assetExtMIME is the allowlist of upload extensions and the only content type
// each extension may contain. SVG is deliberately excluded: it can execute
// script when opened directly from the same origin.
var assetExtMIME = map[string]string{
	".pdf":  "application/pdf",
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".webp": "image/webp",
	".gif":  "image/gif",
}

// assetKind is one of map, extension, or other. Unknown values are rejected.
func normalizeAssetKind(kind string) string {
	return strings.ToLower(strings.TrimSpace(kind))
}

func normalizeAssetExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == ".jpeg" {
		return ".jpg"
	}
	return ext
}

// assetSelect is shared by the list and single-row queries so both expose the
// same joined association names.
const assetSelect = `SELECT a.id, a.hash, a.ext, a.mime_type, a.size_bytes, a.original_name,
	a.title, a.description, a.kind, a.extension_id, a.track_id, a.url, a.thumbnail_url,
	COALESCE(a.created_at, ''), COALESCE(e.name, ''), COALESCE(t.name, '')
	FROM assets a
	LEFT JOIN extensions e ON e.id = a.extension_id
	LEFT JOIN tracks t ON t.id = a.track_id`

type assetRowScanner interface {
	Scan(dest ...any) error
}

func scanAsset(row assetRowScanner, a *models.Asset) error {
	if err := row.Scan(
		&a.ID, &a.Hash, &a.Ext, &a.MIMEType, &a.SizeBytes, &a.OriginalName,
		&a.Title, &a.Description, &a.Kind, &a.ExtensionID, &a.TrackID,
		&a.FilePath, &a.ThumbPath, &a.CreatedAt, &a.ExtensionName, &a.TrackName,
	); err != nil {
		return err
	}
	// Files are only reachable through the API so we never hand out the raw
	// media path.
	a.URL = assetFileURL(a.ID)
	if a.ThumbPath != "" {
		a.ThumbnailURL = assetThumbURL(a.ID)
	}
	return nil
}

// assetFileURL and assetThumbURL are the public, controlled routes for an
// asset's bytes. The underlying media file is not served directly.
func assetFileURL(id int) string  { return fmt.Sprintf("/api/assets/%d/file", id) }
func assetThumbURL(id int) string { return fmt.Sprintf("/api/assets/%d/thumbnail", id) }

type assetFilter struct {
	Kind        string
	ExtensionID int
	TrackID     string
	Query       string
	Limit       int
}

func (h *Handler) queryAssets(f assetFilter) ([]models.Asset, error) {
	where := make([]string, 0, 4)
	args := make([]any, 0, 6)

	if kind := normalizeAssetKind(f.Kind); kind != "" {
		where = append(where, "a.kind = ?")
		args = append(args, kind)
	}
	if f.ExtensionID > 0 {
		where = append(where, "a.extension_id = ?")
		args = append(args, f.ExtensionID)
	}
	if trackID := strings.TrimSpace(f.TrackID); trackID != "" {
		where = append(where, "a.track_id = ?")
		args = append(args, trackID)
	}
	if q := strings.TrimSpace(f.Query); q != "" {
		where = append(where, "(a.title LIKE ? OR a.original_name LIKE ?)")
		like := "%" + q + "%"
		args = append(args, like, like)
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 200
	}
	if limit > 500 {
		limit = 500
	}

	query := assetSelect
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += " ORDER BY a.created_at DESC, a.id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := h.S.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	assets := make([]models.Asset, 0)
	for rows.Next() {
		var a models.Asset
		if err := scanAsset(rows, &a); err != nil {
			continue
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

func (h *Handler) getAssetByID(id int) (*models.Asset, error) {
	var a models.Asset
	if err := scanAsset(h.S.DB.QueryRow(assetSelect+" WHERE a.id = ?", id), &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// validateAssetMeta checks that the kind is known and that any supplied
// associations point at real rows.
func (h *Handler) validateAssetMeta(kind string, extensionID int, trackID string) error {
	switch kind {
	case "map", "extension", "other":
	default:
		return fmt.Errorf("invalid kind")
	}
	if kind == "extension" && extensionID <= 0 {
		return fmt.Errorf("extension_id required for extension assets")
	}
	if extensionID > 0 {
		var count int
		if err := h.S.DB.QueryRow("SELECT COUNT(*) FROM extensions WHERE id = ?", extensionID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("extension not found")
		}
	}
	if trackID != "" {
		var count int
		if err := h.S.DB.QueryRow("SELECT COUNT(*) FROM tracks WHERE id = ?", trackID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return fmt.Errorf("track not found")
		}
	}
	return nil
}

// assetLocalPath maps a /media URL back to a file below media/assets. It
// refuses any URL that would escape the asset subtree.
func (h *Handler) assetLocalPath(mediaURL string) (string, bool) {
	rel := strings.TrimPrefix(mediaURL, "/media/")
	if rel == mediaURL || rel == "" {
		return "", false
	}
	rel = filepath.FromSlash(rel)
	clean := filepath.Clean(rel)
	prefix := "assets" + string(filepath.Separator)
	if clean != "assets" && !strings.HasPrefix(clean, prefix) {
		return "", false
	}
	return filepath.Join(h.S.MediaPath, clean), true
}

// writeAssetFile stores the original bytes and, for images, a thumbnail. The
// returned URLs are the public media paths.
func (h *Handler) writeAssetFile(hash, ext, mimeType string, data []byte) (string, string, error) {
	subDir := hash[:2]
	dir := filepath.Join(h.S.MediaPath, "assets", subDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}

	filename := hash + ext
	if err := os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return "", "", err
	}
	url := fmt.Sprintf("/media/assets/%s/%s", subDir, filename)

	var thumbnailURL string
	if strings.HasPrefix(mimeType, "image/") {
		img, format, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			thumb := resizeImage(img, 480)
			thumbExt, thumbFormat := ".jpg", "jpeg"
			if format == "png" {
				thumbExt, thumbFormat = ".png", "png"
			}
			thumbName := hash + "_thumb" + thumbExt
			if err := saveImage(filepath.Join(dir, thumbName), thumb, thumbFormat); err == nil {
				thumbnailURL = fmt.Sprintf("/media/assets/%s/%s", subDir, thumbName)
			}
		}
	}
	return url, thumbnailURL, nil
}

func (h *Handler) removeAssetFiles(a *models.Asset) {
	for _, stored := range []string{a.FilePath, a.ThumbPath} {
		if stored == "" {
			continue
		}
		if path, ok := h.assetLocalPath(stored); ok {
			_ = os.Remove(path)
		}
	}
}

type assetUploadResponse struct {
	models.Asset
	Duplicate bool `json:"duplicate,omitempty"`
}

// UploadAsset godoc
// @Summary Upload a map or extension asset
// @Description Upload a PDF or image asset (max 25 MB) and optionally tag it to an extension or track
// @Tags Assets
// @Accept mpfd
// @Produce json
// @Param file formData file true "PDF, PNG, JPEG, WebP, or GIF"
// @Param title formData string false "Display title"
// @Param description formData string false "Description"
// @Param kind formData string false "map, extension, or other"
// @Param extension_id formData int false "Associated extension"
// @Param track_id formData string false "Associated track"
// @Param set_as_map_background formData bool false "Use an image map as the track background"
// @Success 200 {object} models.Asset
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/assets [post]
func (h *Handler) UploadAsset(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file field is required"})
		return
	}
	if fileHeader.Size <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file is empty"})
		return
	}
	if fileHeader.Size > maxAssetSize {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file too large (max 25 MB)"})
		return
	}

	originalName := filepath.Base(fileHeader.Filename)
	ext := normalizeAssetExt(filepath.Ext(originalName))
	expectedMIME, allowed := assetExtMIME[ext]
	if !allowed {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unsupported file type (allowed: PDF, PNG, JPG, WebP, GIF)"})
		return
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}
	if int64(len(data)) > maxAssetSize {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file too large (max 25 MB)"})
		return
	}

	detected := http.DetectContentType(data)
	detected = strings.TrimSpace(strings.SplitN(detected, ";", 2)[0])
	if detected != expectedMIME {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "file content does not match its extension"})
		return
	}
	if strings.HasPrefix(expectedMIME, "image/") {
		cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid image file"})
			return
		}
		if cfg.Width > 10000 || cfg.Height > 10000 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "image dimensions too large (max 10000x10000)"})
			return
		}
	}

	kind := normalizeAssetKind(c.PostForm("kind"))
	if kind == "" {
		kind = "other"
	}
	extensionID, _ := strconv.Atoi(c.PostForm("extension_id"))
	trackID := strings.TrimSpace(c.PostForm("track_id"))
	if err := h.validateAssetMeta(kind, extensionID, trackID); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setBackground := c.PostForm("set_as_map_background") == "1" || c.PostForm("set_as_map_background") == "true"
	if setBackground {
		if !strings.HasPrefix(expectedMIME, "image/") {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "only image assets can be used as a track background"})
			return
		}
		if trackID == "" {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "track_id is required when using a map as a background"})
			return
		}
	}

	title := strings.TrimSpace(c.PostForm("title"))
	if title == "" {
		title = strings.TrimSpace(strings.TrimSuffix(originalName, filepath.Ext(originalName)))
	}
	if title == "" {
		title = "Untitled asset"
	}
	description := strings.TrimSpace(c.PostForm("description"))
	if len(title) > 200 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "title is too long (max 200 characters)"})
		return
	}
	if len(description) > 2000 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "description is too long (max 2000 characters)"})
		return
	}
	if len(originalName) > 255 {
		originalName = originalName[:255]
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	// Identical bytes share one stored file. Re-write it in case the media file
	// was removed out-of-band, then return the existing metadata row.
	var duplicateID int
	err = h.S.DB.QueryRow("SELECT id FROM assets WHERE hash = ?", hashStr).Scan(&duplicateID)
	if err == nil {
		if _, _, err := h.writeAssetFile(hashStr, ext, expectedMIME, data); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
			return
		}
		if existing, err := h.getAssetByID(duplicateID); err == nil {
			c.JSON(http.StatusOK, assetUploadResponse{Asset: *existing, Duplicate: true})
			return
		}
	} else if err != sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}

	url, thumbnailURL, err := h.writeAssetFile(hashStr, ext, expectedMIME, data)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}

	result, err := h.S.DB.Exec(`INSERT INTO assets
		(hash, ext, mime_type, size_bytes, original_name, title, description, kind, extension_id, track_id, url, thumbnail_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hashStr, ext, expectedMIME, len(data), originalName, title, description, kind, extensionID, trackID, url, thumbnailURL)
	if err != nil {
		h.removeAssetFiles(&models.Asset{FilePath: url, ThumbPath: thumbnailURL})
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "database error"})
		return
	}
	id64, _ := result.LastInsertId()

	if setBackground {
		if _, err := h.S.DB.Exec("UPDATE tracks SET map_image_url = ?, use_map_image = 1 WHERE id = ?", assetFileURL(int(id64)), trackID); err != nil {
			h.S.DB.Exec("DELETE FROM assets WHERE id = ?", id64)
			h.removeAssetFiles(&models.Asset{FilePath: url, ThumbPath: thumbnailURL})
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to set track background"})
			return
		}
	}

	asset, err := h.getAssetByID(int(id64))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to load asset"})
		return
	}
	c.JSON(http.StatusOK, assetUploadResponse{Asset: *asset})
}

// GetAssets godoc
// @Summary List map and extension assets
// @Description List uploaded assets, optionally filtered by kind, extension, track, or search text
// @Tags Assets
// @Produce json
// @Param kind query string false "map, extension, or other"
// @Param extension_id query int false "Extension ID"
// @Param track_id query string false "Track ID"
// @Param q query string false "Search title or original filename"
// @Param limit query int false "Maximum rows (default 200, max 500)"
// @Success 200 {array} models.Asset
// @Router /api/assets [get]
func (h *Handler) GetAssets(c *gin.Context) {
	extensionID, _ := strconv.Atoi(c.Query("extension_id"))
	limit, _ := strconv.Atoi(c.Query("limit"))

	assets, err := h.queryAssets(assetFilter{
		Kind:        c.Query("kind"),
		ExtensionID: extensionID,
		TrackID:     c.Query("track_id"),
		Query:       c.Query("q"),
		Limit:       limit,
	})
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, assets)
}

// UpdateAsset godoc
// @Summary Update asset metadata
// @Description Update an asset's title, description, kind, extension, or track
// @Tags Assets
// @Accept json
// @Produce json
// @Param body body object true "Asset metadata" SchemaExample({"id":1,"title":"Rain map","kind":"map","extension_id":2,"track_id":"heavy-rain-1"})
// @Success 200 {object} models.Asset
// @Security cookieAuth
// @Router /api/assets [put]
func (h *Handler) UpdateAsset(c *gin.Context) {
	var input struct {
		ID          int     `json:"id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Kind        *string `json:"kind"`
		ExtensionID *int    `json:"extension_id"`
		TrackID     *string `json:"track_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil || input.ID <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	asset, err := h.getAssetByID(input.ID)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}

	if input.Title != nil {
		asset.Title = strings.TrimSpace(*input.Title)
	}
	if input.Description != nil {
		asset.Description = strings.TrimSpace(*input.Description)
	}
	if input.Kind != nil {
		asset.Kind = normalizeAssetKind(*input.Kind)
	}
	if input.ExtensionID != nil {
		asset.ExtensionID = *input.ExtensionID
	}
	if input.TrackID != nil {
		asset.TrackID = strings.TrimSpace(*input.TrackID)
	}

	if asset.Title == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "title required"})
		return
	}
	if len(asset.Title) > 200 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "title is too long (max 200 characters)"})
		return
	}
	if len(asset.Description) > 2000 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "description is too long (max 2000 characters)"})
		return
	}
	if asset.Kind == "" {
		asset.Kind = "other"
	}
	if err := h.validateAssetMeta(asset.Kind, asset.ExtensionID, asset.TrackID); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := h.S.DB.Exec(
		"UPDATE assets SET title = ?, description = ?, kind = ?, extension_id = ?, track_id = ? WHERE id = ?",
		asset.Title, asset.Description, asset.Kind, asset.ExtensionID, asset.TrackID, asset.ID,
	); err != nil {
		internalError(c, err)
		return
	}

	updated, err := h.getAssetByID(asset.ID)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, updated)
}

// DeleteAsset godoc
// @Summary Delete an asset
// @Description Delete an asset row and its stored original and thumbnail
// @Tags Assets
// @Param id query int true "Asset ID"
// @Success 200
// @Security cookieAuth
// @Router /api/assets [delete]
func (h *Handler) DeleteAsset(c *gin.Context) {
	id, err := strconv.Atoi(c.Query("id"))
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	asset, err := h.getAssetByID(id)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}

	if _, err := h.S.DB.Exec("DELETE FROM assets WHERE id = ?", id); err != nil {
		internalError(c, err)
		return
	}
	h.removeAssetFiles(asset)
	c.Status(http.StatusOK)
}

// DownloadAsset godoc
// @Summary Download an asset
// @Description Download the original asset file with an attachment disposition
// @Tags Assets
// @Param id path int true "Asset ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/assets/{id}/download [get]
func (h *Handler) DownloadAsset(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	asset, err := h.getAssetByID(id)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}

	path, ok := h.storedAssetPath(asset.FilePath)
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset file not found"})
		return
	}

	name := filepath.Base(asset.OriginalName)
	name = strings.Map(func(r rune) rune {
		if r < 32 || r == '"' || r == '\\' || r == '/' {
			return '_'
		}
		return r
	}, name)
	if name == "" || name == "." {
		name = "asset" + asset.Ext
	}

	c.Header("Cache-Control", "public, max-age=3600")
	c.FileAttachment(path, name)
}

// storedAssetPath resolves an internal media path to an existing file on disk.
func (h *Handler) storedAssetPath(stored string) (string, bool) {
	if stored == "" {
		return "", false
	}
	path, ok := h.assetLocalPath(stored)
	if !ok {
		return "", false
	}
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

// GetAssetFile godoc
// @Summary View an asset inline
// @Description Serve the original asset file inline for in-browser preview
// @Tags Assets
// @Param id path int true "Asset ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/assets/{id}/file [get]
func (h *Handler) GetAssetFile(c *gin.Context) {
	h.serveAssetBytes(c, false)
}

// GetAssetThumbnail godoc
// @Summary View an asset thumbnail
// @Description Serve the generated thumbnail for an image asset
// @Tags Assets
// @Param id path int true "Asset ID"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/assets/{id}/thumbnail [get]
func (h *Handler) GetAssetThumbnail(c *gin.Context) {
	h.serveAssetBytes(c, true)
}

func (h *Handler) serveAssetBytes(c *gin.Context, thumbnail bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "id required"})
		return
	}

	asset, err := h.getAssetByID(id)
	if err == sql.ErrNoRows {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset not found"})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}

	stored := asset.FilePath
	if thumbnail {
		stored = asset.ThumbPath
	}
	path, ok := h.storedAssetPath(stored)
	if !ok {
		c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "asset file not found"})
		return
	}

	// Inline for previews; the allowlist (no HTML/SVG) keeps this same-origin
	// safe and nosniff stops content-type confusion.
	c.Header("Content-Disposition", "inline")
	if !thumbnail {
		c.Header("Content-Type", asset.MIMEType)
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.File(path)
}
