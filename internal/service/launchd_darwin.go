package service

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const LaunchdServiceName = "com.dotsynx.daemon"

type LaunchdManager struct {
	plistPath string
}

func NewLaunchdManager() *LaunchdManager {
	home, _ := os.UserHomeDir()
	return &LaunchdManager{
		plistPath: filepath.Join(home, "Library", "LaunchAgents", LaunchdServiceName+".plist"),
	}
}

func newPlatformManager() (Manager, error) {
	return NewLaunchdManager(), nil
}

func (m *LaunchdManager) Install(binaryPath string, syncMode string) error {
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			return err
		}
	}

	home, _ := os.UserHomeDir()
	logDir := filepath.Join(home, ".local", "share", "dotsynx")
	_ = os.MkdirAll(logDir, 0755)
	stdoutLog := filepath.Join(logDir, "daemon.stdout.log")
	stderrLog := filepath.Join(logDir, "daemon.stderr.log")

	// Determine command arguments based on syncMode
	// If "boot", run dotsynx sync once at load without continuous daemon loop
	// If "auto", run dotsynx start --daemon
	var argsTag string
	var keepAliveTag string

	if syncMode == "boot" {
		argsTag = fmt.Sprintf(`        <string>%s</string>
        <string>sync</string>`, binaryPath)
		keepAliveTag = "    <key>RunAtLoad</key>\n    <true/>"
	} else {
		argsTag = fmt.Sprintf(`        <string>%s</string>
        <string>start</string>
        <string>--foreground</string>`, binaryPath)
		keepAliveTag = `    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>`
	}

	plistContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>%s</string>
    <key>ProgramArguments</key>
    <array>
%s
    </array>
%s
    <key>StandardOutPath</key>
    <string>%s</string>
    <key>StandardErrorPath</key>
    <string>%s</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/opt/homebrew/bin:%s/.local/bin</string>
    </dict>
</dict>
</plist>
`, LaunchdServiceName, argsTag, keepAliveTag, stdoutLog, stderrLog, home)

	if err := os.MkdirAll(filepath.Dir(m.plistPath), 0755); err != nil {
		return err
	}

	// Unload if previously loaded
	_ = m.Stop()

	if err := os.WriteFile(m.plistPath, []byte(plistContent), 0644); err != nil {
		return fmt.Errorf("failed to write launchd plist: %w", err)
	}

	return m.Start()
}

func (m *LaunchdManager) Uninstall() error {
	_ = m.Stop()
	if _, err := os.Stat(m.plistPath); err == nil {
		return os.Remove(m.plistPath)
	}
	return nil
}

func (m *LaunchdManager) Start() error {
	cmd := exec.Command("launchctl", "load", "-w", m.plistPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load failed: %w (out: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *LaunchdManager) Stop() error {
	cmd := exec.Command("launchctl", "unload", "-w", m.plistPath)
	_ = cmd.Run()
	return nil
}

func (m *LaunchdManager) Status() (*Status, error) {
	st := &Status{
		Platform:    "darwin (launchd)",
		ServiceFile: m.plistPath,
	}

	if _, err := os.Stat(m.plistPath); os.IsNotExist(err) {
		st.Installed = false
		st.Message = "Service is not installed"
		return st, nil
	}

	st.Installed = true

	// Check launchctl list
	cmd := exec.Command("launchctl", "list", LaunchdServiceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		st.Running = false
		st.Message = "Service is stopped"
		return st, nil
	}

	st.Running = true
	st.Message = "Service is running"

	// Parse PID
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"PID\" = ") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				cleanPID := strings.Trim(parts[1], " ;\"")
				pid, _ := strconv.Atoi(cleanPID)
				st.PID = pid
			}
		}
	}

	return st, nil
}
