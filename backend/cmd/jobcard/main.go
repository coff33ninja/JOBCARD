// Command jobcard runs the JobCard backend: HTTP API on localhost, SQLite storage in the user profile.
package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/systray"

	"jobcard/internal/api"
	"jobcard/internal/core"
	"jobcard/internal/domain"
	"jobcard/internal/export"
	"jobcard/internal/mcp"
)

const version = "0.1.0"

//go:embed dist
var bundledUI embed.FS

func main() {
	var (
		addr         = flag.String("addr", "127.0.0.1:8080", "listen address (keep it on localhost: there is no authentication)")
		dataDir      = flag.String("data-dir", "", "override the remembered data folder")
		clientsDir   = flag.String("clients-dir", "", "override the remembered client files folder")
		templatesDir = flag.String("templates-dir", "", "override the remembered editable templates folder")
		mcpMode      = flag.Bool("mcp", false, "run as an MCP server on stdin/stdout instead of the HTTP API (for Claude Desktop)")
		seedPath     = flag.String("seed", "", "DEVELOPMENT ONLY: load sample data from this seed.json into an empty database")
		todayFlag    = flag.String("today", "", "override today's date (YYYY-MM-DD) for testing")
		static       = flag.String("static", "", "override the embedded UI with a folder on disk (development only)")
		showTray     = flag.Bool("tray", runtime.GOOS == "windows", "show a tray icon with Open and Exit actions on supported platforms")
		openBrowser  = flag.Bool("open", true, "open the frontend in the default browser after startup")
		token        = flag.String("token", "", "require this bearer token on every API request (or set JOBCARD_TOKEN); mandatory when --addr is not localhost")
	)
	flag.Parse()

	configRoot, err := os.UserConfigDir()
	if err != nil {
		fatalStartup(fmt.Sprintf("Cannot find the user config directory: %v", err))
	}
	configPath := appConfigFile(configRoot)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		fatalStartup(fmt.Sprintf("Cannot create the application settings directory: %v", err))
	}
	logFile, err := os.OpenFile(filepath.Join(filepath.Dir(configPath), "jobcard.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		fatalStartup(fmt.Sprintf("Cannot open the application log: %v", err))
	}
	log.SetOutput(logFile)
	defer logFile.Close()

	config, err := loadAppConfig(configPath, defaultAppConfig(configRoot))
	if err != nil {
		fatalStartup(fmt.Sprintf("Cannot load application settings: %v", err))
	}
	config = applyFolderOverrides(config, *dataDir, *clientsDir, *templatesDir)
	*dataDir, *clientsDir, *templatesDir = config.DataDir, config.ClientsDir, config.TemplatesDir

	today := func() string { return time.Now().Format("2006-01-02") }
	if *todayFlag != "" {
		if !domain.ValidDate(*todayFlag) {
			fatalStartup("--today must be YYYY-MM-DD")
		}
		fixed := *todayFlag
		today = func() string { return fixed }
	}

	if err := os.MkdirAll(*clientsDir, 0o755); err != nil {
		fatalStartup(fmt.Sprintf("Cannot create the client files folder: %v", err))
	}
	svc, err := core.NewService(*dataDir, *clientsDir, today)
	if err != nil {
		fatalStartup(fmt.Sprintf("Cannot open the JobCard database: %v", err))
	}
	defer svc.Close()

	tpls := export.Templates{Dir: *templatesDir}
	if created, err := tpls.Ensure(); err != nil {
		fatalStartup(fmt.Sprintf("Cannot prepare the Word templates: %v", err))
	} else if len(created) > 0 {
		log.Printf("created default templates: %v", created)
	}

	if *seedPath != "" {
		seeded, err := svc.Seed(*seedPath)
		if err != nil {
			log.Fatalf("seed: %v", err)
		}
		if seeded {
			log.Printf("seeded sample data from %s", *seedPath)
		} else {
			log.Printf("seed skipped: database already has data")
		}
	}

	if *mcpMode {
		log.Printf("JobCard MCP server on stdio | data: %s | clients: %s | templates: %s", *dataDir, *clientsDir, *templatesDir)
		srv := &mcp.Server{Svc: svc, Templates: tpls, Version: version}
		if err := srv.Serve(os.Stdin, os.Stdout); err != nil {
			log.Fatalf("mcp: %v", err)
		}
		return
	}
	if *token == "" {
		*token = os.Getenv("JOBCARD_TOKEN")
	}
	loopback := api.IsLoopbackAddr(*addr)
	if !loopback && *token == "" {
		fatalStartup(fmt.Sprintf("Refusing to listen on %s without authentication. Set --token/JOBCARD_TOKEN or use localhost.", *addr))
	}
	if *static == "" {
		if extracted, err := extractBundledUI(); err == nil {
			*static = extracted
			log.Printf("using embedded frontend bundle from %s", *static)
		} else {
			*static = findStaticDir()
			if *static == "" {
				log.Printf("no bundled frontend found: %v", err)
			}
		}
	}
	uiURL := appURL(*addr)
	srv := &http.Server{Addr: *addr, Handler: (&api.Server{Svc: svc, Templates: tpls, Static: *static, Loopback: loopback, Token: *token}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("JobCard API on %s | data: %s | clients: %s | templates: %s", apiURL(*addr), *dataDir, *clientsDir, *templatesDir)
	if *static != "" {
		log.Printf("JobCard UI served from %s at %s", *static, uiURL)
	} else {
		log.Printf("JobCard UI: no bundled dist folder found; run `npm run build` then restart with `--static` or a dist folder next to the binary")
	}

	listener, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		fatalStartup(fmt.Sprintf("Cannot start the local JobCard server on %s: %v", srv.Addr, err))
	}
	go serveHTTP(srv, listener)
	if *openBrowser && *static != "" {
		if err := openBrowserURL(uiURL); err != nil {
			log.Printf("open browser: %v", err)
			showTrayError("JobCard", fmt.Sprintf("Could not open the web UI. You can still open it from the tray menu.\n\n%s", err))
		}
	}
	if *showTray && runtime.GOOS == "windows" {
		action, exiting := runTray(uiURL, config)
		if exiting {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := srv.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown: %v", err)
			}
			cancel()
			return
		}
		if action.kind != "" {
			restartConfig, err := applyTrayAction(action, config, configPath, srv, svc)
			if err != nil {
				log.Printf("apply tray action: %v", err)
				showTrayError("JobCard settings", err.Error())
				if restartErr := restartApplication(restartConfig); restartErr != nil {
					log.Printf("restart JobCard: %v", restartErr)
					showTrayError("JobCard", fmt.Sprintf("JobCard could not restart. Start it again manually.\n\n%s", restartErr))
				}
				return
			}
			if err := restartApplication(restartConfig); err != nil {
				log.Printf("restart JobCard: %v", err)
				showTrayError("JobCard", fmt.Sprintf("Settings were saved, but JobCard could not restart. Start it again manually.\n\n%s", err))
			}
		}
		return
	}
	if err := waitForShutdown(srv); err != nil {
		log.Fatal(err)
	}
}

