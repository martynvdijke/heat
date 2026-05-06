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

	"heat/app"
	"heat/models"
)

func HandleUpload(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".avif", ".svg", ".bmp", ".tiff", ".tif"}

	validExt := false
	for _, e := range allowedExts {
		if ext == e {
			validExt = true
			break
		}
	}
	if !validExt {
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

	var existingURL string
	err = app.DB.QueryRow("SELECT url FROM uploads WHERE hash = ?", hashStr).Scan(&existingURL)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{"url": existingURL, "duplicate": true})
		return
	}

	saveExt := ext
	if ext == ".jpeg" {
		saveExt = ".jpg"
	}

	subDir := hashStr[:2]
	dir := filepath.Join(app.MediaPath, subDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"})
		return
	}

	filename := hashStr + saveExt
	uploadPath := filepath.Join(dir, filename)
	url := fmt.Sprintf("/media/%s/%s", subDir, filename)
	var thumbnailURL string

	if ext != ".gif" && ext != ".svg" && ext != ".tiff" && ext != ".tif" {
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
				os.WriteFile(uploadPath, data, 0644)
			}

			thumb := resizeImage(img, 300)
			thumbFilename := hashStr + "_thumb" + saveExt
			if err := saveImage(filepath.Join(dir, thumbFilename), thumb, format); err == nil {
				thumbnailURL = fmt.Sprintf("/media/%s/%s", subDir, thumbFilename)
			}
		} else {
			os.WriteFile(uploadPath, data, 0644)
		}
	} else {
		if err := os.WriteFile(uploadPath, data, 0644); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}
	}

	app.DB.Exec("INSERT INTO uploads (hash, ext, url, resized_url, thumbnail_url) VALUES (?, ?, ?, ?, ?)",
		hashStr, ext, url, url, thumbnailURL)

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
	h := bounds.Dy()

	if w <= maxDim && h <= maxDim {
		return img
	}

	ratio := float64(maxDim) / float64(max(w, h))
	newW := int(float64(w) * ratio)
	newH := int(float64(h) * ratio)

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

func GetUploads(c *gin.Context) {
	rows, err := app.DB.Query("SELECT id, hash, ext, url, resized_url, thumbnail_url, COALESCE(created_at, '') FROM uploads ORDER BY created_at DESC LIMIT 50")
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
