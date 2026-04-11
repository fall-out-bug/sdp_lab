// Package main provides the sdp-harness CLI for running SDP phase turns
// and managing session lifecycle through the agentloop Harness.
//
// Usage:
//
//	sdp-harness new --session=<id>
//	  Creates a new session DB at $SDP_DATA_DIR/<id>.db (default: $HOME/.sdp/<id>.db).
//
//	sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]
//	  Runs one phase turn for the given session, streaming events to stdout.
//	  Restores an existing session or fails if session does not exist.
//
// Environment:
//
//	SDP_DATA_DIR  Directory for session DB files. Defaults to $HOME/.sdp.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/agentloop/livegw"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("no subcommand given")
	}

	switch args[0] {
	case "new":
		return cmdNew(args[1:])
	case "run":
		return cmdRun(args[1:])
	case "--help", "-h", "help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand: %q", args[0])
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `sdp-harness — SDP Mini-Harness CLI

Subcommands:
  new  --session=<id>                     Create a new session
  run  --session=<id> --prompt="<text>"   Run one phase turn

Environment:
  SDP_DATA_DIR   Directory for session DB files (default: $HOME/.sdp)`)
}

// dataDir returns the directory where session DB files are stored.
// Uses SDP_DATA_DIR env var; falls back to $HOME/.sdp.
func dataDir() (string, error) {
	if d := os.Getenv("SDP_DATA_DIR"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".sdp"), nil
}

// dbPath returns the path to the session DB file for the given sessionID.
func dbPath(sessionID string) (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create data dir %s: %w", dir, err)
	}
	return filepath.Join(dir, sessionID+".db"), nil
}

// cmdNew implements `sdp-harness new --session=<id>`.
// Creates a fresh session DB and persists the initial Session record.
func cmdNew(args []string) error {
	fs := flag.NewFlagSet("new", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	_, err = agentloop.NewSession(*sessionID, store)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	fmt.Printf("session %q created at %s\n", *sessionID, path)
	return nil
}

// cmdRun implements `sdp-harness run --session=<id> --prompt="<text>" [--token=<tok>]`.
// Restores or creates a session and runs one phase turn, streaming events to stdout.
func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	sessionID := fs.String("session", "", "Session ID (required)")
	prompt := fs.String("prompt", "", "User prompt for this phase turn (required)")
	token := fs.String("token", "", "Owner token (optional; required if session has ownerToken set)")
	fs.SetOutput(os.Stderr)
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *sessionID == "" {
		fs.Usage()
		return fmt.Errorf("--session is required")
	}
	if *prompt == "" {
		fs.Usage()
		return fmt.Errorf("--prompt is required")
	}

	path, err := dbPath(*sessionID)
	if err != nil {
		return err
	}

	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		return fmt.Errorf("open store at %s: %w", path, err)
	}
	defer store.Close()

	// Build a minimal router with no real tools (MVP placeholder).
	// LiveGateway connects to OpenRouter; requires OPENROUTER_API_KEY.
	registry := agentloop.NewToolRegistry(nil)
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	gw, err := livegw.New(apiKey, "")
	if err != nil {
		return fmt.Errorf("create LiveGateway: %w\n(hint: set OPENROUTER_API_KEY env var)", err)
	}
	router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gw, nil)
	gate := agentloop.NewGateEngine(nil, 0) // 0 → 5s default

	// Try to restore an existing session; if not found, error (use `new` first).
	h, err := agentloop.RestoreHarness(*sessionID, *token, store, router, gate, nil)
	if err != nil {
		return fmt.Errorf("restore session %q: %w\n(hint: use 'sdp-harness new --session=%s' to create it)", *sessionID, err, *sessionID)
	}

	// Run the phase turn with a background context (no timeout for MVP CLI).
	ctx := context.Background()
	if err := h.RunPhase(ctx, *prompt, *token); err != nil {
		return fmt.Errorf("run phase: %w", err)
	}

	fmt.Printf("phase turn complete for session %q\n", *sessionID)
	return nil
}
