package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// BackupManager handles creating and restoring timestamped backups
type BackupManager struct {
	BackupDir string
}

// BackupEntry represents an individual backup snapshot
type BackupEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
	FileCount int       `json:"file_count"`
}

// NewBackupManager creates a backup manager
func NewBackupManager(backupDir string) *BackupManager {
	return &BackupManager{BackupDir: backupDir}
}

// Backup creates a timestamped copy of a file or directory before modifying it
func (bm *BackupManager) Backup(sourcePath string) (string, error) {
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return "", nil // Nothing to backup
	}

	timestamp := time.Now().Format("20060102-150405")
	destRoot := filepath.Join(bm.BackupDir, timestamp)

	cleanRel := filepath.Base(sourcePath)
	destPath := filepath.Join(destRoot, cleanRel)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", fmt.Errorf("failed to create backup dir: %w", err)
	}

	if err := copyRecursive(sourcePath, destPath); err != nil {
		return "", fmt.Errorf("backup failed for %s: %w", sourcePath, err)
	}

	return destPath, nil
}

// ListBackups returns all available backup snapshots
func (bm *BackupManager) ListBackups() ([]BackupEntry, error) {
	if _, err := os.Stat(bm.BackupDir); os.IsNotExist(err) {
		return []BackupEntry{}, nil
	}

	entries, err := os.ReadDir(bm.BackupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		t, err := time.Parse("20060102-150405", entry.Name())
		if err != nil {
			continue
		}
		fullPath := filepath.Join(bm.BackupDir, entry.Name())
		count := countFiles(fullPath)

		backups = append(backups, BackupEntry{
			ID:        entry.Name(),
			Timestamp: t,
			Path:      fullPath,
			FileCount: count,
		})
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].Timestamp.After(backups[j].Timestamp)
	})

	return backups, nil
}

func copyRecursive(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}

	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			srcChild := filepath.Join(src, entry.Name())
			dstChild := filepath.Join(dst, entry.Name())
			if err := copyRecursive(srcChild, dstChild); err != nil {
				return err
			}
		}
		return nil
	}

	// Regular file
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func countFiles(dir string) int {
	count := 0
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			count++
		}
		return nil
	})
	return count
}
