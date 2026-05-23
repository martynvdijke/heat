package handlers

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"heat/db"
	"heat/models"
)

// @Summary Get backup settings
// @Description Get the backup configuration settings
// @Tags Backup
// @Produce json
// @Success 200 {object} models.BackupSettings
// @Router /api/backup-settings [get]
func (h *Handler) GetBackupSettings(c *gin.Context) {
	var s models.BackupSettings
	err := h.S.DB.QueryRow("SELECT id, enabled, interval_hrs, retention_count FROM backup_settings WHERE id = 1").Scan(&s.ID, &s.Enabled, &s.IntervalHrs, &s.RetentionCount)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if s.RetentionCount <= 0 {
		s.RetentionCount = 7
	}
	c.JSON(http.StatusOK, s)
}

// @Summary Save backup settings
// @Description Save the backup configuration settings
// @Tags Backup
// @Accept json
// @Produce json
// @Param settings body models.BackupSettings true "Backup settings"
// @Success 200
// @Failure 400 {object} map[string]string
// @Router /api/backup-settings [post]
func (h *Handler) SaveBackupSettings(c *gin.Context) {
	var s models.BackupSettings
	if err := c.ShouldBindJSON(&s); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	enabled := 0
	if s.Enabled {
		enabled = 1
	}
	if s.RetentionCount <= 0 {
		s.RetentionCount = 7
	}
	_, err := h.S.DB.Exec("UPDATE backup_settings SET enabled = ?, interval_hrs = ?, retention_count = ? WHERE id = 1", enabled, s.IntervalHrs, s.RetentionCount)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

// @Summary Create manual backup
// @Description Trigger a manual database backup
// @Tags Backup
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security cookieAuth
// @Router /api/backup/manual [post]
func (h *Handler) TriggerManualBackup(c *gin.Context) {
	if err := db.CreateBackup(); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err := db.PruneBackups(); err != nil {
		log.Printf("[BACKUP] Prune failed: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// @Summary List backups
// @Description List recent backup files
// @Tags Backup
// @Produce json
// @Success 200 {array} object
// @Router /api/backup/list [get]
func (h *Handler) ListBackups(c *gin.Context) {
	backupDir := filepath.Join(filepath.Dir(h.S.DBPath), "backups")
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
