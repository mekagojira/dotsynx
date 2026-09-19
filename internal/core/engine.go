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
		// If local repo has no tracked dotfiles and remote has commits, adopt remote branch
		commitCount, _ := e.Git.CommitCount(ctx)
		lastMsg, _ := e.Git.GetLastCommitMessage(ctx)
		isOnlyInitialCommit := commitCount <= 1 || strings.Contains(lastMsg, "Initial dotsynx repository setup")
		if len(e.Config.Tracked) == 0 && isOnlyInitialCommit {
			_ = e.Git.Fetch(ctx)
			if e.Git.HasRemoteBranch(ctx, e.Config.Branch) {
				logger.Info("Adopting remote branch origin/%s on new device", e.Config.Branch)
				_ = e.Git.ResetHard(ctx, "origin/"+e.Config.Branch)
				_ = e.Git.SetUpstream(ctx, e.Config.Branch)
				_ = e.Tracker.LoadRepoManifest()
			}
		}
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

		// If on a new device with no tracked files and only initial commit, cleanly adopt remote branch
		commitCount, _ := e.Git.CommitCount(ctx)
		lastMsg, _ := e.Git.GetLastCommitMessage(ctx)
		isOnlyInitialCommit := commitCount <= 1 || strings.Contains(lastMsg, "Initial dotsynx repository setup")
		if len(e.Config.Tracked) == 0 && isOnlyInitialCommit && e.Git.HasRemoteBranch(ctx, e.Config.Branch) {
			logger.Info("Aligning local repo with remote origin/%s on new device", e.Config.Branch)
			_ = e.Git.ResetHard(ctx, "origin/"+e.Config.Branch)
			_ = e.Git.SetUpstream(ctx, e.Config.Branch)
			_ = e.Tracker.LoadRepoManifest()
			status, _ = e.Git.Status(ctx)
		}
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
	remoteBranchExists := status.RemoteURL != "" && (e.Git.HasRemoteBranch(ctx, e.Config.Branch) || e.Git.RemoteBranchExists(ctx, e.Config.Branch))
	pullAttempted := false
	pullSucceeded := false

	if remoteBranchExists {
		pullAttempted = true
		if err := e.Git.Pull(ctx, e.Config.Branch); err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "couldn't find remote ref") || strings.Contains(errStr, "no such ref") {
				logger.Info("Remote branch '%s' is empty or not yet created on remote. Proceeding with initial push.", e.Config.Branch)
				pullSucceeded = true
			} else if len(e.Config.Tracked) == 0 && e.Git.HasRemoteBranch(ctx, e.Config.Branch) {
				// Cleanly adopt remote on new device
				logger.Info("Adopting remote branch origin/%s after pull error on new device", e.Config.Branch)
				if resetErr := e.Git.ResetHard(ctx, "origin/"+e.Config.Branch); resetErr == nil {
					_ = e.Git.SetUpstream(ctx, e.Config.Branch)
					_ = e.Tracker.LoadRepoManifest()
					pullSucceeded = true
				} else {
					res.Errors = append(res.Errors, fmt.Sprintf("Pull failed: %v", err))
					logger.Warn("Git pull failed: %v", err)
				}
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
						pullSucceeded = true
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
			pullSucceeded = true
		}
	} else if status.RemoteURL != "" {
		logger.Info("Remote branch '%s' does not exist yet (brand new repository). Skipping pull and preparing initial push.", e.Config.Branch)
	}

	// Always ensure tracked list is synced with repository manifest
	_ = e.Tracker.LoadRepoManifest()

	// 5. Push if remote is configured, we have commits, and push is needed
	if status.RemoteURL != "" && e.Git.HasCommits(ctx) {
		// Only push if pull succeeded or was not needed (never push after a failed pull)
		if !pullAttempted || pullSucceeded {
			status, _ = e.Git.Status(ctx)
			needsPush := !remoteBranchExists || status.Ahead > 0 || hasLocalChanges
			if needsPush {
				if err := e.Git.Push(ctx, e.Config.Branch); err != nil {
					res.Errors = append(res.Errors, fmt.Sprintf("Push failed: %v", err))
					logger.Warn("Git push failed: %v", err)
				} else {
					res.Pushed = true
					logger.Info("Git push to origin/%s succeeded", e.Config.Branch)
				}
			}
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

// ResetToRemote forces the local repository and dotfiles to match remote origin exactly
func (e *Engine) ResetToRemote(ctx context.Context) (*SyncResult, error) {
	res := &SyncResult{
		Timestamp:    time.Now(),
		AppliedFiles: []string{},
		Errors:       []string{},
		Conflicts:    []ConflictItem{},
	}

	if e.Config.RepoURL == "" {
		err := fmt.Errorf("cannot reset to remote: no remote repository URL is configured")
		res.Errors = append(res.Errors, err.Error())
		res.Message = err.Error()
		return res, err
	}

	branch := e.Config.Branch
	if branch == "" {
		branch = "main"
	}

	logger.Sync("Initiating hard reset to remote origin/%s (storage: %s)", branch, e.Config.StorageDir)

	if err := e.EnsureRepoReady(ctx); err != nil {
		res.Errors = append(res.Errors, err.Error())
		res.Message = "Failed to initialize storage repository"
		return res, err
	}

	_ = e.Git.SetRemote(e.Config.RepoURL)

	if err := e.Git.Fetch(ctx); err != nil {
		errMsg := fmt.Sprintf("Git fetch failed: %v", err)
		res.Errors = append(res.Errors, errMsg)
		res.Message = errMsg
		logger.Error("%s", errMsg)
		return res, err
	}

	if !e.Git.HasRemoteBranch(ctx, branch) && !e.Git.RemoteBranchExists(ctx, branch) {
		errMsg := fmt.Sprintf("Remote branch 'origin/%s' does not exist on remote repository", branch)
		res.Errors = append(res.Errors, errMsg)
		res.Message = errMsg
		logger.Error("%s", errMsg)
		return res, fmt.Errorf("%s", errMsg)
	}

	if err := e.Git.ResetHard(ctx, "origin/"+branch); err != nil {
		errMsg := fmt.Sprintf("Git reset --hard failed: %v", err)
		res.Errors = append(res.Errors, errMsg)
		res.Message = errMsg
		logger.Error("%s", errMsg)
		return res, err
	}

	_ = e.Git.Clean(ctx)
	_ = e.Git.SetUpstream(ctx, branch)

	// Reload tracked list entirely from remote repository manifest
	e.Config.Tracked = []config.TrackedItem{}
	_ = e.Tracker.LoadRepoManifest()

	// Apply all symlinks to $HOME
	applied, applyErrs := e.Tracker.ApplyAll()
	res.AppliedFiles = applied
	res.Pulled = true
	for _, ae := range applyErrs {
		res.Errors = append(res.Errors, ae.Error())
	}

	if len(res.Errors) == 0 {
		res.Success = true
		res.Message = fmt.Sprintf("Reset hard to origin/%s succeeded (%d items linked)", branch, len(applied))
		logger.Sync("Reset succeeded: %s", res.Message)
	} else {
		res.Success = false
		res.Message = fmt.Sprintf("Reset completed with %d error(s)", len(res.Errors))
		logger.Error("Reset completed with errors: %s", strings.Join(res.Errors, " | "))
	}

	e.LastSync = res
	e.SyncLogs = append([]*SyncResult{res}, e.SyncLogs...)
	if len(e.SyncLogs) > 20 {
		e.SyncLogs = e.SyncLogs[:20]
	}

	return res, nil
}
