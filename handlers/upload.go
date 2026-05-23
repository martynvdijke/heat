package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/draw"

	"heat/models"
)

// HandleUpload godoc
// @Summary Upload an image
// @Description Upload an image file. Returns the URL and optional thumbnail URL.
// @Tags Upload
// @Accept mpfd
// @Produce json
// @Param image formData file true "Image file"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Security cookieAuth
// @Router /api/upload [post]
func (h *Handler) HandleUpload(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".avif": true, ".svg": true, ".bmp": true,
		".tiff": true, ".tif": true,
	}
	if !allowedExts[ext] {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	mimeType := http.DetectContentType(data)
	if !strings.HasPrefix(mimeType, "image/") {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "File content does not match image type"})
		return
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	saveExt := ext
	if ext == ".jpeg" {
		saveExt = ".jpg"
	}
	if ext == ".tiff" {
		saveExt = ".tif"
	}

	subDir := hashStr[:2]
	dir := filepath.Join(h.S.MediaPath, subDir)

	filename := hashStr + saveExt
	url := fmt.Sprintf("/media/%s/%s", subDir, filename)
	var thumbnailURL string

	result, err := h.S.DB.Exec(
		"INSERT OR IGNORE INTO uploads (hash, ext, url, resized_url, thumbnail_url) VALUES (?, ?, ?, ?, ?)",
		hashStr, ext, url, url, thumbnailURL)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	uploadPath := filepath.Join(dir, filename)

	rows, _ := result.RowsAffected()
	if rows == 0 {
		var existingURL string
		if err := h.S.DB.QueryRow("SELECT url FROM uploads WHERE hash = ?", hashStr).Scan(&existingURL); err == nil {
			if _, statErr := os.Stat(uploadPath); statErr == nil {
				c.JSON(http.StatusOK, gin.H{"url": existingURL, "duplicate": true})
				return
			}
		}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	if ext != ".gif" && ext != ".svg" && ext != ".tif" && ext != ".tiff" {
		img, format, err := image.Decode(bytes.NewReader(data))
		if err == nil {
			size := img.Bounds().Size()
			if size.X > 10000 || size.Y > 10000 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Image dimensions too large (max 10000x10000)"})
				return
			}
			if size.X > 1920 || size.Y > 1920 {
				img = resizeImage(img, 1920)
			}

			if err := saveImage(uploadPath, img, format); err != nil {
				if werr := os.WriteFile(uploadPath, data, 0644); werr != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
					return
				}
			}

			thumb := resizeImage(img, 300)
			thumbFilename := hashStr + "_thumb" + saveExt
			if err := saveImage(filepath.Join(dir, thumbFilename), thumb, format); err == nil {
				thumbnailURL = fmt.Sprintf("/media/%s/%s", subDir, thumbFilename)
			}
		} else {
			if err := os.WriteFile(uploadPath, data, 0644); err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
				return
			}
		}
	} else {
		if err := os.WriteFile(uploadPath, data, 0644); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}
	}

	if thumbnailURL != "" {
		h.S.DB.Exec("UPDATE uploads SET thumbnail_url = ? WHERE hash = ?", thumbnailURL, hashStr)
	}

	c.JSON(http.StatusOK, gin.H{
		"url":           url,
		"resized_url":   url,
		"thumbnail_url": thumbnailURL,
		"hash":          hashStr,
	})
}

func resizeImage(img image.Image, maxDim int) image.Image {
	bounds := img.Bounds()
	w := bounds.Dx()
	ht := bounds.Dy()

	if w <= maxDim && ht <= maxDim {
		return img
	}

	ratio := float64(maxDim) / float64(max(w, ht))
	newW := int(float64(w) * ratio)
	newH := int(float64(ht) * ratio)

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)

	return dst
}

func saveImage(path string, img image.Image, format string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	switch format {
	case "png":
		return png.Encode(f, img)
	default:
		return jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	}
}

// GetUploads godoc
// @Summary List recent uploads
// @Description Get the 50 most recent uploads
// @Tags Upload
// @Produce json
// @Success 200 {array} models.Upload
// @Router /api/uploads [get]
func (h *Handler) GetUploads(c *gin.Context) {
	rows, err := h.S.DB.Query("SELECT id, hash, ext, url, resized_url, thumbnail_url, COALESCE(created_at, '') FROM uploads ORDER BY created_at DESC LIMIT 50")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	uploads := make([]models.Upload, 0)
	for rows.Next() {
		var u models.Upload
		if err := rows.Scan(&u.ID, &u.Hash, &u.Ext, &u.URL, &u.ResizedURL, &u.ThumbnailURL, &u.CreatedAt); err != nil {
			continue
		}
		uploads = append(uploads, u)
	}
	c.JSON(http.StatusOK, uploads)
}
