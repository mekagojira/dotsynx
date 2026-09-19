package tui

import (
	"context"
	"dotsynx"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dotsynx/internal/config"
	"dotsynx/internal/core"
	"dotsynx/internal/daemon"
	"dotsynx/internal/git"
	"dotsynx/internal/logger"
	"dotsynx/internal/service"
	"dotsynx/internal/webserver"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pkg/browser"
)

type Tab int

const (
	TabDashboard Tab = iota
	TabFiles
	TabSuggestions
	TabBrowse
	TabConflicts
	TabLogs
	TabSettings
)

var tabNames = []string{
	"📊 1: Dashboard",
	"📦 2: Tracked Files",
	"✨ 3: Suggestions",
	"📂 4: File Browser",
	"⚔️  5: Conflicts",
	"📜 6: Logs",
	"⚙️  7: Settings",
}

// AppModel is the main Bubbletea state model
type AppModel struct {
	Config      *config.Config
	Engine      *core.Engine
	GitStatus   *git.RepoStatus
	DaemonInfo  *daemon.Info
	ServiceSt   *service.Status
	Suggestions []core.Suggestion
	Conflicts   []core.ConflictItem
	ItemStates  []core.ItemStatus

	CurrentTab Tab
	Theme      Theme
	Styles     Styles
	Width      int
	Height     int

	// List cursors
	FileCursor     int
	SuggestCursor  int
	BrowseCursor   int
	BrowsePath     string
	BrowseEntries  []browseEntry
	ConflictCursor int

	LogEntries      []logger.LogEntry
	LogScrollOffset int
	LogSearchQuery  string
	IsSearchingLogs bool
	WebServer       *webserver.Server
	LastSync        *core.SyncResult
	StatusMessage   string
	IsSyncing       bool
	Err             error
}

type browseEntry struct {
	Name        string
	RelHome     string
	IsDir       bool
	Tracked     bool
	IsSensitive bool
}

// Messages
type syncDoneMsg struct {
	Result *core.SyncResult
	Err    error
}

type refreshDataMsg struct {
	GitStatus   *git.RepoStatus
	DaemonInfo  *daemon.Info
	ServiceSt   *service.Status
	Suggestions []core.Suggestion
	Conflicts   []core.ConflictItem
	ItemStates  []core.ItemStatus
	LogEntries  []logger.LogEntry
}

// NewApp creates an AppModel instance
func NewApp(cfg *config.Config) (*AppModel, error) {
	eng, err := core.NewEngine(cfg)
	if err != nil {
		return nil, err
	}

	theme := CurrentTheme(cfg.Theme)
	styles := MakeStyles(theme)

	home, _ := os.UserHomeDir()

	m := &AppModel{
		Config:     cfg,
		Engine:     eng,
		CurrentTab: TabDashboard,
		Theme:      theme,
		Styles:     styles,
		BrowsePath: home,
		Width:      80,
		Height:     24,
	}

	return m, nil
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadInitialData(),
		tea.EnterAltScreen,
	)
}

