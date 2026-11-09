// Command jobcard runs the JobCard backend: HTTP API on localhost, SQLite storage in the user profile.
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"jobcard/internal/api"
	"jobcard/internal/core"
	"jobcard/internal/domain"
	"jobcard/internal/export"
	"jobcard/internal/mcp"
)

const version = "0.1.0"

func main() {
	var (
		addr         = flag.String("addr", "127.0.0.1:8080", "listen address (keep it on localhost: there is no authentication)")
		dataDir      = flag.String("data-dir", "", "data folder (default: <user config dir>\\JobCard, e.g. %APPDATA%\\JobCard)")
		clientsDir   = flag.String("clients-dir", "", "where client folders and exported files go (default: <data-dir>\\Clients)")
		templatesDir = flag.String("templates-dir", "", "editable .docx templates (default: <data-dir>\\Templates); missing ones are created, existing ones never overwritten")
		mcpMode      = flag.Bool("mcp", false, "run as an MCP server on stdin/stdout instead of the HTTP API (for Claude Desktop); logs go to stderr")
		seedPath     = flag.String("seed", "", "DEVELOPMENT ONLY: load sample data from this seed.json into an empty database")
		todayFlag    = flag.String("today", "", "override today's date (YYYY-MM-DD) for testing")
		static       = flag.String("static", "", "optional folder with the built UI (dist) to serve at /")
		token        = flag.String("token", "", "require this bearer token on every API request (or set JOBCARD_TOKEN); mandatory when --addr is not localhost")
	)
	flag.Parse()

	if *dataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			log.Fatalf("cannot find the user config directory: %v", err)
		}
		*dataDir = filepath.Join(base, "JobCard")
	}
	if *clientsDir == "" {
		*clientsDir = filepath.Join(*dataDir, "Clients")
	}
	today := func() string { return time.Now().Format("2006-01-02") }
	if *todayFlag != "" {
		if !domain.ValidDate(*todayFlag) {
			log.Fatalf("--today must be YYYY-MM-DD")
		}
		fixed := *todayFlag
		today = func() string { return fixed }
	}

	svc, err := core.NewService(*dataDir, *clientsDir, today)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer svc.Close()

	if *templatesDir == "" {
		*templatesDir = filepath.Join(*dataDir, "Templates")
	}
	tpls := export.Templates{Dir: *templatesDir}
	if created, err := tpls.Ensure(); err != nil {
		log.Fatalf("templates: %v", err)
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
		log.Fatalf("refusing to listen on %s without authentication: set --token (or JOBCARD_TOKEN), or use a localhost address", *addr)
	}
	srv := &http.Server{Addr: *addr, Handler: (&api.Server{Svc: svc, Templates: tpls, Static: *static, Loopback: loopback, Token: *token}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("JobCard API on http://%s/api/v1 | data: %s | clients: %s | templates: %s", *addr, *dataDir, *clientsDir, *templatesDir)
	log.Fatal(srv.ListenAndServe())
}
