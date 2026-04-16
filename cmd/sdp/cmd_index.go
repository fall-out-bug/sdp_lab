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
		fmt.Fprintln(os.Stderr, "usage: sdp index <subcommand> [flags]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Commands:")
		fmt.Fprintln(os.Stderr, "  sdp index build <repo-path>              Full index (cold start)")
		fmt.Fprintln(os.Stderr, "  sdp index refresh <repo-path>            Incremental update (changed files only)")
		fmt.Fprintln(os.Stderr, "  sdp index stats <repo-path>              Show index statistics")
		fmt.Fprintln(os.Stderr, "  sdp index manifest <repo-path>           Generate .sdp/manifest.md")
		fmt.Fprintln(os.Stderr, "  sdp index query <repo-path> <query>      Semantic search (FTS + optional vectors)")
		fmt.Fprintln(os.Stderr, "  sdp index deps <repo-path> <module>      Dependency traversal")
		fmt.Fprintln(os.Stderr, "  sdp index find <repo-path> <term>        Exact identifier/keyword lookup")
		fmt.Fprintln(os.Stderr, "  sdp index rank <repo-path>               Compute PageRank scores")
		os.Exit(2)
	}
	switch args[0] {
	case "build":
		runIndexBuild(args[1:])
	case "refresh":
		runIndexRefresh(args[1:])
	case "stats":
		runIndexStats(args[1:])
	case "manifest":
		runIndexManifest(args[1:])
	case "query":
		runIndexQuery(args[1:])
	case "deps":
		runIndexDeps(args[1:])
	case "find":
		runIndexFind(args[1:])
	case "rank":
		runIndexRank(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown index subcommand: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "usage: sdp index <build|refresh|stats|manifest|query|deps|find|rank> [flags]")
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

func runIndexRefresh(args []string) {
	fs := flag.NewFlagSet("index refresh", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	dbPath := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	maxSize := fs.Int64("max-size", 100*1024, "max file size in bytes (default 100KB)")
	langList := fs.String("languages", "", "comma-separated language filter (default: all)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index refresh [--format json|text] <repo-path>")
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

	opts := index.RefreshOptions{
		RepoPath:         repoPath,
		DBPath:           *dbPath,
		MaxFileSizeBytes: *maxSize,
	}
	if *langList != "" {
		opts.Languages = strings.Split(*langList, ",")
	}

	start := time.Now()
	result, err := index.Refresh(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: index refresh failed: %v\n", err)
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
		fmt.Fprintf(os.Stdout, " Refreshed: %d checked, %d updated, %d added, %d removed\n",
			result.FilesChecked, result.FilesUpdated, result.FilesAdded, result.FilesRemoved)
		fmt.Fprintf(os.Stdout, " Index: %d files, %d chunks\n",
			result.TotalFiles, result.TotalChunks)
		fmt.Fprintf(os.Stdout, " Duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Fprintf(os.Stdout, " Database: %s\n", result.DBPath)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

func runIndexStats(args []string) {
	fs := flag.NewFlagSet("index stats", flag.ExitOnError)
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index stats [--db PATH] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	store, err := openIndexStore(repoPath, *db)
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
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index manifest [--output DIR] [--db PATH] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	sdpDir := *output
	if sdpDir == "" {
		sdpDir = filepath.Join(repoPath, ".sdp")
	}

	var dbPath string
	if *db != "" {
		dbPath = *db
	} else {
		dbPath = filepath.Join(sdpDir, "index.db")
	}
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

func runIndexQuery(args []string) {
	fs := flag.NewFlagSet("index query", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	limit := fs.Int("limit", 10, "maximum results to return")
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: sdp index query [--format json|text] [--limit N] [--db PATH] <repo-path> <query>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)
	query := fs.Arg(1)

	store, err := openIndexStore(repoPath, *db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	resp, err := index.SemanticSearch(store, query, *limit, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: query failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(resp, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		fmt.Fprintf(os.Stdout, " Query: %s (%d results in %s)\n", resp.Query, resp.Total, resp.Duration)
		for i, r := range resp.Results {
			fmt.Fprintf(os.Stdout, "\n  %d. %s (%s) [%s]\n", i+1, r.Chunk.SymbolName, r.Chunk.Kind, r.Chunk.FilePath)
			fmt.Fprintf(os.Stdout, "     Lines %d-%d | Score: %.4f | Match: %s\n",
				r.Chunk.LineStart, r.Chunk.LineEnd, r.Score, r.MatchSrc)
			snippet := r.Chunk.Content
			if len(snippet) > 120 {
				snippet = snippet[:120] + "..."
			}
			fmt.Fprintf(os.Stdout, "     %s\n", strings.ReplaceAll(snippet, "\n", " "))
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

func runIndexDeps(args []string) {
	fs := flag.NewFlagSet("index deps", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	reverse := fs.Bool("reverse", false, "show reverse dependencies (who depends on this module)")
	depth := fs.Int("depth", 3, "maximum traversal depth")
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: sdp index deps [--reverse] [--depth N] [--db PATH] <repo-path> <module>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)
	module := fs.Arg(1)

	store, err := openIndexStore(repoPath, *db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	resp, err := index.DepsSearch(store, module, *reverse, *depth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: deps failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(resp, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		direction := "forward"
		if *reverse {
			direction = "reverse"
		}
		fmt.Fprintf(os.Stdout, " Module: %s (%s, depth %d)\n", resp.Module, direction, resp.Depth)
		if len(resp.Results) == 0 {
			fmt.Fprintln(os.Stdout, " No dependencies found.")
		}
		for _, r := range resp.Results {
			hotspot := ""
			if r.IsHotspot {
				hotspot = " [HOTSPOT]"
			}
			fmt.Fprintf(os.Stdout, "  %s%s (%s) LOC:%d BF:%d\n",
				r.ModuleName, hotspot, r.Relation, r.LOC, r.BusFactor)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

func runIndexFind(args []string) {
	fs := flag.NewFlagSet("index find", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	limit := fs.Int("limit", 20, "maximum results to return")
	regex := fs.Bool("regex", false, "treat query as regex pattern")
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "usage: sdp index find [--regex] [--limit N] [--db PATH] <repo-path> <term>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)
	term := fs.Arg(1)

	store, err := openIndexStore(repoPath, *db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	resp, err := index.FindSearch(store, term, *regex, *limit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: find failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		out, jerr := json.MarshalIndent(resp, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		fmt.Fprintf(os.Stdout, " Find: %s (%d results in %s)\n", resp.Query, resp.Total, resp.Duration)
		for i, r := range resp.Results {
			fmt.Fprintf(os.Stdout, "  %d. %s (%s) %s:%d-%d\n",
				i+1, r.Chunk.SymbolName, r.Chunk.Kind,
				r.Chunk.FilePath, r.Chunk.LineStart, r.Chunk.LineEnd)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

func runIndexRank(args []string) {
	fs := flag.NewFlagSet("index rank", flag.ExitOnError)
	format := fs.String("format", "text", "output format: json, text")
	db := fs.String("db", "", "custom database path (default: <repo>/.sdp/index.db)")
	_ = fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp index rank [--db PATH] <repo-path>")
		os.Exit(2)
	}
	repoPath := fs.Arg(0)

	store, err := openIndexStore(repoPath, *db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	updated, err := index.ComputePageRank(store)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: pagerank failed: %v\n", err)
		os.Exit(1)
	}

	switch *format {
	case "json":
		type rankResult struct {
			Updated int `json:"updated"`
		}
		out, jerr := json.MarshalIndent(rankResult{Updated: updated}, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	case "text":
		fmt.Fprintf(os.Stdout, " PageRank: updated %d chunks\n", updated)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", *format)
		os.Exit(2)
	}
}

// openIndexStore opens the index database for a given repo path.
// If dbPath is empty, it defaults to <repo>/.sdp/index.db.
func openIndexStore(repoPath string, dbPath string) (*index.SQLiteStore, error) {
	if dbPath == "" {
		dbPath = filepath.Join(repoPath, ".sdp", "index.db")
	}
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("index not found at %s (run 'sdp index build' first)", dbPath)
	}
	return index.OpenStore(dbPath)
}