func fatalStartup(message string) {
	log.Print(message)
	showTrayError("JobCard could not start", message)
	os.Exit(1)
}

func serveHTTP(srv *http.Server, listener net.Listener) {
	if err := srv.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("http server: %v", err)
	}
}

func waitForShutdown(srv *http.Server) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func apiURL(addr string) string {
	return "http://" + strings.TrimSpace(addr) + "/api/v1"
}

func appURL(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "http://127.0.0.1:8080"
	}
	if host == "" || host == "0.0.0.0" || host == "::" || host == "[::]" {
		host = "127.0.0.1"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return fmt.Sprintf("http://%s:%s", strings.Trim(host, "[]"), port)
}

func findStaticDir() string {
	searches := []string{
		"dist",
		filepath.Join("..", "dist"),
	}
	if exe, err := os.Executable(); err == nil {
		searches = append(searches,
			filepath.Join(filepath.Dir(exe), "dist"),
			filepath.Join(filepath.Dir(exe), "cmd", "jobcard", "dist"),
			filepath.Join(filepath.Dir(exe), "..", "dist"),
		)
	}
	for _, candidate := range searches {
		if candidate == "" {
			continue
		}
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			if _, err := os.Stat(filepath.Join(candidate, "index.html")); err == nil {
				return candidate
			}
		}
	}
	return ""
}

func extractBundledUI() (string, error) {
	entries, err := fs.ReadDir(bundledUI, "dist")
	if err != nil {
		return "", fmt.Errorf("read embedded dist: %w", err)
	}
	if len(entries) == 0 {
		return "", errors.New("embedded dist is empty")
	}
	tempDir, err := os.MkdirTemp("", "jobcard-ui-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	if err := copyEmbeddedDir(bundledUI, "dist", tempDir); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("copy embedded dist: %w", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "index.html")); err != nil {
		_ = os.RemoveAll(tempDir)
		return "", fmt.Errorf("embedded dist missing index.html: %w", err)
	}
	return tempDir, nil
}

