package handlers

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"heat/app"
	"heat/db"
	"heat/models"
)

func GetBackupSettings(c *gin.Context) {
	var s models.BackupSettings
	err := app.DB.QueryRow("SELECT id, enabled, interval_hrs FROM backup_settings WHERE id = 1").Scan(&s.ID, &s.Enabled, &s.IntervalHrs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, s)
}

func SaveBackupSettings(c *gin.Context) {
	var s models.BackupSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	_, err := app.DB.Exec("UPDATE backup_settings SET enabled = ?, interval_hrs = ? WHERE id = 1", enabled, s.IntervalHrs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func TriggerManualBackup(c *gin.Context) {
	if err := db.CreateBackup(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func ListBackups(c *gin.Context) {
	backupDir := filepath.Join(filepath.Dir(app.DBPath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(http.StatusOK, []gin.H{})
		return
	}

	type BackupInfo struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
		Time string `json:"time"`
	}

	var backups []BackupInfo
	for _, e := range entries {
		if !e.IsDir() {
			info, _ := e.Info()
			backups = append(backups, BackupInfo{
				Name: e.Name(),
				Size: info.Size(),
				Time: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}
	c.JSON(http.StatusOK, backups)
}
