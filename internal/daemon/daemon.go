package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"dotsynx/internal/config"
	"dotsynx/internal/core"
	"dotsynx/internal/logger"

	"github.com/robfig/cron/v3"
)

// Daemon coordinates background syncing, periodic schedules, and IPC
type Daemon struct {
	Config     *config.Config
	Engine     *core.Engine
	Cron       *cron.Cron
	HTTP       *http.Server
	Handler    http.Handler
	PIDFile    string
	LastSync   *core.SyncResult
	SyncActive bool
}

// SetHandler sets a custom HTTP handler (e.g. webserver with embedded React UI)
func (d *Daemon) SetHandler(h http.Handler) {
	d.Handler = h
}

// Info summarizes daemon running status
type Info struct {
	Running      bool            `json:"running"`
	PID          int             `json:"pid"`
	SyncMode     config.SyncMode `json:"sync_mode"`
	Interval     string          `json:"interval"`
	LastSyncTime *time.Time      `json:"last_sync_time,omitempty"`
	LastSyncOk   bool            `json:"last_sync_ok"`
	LastSyncMsg  string          `json:"last_sync_msg,omitempty"`
	TrackedCount int             `json:"tracked_count"`
	StorageDir   string          `json:"storage_dir"`
	RemoteURL    string          `json:"repo_url"`
}

// PIDPath returns ~/.local/share/dotsynx/dotsynx.pid
func PIDPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "dotsynx", "dotsynx.pid")
}

// CheckRunning checks if a daemon process is currently running
func CheckRunning() (bool, int) {
	pidFile := PIDPath()
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return false, 0
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 || pid == os.Getpid() {
		return false, 0
	}

	// Check if process exists by sending signal 0
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false, 0
	}

	err = proc.Signal(syscall.Signal(0))
	if err == nil || errors.Is(err, syscall.EPERM) || strings.Contains(err.Error(), "operation not permitted") {
		return true, pid
	}

	// Process dead, clean up stale PID file
	_ = os.Remove(pidFile)
	return false, 0
}

// StopRunning stops the daemon process if running
func StopRunning() error {
	running, pid := CheckRunning()
	if !running || pid <= 0 || pid == os.Getpid() {
		_ = os.Remove(PIDPath())
		return errors.New("daemon is not currently running")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		_ = os.Remove(PIDPath())
		return err
	}

	_ = proc.Signal(syscall.SIGTERM)

	// Wait up to 3 seconds for process to exit
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			_ = os.Remove(PIDPath())
			return nil
		}
	}

	// Force kill if not stopped
	if pid != os.Getpid() {
		_ = proc.Signal(syscall.SIGKILL)
	}
	_ = os.Remove(PIDPath())
	return nil
}

// NewDaemon initializes a daemon instance
func NewDaemon(cfg *config.Config) (*Daemon, error) {
	eng, err := core.NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	d := &Daemon{
		Config:  cfg,
		Engine:  eng,
		Cron:    cron.New(),
		PIDFile: PIDPath(),
	}

	return d, nil
}

// Start launches the daemon service
func (d *Daemon) Start(ctx context.Context) error {
	if running, pid := CheckRunning(); running {
		return fmt.Errorf("daemon is already running (PID %d)", pid)
	}

	if err := os.MkdirAll(filepath.Dir(d.PIDFile), 0755); err != nil {
		return err
	}

	pid := os.Getpid()
	if err := os.WriteFile(d.PIDFile, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	defer os.Remove(d.PIDFile)
	logger.Info("dotsynx daemon started (PID %d, mode: %s)", pid, d.Config.SyncMode)

	// Configure scheduler if in Auto mode
	if d.Config.SyncMode == config.ModeAuto {
		d.setupSchedule()
		d.Cron.Start()
		defer d.Cron.Stop()
	}

	// Setup HTTP handler: prefer rich handler with Web UI if configured
	var httpHandler http.Handler = d.Handler
	if httpHandler == nil {
		mux := http.NewServeMux()
		mux.HandleFunc("/daemon/status", d.handleStatus)
		mux.HandleFunc("/daemon/sync", d.handleSync)
		mux.HandleFunc("/daemon/reset", d.handleReset)
		httpHandler = mux
	}

	port := d.Config.WebPort
	if port == 0 {
		port = 18942
	}
	addr := fmt.Sprintf("127.0.0.1:%d", port)

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Warn("Could not bind daemon Web UI port %s: %v. Running in background sync mode.", addr, err)
	} else {
		d.HTTP = &http.Server{
			Handler: httpHandler,
		}

		go func() {
			if err := d.HTTP.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Printf("Daemon HTTP server error: %v", err)
			}
		}()
	}

	// Perform initial sync on start if Auto or Boot mode
	go func() {
		time.Sleep(1 * time.Second)
		_, _ = d.TriggerSync()
	}()

	// Listen for termination signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	select {
	case <-ctx.Done():
	case <-sigChan:
	}

	logger.Info("dotsynx daemon shutting down...")

	if d.HTTP != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = d.HTTP.Shutdown(shutdownCtx)
	}

	return nil
}

