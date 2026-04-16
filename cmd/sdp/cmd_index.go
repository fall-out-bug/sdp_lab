package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sdp_dev/internal/index"
)

func runIndex(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index <manifest|build|stats> [flags]")
		os.Exit(2)
	}
	switch args[0] {
	case "manifest":
		runIndexManifest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown index subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: sdp index <manifest|build|stats> [flags]")
		os.Exit(2)
	}
}

func runIndexManifest(args []string) {
	fs := flag.NewFlagSet("index manifest", flag.ExitOnError)
	output := fs.String("output", "", "output directory for manifest.md (default: <repo>/.sdp)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index manifest [--output DIR] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	// Determine .sdp directory
	sdpDir := *output
	if sdpDir == "" {
		sdpDir = filepath.Join(repoPath, ".sdp")
	}

	dbPath := filepath.Join(sdpDir, "index.db")
	info, err := os.Stat(dbPath)
	if err != nil || info.IsDir() {
		fmt.Fprintln(os.Stderr, "error: no index database found. Run 'sdp index build' first.")
		os.Exit(1)
	}

	store, err := index.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open index: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	path, err := index.WriteManifest(sdpDir, store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: generate manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "manifest: %s\n", path)
}
