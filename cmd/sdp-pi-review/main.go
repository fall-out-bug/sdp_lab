package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/executil"
	"sdp_dev/internal/pireview"
)

func main() {
	fs := flag.NewFlagSet("sdp-pi-review", flag.ExitOnError)
	scope := fs.String("scope", "auto", "Review scope: auto, working-tree, branch")
	base := fs.String("base", "", "Base ref for branch scope")
	feature := fs.String("feature", "", "Feature ID (e.g. F161)")
	testCmd := fs.String("test-command", "", "Explicit test command")
	writeVerdict := fs.Bool("write-verdict", false, "Write .sdp/review_verdict.json")
	createBeads := fs.Bool("create-beads", false, "Create beads for actionable findings")
	round := fs.Int("round", 1, "Review round number")
	_ = fs.Parse(os.Args[1:])

	projectRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-review: %v\n", err)
		os.Exit(1)
	}

	cfg := pireview.Config{
		ProjectRoot: projectRoot,
		Scope:       pireview.ScopeMode(*scope),
		BaseRef:     *base,
		Feature:     *feature,
		TestCommand: *testCmd,
		Runner:      executil.GetDefaultRunner(),
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-review: %v\n", err)
		os.Exit(2)
	}

	runner, err := pireview.NewRunner(cfg, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-review: %v\n", err)
		os.Exit(1)
	}

	run, verdict, err := runner.Run(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-review: %v\n", err)
		os.Exit(1)
	}
	run.Round = *round

	// Persist run telemetry
	if err := persistRun(projectRoot, run); err != nil {
		fmt.Fprintf(os.Stderr, "sdp-pi-review: persist run: %v\n", err)
	}

	// Write verdict
	if *writeVerdict {
		if err := writeVerdictFile(projectRoot, verdict); err != nil {
			fmt.Fprintf(os.Stderr, "sdp-pi-review: write verdict: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Verdict written to .sdp/review_verdict.json\n")
	}

	// Create beads for findings
	if *createBeads {
		created := createBeadFindings(projectRoot, *feature, *round, verdict.FindingsDetail)
		if len(created) > 0 {
			fmt.Printf("Created %d bead(s) for findings\n", len(created))
		}
	}

	// Print summary
	fmt.Printf("\n## Pi Review Summary\n")
	fmt.Printf("Run: %s\n", run.RunID)
	fmt.Printf("Feature: %s\n", *feature)
	fmt.Printf("Round: %d\n", *round)
	fmt.Printf("Scope: %s (%d files reviewed)\n", *scope, len(run.Scope.ReviewedFiles))
	fmt.Printf("Verdict: %s\n", verdict.Verdict)
	fmt.Printf("Findings: %d P0, %d P1, %d total\n", verdict.P0Count, verdict.P1Count, len(verdict.FindingsDetail))
	fmt.Printf("Models: %d/%d succeeded\n", countOK(run.Models), len(run.Models))

	if verdict.Verdict != "APPROVED" {
		os.Exit(1)
	}
}

func persistRun(root string, run *pireview.ReviewRun) error {
	dir := filepath.Join(root, ".sdp", "runs", "pi-review", run.RunID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "run.json"), data, 0o644)
}

func writeVerdictFile(root string, verdict *pireview.Verdict) error {
	dir := filepath.Join(root, ".sdp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(verdict, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "review_verdict.json"), data, 0o644)
}

func createBeadFindings(root, feature string, round int, findings []pireview.Finding) []string {
	var created []string
	for _, f := range findings {
		if f.Priority != "P0" && f.Priority != "P1" {
			continue
		}

		// Check for existing finding with same dedupe key
		cmd := executil.GetDefaultRunner()
		searchOut, _ := cmd.Output(context.Background(), root,
			"bd", "list", "--status=open", "--labels", "pi-review,round-"+fmt.Sprintf("%d", round))
		if strings.Contains(string(searchOut), f.DedupeKey) {
			continue // already filed
		}

		title := fmt.Sprintf("pi-review %s: %s", f.Priority, f.Title)
		desc := fmt.Sprintf("File: %s:%d-%d\nRationale: %s\nSuggested fix: %s\nDedupe key: %s",
			f.File, f.StartLine, f.EndLine, f.Rationale, f.SuggestedFix, f.DedupeKey)

		labels := fmt.Sprintf("pi-review,review-finding,%s,round-%d", feature, round)
		out, err := cmd.Output(context.Background(), root,
			"bd", "create",
			"--title", title,
			"--description", desc,
			"--type", "bug",
			"--priority", priorityToBeads(f.Priority),
			"--labels", labels,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "create bead: %v\n", err)
			continue
		}
		beadID := strings.TrimSpace(string(out))
		if beadID != "" {
			created = append(created, beadID)
		}
	}
	return created
}

func priorityToBeads(p string) string {
	switch p {
	case "P0":
		return "0"
	case "P1":
		return "1"
	case "P2":
		return "2"
	case "P3":
		return "3"
	default:
		return "2"
	}
}

func countOK(models []pireview.ModelResult) int {
	n := 0
	for _, m := range models {
		if m.Status == "ok" {
			n++
		}
	}
	return n
}
