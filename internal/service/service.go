package service

import (
	"fmt"
	"os/exec"
)

// Status represents the OS service status
type Status struct {
	Installed   bool   `json:"installed"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid,omitempty"`
	Platform    string `json:"platform"`
	ServiceFile string `json:"service_file"`
	Message     string `json:"message"`
}

// Manager defines the interface for OS background service control
type Manager interface {
	Install(binaryPath string, syncMode string) error
	Uninstall() error
	Start() error
	Stop() error
	Status() (*Status, error)
}

// GetManager returns the native OS service manager
func GetManager() (Manager, error) {
	return newPlatformManager()
}

// FindBinaryPath locates the dotsynx executable path
func FindBinaryPath() (string, error) {
	path, err := exec.LookPath("dotsynx")
	if err != nil {
		return "", fmt.Errorf("dotsynx binary not found in PATH: %w", err)
	}
	return path, nil
}
