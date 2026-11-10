package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"jobcard/internal/core"
	"jobcard/internal/export"
)

func TestAppConfigDefaultsAndRoundTrip(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "JobCard")
	defaults := appConfig{
		DataDir:      dataDir,
		ClientsDir:   filepath.Join(dataDir, "Clients"),
		TemplatesDir: filepath.Join(dataDir, "Templates"),
	}
	configPath := filepath.Join(root, "settings", "config.json")

	got, err := loadAppConfig(configPath, defaults)
	if err != nil || got != defaults {
		t.Fatalf("load defaults: got %#v, err %v", got, err)
	}

	custom := appConfig{
		DataDir:      filepath.Join(root, "Business"),
		ClientsDir:   filepath.Join(root, "Business", "Client Files"),
		TemplatesDir: filepath.Join(root, "Business", "Word Templates"),
	}
	if err := saveAppConfig(configPath, custom); err != nil {
		t.Fatal(err)
	}
	custom.ClientsDir = filepath.Join(root, "Moved Clients")
	if err := saveAppConfig(configPath, custom); err != nil {
		t.Fatal(err)
	}
	got, err = loadAppConfig(configPath, defaults)
	if err != nil || got != custom {
		t.Fatalf("load saved config: got %#v, err %v", got, err)
	}
}

