package core

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Suggestion represents a suggested dotfile or directory found on the machine
type Suggestion struct {
	Path        string `json:"path"`         // relative to home (e.g. ".zshrc", ".config/nvim")
	AbsPath     string `json:"abs_path"`     // full system path
	IsDir       bool   `json:"is_dir"`       // directory or file
	Category    string `json:"category"`     // shell, editor, terminal, git, tool, etc.
	Description string `json:"description"`  // context for user
	Exists      bool   `json:"exists"`       // currently exists on the machine
	IsSensitive bool   `json:"is_sensitive"` // whether it matches sensitive pattern (ssh keys, env, secrets)
	SizeHuman   string `json:"size_human"`   // e.g. "4.2 KB"
}

// CatalogItem defines known popular dotfiles
type CatalogItem struct {
	Path        string
	IsDir       bool
	Category    string
	Description string
}

var KnownCatalog = []CatalogItem{
	// Shells
	{Path: ".zshrc", IsDir: false, Category: "Shell", Description: "Zsh configuration file"},
	{Path: ".zshenv", IsDir: false, Category: "Shell", Description: "Zsh environment variables"},
	{Path: ".zprofile", IsDir: false, Category: "Shell", Description: "Zsh login profile"},
	{Path: ".bashrc", IsDir: false, Category: "Shell", Description: "Bash interactive configuration"},
	{Path: ".bash_profile", IsDir: false, Category: "Shell", Description: "Bash login configuration"},
	{Path: ".config/fish", IsDir: true, Category: "Shell", Description: "Fish shell configuration and functions"},
	{Path: ".profile", IsDir: false, Category: "Shell", Description: "POSIX shell profile"},

	// Git
	{Path: ".gitconfig", IsDir: false, Category: "Git", Description: "Git global user configuration"},
	{Path: ".gitignore_global", IsDir: false, Category: "Git", Description: "Git global ignore rules"},
	{Path: ".config/git", IsDir: true, Category: "Git", Description: "Git XDG configuration directory"},

	// Editors
	{Path: ".config/nvim", IsDir: true, Category: "Editor", Description: "Neovim configuration, plugins, and keymaps"},
	{Path: ".vimrc", IsDir: false, Category: "Editor", Description: "Vim traditional configuration"},
	{Path: ".config/helix", IsDir: true, Category: "Editor", Description: "Helix editor configuration and themes"},
	{Path: ".config/zed", IsDir: true, Category: "Editor", Description: "Zed editor settings, keymaps, and themes"},
	{Path: ".config/Code/User/settings.json", IsDir: false, Category: "Editor", Description: "VS Code user settings (Linux)"},
	{Path: "Library/Application Support/Code/User/settings.json", IsDir: false, Category: "Editor", Description: "VS Code user settings (macOS)"},
	{Path: "Library/Application Support/Code/User/keybindings.json", IsDir: false, Category: "Editor", Description: "VS Code keybindings (macOS)"},

	// Terminals & Multiplexers
	{Path: ".tmux.conf", IsDir: false, Category: "Terminal", Description: "Tmux multiplexer configuration"},
	{Path: ".config/tmux", IsDir: true, Category: "Terminal", Description: "Tmux configuration directory"},
	{Path: ".config/alacritty", IsDir: true, Category: "Terminal", Description: "Alacritty GPU terminal emulator config"},
	{Path: ".config/kitty", IsDir: true, Category: "Terminal", Description: "Kitty terminal emulator config"},
	{Path: ".config/wezterm", IsDir: true, Category: "Terminal", Description: "WezTerm terminal emulator configuration"},
	{Path: ".config/ghostty", IsDir: true, Category: "Terminal", Description: "Ghostty terminal emulator configuration"},

	// Prompt & Tools
	{Path: ".config/starship.toml", IsDir: false, Category: "Prompt", Description: "Starship cross-shell prompt configuration"},
	{Path: ".p10k.zsh", IsDir: false, Category: "Prompt", Description: "Powerlevel10k prompt configuration"},
	{Path: ".config/bat", IsDir: true, Category: "Tool", Description: "Bat syntax highlighter config and themes"},
	{Path: ".config/gh", IsDir: true, Category: "Tool", Description: "GitHub CLI configuration"},
	{Path: ".config/ripgrep", IsDir: true, Category: "Tool", Description: "Ripgrep configuration"},
	{Path: ".config/aerospace", IsDir: true, Category: "Window Manager", Description: "AeroSpace tiling window manager (macOS)"},
	{Path: ".config/yabai", IsDir: true, Category: "Window Manager", Description: "Yabai tiling window manager (macOS)"},
	{Path: ".config/skhd", IsDir: true, Category: "Window Manager", Description: "Simple hotkey daemon (macOS)"},
	{Path: ".config/i3", IsDir: true, Category: "Window Manager", Description: "i3 window manager (Linux)"},
	{Path: ".config/sway", IsDir: true, Category: "Window Manager", Description: "Sway Wayland compositor (Linux)"},
	{Path: ".config/hypr", IsDir: true, Category: "Window Manager", Description: "Hyprland Wayland compositor (Linux)"},
}

// Sensitive file patterns that should NOT be synced or warned heavily
var sensitiveRegex = regexp.MustCompile(`(?i)(id_rsa|id_ed25519|id_ecdsa|id_dsa|\.pem|\.key|\.crt|\.p12|\.pfx|\.env|\.netrc|credentials|password|secret|token|\.gnupg|history)`)

// IsSensitive checks if a file path might contain secrets
func IsSensitive(path string) bool {
	return sensitiveRegex.MatchString(filepath.Base(path)) ||
		strings.Contains(path, ".ssh/") ||
		strings.Contains(path, ".gnupg/") ||
		strings.Contains(path, "keyrings")
}

// ScanSuggestions checks the machine against KnownCatalog and scans ~/.config for additional dotfiles
func ScanSuggestions() ([]Suggestion, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var suggestions []Suggestion

	// 1. Check known catalog
	for _, item := range KnownCatalog {
		abs := filepath.Join(home, item.Path)
		info, err := os.Stat(abs)
		exists := err == nil

		if exists {
			seen[item.Path] = true
			suggestions = append(suggestions, Suggestion{
				Path:        item.Path,
				AbsPath:     abs,
				IsDir:       info.IsDir(),
				Category:    item.Category,
				Description: item.Description,
				Exists:      true,
				IsSensitive: IsSensitive(item.Path),
			})
		}
	}

	// 2. Discover user entries inside ~/.config
	configDir := filepath.Join(home, ".config")
	if entries, err := os.ReadDir(configDir); err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") {
				continue
			}
			rel := filepath.Join(".config", name)
			if seen[rel] {
				continue
			}
			abs := filepath.Join(configDir, name)
			isSens := IsSensitive(rel)
			suggestions = append(suggestions, Suggestion{
				Path:        rel,
				AbsPath:     abs,
				IsDir:       entry.IsDir(),
				Category:    "App Config",
				Description: name + " configuration",
				Exists:      true,
				IsSensitive: isSens,
			})
			seen[rel] = true
		}
	}

	return suggestions, nil
}
