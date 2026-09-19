package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dotsynx/internal/config"
	"dotsynx/internal/git"
	"dotsynx/internal/logger"
)

// SyncResult details the outcome of a sync run
type SyncResult struct {
	Success      bool           `json:"success"`
	Pulled       bool           `json:"pulled"`
	Pushed       bool           `json:"pushed"`
	LocalChanges int            `json:"local_changes"`
	AppliedFiles []string       `json:"applied_files"`
	Errors       []string       `json:"errors"`
	Conflicts    []ConflictItem `json:"conflicts"`
	Timestamp    time.Time      `json:"timestamp"`
	Message      string         `json:"message"`
}

// Engine orchestrates dotfiles synchronization
type Engine struct {
	Config   *config.Config
	Git      *git.Client
	Tracker  *Tracker
	Backup   *BackupManager
	Conflict *ConflictManager
	LastSync *SyncResult
	SyncLogs []*SyncResult
}

// NewEngine constructs the dotsynx sync engine
func NewEngine(cfg *config.Config) (*Engine, error) {
	tr, err := NewTracker(cfg)
	if err != nil {
		return nil, err
	}

	g := git.NewClient(cfg.StorageDir)
	bm := NewBackupManager(cfg.BackupDir)
	cm := NewConflictManager(cfg, g, tr)

	return &Engine{
		Config:   cfg,
		Git:      g,
		Tracker:  tr,
		Backup:   bm,
		Conflict: cm,
	}, nil
}

// EnsureRepoReady ensures storage dir exists and git is initialized
func (e *Engine) EnsureRepoReady(ctx context.Context) error {
	if !e.Git.Exists() {
		if e.Config.RepoURL != "" {
			err := e.Git.Clone(ctx, e.Config.RepoURL, e.Config.Branch)
			if err != nil {
				// If clone fails (e.g. empty or non-existent remote repo), init local and set remote
				if err := e.Git.Init(e.Config.Branch); err != nil {
					return err
				}
				_ = e.Git.SetRemote(e.Config.RepoURL)
			}
		} else {
			if err := e.Git.Init(e.Config.Branch); err != nil {
				return err
			}
		}
	} else if e.Config.RepoURL != "" {
		_ = e.Git.SetRemote(e.Config.RepoURL)
	}

	// Sanity check: ensure .git/HEAD never points to .invalid
	headPath := filepath.Join(e.Config.StorageDir, ".git", "HEAD")
	if data, err := os.ReadFile(headPath); err == nil {
		if strings.Contains(string(data), ".invalid") {
			targetBranch := e.Config.Branch
			if targetBranch == "" {
				targetBranch = "main"
			}
			_ = os.WriteFile(headPath, []byte("ref: refs/heads/"+targetBranch+"\n"), 0644)
		}
	}

	// Create a .gitignore in storage repo to prevent accidental secrets
	giPath := e.Config.StorageDir + "/.gitignore"
	if _, err := os.Stat(giPath); os.IsNotExist(err) {
		giContent := `# Ignore potential secrets and OS metadata
.DS_Store
Thumbs.db
*.key
*.pem
*.p12
*.pfx
id_rsa
id_ed25519
.env*
credentials
`
		_ = os.WriteFile(giPath, []byte(giContent), 0644)
	}

	// Ensure there is at least an initial commit so the default branch exists
	if !e.Git.HasCommits(ctx) {
		_ = e.Tracker.WriteRepoManifest()
		_ = e.Git.Add(ctx, ".")
		_ = e.Git.Commit(ctx, "Initial dotsynx repository setup")
	}

	return nil
}