func (m AppModel) loadInitialData() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		gStatus, _ := m.Engine.Git.Status(ctx)
		dInfo, _ := daemon.QueryStatusViaHTTP(m.Config.WebPort)

		var sStatus *service.Status
		if sm, err := service.GetManager(); err == nil {
			sStatus, _ = sm.Status()
		}

		suggs, _ := core.ScanSuggestions()
		conflicts, _ := m.Engine.Conflict.DetectConflicts(ctx)

		var states []core.ItemStatus
		for _, item := range m.Config.Tracked {
			states = append(states, m.Engine.Tracker.InspectItem(item))
		}

		logs, _ := logger.ReadEntries(100, 0)

		return refreshDataMsg{
			GitStatus:   gStatus,
			DaemonInfo:  dInfo,
			ServiceSt:   sStatus,
			Suggestions: suggs,
			Conflicts:   conflicts,
			ItemStates:  states,
			LogEntries:  logs,
		}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case refreshDataMsg:
		m.GitStatus = msg.GitStatus
		m.DaemonInfo = msg.DaemonInfo
		m.ServiceSt = msg.ServiceSt
		m.Suggestions = msg.Suggestions
		m.Conflicts = msg.Conflicts
		m.ItemStates = msg.ItemStates
		m.LogEntries = msg.LogEntries
		m.loadBrowseEntries()
		return m, nil

	case syncDoneMsg:
		m.IsSyncing = false
		m.LastSync = msg.Result
		if msg.Err != nil {
			m.StatusMessage = fmt.Sprintf("❌ Sync failed: %v", msg.Err)
		} else if msg.Result != nil {
			if len(msg.Result.Errors) > 0 {
				m.StatusMessage = fmt.Sprintf("⚠️  Sync finished with %d error(s) — see Dashboard (Tab 1) for details", len(msg.Result.Errors))
			} else {
				m.StatusMessage = fmt.Sprintf("✅ %s", msg.Result.Message)
			}
		}
		return m, m.loadInitialData()

	case tea.KeyMsg:
		if m.CurrentTab == TabLogs && m.IsSearchingLogs {
			switch msg.String() {
			case "esc":
				m.IsSearchingLogs = false
				m.LogSearchQuery = ""
				m.LogScrollOffset = 0
				return m, nil
			case "enter":
				m.IsSearchingLogs = false
				return m, nil
			case "backspace":
				if len(m.LogSearchQuery) > 0 {
					m.LogSearchQuery = m.LogSearchQuery[:len(m.LogSearchQuery)-1]
					m.LogScrollOffset = 0
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.LogSearchQuery += msg.String()
					m.LogScrollOffset = 0
				}
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		// Tab switching
		case "1":
			m.CurrentTab = TabDashboard
		case "2":
			m.CurrentTab = TabFiles
		case "3":
			m.CurrentTab = TabSuggestions
		case "4":
			m.CurrentTab = TabBrowse
			m.loadBrowseEntries()
		case "5":
			m.CurrentTab = TabConflicts
		case "6":
			m.CurrentTab = TabLogs
		case "7":
			m.CurrentTab = TabSettings
		case "tab":
			m.CurrentTab = (m.CurrentTab + 1) % 7
			if m.CurrentTab == TabBrowse {
				m.loadBrowseEntries()
			}

		// Trigger sync
		case "s":
			if !m.IsSyncing {
				m.IsSyncing = true
				m.StatusMessage = "⏳ Synchronizing dotfiles..."
				return m, func() tea.Msg {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					res, err := m.Engine.Sync(ctx)
					return syncDoneMsg{Result: res, Err: err}
				}
			}

		// Navigation
		case "up", "k":
			m.navigateUp()
		case "down", "j":
			m.navigateDown()

		// Actions
		case "enter", " ":
			return m.handleSelection()
		case "d", "x":
			m.handleDeleteOrUntrack()
		case "/":
			if m.CurrentTab == TabLogs {
				m.IsSearchingLogs = true
			}
		case "c":
			if m.CurrentTab == TabLogs {
				m.LogSearchQuery = ""
				m.LogScrollOffset = 0
			}
		case "u":
			m.handleOpenUI()
		}
	}

	return m, nil
}

func (m *AppModel) navigateUp() {
	switch m.CurrentTab {
	case TabFiles:
		if m.FileCursor > 0 {
			m.FileCursor--
		}
	case TabSuggestions:
		if m.SuggestCursor > 0 {
			m.SuggestCursor--
		}
	case TabBrowse:
		if m.BrowseCursor > 0 {
			m.BrowseCursor--
		}
	case TabConflicts:
		if m.ConflictCursor > 0 {
			m.ConflictCursor--
		}
	case TabLogs:
		if m.LogScrollOffset > 0 {
			m.LogScrollOffset--
		}
	}
}

func (m *AppModel) navigateDown() {
	switch m.CurrentTab {
	case TabFiles:
		if m.FileCursor < len(m.Config.Tracked)-1 {
			m.FileCursor++
		}
	case TabSuggestions:
		if m.SuggestCursor < len(m.Suggestions)-1 {
			m.SuggestCursor++
		}
	case TabBrowse:
		if m.BrowseCursor < len(m.BrowseEntries)-1 {
			m.BrowseCursor++
		}
	case TabConflicts:
		if m.ConflictCursor < len(m.Conflicts)-1 {
			m.ConflictCursor++
		}
	case TabLogs:
		if len(m.LogEntries) > 10 && m.LogScrollOffset < len(m.LogEntries)-10 {
			m.LogScrollOffset++
		}
	}
}

