package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/index"
)

func runIndex(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index <build|stats|manifest> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  sdp index build <repo-path>              Full index (cold start)")
		fmt.Fprintln(os.Stderr, "  sdp index stats <repo-path>              Show index statistics")
		fmt.Fprintln(os.Stderr, "  sdp index manifest <repo-path>           Generate .sdp/manifest.md")
		os.Exit(2)
	}
	switch args[0] {
	case "build":
		runIndexBuild(args[1:])
	case "stats":
		runIndexStats(args[1:])
	case "manifest":
		runIndexManifest(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown index subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: sdp index <build|stats|manifest> [flags]")
		os.Exit(2)
	}
}

func runIndexBuild(args []string) {
	fs := flag.NewFlagSet("index build", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	dbPath := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	maxSize := fs.Int64("max-size", 100*1024, "max file size in bytes (default 100KB)")
	langList := fs.String("languages", "", "comma-separated language filter (default: all)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index build [--format json|text] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	info, err := os.Stat(repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %q is not a directory\n", repoPath)
		os.Exit(1)
	}

	opts := index.BuildOptions{
		RepoPath:         repoPath,
		DBPath:           *dbPath,
		MaxFileSizeBytes: *maxSize,
	}
	if *langList != "" {
		opts.Languages = strings.Split(*langList, ",")
	}

	start := time.Now()
	result, err := index.ColdBuild(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: index build failed: %v\n", err)
		os.Exit(1)
	}
	result.Duration = time.Since(start)

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(result, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		fmt.Fprintf(os.Stdout, " Indexed %d files, %d chunks, %d edges\n",
			result.TotalFiles, result.TotalChunks, result.TotalEdges)
		fmt.Fprintf(os.Stdout, " Languages: %v\n", result.Languages)
		fmt.Fprintf(os.Stdout, " Duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Fprintf(os.Stdout, " Database: %s\n", result.DBPath)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

func runIndexStats(args []string) {
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index stats <repo-path>")
		os.Exit(2)
	}
	repoPath := args[0]

	dbPath := filepath.Join(repoPath, ".sdp", "index.db")
	if _, err := os.Stat(dbPath); err != nil {
		fmt.Fprintf(os.Stderr, "error: index not found at %s (run 'sdp index build' first)\n", dbPath)
		os.Exit(1)
	}

	store, err := index.OpenStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: open index: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	stats, err := store.Stats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: stats: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, " Chunks: %d\n", stats.TotalChunks)
	fmt.Fprintf(os.Stdout, " Files:  %d\n", stats.TotalFiles)
	fmt.Fprintf(os.Stdout, " Edges:  %d\n", stats.TotalEdges)
	fmt.Fprintf(os.Stdout, " Languages: %v\n", stats.Languages)

	if v, _ := store.GetMeta("indexed_at"); v != "" {
		fmt.Fprintf(os.Stdout, " Indexed at: %s\n", v)
	}
	if v, _ := store.GetMeta("repo_name"); v != "" {
		fmt.Fprintf(os.Stdout, " Repo: %s\n", v)
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
