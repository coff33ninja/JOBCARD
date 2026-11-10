// Package main contains the JobCard executable's startup configuration helpers.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type appConfig struct {
	DataDir      string `json:"dataDir"`
	ClientsDir   string `json:"clientsDir"`
	TemplatesDir string `json:"templatesDir"`
}

func defaultAppConfig(configRoot string) appConfig {
	dataDir := filepath.Join(configRoot, "JobCard")
	return appConfig{
		DataDir:      dataDir,
		ClientsDir:   filepath.Join(dataDir, "Clients"),
		TemplatesDir: filepath.Join(dataDir, "Templates"),
	}
}

func appConfigFile(configRoot string) string {
	return filepath.Join(configRoot, "JobCardLauncher", "config.json")
}

func applyFolderOverrides(config appConfig, dataDir, clientsDir, templatesDir string) appConfig {
	if dataDir != "" {
		config.DataDir = dataDir
		if clientsDir == "" {
			config.ClientsDir = filepath.Join(dataDir, "Clients")
		}
		if templatesDir == "" {
			config.TemplatesDir = filepath.Join(dataDir, "Templates")
		}
	}
	if clientsDir != "" {
		config.ClientsDir = clientsDir
	}
	if templatesDir != "" {
		config.TemplatesDir = templatesDir
	}
	return config
}

func withoutFolderOverrides(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		skipValue := false
		for _, name := range []string{"--data-dir", "-data-dir", "--clients-dir", "-clients-dir", "--templates-dir", "-templates-dir"} {
			if arg == name {
				skipValue = true
				break
			}
			if strings.HasPrefix(arg, name+"=") {
				skipValue = true
				break
			}
		}
		if skipValue {
			if !strings.Contains(arg, "=") && i+1 < len(args) {
				i++
			}
			continue
		}
		out = append(out, arg)
	}
	return out
}

func restartArgs(args []string, config appConfig) []string {
	out := withoutFolderOverrides(args)
	out = append(out,
		"--data-dir", config.DataDir,
		"--clients-dir", config.ClientsDir,
		"--templates-dir", config.TemplatesDir,
	)
	return out
}

func loadAppConfig(path string, defaults appConfig) (appConfig, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return appConfig{}, fmt.Errorf("read app config: %w", err)
	}
	var config appConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return appConfig{}, fmt.Errorf("parse app config: %w", err)
	}
	if strings.TrimSpace(config.DataDir) == "" {
		config.DataDir = defaults.DataDir
	}
	if strings.TrimSpace(config.ClientsDir) == "" {
		config.ClientsDir = filepath.Join(config.DataDir, "Clients")
	}
	if strings.TrimSpace(config.TemplatesDir) == "" {
		config.TemplatesDir = filepath.Join(config.DataDir, "Templates")
	}
	return config, nil
}

func saveAppConfig(path string, config appConfig) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create app config directory: %w", err)
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app config: %w", err)
	}
	data = append(data, '\n')
	temp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary app config: %w", err)
	}
	tempPath := temp.Name()
	defer os.Remove(tempPath)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return fmt.Errorf("set app config permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return fmt.Errorf("write temporary app config: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close temporary app config: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return fmt.Errorf("write app config: %w", err)
	}
	return nil
}

func rebaseConfiguredPath(path, oldRoot, newRoot string) string {
	rel, err := filepath.Rel(oldRoot, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return path
	}
	return filepath.Join(newRoot, rel)
}

func pathIsWithin(root, candidate string) (bool, error) {
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		if !strings.EqualFold(filepath.VolumeName(root), filepath.VolumeName(candidate)) {
			return false, nil
		}
		return false, err
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel), nil
}

func moveDataDirectory(source, destination string) error {
	source, err := filepath.Abs(filepath.Clean(source))
	if err != nil {
		return fmt.Errorf("resolve current data directory: %w", err)
	}
	destination, err = filepath.Abs(filepath.Clean(destination))
	if err != nil {
		return fmt.Errorf("resolve selected data directory: %w", err)
	}
	if samePath(source, destination) {
		return nil
	}
	withinSource, err := pathIsWithin(source, destination)
	if err != nil {
		return fmt.Errorf("compare data directories: %w", err)
	}
	if withinSource {
		return errors.New("selected data directory cannot be inside the current data directory")
	}

	if entries, err := os.ReadDir(destination); err == nil {
		if len(entries) != 0 {
			return fmt.Errorf("selected data directory is not empty: %s", destination)
		}
		if err := os.Remove(destination); err != nil {
			return fmt.Errorf("prepare selected data directory: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("inspect selected data directory: %w", err)
	}
	sourceInfo, err := os.Stat(source)
	if errors.Is(err, fs.ErrNotExist) {
		return os.MkdirAll(destination, 0o755)
	}
	if err != nil {
		return fmt.Errorf("inspect current data directory: %w", err)
	}
	if !sourceInfo.IsDir() {
		return errors.New("current data path is not a directory")
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return fmt.Errorf("create selected data directory parent: %w", err)
	}
	if err := os.Rename(source, destination); err == nil {
		return nil
	}

	if err := copyDataTree(source, destination); err != nil {
		_ = os.RemoveAll(destination)
		return fmt.Errorf("copy data directory to selected location: %w", err)
	}
	if err := os.RemoveAll(source); err != nil {
		_ = os.RemoveAll(destination)
		return fmt.Errorf("remove old data directory after copy: %w", err)
	}
	return nil
}

func copyDataTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to move symbolic link: %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type in data directory: %s", path)
		}
		if err := copyDataFile(path, target, info.Mode().Perm()); err != nil {
			return err
		}
		return nil
	})
}

func copyDataFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := out.ReadFrom(in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func samePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