func (m *AppModel) handleSelection() (tea.Model, tea.Cmd) {
	switch m.CurrentTab {
	case TabFiles:
		// Toggle enabled/disabled
		if len(m.Config.Tracked) > 0 && m.FileCursor < len(m.Config.Tracked) {
			m.Config.Tracked[m.FileCursor].Enabled = !m.Config.Tracked[m.FileCursor].Enabled
			_ = m.Config.Save("")
			m.StatusMessage = fmt.Sprintf("Updated %s", m.Config.Tracked[m.FileCursor].Path)
			return m, m.loadInitialData()
		}

	case TabSuggestions:
		// Track suggestion
		if len(m.Suggestions) > 0 && m.SuggestCursor < len(m.Suggestions) {
			s := m.Suggestions[m.SuggestCursor]
			err := m.Engine.Tracker.TrackItem(s.Path, s.IsDir, s.Description)
			if err != nil {
				m.StatusMessage = fmt.Sprintf("Failed to track: %v", err)
			} else {
				m.StatusMessage = fmt.Sprintf("Added %s to tracking!", s.Path)
			}
			return m, m.loadInitialData()
		}

	case TabBrowse:
		if len(m.BrowseEntries) == 0 {
			return m, nil
		}
		entry := m.BrowseEntries[m.BrowseCursor]
		if entry.Name == ".." {
			m.BrowsePath = filepath.Dir(m.BrowsePath)
			m.BrowseCursor = 0
			m.loadBrowseEntries()
			return m, nil
		}
		fullPath := filepath.Join(m.BrowsePath, entry.Name)
		if entry.IsDir {
			m.BrowsePath = fullPath
			m.BrowseCursor = 0
			m.loadBrowseEntries()
		} else {
			// Toggle tracking for this file
			if entry.Tracked {
				_ = m.Engine.Tracker.UntrackItem(entry.RelHome, true)
				m.StatusMessage = fmt.Sprintf("Untracked %s", entry.RelHome)
			} else {
				_ = m.Engine.Tracker.TrackItem(entry.RelHome, false, "")
				m.StatusMessage = fmt.Sprintf("Tracked %s", entry.RelHome)
			}
			m.loadBrowseEntries()
			return m, m.loadInitialData()
		}

	case TabConflicts:
		// Resolve with ours
		if len(m.Conflicts) > 0 && m.ConflictCursor < len(m.Conflicts) {
			c := m.Conflicts[m.ConflictCursor]
			ctx := context.Background()
			_ = m.Engine.Conflict.Resolve(ctx, c.Path, config.StrategyOurs)
			m.StatusMessage = fmt.Sprintf("Resolved %s (kept local)", c.Path)
			return m, m.loadInitialData()
		}
	}

	return m, nil
}

func (m *AppModel) handleDeleteOrUntrack() {
	if m.CurrentTab == TabFiles && len(m.Config.Tracked) > 0 && m.FileCursor < len(m.Config.Tracked) {
		item := m.Config.Tracked[m.FileCursor]
		_ = m.Engine.Tracker.UntrackItem(item.Path, true)
		if m.FileCursor >= len(m.Config.Tracked)-1 && m.FileCursor > 0 {
			m.FileCursor--
		}
		m.StatusMessage = fmt.Sprintf("Removed %s from tracking", item.Path)
	}
}

func (m *AppModel) handleOpenUI() {
	port := m.Config.WebPort
	if port == 0 {
		port = 18942
	}
	targetURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Check if already active (e.g. background daemon)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
	if err == nil {
		conn.Close()
		_ = browser.OpenURL(targetURL)
		m.StatusMessage = fmt.Sprintf("🌐 Opened Web UI in your browser: %s", targetURL)
		return
	}

	// Start server if not running
	distFS, _ := dotsynx.GetWebDistFS()
	srv, err := webserver.NewServer(m.Config, distFS)
	if err != nil {
		m.StatusMessage = fmt.Sprintf("❌ Failed to initialize web server: %v", err)
		return
	}
	m.WebServer = srv

	go func() {
		_ = srv.Start()
	}()

	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = browser.OpenURL(targetURL)
	}()

	m.StatusMessage = fmt.Sprintf("🚀 Started Web server & opened in browser: %s", targetURL)
}

