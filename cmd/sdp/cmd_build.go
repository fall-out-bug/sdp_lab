package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	build "sdp_dev/internal/build"
)

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	strict := fs.Bool("strict", false, "route through Phase FSM (plan gate + review gate + eval gate)")
	local := fs.Bool("local", false, "prefer Ollama/local models via dispatch")
	sandbox := fs.String("sandbox", "none", "sandbox type: docker|testcontainers|none")
	dryRun := fs.Bool("dry-run", false, "show plan without executing")
	format := fs.String("format", "text", "output format: json|text")
	output := fs.String("output", "", "output directory for evidence (default: .sdp/evidence/<run_id>)")
	timeout := fs.Duration("timeout", 30*time.Minute, "maximum build duration (0 = no timeout)")

	_ = fs.Parse(reorderFlagsFirst(args, fs))

	validateFormat(*format)

	// Validate --sandbox values.
	switch *sandbox {
	case "docker", "testcontainers", "none":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown sandbox type %q (use docker, testcontainers, or none)\n", *sandbox)
		os.Exit(2)
	}

	// Idea string is required.
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp build \"<idea>\" [--strict] [--local] [--sandbox=<type>] [--dry-run] [--format json|text] [--output DIR] [--timeout DURATION]")
		os.Exit(2)
	}

	idea := fs.Arg(0)
	runID := uuid.New().String()

	evidenceDir := *output
	if evidenceDir == "" {
		evidenceDir = fmt.Sprintf(".sdp/evidence/%s", runID)
	}

	if *dryRun {
		cfg := build.BuildConfig{
			Idea:    idea,
			Sandbox: *sandbox,
			Strict:  *strict,
			Local:   *local,
			Format:  *format,
			RunID:   runID,
		}
		pipeline, err := build.NewDefaultPipeline(cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		result, err := pipeline.DryRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if *format == "json" {
			printJSON(result)
		} else {
			// Print a nice text summary
			fmt.Println("Build Plan (dry-run)")
			fmt.Println("--------------------")
			fmt.Printf("  Run ID:   %s\n", result.RunID)
			fmt.Printf("  Idea:     %s\n", idea)
			fmt.Printf("  Strict:   %v\n", *strict)
			fmt.Printf("  Local:    %v\n", *local)
			fmt.Printf("  Sandbox:  %s\n", *sandbox)
			fmt.Println()
			for _, s := range result.Stages {
				fmt.Printf("  %-10s %s — %s\n", s.Stage, s.Status, s.Output)
			}
		}
		return
	}

	// Wire to internal/build pipeline.
	cfg := build.BuildConfig{
		Idea:      idea,
		Strict:    *strict,
		Local:     *local,
		Sandbox:   *sandbox,
		DryRun:    false,
		Format:    *format,
		OutputDir: evidenceDir,
		RunID:     runID,
	}

	pipeline, err := build.NewDefaultPipeline(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("build [%s] starting\n", runID[:8])
	fmt.Printf("  idea: %s\n", idea)
	fmt.Printf("  strict: %v | local: %v | sandbox: %s\n", *strict, *local, *sandbox)
	fmt.Println()

	ctx := context.Background()
	if *timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	result, err := pipeline.Run(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// Print stage trace.
	for _, s := range result.Stages {
		status := s.Status
		if status == "" {
			status = "unknown"
		}
		fmt.Printf("  %-10s %s (%s)\n", s.Stage, status, s.Duration.Round(1e6))
		if s.Output != "" {
			fmt.Printf("             %s\n", s.Output)
		}
		if s.Error != "" {
			fmt.Printf("             error: %s\n", s.Error)
		}
	}

	fmt.Println()

	if *format == "json" {
		printJSON(result)
	} else {
		fmt.Printf("Summary:  %s\n", result.Summary)
		fmt.Printf("Status:   %s\n", result.Status)
		fmt.Printf("Evidence: %s\n", evidenceDir)
	}

	if result.Status != "success" {
		os.Exit(1)
	}
}

// reorderFlagsFirst moves flag arguments before positional arguments.
// Go's flag package stops parsing at the first non-flag argument, so
// `sdp build "idea" --dry-run` would not recognize --dry-run without this.
// It uses the FlagSet to distinguish bool flags (--strict) from flags that
// take values (--sandbox docker), so it doesn't mistakenly grab the idea
// string as a flag value.
func reorderFlagsFirst(args []string, fs *flag.FlagSet) []string {
	var flags, positional []string
	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			// If flag uses --flag=value form, no separate value needed.
			if strings.Contains(args[i], "=") {
				continue
			}
			// Check if this is a bool flag — bools don't consume the next arg.
			name := strings.TrimLeft(args[i], "-")
			if !isBuildBoolFlag(fs, name) && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flags = append(flags, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}
	return append(flags, positional...)
}

// isBuildBoolFlag checks whether the named flag in the FlagSet is a boolean flag.
func isBuildBoolFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		return false
	}
	if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok {
		return b.IsBoolFlag()
	}
	return false
}
