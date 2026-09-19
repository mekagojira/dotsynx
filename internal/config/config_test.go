package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndSave(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "dotsynx-config-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	cfgFile := filepath.Join(tmpDir, "config.yaml")

	// Load non-existent should return default
	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cfg.RepoURL = "git@github.com:test/dotfiles.git"
	cfg.SyncMode = ModeAuto
	cfg.Interval = "30m"
	cfg.Tracked = []TrackedItem{
		{
			Path:        ".zshrc",
			IsDir:       false,
			Enabled:     true,
			Description: "Zsh configuration",
		},
	}

	if err := cfg.Save(cfgFile); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// Reload
	loaded, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("failed to reload config: %v", err)
	}

	if loaded.RepoURL != cfg.RepoURL {
		t.Errorf("expected RepoURL %s, got %s", cfg.RepoURL, loaded.RepoURL)
	}
	if loaded.SyncMode != ModeAuto {
		t.Errorf("expected SyncMode %s, got %s", ModeAuto, loaded.SyncMode)
	}
	if len(loaded.Tracked) != 1 || loaded.Tracked[0].Path != ".zshrc" {
		t.Errorf("tracked items mismatch")
	}
}

func TestNormalizePath(t *testing.T) {
	home, _ := os.UserHomeDir()

	// 1. Tilde path
	rel, abs, err := NormalizePath("~/.config/nvim")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedRel := filepath.Join(".config", "nvim")
	if rel != expectedRel {
		t.Errorf("expected rel %s, got %s", expectedRel, rel)
	}
	expectedAbs := filepath.Join(home, ".config", "nvim")
	if abs != expectedAbs {
		t.Errorf("expected abs %s, got %s", expectedAbs, abs)
	}

	// 2. Relative path without tilde (e.g. from UI or suggestions: ".config/fish")
	rel2, abs2, err := NormalizePath(".config/fish")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedRel2 := filepath.Join(".config", "fish")
	if rel2 != expectedRel2 {
		t.Errorf("expected rel %s, got %s", expectedRel2, rel2)
	}
	expectedAbs2 := filepath.Join(home, ".config", "fish")
	if abs2 != expectedAbs2 {
		t.Errorf("expected abs %s, got %s", expectedAbs2, abs2)
	}

	// 3. Absolute path in home
	rel3, abs3, err := NormalizePath(expectedAbs2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel3 != expectedRel2 || abs3 != expectedAbs2 {
		t.Errorf("absolute path failed")
	}
}
