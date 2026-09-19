//go:build !darwin && !linux

package service

import "fmt"

func newPlatformManager() (Manager, error) {
	return nil, fmt.Errorf("service management is currently only supported on macOS (launchd) and Linux (systemd)")
}