func TestApplyDataTrayActionMovesDatabaseAndRemembersFolders(t *testing.T) {
	root := t.TempDir()
	oldData := filepath.Join(root, "old-data")
	newData := filepath.Join(root, "new-data")
	configPath := filepath.Join(root, "launcher", "config.json")
	config := appConfig{
		DataDir:      oldData,
		ClientsDir:   filepath.Join(oldData, "Clients"),
		TemplatesDir: filepath.Join(oldData, "Templates"),
	}
	svc, err := core.NewService(config.DataDir, config.ClientsDir, func() string { return "2026-11-09" })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (export.Templates{Dir: config.TemplatesDir}).Ensure(); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	next, err := applyTrayAction(trayAction{kind: "data", path: newData}, config, configPath, server.Config, svc)
	if err != nil {
		t.Fatal(err)
	}
	if next.DataDir != newData || next.ClientsDir != filepath.Join(newData, "Clients") || next.TemplatesDir != filepath.Join(newData, "Templates") {
		t.Fatalf("moved folder settings: %#v", next)
	}
	if _, err := os.Stat(filepath.Join(oldData, "jobcard.sqlite")); !os.IsNotExist(err) {
		t.Fatalf("old database should be moved, stat error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newData, "jobcard.sqlite")); err != nil {
		t.Fatalf("new database missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newData, "Templates", "invoice.docx")); err != nil {
		t.Fatalf("moved template missing: %v", err)
	}
	got, err := loadAppConfig(configPath, appConfig{})
	if err != nil || got != next {
		t.Fatalf("remembered config = %#v, err %v; want %#v", got, err, next)
	}
}

func TestApplyFolderOverridesKeepsRememberedPathsUnlessDataRootChanges(t *testing.T) {
	config := appConfig{
		DataDir:      `G:\Business\JobCard`,
		ClientsDir:   `G:\Business\Clients`,
		TemplatesDir: `G:\Business\Templates`,
	}
	got := applyFolderOverrides(config, `D:\JobCard`, "", "")
	if got.DataDir != `D:\JobCard` || got.ClientsDir != `D:\JobCard\Clients` || got.TemplatesDir != `D:\JobCard\Templates` {
		t.Fatalf("data override did not reset dependent defaults: %#v", got)
	}
	got = applyFolderOverrides(config, "", `D:\Client Files`, "")
	if got.DataDir != config.DataDir || got.ClientsDir != `D:\Client Files` || got.TemplatesDir != config.TemplatesDir {
		t.Fatalf("clients override changed unrelated paths: %#v", got)
	}
}

func TestWithoutFolderOverridesPreservesOtherArguments(t *testing.T) {
	args := []string{"--addr", "127.0.0.1:8081", "--data-dir", "old data", "-clients-dir=D:\\Clients", "--open=false", "--templates-dir", "old templates"}
	got := withoutFolderOverrides(args)
	want := []string{"--addr", "127.0.0.1:8081", "--open=false"}
	if len(got) != len(want) {
		t.Fatalf("filtered args = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("filtered args = %#v, want %#v", got, want)
		}
	}
}

func TestRestartArgsUsesRememberedPathsAndPreservesRuntimeOptions(t *testing.T) {
	config := appConfig{DataDir: `D:\JobCard`, ClientsDir: `D:\Clients`, TemplatesDir: `D:\Templates`}
	args := restartArgs([]string{"--addr", "127.0.0.1:8081", "--data-dir", "old", "--open=false"}, config)
	want := []string{"--addr", "127.0.0.1:8081", "--open=false", "--data-dir", config.DataDir, "--clients-dir", config.ClientsDir, "--templates-dir", config.TemplatesDir}
	if len(args) != len(want) {
		t.Fatalf("restart args = %#v, want %#v", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("restart args = %#v, want %#v", args, want)
		}
	}
}

func TestMoveDataDirectoryMovesContentsAndPreservesSourceOnConflict(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(filepath.Join(source, "Templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	template := filepath.Join(source, "Templates", "invoice.docx")
	if err := os.WriteFile(template, []byte("template"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveDataDirectory(source, destination); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(source); !os.IsNotExist(err) {
		t.Fatalf("source should be moved, stat error: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "Templates", "invoice.docx"))
	if err != nil || string(got) != "template" {
		t.Fatalf("destination content %q, err %v", got, err)
	}

	conflictSource := filepath.Join(root, "conflict-source")
	conflictDestination := filepath.Join(root, "conflict-destination")
	if err := os.MkdirAll(conflictSource, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(conflictDestination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conflictSource, "db.sqlite"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conflictDestination, "db.sqlite"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveDataDirectory(conflictSource, conflictDestination); err == nil {
		t.Fatal("expected destination conflict")
	}
	got, err = os.ReadFile(filepath.Join(conflictSource, "db.sqlite"))
	if err != nil || string(got) != "source" {
		t.Fatalf("source data should remain after conflict, got %q, err %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(conflictDestination, "db.sqlite"))
	if err != nil || string(got) != "keep" {
		t.Fatalf("destination data should not be overwritten, got %q, err %v", got, err)
	}
}

func TestMoveDataDirectoryAcceptsAnEmptyDestinationFolder(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "jobcard.sqlite"), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := moveDataDirectory(source, destination); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(destination, "jobcard.sqlite")); err != nil || string(got) != "db" {
		t.Fatalf("moved database = %q, err %v", got, err)
	}
}

func TestRebaseConfiguredPathOnlyWhenInsideMovedData(t *testing.T) {
	source := filepath.Join(t.TempDir(), "old")
	destination := filepath.Join(t.TempDir(), "new")
	inside := filepath.Join(source, "Templates")
	want := filepath.Join(destination, "Templates")
	if got := rebaseConfiguredPath(inside, source, destination); got != want {
		t.Fatalf("rebased path = %q, want %q", got, want)
	}
	outside := filepath.Join(t.TempDir(), "external-templates")
	if got := rebaseConfiguredPath(outside, source, destination); got != outside {
		t.Fatalf("external path changed: got %q, want %q", got, outside)
	}
}

func TestPathIsWithinAllowsDifferentVolumes(t *testing.T) {
	if filepath.Separator != '\\' {
		t.Skip("different drive letters are specific to Windows")
	}
	within, err := pathIsWithin(`C:\Users\coff3\AppData\Roaming\JobCard`, `G:\My Drive\BUSINESS`)
	if err != nil {
		t.Fatalf("pathIsWithin returned an error for different drives: %v", err)
	}
	if within {
		t.Fatal("a path on another drive cannot be inside the source directory")
	}
}
