package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/models"
)

func HandleUpload(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)

	header, err := c.FormFile("image")
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid file type"})
		return
	}

	file, err := header.Open()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
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

	filename := hashStr + saveExt
	uploadPath := filepath.Join(app.ImagesPath, filename)

	if err := os.WriteFile(uploadPath, data, 0644); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resizedFilename := hashStr + "_resized" + saveExt
	thumbFilename := hashStr + "_thumb" + saveExt
	resizedPath := filepath.Join(app.ImagesPath, resizedFilename)
	thumbPath := filepath.Join(app.ImagesPath, thumbFilename)

	src, err := imaging.Open(uploadPath)
	if err == nil {
		resized := imaging.Fit(src, 1200, 1200, imaging.Lanczos)
		if err := imaging.Save(resized, resizedPath); err != nil {
			log.Printf("[UPLOAD] Failed to save resized: %v", err)
		} else {
			resizedData, _ := os.ReadFile(resizedPath)
			app.StaticCache["/static/images/"+resizedFilename] = resizedData
		}

		thumb := imaging.Thumbnail(src, 150, 150, imaging.Lanczos)
		if err := imaging.Save(thumb, thumbPath); err != nil {
			log.Printf("[UPLOAD] Failed to save thumbnail: %v", err)
		} else {
			thumbData, _ := os.ReadFile(thumbPath)
			app.StaticCache["/static/images/"+thumbFilename] = thumbData
		}
	}

	url := "/static/images/" + filename
	resizedURL := "/static/images/" + resizedFilename
	thumbURL := "/static/images/" + thumbFilename

	app.DB.Exec("INSERT INTO uploads (hash, ext, url, resized_url, thumbnail_url) VALUES (?, ?, ?, ?, ?)",
		hashStr, ext, url, resizedURL, thumbURL)

	app.StaticCache[url] = data

	c.JSON(http.StatusOK, gin.H{
		"url":           url,
		"resized_url":   resizedURL,
		"thumbnail_url": thumbURL,
		"hash":          hashStr,
	})
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
