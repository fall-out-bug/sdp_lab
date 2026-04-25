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
	"sdp_dev/internal/promote"
)

func runBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	strict := fs.Bool("strict", false, "run in strict mode (logs intent; use --promote-to-strict for full Phase FSM bridge)")
	promoteToStrict := fs.Bool("promote-to-strict", false, "run vibecode pipeline, then promote to strict Phase FSM (creates deltas + gates from run evidence)")
	local := fs.Bool("local", false, "prefer Ollama/local models via dispatch")
	sandbox := fs.String("sandbox", "none", "sandbox type: docker|none")
	dryRun := fs.Bool("dry-run", false, "show plan without executing")
	format := fs.String("format", "text", "output format: json|text")
	output := fs.String("output", "", "output directory for evidence (default: .sdp/evidence/<run_id>)")
	timeout := fs.Duration("timeout", 30*time.Minute, "maximum build duration (0 = no timeout)")

	_ = fs.Parse(reorderFlagsFirst(args, fs))

	validateFormat(*format)

	// Validate --sandbox values.
	switch *sandbox {
	case "docker", "none":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown sandbox type %q (use docker or none)\n", *sandbox)
		os.Exit(2)
	}

	// Idea string is required.
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp build \"<idea>\" [--strict] [--promote-to-strict] [--local] [--sandbox=docker|none] [--dry-run] [--format json|text] [--output DIR] [--timeout DURATION]")
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
			DryRun:  true,
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

	// Progress logs go to stderr so --format json produces clean JSON on stdout.
	logf := func(msg string, args ...any) {
		if *format == "json" {
			fmt.Fprintf(os.Stderr, msg, args...)
		} else {
			fmt.Printf(msg, args...)
		}
	}

	logf("build [%s] starting\n", runID[:8])
	logf("  idea: %s\n", idea)
	logf("  strict: %v | local: %v | sandbox: %s\n", *strict, *local, *sandbox)
	logf("\n")

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

	// Print stage trace to logf (stderr in json mode).
	for _, s := range result.Stages {
		status := s.Status
		if status == "" {
			status = "unknown"
		}
		logf("  %-10s %s (%s)\n", s.Stage, status, s.Duration.Round(1e6))
		if s.Output != "" {
			logf("             %s\n", s.Output)
		}
		if s.Error != "" {
			logf("             error: %s\n", s.Error)
		}
	}

	logf("\n")

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

	// --promote-to-strict: after successful vibecode run, promote to Phase FSM.
	if *promoteToStrict {
		logf("promoting run %s to strict Phase FSM...\n", runID[:8])
		promoteResult, err := promote.PromoteFromRun(promote.PromoteOptions{
			RunID:       runID,
			FeatureID:   fmt.Sprintf("promoted-%s", runID[:8]),
			EvidenceDir: evidenceDir,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: promotion failed: %v\n", err)
			os.Exit(1)
		}
		logf("  deltas: %d, gates: %d\n", len(promoteResult.Deltas), len(promoteResult.Gates))
		logf("  phase dir: %s\n", promoteResult.PhaseDir)
		logf("  gates are AWAITING human approval\n")
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
