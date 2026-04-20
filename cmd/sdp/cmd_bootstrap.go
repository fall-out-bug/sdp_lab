package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"sdp_dev/internal/bootstrap"
)

func runBootstrap(args []string) {
	fs := flag.NewFlagSet("bootstrap", flag.ExitOnError)
	dryRun := fs.Bool("dry-run", false, "show what would be generated without writing")
	force := fs.Bool("force", false, "overwrite existing artifacts")
	noVerify := fs.Bool("no-verify", false, "skip build/test/lint verification")
	beads := fs.Bool("beads", false, "enable beads initialization (opt-in)")
	yes := fs.Bool("yes", false, "CI automation: approve final artifacts without DRAFT prefix")
	autoCurate := fs.Bool("auto-curate", false, "CI automation: bypass DRAFT prefix and produce final artifacts")
	format := fs.String("format", "text", "output format: json, text")
	onlyStr := fs.String("only", "", "generate only these artifacts (comma-separated: claude-md,agents-md,policies,hooks,beads)")

	_ = fs.Parse(args)

	// Determine subcommand: "status" or repo path.
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: sdp bootstrap [--dry-run] [--force] [--beads] [--yes] [--auto-curate] [--only TYPES] <repo-path>")
		fmt.Fprintln(os.Stderr, "       sdp bootstrap status <repo-path>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "CI automation flags (--yes, --auto-curate) bypass DRAFT prefix for unattended runs.")
		os.Exit(2)
	}

	// Handle "status" subcommand.
	if fs.Arg(0) == "status" {
		if fs.NArg() < 2 {
			fmt.Fprintln(os.Stderr, "usage: sdp bootstrap status <repo-path>")
			os.Exit(2)
		}
		runBootstrapStatus(fs.Arg(1), *format)
		return
	}

	repoPath := fs.Arg(0)
	validateFormat(*format)

	// Default: UseDraft=true (DRAFT-prefixed files). CI flags bypass this.
	useDraft := !(*yes || *autoCurate)

	cfg := bootstrap.BootstrapConfig{
		RepoPath: repoPath,
		DryRun:   *dryRun,
		Force:    *force,
		NoVerify: *noVerify,
		Beads:    *beads,
		UseDraft: useDraft,
	}

	if *onlyStr != "" {
		cfg.Only = strings.Split(*onlyStr, ",")
	}

	planner := bootstrap.NewPlanner(cfg)

	if *dryRun {
		report, err := planner.DryRun()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		renderBootstrapReport(report, *format)
		return
	}

	report, err := planner.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	renderBootstrapReport(report, *format)
}

func runBootstrapStatus(repoPath string, format string) {
	cfg := bootstrap.BootstrapConfig{RepoPath: repoPath}
	planner := bootstrap.NewPlanner(cfg)
	status, err := planner.Status()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	switch format {
	case "json":
		out, jerr := json.MarshalIndent(status, "", "  ")
		if jerr != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", jerr)
			os.Exit(1)
		}
		fmt.Print(string(out) + "\n")
	default:
		fmt.Print(bootstrap.FormatStatusText(status))
	}
}

func renderBootstrapReport(report *bootstrap.BootstrapReport, format string) {
	switch format {
	case "json":
		out, err := bootstrap.FormatReportJSON(report)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Print(out)
	default:
		// Text format: show notes then artifacts.
		fmt.Fprintf(os.Stdout, "Bootstrap Report — %s\n", report.Repo)
		fmt.Fprintf(os.Stdout, "Version: %s | Duration: %dms\n\n", report.Version, report.DurationMs)

		if len(report.Notes) > 0 {
			for _, n := range report.Notes {
				fmt.Fprintf(os.Stdout, "  %s\n", n)
			}
			fmt.Fprintln(os.Stdout)
		}

		for _, a := range report.Artifacts {
			mark := "[ok]"
			switch a.Status {
			case "dry_run":
				mark = "[plan]"
			case "skipped":
				mark = "[skip]"
			case "error":
				mark = "[err]"
			}
			fmt.Fprintf(os.Stdout, "  %s %-20s %s\n", mark, a.Path, a.Message)
		}

		// Data sources summary.
		fmt.Fprintln(os.Stdout, "\nData Sources:")
		for src, found := range report.DataSources {
			label := "no"
			if found {
				label = "yes"
			}
			fmt.Fprintf(os.Stdout, "  %-12s %s\n", src+":", label)
		}
	}
}

func validateFormat(format string) {
	switch format {
	case "json", "text":
	default:
		fmt.Fprintf(os.Stderr, "error: unknown format %q (use json or text)\n", format)
		os.Exit(2)
	}
}
