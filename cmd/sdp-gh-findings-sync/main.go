// sdp-gh-findings-sync syncs GitHub CI findings to local Beads tasks.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/bridge"

	"github.com/google/uuid"
)

var (
	repo     = flag.String("repo", "", "GitHub repository (owner/repo)")
	branch   = flag.String("branch", "", "Branch to monitor (default: all branches)")
	runID    = flag.Int64("run", 0, "Specific workflow run ID to sync")
	localDir = flag.String("dir", "", "Local directory with findings JSON files")
	poll     = flag.Bool("poll", false, "Enable polling mode")
	interval = flag.Duration("interval", 5*time.Minute, "Polling interval")
	prefix   = flag.String("prefix", "sdplab-", "Beads issue prefix")
	labels   = flag.String("labels", "", "Comma-separated default labels")
	dryRun     = flag.Bool("dry-run", false, "Show what would be created without creating")
	output     = flag.String("output", "", "Output file for sync report")
	issues     = flag.Bool("issues", false, "Sync GitHub Issues as well as CI artifacts")
	issueLabel = flag.String("issue-label", "", "Filter GitHub Issues by label (comma-separated)")
	issueState = flag.String("issue-state", "open", "GitHub issue state filter (open, closed, all)")
)

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nInterrupted, shutting down...")
		cancel()
	}()

	if *repo == "" && *localDir == "" {
		fmt.Fprintln(os.Stderr, "Error: --repo or --dir is required")
		os.Exit(1)
	}

	sink := bridge.NewBeadsSink(*prefix, *dryRun, parseLabels(*labels))

	fmt.Println("Loading existing findings...")
	if err := sink.LoadExistingFindings(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load existing findings: %v\n", err)
	}

	if *poll {
		runPollingMode(ctx, sink)
	} else {
		runOneShotMode(ctx, sink)
	}

	sink.PrintSummary()

	if *output != "" {
		report, err := sink.GenerateReport()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		} else if err := os.WriteFile(*output, report, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		} else {
			fmt.Printf("Report written to: %s\n", *output)
		}
	}
}

func runOneShotMode(ctx context.Context, sink *bridge.BeadsSink) {
	if *localDir != "" {
		fmt.Printf("Loading findings from: %s\n", *localDir)
		syncLocalFindings(ctx, sink, *localDir)
	} else if *runID > 0 {
		fmt.Printf("Syncing findings from run %d...\n", *runID)
		syncFromRun(ctx, sink, *repo, *runID)
	} else {
		fmt.Println("Fetching latest workflow runs...")
		syncLatestRuns(ctx, sink, *repo, *branch)
	}

	if *issues && *repo != "" {
		fmt.Println("Syncing GitHub issues...")
		syncGitHubIssues(ctx, sink, *repo)
	}
}

func runPollingMode(ctx context.Context, sink *bridge.BeadsSink) {
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	runOneShotMode(ctx, sink)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fmt.Printf("\n[%s] Polling for new findings...\n", time.Now().Format(time.RFC3339))
			runOneShotMode(ctx, sink)
		}
	}
}

func syncLocalFindings(ctx context.Context, sink *bridge.BeadsSink, dir string) {
	findings, types, err := bridge.LoadLocalFindings(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading findings: %v\n", err)
		return
	}

	for i, f := range findings {
		switch types[i] {
		case "protocol":
			pf := f.(*bridge.ProtocolFindings)
			fmt.Printf("Processing %d protocol findings from %s...\n", len(pf.Findings), pf.Source.CheckName)
			if err := sink.SyncProtocolFindings(ctx, pf); err != nil {
				fmt.Fprintf(os.Stderr, "Error syncing protocol findings: %v\n", err)
			}
		case "docs":
			df := f.(*bridge.DocsFindings)
			fmt.Printf("Processing %d docs findings from %s...\n", len(df.Findings), df.Source.CheckName)
			if err := sink.SyncDocsFindings(ctx, df); err != nil {
				fmt.Fprintf(os.Stderr, "Error syncing docs findings: %v\n", err)
			}
		}
	}
}

func syncFromRun(ctx context.Context, sink *bridge.BeadsSink, repo string, runID int64) {
	client := bridge.NewGitHubClient(repo)

	tmpDir := filepath.Join(os.TempDir(), "sdp-findings", uuid.New().String())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		return
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	files, err := client.FetchArtifacts(ctx, runID, tmpDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching artifacts: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No finding artifacts found in run")
		return
	}

	for _, file := range files {
		f, t, err := bridge.ParseFindingsFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", file, err)
			continue
		}

		switch t {
		case "protocol":
			if err := sink.SyncProtocolFindings(ctx, f.(*bridge.ProtocolFindings)); err != nil {
				fmt.Fprintf(os.Stderr, "Error syncing: %v\n", err)
			}
		case "docs":
			if err := sink.SyncDocsFindings(ctx, f.(*bridge.DocsFindings)); err != nil {
				fmt.Fprintf(os.Stderr, "Error syncing: %v\n", err)
			}
		}
	}
}

func syncLatestRuns(ctx context.Context, sink *bridge.BeadsSink, repo, branch string) {
	client := bridge.NewGitHubClient(repo)

	runs, err := client.GetLatestWorkflowRuns(ctx, branch, 5)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching runs: %v\n", err)
		return
	}

	for _, run := range runs {
		if run.Status != "completed" {
			continue
		}

		fmt.Printf("  Run %d: %s (%s)\n", run.ID, run.Name, run.Conclusion)

		if run.Conclusion == "failure" || run.Conclusion == "success" {
			syncFromRun(ctx, sink, repo, run.ID)
		}
	}
}

func parseLabels(s string) []string {
	if s == "" {
		return nil
	}
	var labels []string
	for _, l := range strings.Split(s, ",") {
		l = strings.TrimSpace(l)
		if l != "" {
			labels = append(labels, l)
		}
	}
	return labels
}

func syncGitHubIssues(ctx context.Context, sink *bridge.BeadsSink, repo string) {
	// Note: This function reads global flag variables (*issueLabel, *issueState) directly.
	// This is the established pattern throughout main.go — all sync* functions read
	// their configuration from package-level flag vars rather than receiving params.
	// See syncLocalFindings, syncFromRun, syncLatestRuns for the same convention.
	client := bridge.NewGitHubClient(repo)

	labels := parseLabels(*issueLabel)
	// Pass 0 as limit to fetch all issues with pagination
	ghIssues, err := client.FetchIssues(ctx, labels, *issueState, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching GitHub issues: %v\n", err)
		return
	}

	if len(ghIssues) == 0 {
		fmt.Println("No matching GitHub issues found")
		return
	}

	fmt.Printf("Processing %d GitHub issues...\n", len(ghIssues))
	if err := sink.SyncGitHubIssues(ctx, repo, ghIssues); err != nil {
		fmt.Fprintf(os.Stderr, "Error syncing GitHub issues: %v\n", err)
	}
}
