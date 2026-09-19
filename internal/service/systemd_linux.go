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

const SystemdServiceName = "dotsynx.service"

type SystemdManager struct {
	unitPath string
}

func NewSystemdManager() *SystemdManager {
	home, _ := os.UserHomeDir()
	return &SystemdManager{
		unitPath: filepath.Join(home, ".config", "systemd", "user", SystemdServiceName),
	}
}

func newPlatformManager() (Manager, error) {
	return NewSystemdManager(), nil
}

func (m *SystemdManager) Install(binaryPath string, syncMode string) error {
	if binaryPath == "" {
		var err error
		binaryPath, err = os.Executable()
		if err != nil {
			return err
		}
	}

	execCmd := fmt.Sprintf("%s start --foreground", binaryPath)
	restartPolicy := "Restart=on-failure\nRestartSec=10s"

	if syncMode == "boot" {
		execCmd = fmt.Sprintf("%s sync", binaryPath)
		restartPolicy = "Type=oneshot"
	}

	unitContent := fmt.Sprintf(`[Unit]
Description=dotsynx dotfile synchronization agent
After=network-online.target
Wants=network-online.target

[Service]
ExecStart=%s
%s
Environment="PATH=/usr/local/bin:/usr/bin:/bin:%s/.local/bin"

[Install]
WantedBy=default.target
`, execCmd, restartPolicy, os.Getenv("HOME"))

	if err := os.MkdirAll(filepath.Dir(m.unitPath), 0755); err != nil {
		return err
	}

	_ = m.Stop()

	if err := os.WriteFile(m.unitPath, []byte(unitContent), 0644); err != nil {
		return fmt.Errorf("failed to write systemd unit: %w", err)
	}

	// Reload systemd user daemon
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	_ = exec.Command("systemctl", "--user", "enable", SystemdServiceName).Run()

	return m.Start()
}

func (m *SystemdManager) Uninstall() error {
	_ = m.Stop()
	_ = exec.Command("systemctl", "--user", "disable", SystemdServiceName).Run()
	if _, err := os.Stat(m.unitPath); err == nil {
		_ = os.Remove(m.unitPath)
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}

func (m *SystemdManager) Start() error {
	cmd := exec.Command("systemctl", "--user", "start", SystemdServiceName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl start failed: %w (out: %s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (m *SystemdManager) Stop() error {
	cmd := exec.Command("systemctl", "--user", "stop", SystemdServiceName)
	_ = cmd.Run()
	return nil
}

func (m *SystemdManager) Status() (*Status, error) {
	st := &Status{
		Platform:    "linux (systemd user)",
		ServiceFile: m.unitPath,
	}

	if _, err := os.Stat(m.unitPath); os.IsNotExist(err) {
		st.Installed = false
		st.Message = "Service unit not installed"
		return st, nil
	}

	st.Installed = true

	// Check is-active
	cmd := exec.Command("systemctl", "--user", "is-active", SystemdServiceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	_ = cmd.Run()

	statusStr := strings.TrimSpace(out.String())
	if statusStr == "active" {
		st.Running = true
		st.Message = "Service is active and running"

		// Get PID
		pidCmd := exec.Command("systemctl", "--user", "show", "--property", "MainPID", "--value", SystemdServiceName)
		var pidOut bytes.Buffer
		pidCmd.Stdout = &pidOut
		if err := pidCmd.Run(); err == nil {
			pid, _ := strconv.Atoi(strings.TrimSpace(pidOut.String()))
			st.PID = pid
		}
	} else {
		st.Running = false
		st.Message = fmt.Sprintf("Service is %s", statusStr)
	}

	return st, nil
}
