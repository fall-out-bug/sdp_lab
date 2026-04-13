package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"sdp_dev/internal/strataudit"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "init":
		runInit(os.Args[2:])
	case "run":
		runRun(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: sdp-strataudit <command> [options]

Commands:
  init    Create strataudit.yaml template
  run     Run full audit pipeline

Run options:
  --dir   Project root directory (default: .)
  --config  Config file path (default: strataudit.yaml)`)
}

func runInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	dir := fs.String("dir", ".", "project directory")
	_ = fs.Parse(args)

	path := filepath.Join(*dir, "strataudit.yaml")
	if err := os.MkdirAll(*dir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(os.Stderr, "error: %s already exists\n", path)
		os.Exit(1)
	}

	tmpl := strataudit.DefaultConfigYAML()
	data, _ := yaml.Marshal(tmpl)
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Created %s\n", path)
}

func runRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dir := fs.String("dir", ".", "project root directory")
	configPath := fs.String("config", "strataudit.yaml", "config file name")
	resume := fs.Bool("resume", false, "resume from last completed stage")
	_ = fs.Parse(args)

	cfgPath := filepath.Join(*dir, *configPath)
	cfg, err := strataudit.LoadConfig(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config validation error: %v\n", err)
		os.Exit(1)
	}

	// Resolve output dir as absolute path relative to --dir
	absDir, err := filepath.Abs(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving dir: %v\n", err)
		os.Exit(1)
	}
	cfg.Output.Dir = filepath.Join(absDir, cfg.Output.Dir)

	dbPath := filepath.Join(cfg.Output.Dir, "strataudit.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "error creating output dir: %v\n", err)
		os.Exit(1)
	}

	store, err := strataudit.NewSQLiteStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error opening store: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = store.Close() }()

	ctx := context.Background()
	runtime, err := cfg.ResolveRuntime()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error resolving runtime: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("StratAudit config loaded: %d levels, %d source dirs\n", len(cfg.Levels), len(cfg.Project.SourceDirs))
	fmt.Printf("Store: %s\n", dbPath)

	result, err := strataudit.RunPipeline(ctx, cfg, store, runtime, strataudit.PipelineOpts{
		Resume: *resume,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pipeline error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n=== StratAudit Complete ===\n")
	fmt.Printf("Ingest:   %d new, %d updated, %d unchanged\n", result.Ingest.New, result.Ingest.Updated, result.Ingest.Unchanged)
	fmt.Printf("Extract:  %d entities from %d documents\n", result.Extract.EntitiesExtracted, result.Extract.Documents)
	fmt.Printf("Link:     %d traces from %d candidates (%d level pairs)\n", result.Link.TracesCreated, result.Link.CandidatesGenerated, result.Link.Pairs)
	fmt.Printf("Analyze:  %d findings\n", result.Analyze.Findings)
	fmt.Printf("Duration: %s\n", result.Duration.Round(time.Millisecond))
	fmt.Printf("Output:   %s\n", cfg.Output.Dir)
}
