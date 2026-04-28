package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/executor"
)

func main() {
	projectRoot := flag.String("project-root", ".", "project root")
	interval := flag.Duration("interval", 30*time.Second, "time between orchestration cycles")
	maxCycles := flag.Int("max-cycles", 0, "maximum cycles to run; 0 means infinite")
	flag.Parse()

	store, err := control.OpenFromEnv(*projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open control store: %v\n", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := executor.RunOrchestrateLoopV2(ctx, store, *projectRoot, *interval, *maxCycles); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "error: run orchestrate loop: %v\n", err)
		os.Exit(1)
	}
}
