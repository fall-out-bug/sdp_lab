package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/executor"
)

func runOrchestrate(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp orchestrate once")
		os.Exit(2)
	}
	switch args[0] {
	case "loop":
		runOrchestrateLoop(args[1:])
	case "once":
		runOrchestrateOnce(args[1:])
	default:
		fmt.Fprintln(os.Stderr, "usage: sdp orchestrate once")
		os.Exit(2)
	}
}

func runOrchestrateOnce(args []string) {
	fs := flag.NewFlagSet("orchestrate-once", flag.ExitOnError)
	_ = fs.Parse(args)

	store := openStore()
	result, err := store.OrchestrateOnce()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: orchestrate once: %v\n", err)
		os.Exit(1)
	}

	switch result.Action {
	case "ingested":
		fmt.Printf("✅ %s\n", result.Message)
		if result.IngestedCard != nil {
			fmt.Printf("   Card: %s/%s\n", result.IngestedCard.ProjectID, result.IngestedCard.ID)
		}
	case "dispatched":
		fmt.Printf("✅ %s\n", result.Message)
		if result.DispatchedCard != nil {
			fmt.Printf("   Project: %s | Card: %s\n", result.DispatchedCard.ProjectID, result.DispatchedCard.ID)
		}
		if result.ExecutorRole != "" {
			fmt.Printf("   Executor: %s\n", result.ExecutorRole)
		}
		if result.PacketPath != "" {
			fmt.Printf("   Packet: %s\n", result.PacketPath)
		}
	default:
		fmt.Printf("⏸️  %s\n", result.Message)
		if result.NoActionReason != "" {
			fmt.Printf("   Reason: %s\n", result.NoActionReason)
		}
	}

	printJSON(result)
}

func runOrchestrateLoop(args []string) {
	fs := flag.NewFlagSet("orchestrate-loop", flag.ExitOnError)
	cycles := fs.Int("cycles", 1, "number of cycles to run")
	interval := fs.Duration("interval", 0, "interval between cycles")
	_ = fs.Parse(args)

	store := openStore()
	projectRoot := store.ProjectRoot
	ctx := context.Background()

	err := executor.RunOrchestrateLoopV2(ctx, store, projectRoot, *interval, *cycles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: orchestrate loop: %v\n", err)
		os.Exit(1)
	}
}