// Sync performs a full sync operation
func (e *Engine) Sync(ctx context.Context) (*SyncResult, error) {
	res := &SyncResult{
		Timestamp:    time.Now(),
		AppliedFiles: []string{},
		Errors:       []string{},
		Conflicts:    []ConflictItem{},
	}

	if err := e.EnsureRepoReady(ctx); err != nil {
		res.Errors = append(res.Errors, err.Error())
		res.Message = "Failed to initialize storage repository"
		logger.Error("Storage repo init failed: %v", err)
		return res, err
	}

	logger.Sync("Starting sync cycle (storage: %s, mode: %s)", e.Config.StorageDir, e.Config.SyncMode)

	// 1. Check current git status
	status, err := e.Git.Status(ctx)
	if err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res, err
	}

	// 2. If remote configured, fetch and check divergence
	if status.RemoteURL != "" {
		_ = e.Git.Fetch(ctx)
		status, _ = e.Git.Status(ctx)
	}

	// 3. Stage and commit local dotfile modifications if any
	_ = e.Tracker.WriteRepoManifest()
	hasLocalChanges := !status.Clean && (len(status.Modified) > 0 || len(status.Untracked) > 0)

	if !e.Git.HasCommits(ctx) || hasLocalChanges {
		if err := e.Git.Add(ctx, "."); err == nil {
			hostname, _ := os.Hostname()
			if hostname == "" {
				hostname = "local"
			}
			msg := fmt.Sprintf("dotsynx update from %s at %s", hostname, time.Now().Format("2006-01-02 15:04:05"))
			if !e.Git.HasCommits(ctx) {
				msg = fmt.Sprintf("Initial dotfiles sync from %s", hostname)
			}
			if err := e.Git.Commit(ctx, msg); err != nil {
				logger.Error("Git commit failed: %v", err)
				res.Errors = append(res.Errors, fmt.Sprintf("Commit failed: %v", err))
			} else {
				res.LocalChanges = len(status.Modified) + len(status.Untracked)
				logger.Info("Committed changes: %s", msg)
			}
		} else {
			logger.Error("Git add failed: %v", err)
		}
	}

	// 4. Pull if remote exists and branch exists on remote
	remoteBranchExists := e.Git.HasRemoteBranch(ctx, e.Config.Branch) || e.Git.RemoteBranchExists(ctx, e.Config.Branch)

	if status.RemoteURL != "" && remoteBranchExists {
		if status.Behind > 0 || hasLocalChanges {
			if err := e.Git.Pull(ctx, e.Config.Branch); err != nil {
				errStr := err.Error()
				if strings.Contains(errStr, "couldn't find remote ref") || strings.Contains(errStr, "no such ref") {
					logger.Info("Remote branch '%s' is empty or not yet created on remote. Proceeding with initial push.", e.Config.Branch)
				} else {
					// Check for conflict
					conflicts, confErr := e.Conflict.DetectConflicts(ctx)
					if confErr == nil && len(conflicts) > 0 {
						res.Conflicts = conflicts
						res.Message = fmt.Sprintf("Merge conflict detected in %d file(s)", len(conflicts))

						// If non-interactive auto strategy is configured, resolve immediately
						if e.Config.ConflictStrategy == config.StrategyOurs || e.Config.ConflictStrategy == config.StrategyTheirs {
							for _, c := range conflicts {
								_ = e.Conflict.Resolve(ctx, c.Path, e.Config.ConflictStrategy)
							}
						} else {
							return res, nil
						}
					} else {
						res.Errors = append(res.Errors, fmt.Sprintf("Pull failed: %v", err))
						logger.Warn("Git pull failed: %v", err)
					}
				}
			} else {
				res.Pulled = true
				// Merge any dotfiles tracked by other computers
				_ = e.Tracker.LoadRepoManifest()
			}
		}
	} else if status.RemoteURL != "" && !remoteBranchExists {
		logger.Info("Remote branch '%s' does not exist yet (brand new repository). Skipping pull and preparing initial push.", e.Config.Branch)
	}

	// 5. Push if ahead or if remote branch does not exist yet
	status, _ = e.Git.Status(ctx)
	needsPush := !remoteBranchExists || status.Ahead > 0 || hasLocalChanges
	if e.Git.HasCommits(ctx) && needsPush {
		if err := e.Git.Push(ctx, e.Config.Branch); err != nil {
			res.Errors = append(res.Errors, fmt.Sprintf("Push failed: %v", err))
			logger.Warn("Git push failed: %v", err)
		} else {
			res.Pushed = true
			logger.Info("Git push to origin/%s succeeded", e.Config.Branch)
		}
	}

	// 6. Apply all symlinks to $HOME
	applied, applyErrs := e.Tracker.ApplyAll()
	res.AppliedFiles = applied
	for _, ae := range applyErrs {
		res.Errors = append(res.Errors, ae.Error())
	}

	if len(res.Errors) == 0 {
		res.Success = true
		res.Message = fmt.Sprintf("Sync completed successfully (%d items linked)", len(applied))
		logger.Sync("Sync succeeded: %s", res.Message)
	} else {
		res.Success = false
		res.Message = fmt.Sprintf("Sync completed with %d error(s)", len(res.Errors))
		logger.Error("Sync completed with errors: %s", strings.Join(res.Errors, " | "))
	}

	e.LastSync = res
	e.SyncLogs = append([]*SyncResult{res}, e.SyncLogs...)
	if len(e.SyncLogs) > 20 {
		e.SyncLogs = e.SyncLogs[:20]
	}

	return res, nil
}
