package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dotsynx/internal/config"

	"gopkg.in/yaml.v3"
)

// FileLinkStatus describes the connection between $HOME and the git repo
type FileLinkStatus string

const (
	StatusLinked     FileLinkStatus = "linked"    // Properly symlinked to repo
	StatusBroken     FileLinkStatus = "broken"    // Symlink points to non-existent target
	StatusUnlinked   FileLinkStatus = "unlinked"  // Regular file exists in $HOME, not linked to repo
	StatusMissing    FileLinkStatus = "missing"   // Does not exist in $HOME, but exists in repo
	StatusNotTracked FileLinkStatus = "untracked" // Exists in $HOME, not in repo
)

// ItemStatus contains comprehensive state of a tracked dotfile
type ItemStatus struct {
	Item         config.TrackedItem `json:"item"`
	HomeAbsPath  string             `json:"home_abs_path"`
	RepoAbsPath  string             `json:"repo_abs_path"`
	LinkStatus   FileLinkStatus     `json:"link_status"`
	HomeExists   bool               `json:"home_exists"`
	RepoExists   bool               `json:"repo_exists"`
	HasDiff      bool               `json:"has_diff"`
	ErrorMessage string             `json:"error_message,omitempty"`
}

// Tracker handles linking and checking dotfiles between $HOME and the repository
type Tracker struct {
	Config  *config.Config
	HomeDir string
	Backup  *BackupManager
}

// NewTracker returns a dotfile tracker
func NewTracker(cfg *config.Config) (*Tracker, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &Tracker{
		Config:  cfg,
		HomeDir: home,
		Backup:  NewBackupManager(cfg.BackupDir),
	}, nil
}

// TargetRepoPath returns the absolute path inside the git repo for a tracked item
func (t *Tracker) TargetRepoPath(item config.TrackedItem) string {
	rel := item.RepoPath
	if rel == "" {
		rel = item.Path
	}
	// Avoid leading slashes
	rel = strings.TrimPrefix(rel, "/")
	return filepath.Join(t.Config.StorageDir, rel)
}

// TargetHomePath returns the absolute path in $HOME for a tracked item
func (t *Tracker) TargetHomePath(item config.TrackedItem) string {
	clean := strings.TrimPrefix(item.Path, "/")
	return filepath.Join(t.HomeDir, clean)
}

// InspectItem checks the current state of a tracked item
func (t *Tracker) InspectItem(item config.TrackedItem) ItemStatus {
	homeAbs := t.TargetHomePath(item)
	repoAbs := t.TargetRepoPath(item)

	status := ItemStatus{
		Item:        item,
		HomeAbsPath: homeAbs,
		RepoAbsPath: repoAbs,
	}

	_, repoErr := os.Stat(repoAbs)
	status.RepoExists = repoErr == nil

	homeInfo, homeErr := os.Lstat(homeAbs)
	status.HomeExists = homeErr == nil

	if !status.HomeExists {
		if status.RepoExists {
			status.LinkStatus = StatusMissing
		} else {
			status.LinkStatus = StatusNotTracked
		}
		return status
	}

	// Check if it is a symlink
	if homeInfo.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(homeAbs)
		if err != nil {
			status.LinkStatus = StatusBroken
			status.ErrorMessage = err.Error()
			return status
		}

		// Target could be relative or absolute
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(homeAbs), target)
		}
		target = filepath.Clean(target)

		if target == filepath.Clean(repoAbs) {
			if status.RepoExists {
				status.LinkStatus = StatusLinked
			} else {
				status.LinkStatus = StatusBroken
			}
		} else {
			status.LinkStatus = StatusUnlinked
			status.ErrorMessage = fmt.Sprintf("points to %s instead of repo", target)
		}
		return status
	}

	// It's a real file or directory, not a symlink
	status.LinkStatus = StatusUnlinked
	return status
}

