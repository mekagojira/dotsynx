package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SyncMode defines how and when dotsynx synchronizes
type SyncMode string

const (
	ModeManual SyncMode = "manual"
	ModeAuto   SyncMode = "auto"
	ModeBoot   SyncMode = "boot"
)

// ConflictStrategy defines the default behavior on conflicts
type ConflictStrategy string

const (
	StrategyInteractive ConflictStrategy = "interactive"
	StrategyOurs        ConflictStrategy = "ours"
	StrategyTheirs      ConflictStrategy = "theirs"
)

// TrackedItem represents a synced file or directory
type TrackedItem struct {
	// Source is the relative path from Home (e.g. ".zshrc", ".config/nvim")
	Path string `yaml:"path" json:"path"`
	// TargetInRepo is where it's stored in the git repo. If empty, matches Path
	RepoPath string `yaml:"repo_path,omitempty" json:"repo_path,omitempty"`
	// IsDir specifies if the item is a directory
	IsDir bool `yaml:"is_dir" json:"is_dir"`
	// Enabled determines if this item is currently synced
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Description provides context for the dotfile
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// Config is the main configuration structure for dotsynx
type Config struct {
	RepoURL          string           `yaml:"repo_url" json:"repo_url"`
	Branch           string           `yaml:"branch" json:"branch"`
	StorageDir       string           `yaml:"storage_dir" json:"storage_dir"`
	BackupDir        string           `yaml:"backup_dir" json:"backup_dir"`
	SyncMode         SyncMode         `yaml:"sync_mode" json:"sync_mode"`
	Interval         string           `yaml:"interval" json:"interval"` // e.g., "15m", "1h", or cron "0 * * * *"
	ConflictStrategy ConflictStrategy `yaml:"conflict_strategy" json:"conflict_strategy"`
	Theme            string           `yaml:"theme" json:"theme"` // "system", "dark", "light"
	Tracked          []TrackedItem    `yaml:"tracked" json:"tracked"`
	WebPort          int              `yaml:"web_port" json:"web_port"`
}

// DefaultConfigPath returns ~/.local/share/dotsynx/config.yaml
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	primary := filepath.Join(home, ".local", "share", "dotsynx", "config.yaml")
	legacy := filepath.Join(home, ".config", "dotsynx", "config.yaml")

	// If primary exists, always use primary
	if _, err := os.Stat(primary); err == nil {
		return primary
	}

	// If legacy exists, migrate content to primary
	if data, err := os.ReadFile(legacy); err == nil {
		_ = os.MkdirAll(filepath.Dir(primary), 0755)
		_ = os.WriteFile(primary, data, 0644)
		return primary
	}

	return primary
}

// DefaultStorageDir returns ~/.local/share/dotsynx/repo
func DefaultStorageDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "dotsynx", "repo")
}

// DefaultBackupDir returns ~/.local/share/dotsynx/backups
func DefaultBackupDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	return filepath.Join(home, ".local", "share", "dotsynx", "backups")
}

// NewDefault returns a standard default configuration
func NewDefault() *Config {
	return &Config{
		RepoURL:          "",
		Branch:           "main",
		StorageDir:       DefaultStorageDir(),
		BackupDir:        DefaultBackupDir(),
		SyncMode:         ModeManual,
		Interval:         "15m",
		ConflictStrategy: StrategyInteractive,
		Theme:            "system",
		Tracked:          []TrackedItem{},
		WebPort:          18942,
	}
}

// Load reads the config from the given path or default if empty
func Load(customPath string) (*Config, error) {
	path := customPath
	if path == "" {
		path = DefaultConfigPath()
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		cfg := NewDefault()
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := NewDefault()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Expand paths
	cfg.StorageDir = expandHome(cfg.StorageDir)
	cfg.BackupDir = expandHome(cfg.BackupDir)

	return cfg, nil
}

// Save writes the config to disk atomically
func (c *Config) Save(customPath string) error {
	path := customPath
	if path == "" {
		path = DefaultConfigPath()
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	tmpFile := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary config: %w", err)
	}

	if err := os.Rename(tmpFile, path); err != nil {
		_ = os.Remove(tmpFile)
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

// NormalizePath normalizes a user path relative to home
func NormalizePath(inputPath string) (relHome string, absPath string, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	cleaned := strings.TrimSpace(inputPath)
	var abs string

	if filepath.IsAbs(cleaned) {
		abs = filepath.Clean(cleaned)
	} else if strings.HasPrefix(cleaned, "~") {
		abs = filepath.Clean(filepath.Join(home, strings.TrimPrefix(cleaned, "~")))
	} else {
		// Relative path: In a dotfiles manager, all relative paths are relative to $HOME
		abs = filepath.Clean(filepath.Join(home, cleaned))
	}

	rel, err := filepath.Rel(home, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", abs, fmt.Errorf("path %s must be inside home directory %s", abs, home)
	}

	return rel, abs, nil
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") || p == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
