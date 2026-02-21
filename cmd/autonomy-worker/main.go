package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/policy"
	"sdp_dev/internal/safeid"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Print selected task without changes")
	debug := flag.Bool("debug", false, "Print candidate selection diagnostics")
	flag.Parse()

	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	loadWorkstreamConfig(root)

	byID, err := listIssues()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	picked, err := pickCandidate(byID, *debug)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if picked == nil {
		fmt.Println("No eligible autonomy tasks found")
		return
	}
	if err := safeid.ValidateIssueID(picked.ID); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model, err := modelFromLabels(picked.Labels)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	decision := policy.Decide(policy.DecisionRequest{
		IssueID:        picked.ID,
		Title:          picked.Title,
		Lane:           laneFromLabels(picked.Labels),
		PreferredModel: model,
		ChangedPaths:   []string{picked.SpecID},
	})

	branch := decision.BranchName

	output := map[string]string{"issue_id": picked.ID, "title": picked.Title, "model": model, "branch": branch}
	if *dryRun {
		b, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(b))
		return
	}

	if _, err := runBD("update", picked.ID, "--status", "in_progress"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runPacket := map[string]any{
		"issue_id":       picked.ID,
		"title":          picked.Title,
		"model":          decision.SelectedModel,
		"fallback":       decision.FallbackChain,
		"branch":         branch,
		"started_at":     time.Now().UTC().Format(time.RFC3339),
		"status":         "in_progress",
		"flow":           "in_progress",
		"risk_class":     decision.RiskClass,
		"lane":           decision.Lane,
		"policy_verdict": decision.PolicyVerdict,
		"policy_stack":   []string{"go-first", "strict-evidence", "model-allowlist"},
	}
	runPath := filepath.Join(root, ".sdp", "runs", picked.ID+".json")
	if err := writeJSON(runPath, runPacket); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	evidencePath, err := populateEvidence(root, picked, branch, decision)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if decision.EscalationRequired {
		runPacket["flow"] = "escalated"
		_, _ = runBD("update", picked.ID, "--append-notes", "protocol flow -> escalated")
	}

	note := fmt.Sprintf("autonomy-worker(go): claimed; verdict=%s; model=%s; branch=%s; packet=%s; evidence=%s", decision.PolicyVerdict, decision.SelectedModel, branch, runPath, evidencePath)
	if err := appendNote(picked.ID, note); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	b, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(b))
}
