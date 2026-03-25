package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"sdp_dev/internal/control"
	"sdp_dev/internal/executor"
)

func main() {
	projectRoot := flag.String("project-root", ".", "project root")
	interval := flag.Duration("interval", 30*time.Second, "time between orchestration cycles")
	maxCycles := flag.Int("max-cycles", 0, "maximum cycles to run; 0 means infinite")
	execute := flag.Bool("execute", false, "wire the executor bridge and run dispatched work")
	flag.Parse()

	store, err := control.OpenFromEnv(*projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err)
		os.Exit(1)
	}

	var bridge *executor.ExecutorBridge
	if *execute {
		bridge = &executor.ExecutorBridge{Store: store, ProjectRoot: *projectRoot}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := executor.RunOrchestrateLoop(ctx, store, bridge, *projectRoot, *interval, *maxCycles); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "error: run orchestrate loop: %v\n", err)
		os.Exit(1)
	}
}
