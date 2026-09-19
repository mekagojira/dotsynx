package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Client wraps git operations on a repository directory
type Client struct {
	RepoDir string
}

// RepoStatus holds git repository sync and branch status
type RepoStatus struct {
	Initialized bool
	Branch      string
	RemoteURL   string
	Ahead       int
	Behind      int
	Clean       bool
	HasConflict bool
	Modified    []string
	Untracked   []string
}

// NewClient returns a new Git client pointing to repoDir
func NewClient(repoDir string) *Client {
	return &Client{
		RepoDir: repoDir,
	}
}

func gitEnv() []string {
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o StrictHostKeyChecking=accept-new",
		"GIT_EDITOR=true",
		"EDITOR=true",
	)
}

func (c *Client) buildCmd(ctx context.Context, dir string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	cmd.Stdin = bytes.NewReader(nil) // Never block waiting on terminal stdin
	return cmd
}

func (c *Client) CleanStaleLock() {
	lockPath := filepath.Join(c.RepoDir, ".git", "index.lock")
	if info, err := os.Stat(lockPath); err == nil {
		// If lock exists and is older than 2 seconds, remove it to unblock git
		if time.Since(info.ModTime()) > 2*time.Second {
			_ = os.Remove(lockPath)
		}
	}
}

// run executes a git command in RepoDir
func (c *Client) run(ctx context.Context, args ...string) (string, error) {
	c.CleanStaleLock()
	cmd := c.buildCmd(ctx, c.RepoDir, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w (stderr: %s)", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}

	return strings.TrimSpace(stdout.String()), nil
}

// IsInstalled checks if git is available in PATH
func IsInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// Exists checks if RepoDir contains a .git directory
func (c *Client) Exists() bool {
	info, err := os.Stat(c.RepoDir + "/.git")
	return err == nil && info.IsDir()
}

// Init creates a new git repository in RepoDir
func (c *Client) Init(defaultBranch string) error {
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if err := os.MkdirAll(c.RepoDir, 0755); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := c.run(ctx, "init", "-b", defaultBranch)
	if err != nil {
		_, err = c.run(ctx, "init")
	}
	// Ensure HEAD points to default branch
	headFile := filepath.Join(c.RepoDir, ".git", "HEAD")
	_ = os.WriteFile(headFile, []byte("ref: refs/heads/"+defaultBranch+"\n"), 0644)
	return err
}

// Clone clones a remote repository into RepoDir
func (c *Client) Clone(ctx context.Context, remoteURL string, branch string) error {
	if err := os.MkdirAll(filepath.Dir(c.RepoDir), 0755); err != nil {
		return err
	}

	args := []string{"clone"}
	if branch != "" {
		args = append(args, "-b", branch)
	}
	args = append(args, remoteURL, c.RepoDir)

	cmd := c.buildCmd(ctx, filepath.Dir(c.RepoDir), args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to clone repo: %w (stderr: %s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

// SetRemote sets or updates the 'origin' remote URL
func (c *Client) SetRemote(remoteURL string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := c.run(ctx, "remote", "get-url", "origin")
	if err != nil {
		_, err = c.run(ctx, "remote", "add", "origin", remoteURL)
		return err
	}
	_, err = c.run(ctx, "remote", "set-url", "origin", remoteURL)
	return err
}

// GetRemote returns the URL of origin
func (c *Client) GetRemote() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.run(ctx, "remote", "get-url", "origin")
}

// HasCommits checks if the repository has at least one commit (HEAD exists)
func (c *Client) HasCommits(ctx context.Context) bool {
	_, err := c.run(ctx, "rev-parse", "--verify", "HEAD")
	return err == nil
}

// CurrentBranch returns the currently checked-out branch
func (c *Client) CurrentBranch() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	branch, err := c.run(ctx, "branch", "--show-current")
	if err != nil || branch == "" || branch == ".invalid" {
		// Read directly from .git/HEAD for unborn branches or broken refs
		headFile := filepath.Join(c.RepoDir, ".git", "HEAD")
		if data, err := os.ReadFile(headFile); err == nil {
			str := strings.TrimSpace(string(data))
			if strings.HasPrefix(str, "ref: refs/heads/") {
				clean := strings.TrimPrefix(str, "ref: refs/heads/")
				if clean != "" && clean != ".invalid" {
					return clean, nil
				}
			}
		}
		return "main", nil
	}
	return branch, nil
}

// HasRemoteBranch checks if origin/<branch> exists locally in git refs
func (c *Client) HasRemoteBranch(ctx context.Context, branch string) bool {
	if branch == "" {
		branch = "main"
	}
	_, err := c.run(ctx, "rev-parse", "--verify", "origin/"+branch)
	return err == nil
}

// RemoteBranchExists queries the remote via ls-remote to check if the branch exists on remote
func (c *Client) RemoteBranchExists(ctx context.Context, branch string) bool {
	if branch == "" {
		branch = "main"
	}
	out, err := c.run(ctx, "ls-remote", "--heads", "origin", branch)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) != ""
}

// RemoteDefaultBranch queries origin for its default branch (e.g. main or master)
func (c *Client) RemoteDefaultBranch(ctx context.Context) string {
	out, err := c.run(ctx, "ls-remote", "--symref", "origin", "HEAD")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ref: refs/heads/") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				branch := strings.TrimPrefix(parts[0], "ref: refs/heads/")
				if branch != "" {
					return branch
				}
			}
		}
	}
	return ""
}