// TrackItem copies a local dotfile into the repo (if not already there), creates a backup, and symlinks it
func (t *Tracker) TrackItem(relHomePath string, isDir bool, description string) error {
	relHome, homeAbs, err := config.NormalizePath(relHomePath)
	if err != nil {
		return err
	}

	// Auto-detect isDir if the source exists on disk
	if statInfo, statErr := os.Stat(homeAbs); statErr == nil {
		isDir = statInfo.IsDir()
	}

	// Check if already in config
	for i, item := range t.Config.Tracked {
		if item.Path == relHome {
			t.Config.Tracked[i].Enabled = true
			if description != "" {
				t.Config.Tracked[i].Description = description
			}
			_ = t.Config.Save("")
			_ = t.WriteRepoManifest()
			return t.ApplyItem(t.Config.Tracked[i])
		}
	}

	item := config.TrackedItem{
		Path:        relHome,
		RepoPath:    relHome,
		IsDir:       isDir,
		Enabled:     true,
		Description: description,
	}

	repoAbs := t.TargetRepoPath(item)
	_, repoErr := os.Stat(repoAbs)
	repoExists := repoErr == nil

	// Check if source exists in $HOME
	homeInfo, homeErr := os.Lstat(homeAbs)
	if homeErr != nil && !repoExists {
		return fmt.Errorf("file or directory does not exist: %s", homeAbs)
	}

	// Copy into repo if not already there
	if homeErr == nil && !repoExists {
		sourceToCopy := homeAbs
		if homeInfo.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(homeAbs)
			if filepath.Clean(target) == filepath.Clean(repoAbs) {
				repoExists = true
			} else if realTarget, err := filepath.EvalSymlinks(homeAbs); err == nil {
				sourceToCopy = realTarget
			}
		}

		if !repoExists {
			if err := os.MkdirAll(filepath.Dir(repoAbs), 0755); err != nil {
				return fmt.Errorf("failed to create repo dir: %w", err)
			}
			if err := copyRecursive(sourceToCopy, repoAbs); err != nil {
				return fmt.Errorf("failed to copy dotfile to repo: %w", err)
			}
		}
	}

	t.Config.Tracked = append(t.Config.Tracked, item)
	if err := t.Config.Save(""); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}
	_ = t.WriteRepoManifest()

	return t.ApplyItem(item)
}

// WriteRepoManifest saves the tracked items list inside the git repo
func (t *Tracker) WriteRepoManifest() error {
	if _, err := os.Stat(t.Config.StorageDir); os.IsNotExist(err) {
		return nil
	}
	manifestPath := filepath.Join(t.Config.StorageDir, "dotsynx.manifest.yaml")
	data, err := yaml.Marshal(t.Config.Tracked)
	if err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

// LoadRepoManifest merges any tracked items from git repository into local config
func (t *Tracker) LoadRepoManifest() error {
	manifestPath := filepath.Join(t.Config.StorageDir, "dotsynx.manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return t.AutoImportFromRepo()
		}
		return nil
	}

	var remoteTracked []config.TrackedItem
	if err := yaml.Unmarshal(data, &remoteTracked); err != nil {
		return err
	}

	existingMap := make(map[string]int)
	for i, it := range t.Config.Tracked {
		existingMap[it.Path] = i
	}

	changed := false
	for _, rem := range remoteTracked {
		if idx, exists := existingMap[rem.Path]; exists {
			if rem.Description != "" && t.Config.Tracked[idx].Description == "" {
				t.Config.Tracked[idx].Description = rem.Description
				changed = true
			}
		} else {
			t.Config.Tracked = append(t.Config.Tracked, rem)
			changed = true
		}
	}

	if changed {
		return t.Config.Save("")
	}
	return nil
}