func (m *AppModel) loadBrowseEntries() {
	home, _ := os.UserHomeDir()
	if m.BrowsePath == "" {
		m.BrowsePath = home
	}

	entries, err := os.ReadDir(m.BrowsePath)
	if err != nil {
		m.BrowseEntries = []browseEntry{}
		return
	}

	trackedMap := make(map[string]bool)
	for _, it := range m.Config.Tracked {
		trackedMap[it.Path] = true
	}

	var list []browseEntry
	if m.BrowsePath != home {
		list = append(list, browseEntry{
			Name:    "..",
			RelHome: "",
			IsDir:   true,
		})
	}

	for _, e := range entries {
		fullPath := filepath.Join(m.BrowsePath, e.Name())
		rel, _ := filepath.Rel(home, fullPath)
		list = append(list, browseEntry{
			Name:        e.Name(),
			RelHome:     rel,
			IsDir:       e.IsDir(),
			Tracked:     trackedMap[rel],
			IsSensitive: core.IsSensitive(fullPath),
		})
	}

	m.BrowseEntries = list
}

func (m AppModel) View() string {
	var b strings.Builder

	// 1. Header with Logo & Mode Indicator
	header := m.renderHeader()
	b.WriteString(header + "\n")

	// 2. Tab Bar
	tabs := m.renderTabs()
	b.WriteString(tabs + "\n\n")

	// 3. Main Content
	var content string
	switch m.CurrentTab {
	case TabDashboard:
		content = m.renderDashboard()
	case TabFiles:
		content = m.renderFiles()
	case TabSuggestions:
		content = m.renderSuggestions()
	case TabBrowse:
		content = m.renderBrowse()
	case TabConflicts:
		content = m.renderConflicts()
	case TabLogs:
		content = m.renderLogs()
	case TabSettings:
		content = m.renderSettings()
	}
	b.WriteString(content + "\n")

	// 4. Status Bar / Flash Message
	if m.StatusMessage != "" {
		statusCard := m.Styles.Card.Copy().
			BorderForeground(m.Theme.Accent).
			Padding(0, 1).
			Render(m.StatusMessage)
		b.WriteString(statusCard + "\n")
	}

	// 5. Footer Controls
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m AppModel) renderHeader() string {
	title := m.Styles.HeaderTitle.Render("⚡ DOTSYNX")
	sub := m.Styles.HeaderSub.Render("Multi-device dotfile sync agent")

	modeBadge := m.Styles.BadgeInfo.Render(fmt.Sprintf("Mode: %s", strings.ToUpper(string(m.Config.SyncMode))))
	if m.Config.SyncMode == config.ModeAuto {
		modeBadge = m.Styles.BadgeSuccess.Render("Mode: AUTO")
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, title, " ", sub, "  ", modeBadge)
}

