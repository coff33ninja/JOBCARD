//go:build windows

// Package main contains the JobCard executable's Windows desktop integration.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

func openBrowserURL(rawURL string) error {
	return shellOpen(rawURL)
}

func openLocation(path string) error {
	return shellOpen(path)
}

func shellOpen(target string) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return fmt.Errorf("encode browser action: %w", err)
	}
	targetPtr, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return fmt.Errorf("encode shell target: %w", err)
	}

	shellExecute := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	result, _, _ := shellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(targetPtr)),
		0,
		0,
		1,
	)
	if result <= 32 {
		return fmt.Errorf("could not open %q with ShellExecuteW (result %d)", target, result)
	}
	return nil
}

func selectFolder(initial, description string) (string, error) {
	const script = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
$dialog = New-Object System.Windows.Forms.FolderBrowserDialog
$dialog.Description = $env:JOBCARD_PICKER_DESCRIPTION
$dialog.SelectedPath = $env:JOBCARD_PICKER_INITIAL
$dialog.ShowNewFolderButton = $true
if ($dialog.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
    [Console]::Out.Write($dialog.SelectedPath)
}`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Env = append(os.Environ(),
		"JOBCARD_PICKER_INITIAL="+initial,
		"JOBCARD_PICKER_DESCRIPTION="+description,
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("open Windows folder picker: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func showTrayError(title, message string) {
	titlePtr, titleErr := syscall.UTF16PtrFromString(title)
	messagePtr, messageErr := syscall.UTF16PtrFromString(message)
	if titleErr != nil || messageErr != nil {
		return
	}
	messageBox := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW")
	messageBox.Call(0, uintptr(unsafe.Pointer(messagePtr)), uintptr(unsafe.Pointer(titlePtr)), 0x10)
}

func restartApplication(config appConfig) error {
	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find JobCard executable: %w", err)
	}
	cmd := exec.Command(executable, restartArgs(os.Args[1:], config)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start JobCard: %w", err)
	}
	return nil
}
