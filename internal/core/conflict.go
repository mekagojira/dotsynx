package core

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"dotsynx/internal/config"
	"dotsynx/internal/git"
)

// ConflictItem represents an unresolved git or file conflict
type ConflictItem struct {
	Path       string `json:"path"`
	Diff       string `json:"diff"`
	RepoPath   string `json:"repo_path"`
	HomePath   string `json:"home_path"`
	CanAutoFix bool   `json:"can_auto_fix"`
}

// ConflictManager handles detecting and resolving conflicts
type ConflictManager struct {
	Config  *config.Config
	Git     *git.Client
	Tracker *Tracker
}

// NewConflictManager creates a conflict manager
func NewConflictManager(cfg *config.Config, g *git.Client, tr *Tracker) *ConflictManager {
	return &ConflictManager{
		Config:  cfg,
		Git:     g,
		Tracker: tr,
	}
}

// DetectConflicts checks if git has conflicts or diverged files
func (cm *ConflictManager) DetectConflicts(ctx context.Context) ([]ConflictItem, error) {
	status, err := cm.Git.Status(ctx)
	if err != nil {
		return nil, err
	}

	var conflicts []ConflictItem

	if status.HasConflict {
		// Get diffs for conflicting files
		for _, file := range append(status.Modified, status.Untracked...) {
			diff, _ := cm.Git.Diff(ctx, file)
			if diff != "" {
				conflicts = append(conflicts, ConflictItem{
					Path:       file,
					Diff:       diff,
					RepoPath:   filepath.Join(cm.Config.StorageDir, file),
					HomePath:   filepath.Join(cm.Tracker.HomeDir, file),
					CanAutoFix: true,
				})
			}
		}
	}

	return conflicts, nil
}

// Resolve resolves a conflict on a file using 'ours' or 'theirs'
func (cm *ConflictManager) Resolve(ctx context.Context, relPath string, strategy config.ConflictStrategy) error {
	var gitStrategy string
	switch strategy {
	case config.StrategyOurs:
		gitStrategy = "ours"
	case config.StrategyTheirs:
		gitStrategy = "theirs"
	default:
		return fmt.Errorf("unsupported resolution strategy: %s", strategy)
	}

	// Make a safety backup first
	fullRepoPath := filepath.Join(cm.Config.StorageDir, relPath)
	_, _ = cm.Tracker.Backup.Backup(fullRepoPath)

	err := cm.Git.CheckoutStrategy(ctx, relPath, gitStrategy)
	if err != nil {
		return fmt.Errorf("git checkout strategy failed: %w", err)
	}

	// Commit the resolution
	msg := fmt.Sprintf("Resolved conflict in %s using %s strategy at %s", relPath, strategy, time.Now().Format("2006-01-02 15:04"))
	if err := cm.Git.Commit(ctx, msg); err != nil {
		// It might be clean or staged already
		_ = cm.Git.Add(ctx, ".")
		_ = cm.Git.Commit(ctx, msg)
	}

	// Reapply symlink if needed
	for _, it := range cm.Config.Tracked {
		if it.Path == relPath {
			_ = cm.Tracker.ApplyItem(it)
			break
		}
	}

	return nil
}