// ResetHard resets the current branch to a target ref
func (c *Client) ResetHard(ctx context.Context, target string) error {
	_, err := c.run(ctx, "reset", "--hard", target)
	return err
}

// Clean removes untracked files and directories from the working tree
func (c *Client) Clean(ctx context.Context) error {
	_, err := c.run(ctx, "clean", "-fd")
	return err
}

// SetUpstream sets the upstream tracking branch for branch
func (c *Client) SetUpstream(ctx context.Context, branch string) error {
	if branch == "" {
		branch = "main"
	}
	_, err := c.run(ctx, "branch", "--set-upstream-to=origin/"+branch, branch)
	return err
}

// CommitCount returns the number of commits in HEAD
func (c *Client) CommitCount(ctx context.Context) (int, error) {
	out, err := c.run(ctx, "rev-list", "--count", "HEAD")
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(out))
}

// GetLastCommitMessage returns the commit message of HEAD
func (c *Client) GetLastCommitMessage(ctx context.Context) (string, error) {
	return c.run(ctx, "log", "-1", "--pretty=%B")
}

// Fetch fetches from origin
func (c *Client) Fetch(ctx context.Context) error {
	_, err := c.run(ctx, "fetch", "origin")
	return err
}

// Status inspects the repo and remote state
func (c *Client) Status(ctx context.Context) (*RepoStatus, error) {
	if !c.Exists() {
		return &RepoStatus{Initialized: false}, nil
	}

	branch, _ := c.CurrentBranch()
	remoteURL, _ := c.GetRemote()

	status := &RepoStatus{
		Initialized: true,
		Branch:      branch,
		RemoteURL:   remoteURL,
		Clean:       true,
		Modified:    []string{},
		Untracked:   []string{},
	}

	// porcelain status
	out, err := c.run(ctx, "status", "--porcelain")
	if err == nil && out != "" {
		lines := strings.Split(out, "\n")
		for _, line := range lines {
			if len(line) < 3 {
				continue
			}
			status.Clean = false
			code := line[:2]
			file := strings.TrimSpace(line[3:])

			if strings.Contains(code, "U") || code == "DD" || code == "AA" {
				status.HasConflict = true
			}
			if strings.HasPrefix(code, "??") {
				status.Untracked = append(status.Untracked, file)
			} else {
				status.Modified = append(status.Modified, file)
			}
		}
	}

	// Ahead / behind count against origin
	if remoteURL != "" && branch != "" {
		revRange := fmt.Sprintf("%s...origin/%s", branch, branch)
		counts, err := c.run(ctx, "rev-list", "--left-right", "--count", revRange)
		if err == nil {
			parts := strings.Fields(counts)
			if len(parts) == 2 {
				ahead, _ := strconv.Atoi(parts[0])
				behind, _ := strconv.Atoi(parts[1])
				status.Ahead = ahead
				status.Behind = behind
			}
		}
	}

	return status, nil
}

// Add stages files or all changes
func (c *Client) Add(ctx context.Context, paths ...string) error {
	if len(paths) == 0 {
		paths = []string{"."}
	}
	args := append([]string{"add"}, paths...)
	_, err := c.run(ctx, args...)
	return err
}

// Commit creates a commit
func (c *Client) Commit(ctx context.Context, message string) error {
	// Ensure author info exists for non-configured environments
	_, _ = c.run(ctx, "config", "user.name")
	_, err := c.run(ctx, "config", "user.email")
	if err != nil {
		_, _ = c.run(ctx, "config", "user.name", "dotsynx agent")
		_, _ = c.run(ctx, "config", "user.email", "dotsynx@localhost")
	}

	_, err = c.run(ctx, "commit", "-m", message)
	return err
}

// Push pushes the current branch to origin
func (c *Client) Push(ctx context.Context, branch string) error {
	remote, err := c.GetRemote()
	if err != nil || remote == "" {
		return errors.New("remote 'origin' is not configured")
	}
	if branch == "" {
		var err error
		branch, err = c.CurrentBranch()
		if err != nil {
			branch = "main"
		}
	}
	_, err = c.run(ctx, "push", "-u", "origin", branch)
	return err
}

// Pull pulls changes from origin
func (c *Client) Pull(ctx context.Context, branch string) error {
	remote, err := c.GetRemote()
	if err != nil || remote == "" {
		return errors.New("remote 'origin' is not configured")
	}
	if branch == "" {
		var err error
		branch, err = c.CurrentBranch()
		if err != nil {
			branch = "main"
		}
	}
	_, err = c.run(ctx, "pull", "--rebase=false", "--allow-unrelated-histories", "--no-edit", "origin", branch)
	return err
}

// Diff gets the diff of a specific file or all files
func (c *Client) Diff(ctx context.Context, file string) (string, error) {
	args := []string{"diff"}
	if file != "" {
		args = append(args, "--", file)
	}
	return c.run(ctx, args...)
}

// CheckoutStrategy resolves a file conflict using either 'ours' or 'theirs'
func (c *Client) CheckoutStrategy(ctx context.Context, file string, strategy string) error {
	var flag string
	switch strategy {
	case "ours":
		flag = "--ours"
	case "theirs":
		flag = "--theirs"
	default:
		return errors.New("invalid strategy: must be 'ours' or 'theirs'")
	}

	_, err := c.run(ctx, "checkout", flag, "--", file)
	if err != nil {
		return err
	}
	// Mark as resolved
	return c.Add(ctx, file)
}
