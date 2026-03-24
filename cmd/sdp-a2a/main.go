package main

import (
	"flag"
	"log"
	"net/http"

	"sdp_dev/internal/a2a"
	"sdp_dev/internal/control"
	"sdp_dev/internal/executor"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	projectRoot := flag.String("project-root", ".", "project root")
	apiKey := flag.String("api-key", "", "optional bearer token for API access")
	flag.Parse()

	store, err := control.Open(*projectRoot)
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
	if err := http.ListenAndServe(*addr, server); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
