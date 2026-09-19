package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"dotsynx"
	"dotsynx/internal/config"
	"dotsynx/internal/core"
	"dotsynx/internal/daemon"
	"dotsynx/internal/logger"
	"dotsynx/internal/service"
	"dotsynx/internal/tui"
	"dotsynx/internal/webserver"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

var (
	cfgPath    string
	foreground bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "dotsynx",
		Short: "dotsynx: Multi-device dotfile sync agent",
		Long: `dotsynx is a modern, automated dotfile synchronization tool for macOS and Linux.
It supports Git-backed synchronization, public and private repositories,
conflict detection, background daemon/services, interactive TUI, and embedded Web UI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// When invoked without arguments, launch the interactive TUI
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			app, err := tui.NewApp(cfg)
			if err != nil {
				return err
			}

			p := tea.NewProgram(app, tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	rootCmd.PersistentFlags().StringVarP(&cfgPath, "config", "c", "", "Path to custom config file (default ~/.local/share/dotsynx/config.yaml)")

	// Subcommands
	rootCmd.AddCommand(newUICommand())
	rootCmd.AddCommand(newStartCommand())
	rootCmd.AddCommand(newStopCommand())
	rootCmd.AddCommand(newRestartCommand())
	rootCmd.AddCommand(newSyncCommand())
	rootCmd.AddCommand(newStatusCommand())
	rootCmd.AddCommand(newAddCommand())
	rootCmd.AddCommand(newRemoveCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newSuggestCommand())
	rootCmd.AddCommand(newServiceCommand())
	rootCmd.AddCommand(newLogsCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// dotsynx ui
func newUICommand() *cobra.Command {
	var port int
	var noOpen bool

	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Launch the webview interface in your browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if port > 0 {
				cfg.WebPort = port
			}
			webPort := cfg.WebPort
			if webPort == 0 {
				webPort = 18942
			}
			targetURL := fmt.Sprintf("http://127.0.0.1:%d", webPort)

			// If daemon or webserver is already running, open the browser directly
			running, pid := daemon.CheckRunning()
			if running || isPortInUse(webPort) {
				if pid > 0 {
					fmt.Printf("🌐 dotsynx daemon is active (PID %d)\n", pid)
				} else {
					fmt.Printf("🌐 dotsynx server is active on port %d\n", webPort)
				}
				fmt.Printf("🚀 Opening Web UI at %s\n", targetURL)
				if !noOpen {
					_ = browser.OpenURL(targetURL)
				}
				return nil
			}

			distFS, err := dotsynx.GetWebDistFS()
			if err != nil {
				fmt.Printf("Warning: Embedded assets not found: %v\n", err)
			}

			srv, err := webserver.NewServer(cfg, distFS)
			if err != nil {
				return err
			}

			fmt.Printf("🌐 Starting dotsynx Web UI at %s\n", targetURL)

			if !noOpen {
				go func() {
					time.Sleep(200 * time.Millisecond)
					_ = browser.OpenURL(targetURL)
				}()
			}

			return srv.Start()
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 18942, "Port for web server")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "Do not open browser automatically")
	return cmd
}

// dotsynx start
func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the background sync daemon and Web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			if foreground {
				fmt.Println("🚀 Starting dotsynx daemon in foreground...")
				d, err := daemon.NewDaemon(cfg)
				if err != nil {
					return err
				}

				distFS, _ := dotsynx.GetWebDistFS()
				srv, _ := webserver.NewServer(cfg, distFS)
				if srv != nil {
					d.SetHandler(srv.Handler())
				}

				return d.Start(context.Background())
			}

			// Check if running already
			if running, pid := daemon.CheckRunning(); running {
				port := cfg.WebPort
				if port == 0 {
					port = 18942
				}
				fmt.Printf("⚠️  dotsynx daemon is already running (PID %d)\n", pid)
				fmt.Printf("🌐 Web UI available at http://127.0.0.1:%d\n", port)
				return nil
			}

			return startDaemonProcess(cfgPath)
		},
	}

	cmd.Flags().BoolVarP(&foreground, "foreground", "f", false, "Run daemon in foreground")
	return cmd
}

// dotsynx restart
func newRestartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "restart",
		Short: "Restart the background sync daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if running, _ := daemon.CheckRunning(); running {
				fmt.Println("🛑 Stopping running dotsynx daemon...")
				_ = daemon.StopRunning()
				time.Sleep(300 * time.Millisecond)
			}
			return startDaemonProcess(cfgPath)
		},
	}
}

func startDaemonProcess(customCfgPath string) error {
	cfg, err := config.Load(customCfgPath)
	if err != nil {
		return err
	}

	binPath, err := os.Executable()
	if err != nil {
		return err
	}

	bgCmd := exec.Command(binPath, "start", "--foreground")
	if customCfgPath != "" {
		bgCmd.Args = append(bgCmd.Args, "--config", customCfgPath)
	}
	bgCmd.Dir = "/"
	bgCmd.Env = os.Environ()

	// Redirect daemon logs
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".local", "share", "dotsynx", "daemon.log")
	_ = os.MkdirAll(filepath.Dir(logPath), 0755)
	if logF, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
		bgCmd.Stdout = logF
		bgCmd.Stderr = logF
	}

	if err := bgCmd.Start(); err != nil {
		return fmt.Errorf("failed to spawn daemon process: %w", err)
	}

	// Wait up to 1.5s for process to initialize PID and bind port
	for i := 0; i < 15; i++ {
		time.Sleep(100 * time.Millisecond)
		if running, _ := daemon.CheckRunning(); running {
			break
		}
	}

	port := cfg.WebPort
	if port == 0 {
		port = 18942
	}

	fmt.Printf("✅ dotsynx daemon started successfully (PID %d)\n", bgCmd.Process.Pid)
	fmt.Printf("🌐 Web UI available at http://127.0.0.1:%d\n", port)
	return nil
}

func isPortInUse(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

// dotsynx stop
func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background sync daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := daemon.StopRunning(); err != nil {
				fmt.Printf("Notice: %v\n", err)
				return nil
			}
			fmt.Println("🛑 dotsynx daemon stopped.")
			return nil
		},
	}
}

// dotsynx sync
func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Run an immediate dotfile synchronization",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			fmt.Println("⚡ Synchronizing dotfiles...")

			// If daemon is running, request daemon to sync over HTTP so state stays unified
			if running, _ := daemon.CheckRunning(); running {
				res, err := daemon.RequestSyncViaHTTP(cfg.WebPort)
				if err != nil {
					fmt.Printf("Daemon sync request failed (%v); falling back to direct sync...\n", err)
				} else {
					printSyncResult(res)
					return nil
				}
			}

			// Direct sync
			eng, err := core.NewEngine(cfg)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()

			res, err := eng.Sync(ctx)
			printSyncResult(res)
			return err
		},
	}
}

func printSyncResult(res *core.SyncResult) {
	if res == nil {
		fmt.Println("❌ No result returned")
		return
	}

	if res.Success {
		fmt.Printf("✅ %s\n", res.Message)
	} else {
		fmt.Printf("⚠️  %s\n", res.Message)
	}

	if len(res.Conflicts) > 0 {
		fmt.Printf("\n⚠️  Conflicts detected in %d file(s):\n", len(res.Conflicts))
		for _, c := range res.Conflicts {
			fmt.Printf("   - %s\n", c.Path)
		}
		fmt.Println("Run 'dotsynx' (TUI) or 'dotsynx ui' (Web) to inspect diffs and resolve.")
	}

	if len(res.Errors) > 0 {
		fmt.Println("\nErrors encountered:")
		for _, e := range res.Errors {
			fmt.Printf("   - %s\n", e)
		}
	}
}

// dotsynx status
func newStatusCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current synchronization and daemon status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			eng, err := core.NewEngine(cfg)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			fmt.Println("═══════════════════════════════════════════════")
			fmt.Println("              DOTSYNX STATUS                   ")
			fmt.Println("═══════════════════════════════════════════════")

			// Daemon status
			running, pid := daemon.CheckRunning()
			if running {
				fmt.Printf("🤖 Daemon:       RUNNING (PID %d)\n", pid)
			} else {
				fmt.Println("🤖 Daemon:       STOPPED")
			}

			// OS Service
			if mgr, err := service.GetManager(); err == nil {
				st, _ := mgr.Status()
				if st != nil && st.Installed {
					if st.Running {
						fmt.Printf("⚙️  OS Service:   ACTIVE (%s, PID %d)\n", st.Platform, st.PID)
					} else {
						fmt.Printf("⚙️  OS Service:   INSTALLED (STOPPED) - %s\n", st.Platform)
					}
				} else {
					fmt.Println("⚙️  OS Service:   NOT INSTALLED")
				}
			}

			// Sync settings
			fmt.Printf("⏱️  Sync Mode:    %s", strings.ToUpper(string(cfg.SyncMode)))
			if cfg.SyncMode == config.ModeAuto {
				fmt.Printf(" (interval: %s)\n", cfg.Interval)
			} else {
				fmt.Println()
			}

			// Git status
			repoURL := cfg.RepoURL
			if repoURL == "" {
				repoURL = "Local only (no remote git set)"
			}
			fmt.Printf("🔗 Remote Repo:  %s\n", repoURL)
			fmt.Printf("🌿 Branch:       %s\n", cfg.Branch)
			fmt.Printf("📦 Storage Dir:  %s\n", cfg.StorageDir)

			status, err := eng.Git.Status(ctx)
			if err == nil && status.Initialized {
				fmt.Printf("📊 Git Status:   Ahead: %d, Behind: %d, Modified: %d\n", status.Ahead, status.Behind, len(status.Modified))
			}

			// Tracked dotfiles
			fmt.Printf("📁 Tracked:      %d items configured\n", len(cfg.Tracked))
			for _, it := range cfg.Tracked {
				state := eng.Tracker.InspectItem(it)
				statusBadge := fmt.Sprintf("[%s]", strings.ToUpper(string(state.LinkStatus)))
				fmt.Printf("   • ~/%-25s %s\n", it.Path, statusBadge)
			}

			fmt.Println("═══════════════════════════════════════════════")
			return nil
		},
	}
}

// dotsynx add <path>
func newAddCommand() *cobra.Command {
	var isDir bool
	var desc string

	cmd := &cobra.Command{
		Use:   "add <path>",
		Short: "Add a dotfile or directory to synchronization",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			eng, err := core.NewEngine(cfg)
			if err != nil {
				return err
			}

			target := args[0]
			// Inspect if it is directory
			_, abs, err := config.NormalizePath(target)
			if err == nil {
				if info, statErr := os.Stat(abs); statErr == nil {
					if info.IsDir() {
						isDir = true
					}
				}
			}

			err = eng.Tracker.TrackItem(target, isDir, desc)
			if err != nil {
				return fmt.Errorf("failed to track item: %w", err)
			}

			fmt.Printf("✅ Successfully tracked %s into dotsynx\n", target)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&isDir, "dir", "d", false, "Specify that path is a directory")
	cmd.Flags().StringVarP(&desc, "desc", "m", "", "Short description for this dotfile")
	return cmd
}

// dotsynx remove <path>
func newRemoveCommand() *cobra.Command {
	var keepLocal bool

	cmd := &cobra.Command{
		Use:   "remove <path>",
		Short: "Untrack a dotfile or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			eng, err := core.NewEngine(cfg)
			if err != nil {
				return err
			}

			relHome, _, _ := config.NormalizePath(args[0])
			err = eng.Tracker.UntrackItem(relHome, keepLocal)
			if err != nil {
				return err
			}

			fmt.Printf("✅ Untracked %s\n", relHome)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&keepLocal, "keep", "k", true, "Preserve file in $HOME as a regular file")
	return cmd
}

// dotsynx list
func newListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all tracked dotfiles and link states",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			eng, err := core.NewEngine(cfg)
			if err != nil {
				return err
			}

			if len(cfg.Tracked) == 0 {
				fmt.Println("No dotfiles currently tracked. Use 'dotsynx add <path>' or 'dotsynx suggest'.")
				return nil
			}

			fmt.Printf("%-30s %-12s %-10s %s\n", "PATH IN $HOME", "TYPE", "STATUS", "DESCRIPTION")
			fmt.Println(strings.Repeat("-", 75))

			for _, it := range cfg.Tracked {
				st := eng.Tracker.InspectItem(it)
				itemType := "file"
				if it.IsDir {
					itemType = "directory"
				}
				fmt.Printf("~/%-28s %-12s %-10s %s\n", it.Path, itemType, st.LinkStatus, it.Description)
			}
			return nil
		},
	}
}

// dotsynx suggest
func newSuggestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "suggest",
		Short: "Scan and suggest dotfiles on this computer",
		RunE: func(cmd *cobra.Command, args []string) error {
			suggs, err := core.ScanSuggestions()
			if err != nil {
				return err
			}

			cfg, _ := config.Load(cfgPath)
			trackedMap := make(map[string]bool)
			if cfg != nil {
				for _, it := range cfg.Tracked {
					trackedMap[it.Path] = true
				}
			}

			fmt.Println("🔍 Detected dotfiles on this machine:")
			fmt.Printf("%-30s %-15s %-10s %s\n", "PATH", "CATEGORY", "STATUS", "DESCRIPTION")
			fmt.Println(strings.Repeat("-", 80))

			for _, s := range suggs {
				status := "available"
				if trackedMap[s.Path] {
					status = "TRACKED"
				}
				if s.IsSensitive {
					status = "SENSITIVE"
				}
				fmt.Printf("~/%-28s %-15s %-10s %s\n", s.Path, s.Category, status, s.Description)
			}

			fmt.Println("\nTo track any of these, run: dotsynx add <path>")
			return nil
		},
	}
}

// dotsynx service [install|uninstall|status]
func newServiceCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "service",
		Short: "Manage OS-level background service (macOS launchd / Linux systemd)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "install",
		Short: "Install dotsynx as a background user service",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			mgr, err := service.GetManager()
			if err != nil {
				return err
			}

			binPath, _ := os.Executable()
			err = mgr.Install(binPath, string(cfg.SyncMode))
			if err != nil {
				return fmt.Errorf("failed to install service: %w", err)
			}

			fmt.Printf("✅ Successfully installed dotsynx service for %s\n", runtime.GOOS)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall dotsynx OS service",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := service.GetManager()
			if err != nil {
				return err
			}

			if err := mgr.Uninstall(); err != nil {
				return fmt.Errorf("failed to uninstall service: %w", err)
			}

			fmt.Println("✅ Successfully uninstalled dotsynx service.")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "status",
		Short: "Check OS service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := service.GetManager()
			if err != nil {
				return err
			}

			st, err := mgr.Status()
			if err != nil {
				return err
			}

			fmt.Printf("Platform: %s\n", st.Platform)
			fmt.Printf("Service File: %s\n", st.ServiceFile)
			fmt.Printf("Installed: %v\n", st.Installed)
			fmt.Printf("Running: %v\n", st.Running)
			if st.PID > 0 {
				fmt.Printf("PID: %d\n", st.PID)
			}
			fmt.Printf("Message: %s\n", st.Message)
			return nil
		},
	})

	return cmd
}

// dotsynx logs [-f] [-n lines] [--since duration]
func newLogsCommand() *cobra.Command {
	var (
		follow   bool
		lines    int
		sinceStr string
		clear    bool
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View dotsynx server and sync logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			if clear {
				p := logger.DefaultLogPath()
				if err := os.WriteFile(p, []byte(""), 0644); err != nil {
					return fmt.Errorf("failed to clear logs: %w", err)
				}
				fmt.Println("Cleared dotsynx log file.")
				return nil
			}

			var sinceDur time.Duration
			if sinceStr != "" {
				var err error
				sinceDur, err = time.ParseDuration(sinceStr)
				if err != nil {
					return fmt.Errorf("invalid duration format for --since (e.g. 10m, 1h, 24h): %w", err)
				}
			}

			if follow {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				return logger.Tail(ctx, lines, os.Stdout)
			}

			entries, err := logger.ReadEntries(lines, sinceDur)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No log entries recorded yet.")
				return nil
			}

			for _, e := range entries {
				fmt.Println(e.Raw)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (stream live logs)")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "Number of recent lines to display")
	cmd.Flags().StringVar(&sinceStr, "since", "", "Show logs since relative duration (e.g. 10m, 1h, 24h)")
	cmd.Flags().BoolVar(&clear, "clear", false, "Clear log file")

	return cmd
}
