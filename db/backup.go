package db

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func CreateBackup() error {
	backupDir := filepath.Join(filepath.Dir(srv.DBPath), "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return err
	}
	backupPath := filepath.Join(backupDir, "heat_backup_"+time.Now().Format("20060102_150405")+".db")
	_, err := srv.DB.Exec("VACUUM INTO ?", backupPath)
	return err
}

func PruneBackups() error {
	var retentionCount int
	err := srv.DB.QueryRow("SELECT retention_count FROM backup_settings WHERE id = 1").Scan(&retentionCount)
	if err != nil || retentionCount <= 0 {
		retentionCount = 7
	}

	backupDir := filepath.Join(filepath.Dir(srv.DBPath), "backups")
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil
	}

	var backups []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "heat_backup_") && strings.HasSuffix(e.Name(), ".db") {
			backups = append(backups, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(backups)))

	if len(backups) <= retentionCount {
		return nil
	}

	for _, name := range backups[retentionCount:] {
		os.Remove(filepath.Join(backupDir, name))
	}
	return nil
}