func (m AppModel) renderTabs() string {
	var renderedTabs []string
	for i, name := range tabNames {
		if Tab(i) == m.CurrentTab {
			renderedTabs = append(renderedTabs, m.Styles.TabActive.Render(name))
		} else {
			renderedTabs = append(renderedTabs, m.Styles.TabInactive.Render(name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
}

func (m AppModel) renderDashboard() string {
	var lines []string

	// Repo summary
	repoURL := m.Config.RepoURL
	if repoURL == "" {
		repoURL = "Not connected (Local only)"
	}
	lines = append(lines, fmt.Sprintf("📦 Storage Repo:  %s", m.Config.StorageDir))
	lines = append(lines, fmt.Sprintf("🔗 Remote URL:    %s", repoURL))
	lines = append(lines, fmt.Sprintf("🌿 Branch:        %s", m.Config.Branch))

	if m.GitStatus != nil {
		lines = append(lines, fmt.Sprintf("📊 Git State:     Ahead: %d, Behind: %d, Modified: %d",
			m.GitStatus.Ahead, m.GitStatus.Behind, len(m.GitStatus.Modified)))
	}

	// Daemon summary
	daemonText := "Stopped"
	if m.DaemonInfo != nil && m.DaemonInfo.Running {
		daemonText = fmt.Sprintf("Active (PID %d)", m.DaemonInfo.PID)
	}
	lines = append(lines, fmt.Sprintf("🤖 Daemon:        %s", daemonText))

	// Service summary
	if m.ServiceSt != nil {
		svcText := "Not installed"
		if m.ServiceSt.Installed {
			if m.ServiceSt.Running {
				svcText = fmt.Sprintf("Running (%s)", m.ServiceSt.Platform)
			} else {
				svcText = fmt.Sprintf("Installed (Stopped) - %s", m.ServiceSt.Platform)
			}
		}
		lines = append(lines, fmt.Sprintf("⚙️  OS Service:    %s", svcText))
	}

	lines = append(lines, fmt.Sprintf("📁 Tracked Items: %d configured", len(m.Config.Tracked)))

	// Sync logs / Errors section
	if m.LastSync != nil {
		lines = append(lines, "")
		if len(m.LastSync.Errors) > 0 {
			lines = append(lines, fmt.Sprintf("🚨 LAST SYNC ISSUES (%d errors at %s):", len(m.LastSync.Errors), m.LastSync.Timestamp.Format("15:04:05")))
			for _, errStr := range m.LastSync.Errors {
				lines = append(lines, fmt.Sprintf("   ⚠️  %s", errStr))
			}
		} else if m.LastSync.Success {
			lines = append(lines, fmt.Sprintf("✅ Last sync succeeded at %s (%d files linked)", m.LastSync.Timestamp.Format("15:04:05"), len(m.LastSync.AppliedFiles)))
		}
	}

	body := strings.Join(lines, "\n")
	return m.Styles.Card.Width(m.Width - 4).Render(body)
}

func (m AppModel) renderFiles() string {
	if len(m.Config.Tracked) == 0 {
		return m.Styles.Card.Width(m.Width - 4).Render("📦 No dotfiles tracked yet!\nGo to '✨ Suggestions' (Tab 3) or '📂 File Browser' (Tab 4) to add your dotfiles.")
	}

	maxVisible := m.Height - 11
	if maxVisible < 6 {
		maxVisible = 6
	}

	start, end := getVisibleWindow(len(m.Config.Tracked), m.FileCursor, maxVisible)

	var rows []string
	header := fmt.Sprintf("📦 Tracked Dotfiles (%d items total)", len(m.Config.Tracked))
	if len(m.Config.Tracked) > maxVisible {
		header += fmt.Sprintf(" • [Item %d/%d]", m.FileCursor+1, len(m.Config.Tracked))
	}
	rows = append(rows, header+"\n")

	for i := start; i < end; i++ {
		it := m.Config.Tracked[i]
		statusStr := "✅ [LINKED]"
		if !it.Enabled {
			statusStr = "⏸️  [PAUSED]"
		}

		icon := getFileIcon(it.Path, it.IsDir, false)

		desc := it.Description
		if desc == "" {
			desc = "Personal config"
		}

		line := fmt.Sprintf("%s ~/%-28s %-12s %s", icon, truncateString(it.Path, 28), statusStr, desc)
		if i == m.FileCursor {
			rows = append(rows, m.Styles.SelectedRow.Render("❯ "+line))
		} else {
			rows = append(rows, m.Styles.NormalRow.Render("  "+line))
		}
	}

	help := m.Styles.HelpText.Render("Commands: [Space/Enter] Toggle Sync  •  [d/x] Untrack  •  [j/k] Scroll  •  [s] Sync Now")
	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(rows, "\n") + "\n\n" + help)
}

func (m AppModel) renderSuggestions() string {
	if len(m.Suggestions) == 0 {
		return m.Styles.Card.Width(m.Width - 4).Render("✨ Scanning complete. No additional standard dotfiles found.")
	}

	maxVisible := m.Height - 11
	if maxVisible < 6 {
		maxVisible = 6
	}

	start, end := getVisibleWindow(len(m.Suggestions), m.SuggestCursor, maxVisible)

	var rows []string
	header := fmt.Sprintf("✨ Auto-Discovered Dotfiles (%d detected)", len(m.Suggestions))
	if len(m.Suggestions) > maxVisible {
		header += fmt.Sprintf(" • [Item %d/%d]", m.SuggestCursor+1, len(m.Suggestions))
	}
	rows = append(rows, header+"\n")

	for i := start; i < end; i++ {
		s := m.Suggestions[i]
		icon := getFileIcon(s.Path, s.IsDir, s.IsSensitive)

		sensBadge := ""
		if s.IsSensitive {
			sensBadge = " [🔒 SENSITIVE]"
		}

		line := fmt.Sprintf("%s ~/%-26s [%s] %s%s", icon, truncateString(s.Path, 26), s.Category, s.Description, sensBadge)
		if i == m.SuggestCursor {
			rows = append(rows, m.Styles.SelectedRow.Render("❯ "+line))
		} else {
			rows = append(rows, m.Styles.NormalRow.Render("  "+line))
		}
	}

	help := m.Styles.HelpText.Render("Commands: [Enter/Space] Track Item  •  [j/k] Scroll  •  [s] Sync Now")
	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(rows, "\n") + "\n\n" + help)
}

func (m AppModel) renderBrowse() string {
	maxVisible := m.Height - 11
	if maxVisible < 6 {
		maxVisible = 6
	}

	start, end := getVisibleWindow(len(m.BrowseEntries), m.BrowseCursor, maxVisible)

	home, _ := os.UserHomeDir()
	displayPath := m.BrowsePath
	if strings.HasPrefix(displayPath, home) {
		displayPath = "~" + strings.TrimPrefix(displayPath, home)
	}

	scrollInfo := ""
	if len(m.BrowseEntries) > maxVisible {
		scrollInfo = fmt.Sprintf(" • [Item %d/%d]", m.BrowseCursor+1, len(m.BrowseEntries))
	}

	var rows []string
	rows = append(rows, fmt.Sprintf("📂 Path: %s (%d items)%s\n", displayPath, len(m.BrowseEntries), scrollInfo))

	for i := start; i < end; i++ {
		e := m.BrowseEntries[i]
		icon := getFileIcon(e.Name, e.IsDir, e.IsSensitive)

		status := " [➕ Add]"
		if e.Name == ".." {
			status = " [Open ↵]"
		} else if e.Tracked {
			status = " [✅ Tracked]"
		}

		typeLabel := "📄 file"
		if e.IsDir {
			typeLabel = "📁 dir "
		}

		line := fmt.Sprintf("%s %-32s %-8s %s", icon, truncateString(e.Name, 32), typeLabel, status)
		if i == m.BrowseCursor {
			rows = append(rows, m.Styles.SelectedRow.Render("❯ "+line))
		} else {
			rows = append(rows, m.Styles.NormalRow.Render("  "+line))
		}
	}

	help := m.Styles.HelpText.Render("Commands: [Enter] Open Folder / Toggle Track  •  [j/k] Scroll  •  [s] Sync Now")
	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(rows, "\n") + "\n\n" + help)
}

func (m AppModel) renderConflicts() string {
	if len(m.Conflicts) == 0 {
		return m.Styles.Card.Width(m.Width - 4).Render("✨ No active conflicts! Your dotfiles are in sync.")
	}

	var rows []string
	for i, c := range m.Conflicts {
		line := fmt.Sprintf("⚠️  Conflict in: %s", c.Path)
		if i == m.ConflictCursor {
			rows = append(rows, m.Styles.SelectedRow.Render("❯ "+line))
		} else {
			rows = append(rows, m.Styles.NormalRow.Render("  "+line))
		}
		// Render diff snippet
		diffPreview := c.Diff
		if len(diffPreview) > 200 {
			diffPreview = diffPreview[:200] + "\n..."
		}
		rows = append(rows, m.Styles.NormalRow.Render(diffPreview))
	}

	help := m.Styles.HelpText.Render("Commands: [Enter] Resolve (Keep Local)  •  [j/k] Navigate")
	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(rows, "\n") + "\n\n" + help)
}