// TriggerSync executes a sync cycle safely
func (d *Daemon) TriggerSync() (*core.SyncResult, error) {
	if d.SyncActive {
		return nil, errors.New("a sync operation is already in progress")
	}

	d.SyncActive = true
	defer func() { d.SyncActive = false }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := d.Engine.Sync(ctx)
	d.LastSync = res
	return res, err
}

// TriggerReset executes a hard reset to remote safely
func (d *Daemon) TriggerReset() (*core.SyncResult, error) {
	if d.SyncActive {
		return nil, errors.New("a sync or reset operation is already in progress")
	}

	d.SyncActive = true
	defer func() { d.SyncActive = false }()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	res, err := d.Engine.ResetToRemote(ctx)
	d.LastSync = res
	return res, err
}

func (d *Daemon) setupSchedule() {
	interval := d.Config.Interval
	if interval == "" {
		interval = "15m"
	}

	// Check if interval is standard duration (e.g. 15m, 1h, 30m)
	if dur, err := time.ParseDuration(interval); err == nil {
		spec := fmt.Sprintf("@every %s", dur.String())
		_, _ = d.Cron.AddFunc(spec, func() {
			_, _ = d.TriggerSync()
		})
		return
	}

	// Otherwise treat as cron expression (e.g. "0 * * * *")
	_, err := d.Cron.AddFunc(interval, func() {
		_, _ = d.TriggerSync()
	})
	if err != nil {
		// Fallback to 15m if invalid cron
		_, _ = d.Cron.AddFunc("@every 15m", func() {
			_, _ = d.TriggerSync()
		})
	}
}

func (d *Daemon) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var lastTime *time.Time
	var lastOk bool
	var lastMsg string

	if d.LastSync != nil {
		lastTime = &d.LastSync.Timestamp
		lastOk = d.LastSync.Success
		lastMsg = d.LastSync.Message
	}

	info := Info{
		Running:      true,
		PID:          os.Getpid(),
		SyncMode:     d.Config.SyncMode,
		Interval:     d.Config.Interval,
		LastSyncTime: lastTime,
		LastSyncOk:   lastOk,
		LastSyncMsg:  lastMsg,
		TrackedCount: len(d.Config.Tracked),
		StorageDir:   d.Config.StorageDir,
		RemoteURL:    d.Config.RepoURL,
	}

	_ = json.NewEncoder(w).Encode(info)
}

func (d *Daemon) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	res, err := d.TriggerSync()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   err.Error(),
			"details": res,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

func (d *Daemon) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	res, err := d.TriggerReset()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   err.Error(),
			"details": res,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

// RequestSyncViaHTTP asks a running daemon over HTTP to trigger a sync
func RequestSyncViaHTTP(port int) (*core.SyncResult, error) {
	if port == 0 {
		port = 18942
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/daemon/sync", port), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res core.SyncResult
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse daemon response: %w", err)
	}

	return &res, nil
}

// RequestResetViaHTTP asks a running daemon over HTTP to trigger a hard reset to remote
func RequestResetViaHTTP(port int) (*core.SyncResult, error) {
	if port == 0 {
		port = 18942
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Post(fmt.Sprintf("http://127.0.0.1:%d/daemon/reset", port), "application/json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res core.SyncResult
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("failed to parse daemon response: %w", err)
	}

	return &res, nil
}

// QueryStatusViaHTTP queries running daemon info
func QueryStatusViaHTTP(port int) (*Info, error) {
	if port == 0 {
		port = 18942
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/daemon/status", port))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info Info
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}
