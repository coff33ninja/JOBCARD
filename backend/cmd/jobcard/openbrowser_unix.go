//go:build !windows

// Package main contains the JobCard executable's desktop integration.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

func openBrowserURL(rawURL string) error {
	return openLocation(rawURL)
}

func openLocation(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start browser: %w", err)
	}
	return nil
}

func selectFolder(string, string) (string, error) {
	return "", fmt.Errorf("folder selection is only supported in the Windows tray app")
}

func showTrayError(_, _ string) {}

func restartApplication(config appConfig) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find JobCard executable: %w", err)
	}
	cmd := exec.Command(executable, restartArgs(os.Args[1:], config)...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start JobCard: %w", err)
	}
	return nil
}