func (m AppModel) renderLogs() string {
	if len(m.LogEntries) == 0 {
		return m.Styles.Card.Width(m.Width - 4).Render("📜 No log entries recorded yet.\nLogs are written to: " + logger.DefaultLogPath())
	}

	// Filter entries by search query
	var matched []logger.LogEntry
	query := strings.ToLower(strings.TrimSpace(m.LogSearchQuery))
	for _, entry := range m.LogEntries {
		if query == "" || strings.Contains(strings.ToLower(entry.Raw), query) {
			matched = append(matched, entry)
		}
	}

	maxVisible := m.Height - 12
	if maxVisible < 6 {
		maxVisible = 6
	}

	start, end := getVisibleWindow(len(matched), m.LogScrollOffset, maxVisible)

	var rows []string
	searchStatus := ""
	if m.IsSearchingLogs {
		searchStatus = fmt.Sprintf("🔍 Search: %s█ (Press Enter to apply, Esc to clear)", m.LogSearchQuery)
	} else if query != "" {
		searchStatus = fmt.Sprintf("🔍 Filter: \"%s\" (%d matched) • [Press / to edit, c to clear]", m.LogSearchQuery, len(matched))
	} else {
		searchStatus = fmt.Sprintf("📜 System Logs (%d entries) — Descending (Newest First) • [Press / to search]", len(m.LogEntries))
	}
	rows = append(rows, searchStatus+"\n")

	if len(matched) == 0 {
		rows = append(rows, fmt.Sprintf("  No logs match query \"%s\"", m.LogSearchQuery))
	} else {
		for i := start; i < end; i++ {
			entry := matched[i]
			timeStr := entry.Timestamp.Format("15:04:05")
			var levelBadge string
			switch entry.Level {
			case logger.LevelError:
				levelBadge = m.Styles.BadgeDanger.Render(string(entry.Level))
			case logger.LevelWarn:
				levelBadge = m.Styles.BadgeWarning.Render(string(entry.Level))
			case logger.LevelSync:
				levelBadge = m.Styles.BadgeSuccess.Render(string(entry.Level))
			default:
				levelBadge = m.Styles.BadgeInfo.Render(string(entry.Level))
			}

			line := fmt.Sprintf("%s %s %s", timeStr, levelBadge, entry.Message)
			if i == m.LogScrollOffset {
				rows = append(rows, m.Styles.SelectedRow.Render("❯ "+line))
			} else {
				rows = append(rows, m.Styles.NormalRow.Render("  "+line))
			}
		}
	}

	help := m.Styles.HelpText.Render("Commands: [/] Search Logs  •  [c] Clear Filter  •  [j/k] Scroll  •  [1-7] Switch Tabs  •  [s] Sync Now")
	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(rows, "\n") + "\n\n" + help)
}

