package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/pr"
)

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func ensureRepoInitialized() error {
	out, err := run("gh", "repo", "view", "--json", "defaultBranchRef", "--jq", ".defaultBranchRef.name")
	if err != nil {
		return err
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" {
		return fmt.Errorf("repository has no default branch; initialize remote with first commit before PR publishing")
	}
	return nil
}

func main() {
	issueID := flag.String("issue", "", "Issue ID")
	prTitle := flag.String("title", "", "PR title")
	bodyFile := flag.String("body-file", "", "Path to PR body markdown file")
	head := flag.String("head", "", "Head branch")
	base := flag.String("base", "", "Base branch")
	evidencePath := flag.String("evidence", "", "Evidence JSON path (default .sdp/evidence/<issue>.json)")
	dryRun := flag.Bool("dry-run", false, "Print command without executing gh")
	flag.Parse()

	if *issueID == "" || *prTitle == "" || *head == "" {
		fmt.Fprintln(os.Stderr, "--issue, --title, and --head are required")
		os.Exit(2)
	}

	args := []string{"pr", "create", "--title", *prTitle, "--head", *head}
	if *base != "" {
		args = append(args, "--base", *base)
	}
	if *bodyFile != "" {
		args = append(args, "--body-file", *bodyFile)
	}

	if *dryRun {
		out := map[string]any{"issue": *issueID, "command": append([]string{"gh"}, args...)}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return
	}

	if err := ensureRepoInitialized(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	out, err := run("gh", args...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	prURL := strings.TrimSpace(string(bytes.TrimSpace(out)))
	if prURL == "" {
		fmt.Fprintln(os.Stderr, "gh did not return PR URL")
		os.Exit(1)
	}

	path := *evidencePath
	if path == "" {
		wd, _ := os.Getwd()
		path = filepath.Join(wd, ".sdp", "evidence", *issueID+".json")
	}
	if err := pr.WritePRURLToEvidence(path, prURL); err != nil {
		fmt.Fprintf(os.Stderr, "update evidence: %v\n", err)
		os.Exit(1)
	}

	if _, err := run("bd", "update", *issueID, "--append-notes", "PR created: "+prURL); err != nil {
		fmt.Fprintf(os.Stderr, "update beads note: %v\n", err)
		os.Exit(1)
	}

	result := map[string]any{"issue": *issueID, "pr_url": prURL, "evidence": path}
	b, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(b))
}
