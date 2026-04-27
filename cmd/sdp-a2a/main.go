//go:build sdp_experimental

package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/a2a"
	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/executor"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	projectRoot := flag.String("project-root", ".", "project root")
	apiKey := flag.String("api-key", "", "optional bearer token for API access")
	flag.Parse()

	store, err := control.OpenFromEnv(*projectRoot)
	if err != nil {
		log.Fatalf("open control store: %v", err)
	}

	server := &a2a.Server{
		Store:       store,
		Bridge:      &executor.ExecutorBridge{Store: store, ProjectRoot: *projectRoot},
		ProjectRoot: *projectRoot,
		Addr:        *addr,
		APIKey:      *apiKey,
	}

	log.Printf("sdp-a2a listening on %s", *addr)
	srv := &http.Server{
		Addr:         *addr,
		Handler:      server,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