// AutoImportFromRepo scans storage repo for dotfiles when dotsynx.manifest.yaml is absent
func (t *Tracker) AutoImportFromRepo() error {
	if _, err := os.Stat(t.Config.StorageDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(t.Config.StorageDir)
	if err != nil {
		return nil
	}

	ignoreList := map[string]bool{
		".git":                  true,
		".gitignore":            true,
		"dotsynx.manifest.yaml": true,
		".DS_Store":             true,
		"Thumbs.db":             true,
		"README.md":             true,
		"readme.md":             true,
		"LICENSE":               true,
		"license":               true,
		"Makefile":              true,
		"install.sh":            true,
		"bin":                   true,
		"web":                   true,
	}

	existingMap := make(map[string]bool)
	for _, it := range t.Config.Tracked {
		existingMap[it.Path] = true
	}

	changed := false
	for _, e := range entries {
		name := e.Name()
		if ignoreList[name] || strings.HasPrefix(name, "README") || strings.HasPrefix(name, "LICENSE") {
			continue
		}

		if name == ".config" && e.IsDir() {
			configEntries, cErr := os.ReadDir(filepath.Join(t.Config.StorageDir, ".config"))
			if cErr == nil {
				for _, ce := range configEntries {
					if ignoreList[ce.Name()] {
						continue
					}
					relPath := filepath.Join(".config", ce.Name())
					if !existingMap[relPath] {
						t.Config.Tracked = append(t.Config.Tracked, config.TrackedItem{
							Path:        relPath,
							RepoPath:    relPath,
							IsDir:       ce.IsDir(),
							Enabled:     true,
							Description: fmt.Sprintf("%s configuration", ce.Name()),
						})
						existingMap[relPath] = true
						changed = true
					}
				}
			}
			continue
		}

		if !existingMap[name] {
			t.Config.Tracked = append(t.Config.Tracked, config.TrackedItem{
				Path:        name,
				RepoPath:    name,
				IsDir:       e.IsDir(),
				Enabled:     true,
				Description: fmt.Sprintf("%s configuration", strings.TrimPrefix(name, ".")),
			})
			existingMap[name] = true
			changed = true
		}
	}

	if changed {
		_ = t.WriteRepoManifest()
		return t.Config.Save("")
	}
	return nil
}

// UntrackItem removes item from config. If keepFile is true, turns the symlink into a regular file
func (t *Tracker) UntrackItem(relPath string, keepFile bool) error {
	var remaining []config.TrackedItem
	var targetItem *config.TrackedItem

	for _, it := range t.Config.Tracked {
		if it.Path == relPath {
			targetItem = &it
		} else {
			remaining = append(remaining, it)
		}
	}

	if targetItem == nil {
		return fmt.Errorf("item %s not found in tracked list", relPath)
	}

	t.Config.Tracked = remaining
	if err := t.Config.Save(""); err != nil {
		return err
	}

	if keepFile {
		homeAbs := t.TargetHomePath(*targetItem)
		repoAbs := t.TargetRepoPath(*targetItem)

		// Check if it's currently a symlink
		if info, err := os.Lstat(homeAbs); err == nil && (info.Mode()&os.ModeSymlink != 0) {
			_ = os.Remove(homeAbs)
			if _, repoErr := os.Stat(repoAbs); repoErr == nil {
				_ = copyRecursive(repoAbs, homeAbs)
			}
		}
	}

	return nil
}

// ApplyItem ensures the symlink from $HOME to repo exists safely
func (t *Tracker) ApplyItem(item config.TrackedItem) error {
	if !item.Enabled {
		return nil
	}

	homeAbs := t.TargetHomePath(item)
	repoAbs := t.TargetRepoPath(item)

	// Ensure repo file exists
	if _, err := os.Stat(repoAbs); os.IsNotExist(err) {
		return fmt.Errorf("source in repo does not exist: %s", repoAbs)
	}

	// Check existing file in home
	if info, err := os.Lstat(homeAbs); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(homeAbs)
			if err == nil && filepath.Clean(target) == filepath.Clean(repoAbs) {
				return nil // Already correctly linked
			}
			_ = os.Remove(homeAbs) // Remove incorrect symlink
		} else {
			// Real file or directory exists; create a backup before replacing with symlink
			backupPath, err := t.Backup.Backup(homeAbs)
			if err != nil {
				return fmt.Errorf("failed to backup existing file before linking: %w", err)
			}
			_ = backupPath
			// Remove the existing file/dir
			if err := os.RemoveAll(homeAbs); err != nil {
				return fmt.Errorf("failed to remove existing file: %w", err)
			}
		}
	}

	// Ensure parent directory exists in Home
	if err := os.MkdirAll(filepath.Dir(homeAbs), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir: %w", err)
	}

	// Create symlink
	if err := os.Symlink(repoAbs, homeAbs); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

// ApplyAll applies all tracked and enabled dotfiles
func (t *Tracker) ApplyAll() ([]string, []error) {
	var applied []string
	var errs []error

	for _, item := range t.Config.Tracked {
		if !item.Enabled {
			continue
		}
		if err := t.ApplyItem(item); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", item.Path, err))
		} else {
			applied = append(applied, item.Path)
		}
	}

	return applied, errs
}
