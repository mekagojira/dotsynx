package webserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"dotsynx/internal/config"
	"dotsynx/internal/core"
	"dotsynx/internal/daemon"
	"dotsynx/internal/logger"
	"dotsynx/internal/service"
)

type Server struct {
	Config *config.Config
	Engine *core.Engine
	DistFS fs.FS
	Port   int
	Server *http.Server
}

func NewServer(cfg *config.Config, distFS fs.FS) (*Server, error) {
	eng, err := core.NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	port := cfg.WebPort
	if port == 0 {
		port = 18942
	}

	return &Server{
		Config: cfg,
		Engine: eng,
		DistFS: distFS,
		Port:   port,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/reset-remote", s.handleResetRemote)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/files", s.handleFiles)
	mux.HandleFunc("/api/files/track", s.handleTrack)
	mux.HandleFunc("/api/files/untrack", s.handleUntrack)
	mux.HandleFunc("/api/files/toggle", s.handleToggle)
	mux.HandleFunc("/api/suggestions", s.handleSuggestions)
	mux.HandleFunc("/api/browse", s.handleBrowse)
	mux.HandleFunc("/api/conflicts", s.handleConflicts)
	mux.HandleFunc("/api/conflicts/resolve", s.handleResolveConflict)
	mux.HandleFunc("/api/backups", s.handleBackups)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/daemon/start", s.handleDaemonStart)
	mux.HandleFunc("/api/daemon/stop", s.handleDaemonStop)

	// Backward-compatible daemon endpoints
	mux.HandleFunc("/daemon/status", s.handleStatus)
	mux.HandleFunc("/daemon/sync", s.handleSync)
	mux.HandleFunc("/daemon/reset", s.handleResetRemote)

	// Static assets handler
	var fileServer http.Handler
	if s.DistFS != nil {
		fileServer = http.FileServer(http.FS(s.DistFS))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/daemon/") {
			http.NotFound(w, r)
			return
		}

		if fileServer != nil {
			// Check if file exists in DistFS
			f, err := s.DistFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// Fallback to index.html for SPA router
			indexFile, err := s.DistFS.Open("index.html")
			if err == nil {
				defer indexFile.Close()
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				http.ServeContent(w, r, "index.html", time.Now(), indexFile.(ioReadSeeker))
				return
			}
		}

		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>dotsynx</title></head><body><h1>dotsynx API server is running. Web assets not embedded.</h1></body></html>`))
	})

	return corsMiddleware(mux)
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	s.Server = &http.Server{
		Addr:    addr,
		Handler: s.Handler(),
	}

	return s.Server.ListenAndServe()
}

type ioReadSeeker interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	gitStatus, _ := s.Engine.Git.Status(ctx)
	daemonRunning, pid := daemon.CheckRunning()

	var svcStatus *service.Status
	if mgr, err := service.GetManager(); err == nil {
		svcStatus, _ = mgr.Status()
	}

	conflicts, _ := s.Engine.Conflict.DetectConflicts(ctx)
	if conflicts == nil {
		conflicts = []core.ConflictItem{}
	}

	if s.Config.Tracked == nil {
		s.Config.Tracked = []config.TrackedItem{}
	}

	syncLogs := s.Engine.SyncLogs
	if syncLogs == nil {
		syncLogs = []*core.SyncResult{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"git": gitStatus,
		"daemon": map[string]any{
			"running": daemonRunning,
			"pid":     pid,
		},
		"service":   svcStatus,
		"conflicts": conflicts,
		"config":    s.Config,
		"last_sync": s.Engine.LastSync,
		"sync_logs": syncLogs,
	})
}

func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	res, err := s.Engine.Sync(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":  err.Error(),
			"result": res,
		})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleResetRemote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Minute)
	defer cancel()

	res, err := s.Engine.ResetToRemote(ctx)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":  err.Error(),
			"result": res,
		})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, s.Config)
	case http.MethodPost:
		var req struct {
			config.Config `json:",inline"`
			ResetToRemote bool `json:"reset_to_remote"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		s.Config.RepoURL = strings.TrimSpace(req.RepoURL)
		s.Config.Branch = strings.TrimSpace(req.Branch)
		s.Config.SyncMode = req.SyncMode
		s.Config.Interval = req.Interval
		s.Config.ConflictStrategy = req.ConflictStrategy
		s.Config.Theme = req.Theme

		if err := s.Config.Save(""); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update git remote if changed
		if s.Config.RepoURL != "" {
			_ = s.Engine.Git.SetRemote(s.Config.RepoURL)
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			if req.ResetToRemote {
				_, _ = s.Engine.ResetToRemote(ctx)
			} else {
				_ = s.Engine.EnsureRepoReady(ctx)
			}
			cancel()
		}

		writeJSON(w, http.StatusOK, s.Config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	results := []core.ItemStatus{}
	for _, it := range s.Config.Tracked {
		results = append(results, s.Engine.Tracker.InspectItem(it))
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleTrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path        string `json:"path"`
		IsDir       bool   `json:"is_dir"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Engine.Tracker.TrackItem(req.Path, req.IsDir, req.Description); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleUntrack(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path     string `json:"path"`
		KeepFile bool   `json:"keep_file"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.Engine.Tracker.UntrackItem(req.Path, req.KeepFile); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	found := false
	for i, it := range s.Config.Tracked {
		if it.Path == req.Path {
			s.Config.Tracked[i].Enabled = !s.Config.Tracked[i].Enabled
			found = true
			if s.Config.Tracked[i].Enabled {
				_ = s.Engine.Tracker.ApplyItem(s.Config.Tracked[i])
			}
			break
		}
	}

	if !found {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}

	_ = s.Config.Save("")
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleSuggestions(w http.ResponseWriter, r *http.Request) {
	suggs, err := core.ScanSuggestions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if suggs == nil {
		suggs = []core.Suggestion{}
	}
	writeJSON(w, http.StatusOK, suggs)
}

func (s *Server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	home, _ := os.UserHomeDir()
	reqPath := r.URL.Query().Get("path")
	if reqPath == "" {
		reqPath = home
	} else if strings.HasPrefix(reqPath, "~") {
		reqPath = filepath.Join(home, strings.TrimPrefix(reqPath, "~"))
	}

	abs, err := filepath.Abs(reqPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	type fileItem struct {
		Name        string `json:"name"`
		RelHome     string `json:"rel_home"`
		AbsPath     string `json:"abs_path"`
		IsDir       bool   `json:"is_dir"`
		IsSensitive bool   `json:"is_sensitive"`
		IsTracked   bool   `json:"is_tracked"`
		Size        int64  `json:"size"`
	}

	trackedMap := make(map[string]bool)
	for _, it := range s.Config.Tracked {
		trackedMap[it.Path] = true
	}

	var items []fileItem
	for _, e := range entries {
		itemAbs := filepath.Join(abs, e.Name())
		rel, _ := filepath.Rel(home, itemAbs)
		info, _ := e.Info()
		var sz int64
		if info != nil {
			sz = info.Size()
		}

		items = append(items, fileItem{
			Name:        e.Name(),
			RelHome:     rel,
			AbsPath:     itemAbs,
			IsDir:       e.IsDir(),
			IsSensitive: core.IsSensitive(itemAbs),
			IsTracked:   trackedMap[rel],
			Size:        sz,
		})
	}

	parent := filepath.Dir(abs)
	writeJSON(w, http.StatusOK, map[string]any{
		"current_path": abs,
		"parent_path":  parent,
		"home_path":    home,
		"entries":      items,
	})
}

func (s *Server) handleConflicts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	conflicts, err := s.Engine.Conflict.DetectConflicts(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conflicts == nil {
		conflicts = []core.ConflictItem{}
	}
	writeJSON(w, http.StatusOK, conflicts)
}

func (s *Server) handleResolveConflict(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Path     string                  `json:"path"`
		Strategy config.ConflictStrategy `json:"strategy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.Engine.Conflict.Resolve(ctx, req.Path, req.Strategy); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleBackups(w http.ResponseWriter, r *http.Request) {
	backups, err := s.Engine.Backup.ListBackups()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if backups == nil {
		backups = []core.BackupEntry{}
	}
	writeJSON(w, http.StatusOK, backups)
}

func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	linesStr := r.URL.Query().Get("lines")
	lines := 100
	if linesStr != "" {
		if val, err := strconv.Atoi(linesStr); err == nil && val > 0 {
			lines = val
		}
	}

	sinceStr := r.URL.Query().Get("since")
	var sinceDuration time.Duration
	if sinceStr != "" {
		sinceDuration, _ = time.ParseDuration(sinceStr)
	}

	entries, err := logger.ReadEntries(lines, sinceDuration)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []logger.LogEntry{}
	}

	writeJSON(w, http.StatusOK, entries)
}

func (s *Server) handleDaemonStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if sm, err := service.GetManager(); err == nil {
		if err := sm.Start(); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (s *Server) handleDaemonStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = daemon.StopRunning()
	if sm, err := service.GetManager(); err == nil {
		_ = sm.Stop()
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}
