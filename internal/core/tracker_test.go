package core

import (
	"os"
	"path/filepath"
	"testing"

	"dotsynx/internal/config"
)

func TestTrackerAndBackup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dotsynx-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	storageDir := filepath.Join(tmpDir, "repo")
	backupDir := filepath.Join(tmpDir, "backups")
	homeDir := filepath.Join(tmpDir, "home")

	_ = os.MkdirAll(storageDir, 0755)
	_ = os.MkdirAll(backupDir, 0755)
	_ = os.MkdirAll(homeDir, 0755)

	// Create dummy dotfile in home
	sampleFile := filepath.Join(homeDir, ".dummyrc")
	if err := os.WriteFile(sampleFile, []byte("theme=dark\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		StorageDir: storageDir,
		BackupDir:  backupDir,
		Tracked:    []config.TrackedItem{},
	}

	tracker := &Tracker{
		Config:  cfg,
		HomeDir: homeDir,
		Backup:  NewBackupManager(backupDir),
	}

	// Test backup
	bPath, err := tracker.Backup.Backup(sampleFile)
	if err != nil {
		t.Fatalf("backup failed: %v", err)
	}
	if _, err := os.Stat(bPath); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}

	// Copy to repo manually to simulate tracking
	repoSample := filepath.Join(storageDir, ".dummyrc")
	if err := os.WriteFile(repoSample, []byte("theme=dark\n"), 0644); err != nil {
		t.Fatal(err)
	}

	item := config.TrackedItem{
		Path:    ".dummyrc",
		Enabled: true,
	}

	// Apply link
	if err := tracker.ApplyItem(item); err != nil {
		t.Fatalf("ApplyItem failed: %v", err)
	}

	// Verify it is a symlink
	info, err := os.Lstat(sampleFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("expected symlink, got regular file")
	}

	target, _ := os.Readlink(sampleFile)
	if target != repoSample {
		t.Errorf("symlink target mismatch: expected %s, got %s", repoSample, target)
	}

	// Inspect status
	st := tracker.InspectItem(item)
	if st.LinkStatus != StatusLinked {
		t.Errorf("expected StatusLinked, got %s", st.LinkStatus)
	}
}
