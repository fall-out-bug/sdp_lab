package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/executor"
	"sdp_dev/internal/orchestrate"
)

func runDispatch(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp dispatch card")
		os.Exit(2)
	}
	switch args[0] {
	case "card":
		runDispatchCard(args[1:])
	case "next":
		runDispatchNext(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown dispatch subcommand: %s\n", args[0])
		os.Exit(1)
	}
}

func runDispatchCard(args []string) {
	fs := flag.NewFlagSet("dispatch-card", flag.ExitOnError)
	project := fs.String("project", "", "project id")
	id := fs.String("id", "", "card id")
	_ = fs.Parse(args)
	if *project == "" || *id == "" {
		fmt.Fprintln(os.Stderr, "error: --project and --id are required")
		os.Exit(2)
	}
	store := openStore()
	card, err := store.DispatchCard(*project, *id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: dispatch card: %v\n", err)
		os.Exit(1)
	}
	printJSON(card)
}

func runDispatchNext(args []string) {
	fs := flag.NewFlagSet("dispatch-next", flag.ExitOnError)
	execute := fs.Bool("execute", false, "Dispatch, execute through the bridge, then auto-ingest the result")
	_ = fs.Parse(args)

	store := openStore()
	result, err := store.DispatchNext()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: dispatch next: %v\n", err)
		os.Exit(1)
	}

	if result.Success {
		fmt.Printf("✅ %s\n", result.Message)
		if result.ProjectID != "" && result.CardID != "" {
			fmt.Printf("   Project: %s | Card: %s\n", result.ProjectID, result.CardID)
		}
		if result.ExecutorRole != "" {
			fmt.Printf("   Executor: %s\n", result.ExecutorRole)
		}
		if result.PacketPath != "" {
			fmt.Printf("   Packet: %s\n", result.PacketPath)
		}
		if *execute {
			bridge := &executor.ExecutorBridge{Store: store, Invoker: orchestrate.DefaultLLMInvoker, ProjectRoot: store.ProjectRoot}
			execResult, err := bridge.DispatchAndRun(context.Background(), result.ProjectID, result.CardID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: execute dispatched card: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("   Execution result: %s\n", execResult.Status)
			if _, err := store.OrchestrateOnce(); err != nil {
				fmt.Fprintf(os.Stderr, "error: auto-ingest executor result: %v\n", err)
				os.Exit(1)
			}
		}
	} else {
		fmt.Printf("⏸️  %s\n", result.Message)
		if result.NoDispatchableReason != "" {
			fmt.Printf("   Reason: %s\n", result.NoDispatchableReason)
		}
	}

	printJSON(result)
}