func (m AppModel) renderSettings() string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Sync Mode:          %s (auto, manual, boot)", m.Config.SyncMode))
	lines = append(lines, fmt.Sprintf("Auto Interval:      %s", m.Config.Interval))
	lines = append(lines, fmt.Sprintf("Conflict Strategy:  %s", m.Config.ConflictStrategy))
	lines = append(lines, fmt.Sprintf("Theme:              %s (system, dark, light)", m.Config.Theme))
	lines = append(lines, fmt.Sprintf("Config File:        %s", config.DefaultConfigPath()))
	lines = append(lines, fmt.Sprintf("Backup Directory:   %s", m.Config.BackupDir))
	lines = append(lines, "\nTo edit these settings, open the web UI (`dotsynx ui`) or edit the YAML config file directly.")

	return m.Styles.Card.Width(m.Width - 4).Render(strings.Join(lines, "\n"))
}

func (m AppModel) renderFooter() string {
	keys := []string{
		"[s] Sync Now",
		"[1-7/Tab] Switch Tabs",
		"[u] Open Web UI",
		"[q] Quit",
	}
	return m.Styles.HelpText.Render(strings.Join(keys, "  •  "))
}

func getVisibleWindow(totalCount, cursor, maxVisible int) (start, end int) {
	if totalCount <= maxVisible {
		return 0, totalCount
	}
	start = cursor - (maxVisible / 2)
	if start < 0 {
		start = 0
	}
	end = start + maxVisible
	if end > totalCount {
		end = totalCount
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}
	return start, end
}

func truncateString(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-3] + "..."
}

func getFileIcon(name string, isDir bool, isSensitive bool) string {
	if isSensitive {
		return "🔒"
	}
	if isDir {
		if name == ".." {
			return "⬆️ "
		}
		switch name {
		case ".config":
			return "⚙️ "
		case ".git":
			return "🐙"
		case ".ssh":
			return "🔑"
		case "fish":
			return "🐚"
		case "nvim", "vim", "zed", "helix":
			return "📝"
		default:
			return "📁"
		}
	}

	ext := strings.ToLower(filepath.Ext(name))
	base := strings.ToLower(filepath.Base(name))

	switch {
	case strings.HasPrefix(base, ".zsh") || strings.HasPrefix(base, ".bash") || ext == ".fish" || ext == ".sh":
		return "🐚"
	case strings.HasPrefix(base, ".git") || base == ".gitignore":
		return "🐙"
	case base == ".tmux.conf" || ext == ".conf":
		return "💻"
	case ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".toml" || ext == ".ini":
		return "⚙️ "
	case ext == ".md" || ext == ".txt":
		return "📄"
	case ext == ".lua" || ext == ".vim" || base == ".vimrc":
		return "📝"
	case ext == ".go":
		return "🐹"
	case ext == ".rs":
		return "🦀"
	case ext == ".py":
		return "🐍"
	case ext == ".js" || ext == ".ts":
		return "⚡"
	default:
		return "📄"
	}
}
