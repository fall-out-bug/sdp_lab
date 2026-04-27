package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fall-out-bug/sdp_lab/internal/convoy"
	"github.com/fall-out-bug/sdp_lab/internal/guard"
	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "sync":
		syncCmd(args)
	case "scope":
		scopeCmd(args)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("gt-adapter - Gas Town to SDP bridge")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gt-adapter sync <subcommand>    Sync Gas Town state to SDP")
	fmt.Println("    convoys                      Sync convoys to workstreams")
	fmt.Println("  gt-adapter scope <task-json>   Write guard scope file for worktree")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  SDP_ROOT        Project root (auto-detected if empty)")
	fmt.Println("  GT_DATA_DIR     Gas Town data directory")
}

func syncCmd(args []string) {
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "error: sync requires a subcommand\n")
		fmt.Println("Available subcommands: convoys")
		os.Exit(1)
	}

	subcommand := args[0]
	switch subcommand {
	case "convoys":
		if err := syncConvoys(); err != nil {
			fmt.Fprintf(os.Stderr, "error: sync convoys: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown sync subcommand %q\n", subcommand)
		fmt.Println("Available subcommands: convoys")
		os.Exit(1)
	}
}

func scopeCmd(args []string) {
	flagSet := flag.NewFlagSet("scope", flag.ExitOnError)
	worktreePath := flagSet.String("worktree", "", "Path to worktree (default: current directory)")
	if err := flagSet.Parse(args); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse scope flags: %v\n", err)
		os.Exit(2)
	}

	if flagSet.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "error: scope requires task JSON\n")
		os.Exit(1)
	}

	taskJSON := flagSet.Arg(0)
	if err := writeScopeFile(taskJSON, *worktreePath); err != nil {
		fmt.Fprintf(os.Stderr, "error: write scope file: %v\n", err)
		os.Exit(1)
	}
}

func syncConvoys() error {
	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	root, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	// Parse convoy list
	parser := convoy.NewParser()
	response, err := parser.ParseConvoyList()
	if err != nil {
		return fmt.Errorf("parse convoy list: %w", err)
	}

	// Generate workstream files
	backlogDir := filepath.Join(root, "docs/workstreams/backlog")
	gen := convoy.NewWorkstreamGenerator(backlogDir)
	generated, err := gen.GenerateAll(response.Convoys)
	if err != nil {
		return fmt.Errorf("generate workstreams: %w", err)
	}

	fmt.Printf("Generated %d workstream file(s)\n", len(generated))
	for _, path := range generated {
		fmt.Printf("  - %s\n", path)
	}

	return nil
}

func writeScopeFile(taskJSON, worktreePath string) error {
	// Find project root
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get cwd: %w", err)
	}
	root, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	// Default worktree path to current directory if not specified
	if worktreePath == "" {
		worktreePath = wd
	}

	// Parse task metadata
	var task guard.TaskMetadata
	if err := json.Unmarshal([]byte(taskJSON), &task); err != nil {
		return fmt.Errorf("parse task JSON: %w", err)
	}

	// Write scope file
	writer := guard.NewScopeWriter(root)
	if err := writer.WriteScopeFile(worktreePath, task); err != nil {
		return fmt.Errorf("write scope file: %w", err)
	}

	fmt.Printf("Guard scope file written to %s\n", filepath.Join(worktreePath, ".sdp/guard-scope.json"))
	return nil
}
