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
)

func main() {
	var (
		addr       = flag.String("addr", "127.0.0.1:8080", "listen address (keep it on localhost: there is no authentication)")
		dataDir    = flag.String("data-dir", "", "data folder (default: <user config dir>\\JobCard, e.g. %APPDATA%\\JobCard)")
		clientsDir = flag.String("clients-dir", "", "where client folders and exported files go (default: <data-dir>\\Clients)")
		seedPath   = flag.String("seed", "", "DEVELOPMENT ONLY: load sample data from this seed.json into an empty database")
		todayFlag  = flag.String("today", "", "override today's date (YYYY-MM-DD) for testing")
		static     = flag.String("static", "", "optional folder with the built UI (dist) to serve at /")
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

	srv := &http.Server{Addr: *addr, Handler: (&api.Server{Svc: svc, Static: *static}).Handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("JobCard API on http://%s/api/v1 | data: %s | clients: %s", *addr, *dataDir, *clientsDir)
	log.Fatal(srv.ListenAndServe())
}
