# ⚡ dotsynx

A modern, fast, and automated dotfile synchronization agent for **macOS** and **Linux**, written in **Go** and **React + Vite** in a unified monorepo.

`dotsynx` compiles into a **single, zero-dependency binary** that embeds the complete webview and delivers an interactive TUI, CLI, background daemon, and OS service integration (`launchd` on macOS, `systemd` on Linux).

---

## ✨ Features

- **Single Portable Binary:** React + Vite webview assets are embedded into the Go binary with `//go:embed`.
- **Modern Terminal UI (TUI):** Built with Bubble Tea & Lip Gloss, adapting automatically to your terminal's light or dark mode.
- **Modern Web UI (GUI):** Runs on `127.0.0.1:18942`, featuring dashboard statistics, light/dark mode (following OS or manual), visual git status, and interactive file pickers.
- **Git-Backed Multi-Machine Sync:** Synchronize across unlimited machines using any Git host (GitHub, GitLab, self-hosted, public or private repos with SSH or token auth).
- **Dotfile Suggestion Engine:** Automatically scans your machine and suggests standard dotfiles (Zsh, Bash, Fish, Neovim, Zed, Helix, VS Code, Tmux, Starship, Kitty, Alacritty, Window Managers, etc.) while warning against sensitive files (`id_rsa`, `.env`, tokens).
- **Intuitive File & Directory Browser:** Explore your `$HOME` directory directly within the TUI or Web UI to track arbitrary files or folders with one click.
- **3 Synchronization Modes:**
  1. **Manual:** Immediate sync on demand (`dotsynx sync`, or via TUI/Web UI).
  2. **Auto:** Periodic background daemon running on an interval (e.g. `15m`, `1h`) or cron schedule.
  3. **At Boot / Login:** OS-level service (`launchd` on macOS, `systemd` on Linux) synchronizing dotfiles upon system startup or login.
- **Conflict Resolution & Safety Backups:**
  - Automated timestamped snapshots in `~/.local/share/dotsynx/backups/` before any file is touched or symlinked.
  - Visual side-by-side / unified diff viewer.
  - One-click resolution strategies (`ours` / `theirs` / `interactive`).

---

## 🚀 Quick Start

### 1. Build

```bash
make build
```

This builds the React frontend and compiles the single `bin/dotsynx` binary.

### 2. Run TUI

Simply run `dotsynx` without arguments:

```bash
./bin/dotsynx
```

### 3. Run Webview (GUI)

Launch the embedded local web interface and automatically open your default browser:

```bash
./bin/dotsynx ui
```

### 4. Basic CLI Usage

# Basic CLI Usage

```bash
# Show status of daemon, git, service, and tracked items
dotsynx status

# Start the background daemon (also serves Web UI at http://127.0.0.1:18942)
dotsynx start

# Restart the daemon
dotsynx restart

# Stop the daemon
dotsynx stop

# Open the Web UI in your browser (connects to running daemon or starts UI)
dotsynx ui

# View server and sync logs
dotsynx logs
dotsynx logs -f                # Follow live log stream
dotsynx logs -n 100            # Show last 100 lines
dotsynx logs --since 30m       # Show logs from last 30 minutes
dotsynx logs --clear           # Clear log file

# Start the background daemon
dotsynx start

# Stop the background daemon
dotsynx stop

# Install as an OS boot service (launchd on macOS / systemd on Linux)
dotsynx service install
```

---

## 🛠️ Monorepo Structure

```
dotsync/
├── cmd/
│   └── dotsynx/
│       └── main.go                 # Cobra CLI router (TUI, UI, start, stop, sync, service)
├── internal/
│   ├── config/                     # Configuration loader (~/.config/dotsynx/config.yaml)
│   ├── core/
│   │   ├── engine.go               # Sync orchestrator (git pull, diff, apply, commit, push)
│   │   ├── tracker.go              # Symlink engine & link state inspector
│   │   ├── scanner.go              # Dotfile suggestion catalog & security filter
│   │   ├── conflict.go             # Conflict detection & checkout strategies
│   │   └── backup.go               # Timestamped backup manager
│   ├── git/                        # Git client wrapper
│   ├── daemon/                     # Background daemon & scheduler
│   ├── service/                    # macOS launchd and Linux systemd service managers
│   ├── tui/                        # Bubble Tea & Lip Gloss terminal interface
│   └── webserver/                  # Local REST API & embedded asset server
├── web/                            # React + Vite + Tailwind CSS frontend
│   ├── src/
│   │   ├── components/             # Dashboard, TrackedFiles, Suggestions, Browser, Conflicts
│   │   ├── App.tsx
│   │   └── api.ts
│   └── dist/                       # Compiled frontend assets
├── web_embed.go                    # Go embed directive embedding web/dist
├── Makefile                        # Unified build commands
├── go.mod
└── go.sum
```

---

## ⚙️ Configuration (`~/.config/dotsynx/config.yaml`)

```yaml
repo_url: "git@github.com:username/dotfiles.git"
branch: "main"
storage_dir: "~/.local/share/dotsynx/repo"
backup_dir: "~/.local/share/dotsynx/backups"
sync_mode: "auto" # manual, auto, boot
interval: "15m" # 15m, 1h, or cron expression
conflict_strategy: "interactive" # interactive, ours, theirs
theme: "system" # system, dark, light
web_port: 18942
tracked:
  - path: ".zshrc"
    is_dir: false
    enabled: true
    description: "Zsh configuration"
  - path: ".config/nvim"
    is_dir: true
    enabled: true
    description: "Neovim configuration"
```

---

## 🛡️ Conflict & Backup Mechanics

1. **Before any file modification**, a snapshot is saved to `~/.local/share/dotsynx/backups/<YYYYMMDD-HHMMSS>/`.
2. Divergence between machines is detected via Git branch comparison (`ahead` / `behind`) and local dirty state.
3. If conflicts occur:
   - In **TUI** (Tab 5) or **Web UI** (Conflicts Tab), review the color-coded diff.
   - Choose **Keep Local (Ours)** or **Use Remote (Theirs)** with a single click or keystroke.

---

## 📄 License

MIT