func copyEmbeddedDir(root fs.FS, srcDir, dstDir string) error {
	return fs.WalkDir(root, srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dstDir, filepath.FromSlash(rel))
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := fs.ReadFile(root, path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

type trayAction struct {
	kind string
	path string
}

func runTray(ui string, config appConfig) (trayAction, bool) {
	var action trayAction
	exiting := false
	systray.Run(func() {
		systray.SetTitle("JobCard")
		systray.SetTooltip("JobCard")
		openItem := systray.AddMenuItem("Open JobCard", "Open the JobCard web UI")
		systray.AddSeparator()
		openData := systray.AddMenuItem("Open data folder", config.DataDir)
		changeData := systray.AddMenuItem("Move data to folder...", "Move current JobCard data; destination must be empty")
		openClients := systray.AddMenuItem("Open client files folder", config.ClientsDir)
		changeClients := systray.AddMenuItem("Choose client files folder...", "Change where exported client files are saved")
		openTemplates := systray.AddMenuItem("Open templates folder", config.TemplatesDir)
		changeTemplates := systray.AddMenuItem("Choose templates folder...", "Change which editable Word templates are used")
		systray.AddSeparator()
		exitItem := systray.AddMenuItem("Exit", "Exit the JobCard app")
		for {
			select {
			case <-openItem.ClickedCh:
				if err := openBrowserURL(ui); err != nil {
					log.Printf("open JobCard: %v", err)
					showTrayError("JobCard", err.Error())
				}
			case <-openData.ClickedCh:
				openTrayFolder(config.DataDir)
			case <-changeData.ClickedCh:
				action = chooseTrayFolder("data", config.DataDir, "Choose JobCard data folder")
				if action.kind != "" {
					systray.Quit()
					return
				}
			case <-openClients.ClickedCh:
				openTrayFolder(config.ClientsDir)
			case <-changeClients.ClickedCh:
				action = chooseTrayFolder("clients", config.ClientsDir, "Choose client files folder")
				if action.kind != "" {
					systray.Quit()
					return
				}
			case <-openTemplates.ClickedCh:
				openTrayFolder(config.TemplatesDir)
			case <-changeTemplates.ClickedCh:
				action = chooseTrayFolder("templates", config.TemplatesDir, "Choose editable templates folder")
				if action.kind != "" {
					systray.Quit()
					return
				}
			case <-exitItem.ClickedCh:
				exiting = true
				systray.Quit()
				return
			}
		}
	}, func() {})
	return action, exiting
}

func chooseTrayFolder(kind, current, description string) trayAction {
	selected, err := selectFolder(current, description)
	if err != nil {
		log.Printf("choose %s folder: %v", kind, err)
		showTrayError("JobCard settings", err.Error())
		return trayAction{}
	}
	if selected == "" || samePath(selected, current) {
		return trayAction{}
	}
	return trayAction{kind: kind, path: selected}
}

func openTrayFolder(path string) {
	if err := os.MkdirAll(path, 0o755); err != nil {
		log.Printf("create folder %s: %v", path, err)
		showTrayError("JobCard", err.Error())
		return
	}
	if err := openLocation(path); err != nil {
		log.Printf("open folder %s: %v", path, err)
		showTrayError("JobCard", err.Error())
	}
}

func applyTrayAction(action trayAction, config appConfig, configPath string, srv *http.Server, svc *core.Service) (appConfig, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	shutdownErr := srv.Shutdown(ctx)
	cancel()
	if shutdownErr != nil {
		return config, fmt.Errorf("stop the JobCard server before changing folders: %w", shutdownErr)
	}
	if err := svc.Close(); err != nil {
		return config, fmt.Errorf("close current database: %w", err)
	}

	next := config
	switch action.kind {
	case "data":
		if err := moveDataDirectory(config.DataDir, action.path); err != nil {
			return config, err
		}
		next.DataDir = action.path
		next.ClientsDir = rebaseConfiguredPath(config.ClientsDir, config.DataDir, action.path)
		next.TemplatesDir = rebaseConfiguredPath(config.TemplatesDir, config.DataDir, action.path)
	case "clients":
		if err := os.MkdirAll(action.path, 0o755); err != nil {
			return config, fmt.Errorf("create client files folder: %w", err)
		}
		next.ClientsDir = action.path
	case "templates":
		if err := os.MkdirAll(action.path, 0o755); err != nil {
			return config, fmt.Errorf("create templates folder: %w", err)
		}
		next.TemplatesDir = action.path
	default:
		return config, fmt.Errorf("unknown tray action %q", action.kind)
	}
	if err := saveAppConfig(configPath, next); err != nil {
		if action.kind == "data" {
			if rollbackErr := moveDataDirectory(next.DataDir, config.DataDir); rollbackErr != nil {
				return next, fmt.Errorf("save settings: %v; restore old data location: %w", err, rollbackErr)
			}
		}
		return config, err
	}
	return next, nil
}
