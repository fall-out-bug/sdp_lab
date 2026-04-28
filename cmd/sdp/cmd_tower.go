package main

import (
	"fmt"
	"os"

	"github.com/fall-out-bug/sdp_lab/internal/tower"
)

func runTower(args []string) {
	port := "8090"
	for i, a := range args {
		if a == "--port" && i+1 < len(args) {
			port = args[i+1]
		}
	}

	projectRoot := "."
	if d := os.Getenv("SDP_PROJECT_ROOT"); d != "" {
		projectRoot = d
	}

	if err := tower.Serve(projectRoot, port); err != nil {
		fmt.Fprintf(os.Stderr, "tower: %v\n", err)
		os.Exit(1)
	}
}
