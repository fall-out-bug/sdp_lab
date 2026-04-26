package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"sdp_dev/internal/backlog"
)

// fidExtractRe matches the first F-identifier in a title string.
var fidExtractRe = regexp.MustCompile(`F[0-9]+(?:-[0-9]+)?`)

// beadEntry mirrors the JSON fields returned by `bd list --json`.
type beadEntry struct {
	ID              string `json:"id"`
	Title           string `json:"title"`
	Status          string `json:"status"`
	IssueType       string `json:"issue_type"`
	DependencyCount int    `json:"dependency_count"`
}

// runDoctorBacklog implements `sdp doctor backlog`.
//
// It calls `bd list --type=feature --json` (and a second pass for epics),
// converts the results into []backlog.Feature, runs backlog.Audit, prints
// the report, and exits 0 (clean) or 1 (findings).
func runDoctorBacklog(args []string) {
	strict := false
	includeStatus := "open"

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strict":
			strict = true
		case "--include-status":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "error: --include-status requires a value")
				os.Exit(2)
			}
			includeStatus = args[i+1]
			i++
		case "-h", "--help":
			fmt.Println("usage: sdp doctor backlog [--strict] [--include-status <status>]")
			fmt.Println()
			fmt.Println("  --strict                   Also flag features with ws status=design-pending and no children")
			fmt.Println("  --include-status <status>  Comma-separated beads statuses to check (default: open)")
			fmt.Println()
			fmt.Println("Exits 0 if no findings, 1 if any findings detected.")
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "error: unknown flag: %s\n", args[i])
			os.Exit(2)
		}
	}

	// Verify bd is on PATH.
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: bd not found on PATH — install beads CLI before running sdp doctor backlog")
		os.Exit(1)
	}

	// Resolve repo root from the manifest path (same approach as runDoctorAdapters).
	repoRoot := resolveRepoRoot()

	// Fetch features from bd (type=feature) and epics (type=epic), merge.
	features, err := fetchBeadsFeatures(bdPath, includeStatus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: fetch beads features: %v\n", err)
		os.Exit(1)
	}

	// Parse comma-separated statuses.
	statuses := strings.Split(includeStatus, ",")
	for i, s := range statuses {
		statuses[i] = strings.TrimSpace(s)
	}

	opts := backlog.AuditOpts{
		RepoRoot:      repoRoot,
		IncludeStatus: statuses,
		Strict:        strict,
	}

	result := backlog.Audit(opts, features)
	fmt.Print(backlog.FormatReport(result))

	if len(result.Findings) > 0 {
		os.Exit(1)
	}
}

// resolveRepoRoot returns the repository root by walking up from the executable
// or, failing that, from the working directory.
func resolveRepoRoot() string {
	// Walk up from cwd looking for go.mod / AGENTS.md.
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	dir := cwd
	for {
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return cwd
}

// fileExists returns true if the named file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// fetchBeadsFeatures calls `bd list --type=feature --json` and
// `bd list --type=epic --json`, parses the JSON, and returns a merged
// []backlog.Feature.
func fetchBeadsFeatures(bdPath, status string) ([]backlog.Feature, error) {
	var all []backlog.Feature
	for _, issueType := range []string{"feature", "epic"} {
		entries, err := bdList(bdPath, issueType)
		if err != nil {
			return nil, fmt.Errorf("bd list --type=%s: %w", issueType, err)
		}
		for _, e := range entries {
			fid := fidExtractRe.FindString(e.Title)
			all = append(all, backlog.Feature{
				BeadID:    e.ID,
				FID:       fid,
				Title:     e.Title,
				Status:    e.Status,
				IssueType: e.IssueType,
				DepCount:  e.DependencyCount,
			})
		}
	}
	return all, nil
}

// bdList runs `bd list --type=<issueType> --json --flat --all` and decodes the result.
func bdList(bdPath, issueType string) ([]beadEntry, error) {
	cmd := exec.Command(bdPath, "list", "--type="+issueType, "--json", "--flat", "--all")
	out, err := cmd.Output()
	if err != nil {
		// Try to surface stderr.
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("exit %d: %s", ee.ExitCode(), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	var entries []beadEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil, fmt.Errorf("parse JSON: %w", err)
	}
	return entries, nil
}
