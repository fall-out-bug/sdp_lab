package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"sdp_dev/internal/safeid"
)

type claimResult struct {
	IssueID string `json:"issue_id"`
	Title   string `json:"title"`
	Model   string `json:"model"`
	Branch  string `json:"branch"`
}

type issueDetail struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Labels             []string `json:"labels"`
	SpecID             string   `json:"spec_id"`
	Description        string   `json:"description"`
	AcceptanceCriteria string   `json:"acceptance_criteria"`
}

func main() {
	flowStartedAt := time.Now()
	claimOut, claimFallback, err := runComponentWithFallback("autonomy-worker", "./cmd/autonomy-worker")
	if err != nil {
		if strings.Contains(err.Error(), "No eligible autonomy tasks found") {
			fmt.Println("No eligible autonomy tasks found")
			emitWorkerObservability("", "plan", "blocked", "unknown", flowStartedAt, 0, claimFallback, false, "", "")
			return
		}
		emitWorkerObservability("", "plan", "failed", "unknown", flowStartedAt, 0, claimFallback, true, "", "")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.Contains(string(claimOut), "No eligible autonomy tasks found") {
		fmt.Println("No eligible autonomy tasks found")
		emitWorkerObservability("", "plan", "blocked", "unknown", flowStartedAt, 0, claimFallback, false, "", "")
		return
	}
	claim, err := parseClaim(claimOut)
	if err != nil {
		emitWorkerObservability("", "plan", "failed", "unknown", flowStartedAt, 0, claimFallback, true, "", "")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := safeid.ValidateIssueID(claim.IssueID); err != nil {
		emitWorkerObservability("", "plan", "failed", "unknown", flowStartedAt, 0, claimFallback, true, "", "")
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	evidenceContextLink, prURL := extractLinkage(claim.IssueID)
	emitWorkerObservability(claim.IssueID, "plan", "running", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)

	issue, err := loadIssue(claim.IssueID)
	if err != nil {
		emitWorkerObservability(claim.IssueID, "intake", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	workstream := resolveWorkstream(issue.Labels)
	if workstream == "" {
		emitWorkerObservability(claim.IssueID, "plan", "escalated", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintf(os.Stderr, "unsupported workstream labels for issue %s\n", claim.IssueID)
		os.Exit(1)
	}

	discardBeadsSyncNoise()

	if _, err := run("git", "checkout", "-b", claim.Branch); err != nil {
		_, err = run("git", "checkout", claim.Branch)
		if err != nil {
			emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	changedFiles := applyWorkstreamFlow(workstream, claim.IssueID, issue)

	testsPassed := true
	if _, err := run("go", "test", "./..."); err != nil {
		testsPassed = false
		emitWorkerObservability(claim.IssueID, "verify", "failed", claim.Model, flowStartedAt, 0, claimFallback, workstream == "oneshot-swarm-orchestrator", evidenceContextLink, prURL)
		if workstream != "oneshot-swarm-orchestrator" {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	if testsPassed {
		emitWorkerObservability(claim.IssueID, "verify", "success", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
	}

	onesNote, err := updateEvidence(claim.IssueID, claim.Branch, workstream, changedFiles, testsPassed)
	if err != nil {
		emitWorkerObservability(claim.IssueID, "verify", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if strings.TrimSpace(onesNote) != "" {
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", onesNote)
	}
	if !testsPassed {
		emitWorkerObservability(claim.IssueID, "verify", "escalated", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", "worker: go test failed; oneshot verification emitted recovery plan")
		fmt.Fprintln(os.Stderr, "go test ./... failed")
		os.Exit(1)
	}

	if _, err := runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "review", "--apply"); err != nil {
		emitWorkerObservability(claim.IssueID, "review", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := runComponent("pr-gate", "./cmd/pr-gate", "--issue", claim.IssueID, "--prepublish"); err != nil {
		emitWorkerObservability(claim.IssueID, "review", "blocked", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := []string{"add"}
	args = append(args, changedFiles...)
	args = append(args, ".beads/issues.jsonl")
	args = append(args, ".beads/metadata.json")
	if _, err := run("git", args...); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	staged, err := hasStagedChanges()
	if err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if !staged {
		emitWorkerObservability(claim.IssueID, "execute", "blocked", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)
		_, _ = run("bd", "update", claim.IssueID, "--append-notes", "worker: no code diff produced; likely already implemented")
		_, _ = runComponent("beads-fsm", "./cmd/beads-fsm", "--issue", claim.IssueID, "--to", "blocked", "--apply")
		out, _ := json.MarshalIndent(map[string]any{
			"issue":  claim.IssueID,
			"branch": claim.Branch,
			"status": "blocked",
		}, "", "  ")
		fmt.Println(string(out))
		return
	}
	commitBody := commitBodyForWorkstream(workstream)
	if _, err := run("git", "commit", "-m", "worker: implement "+claim.IssueID, "-m", commitBody); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := run("git", "push", "-u", "origin", claim.Branch); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	bodyPath := writePRBody(claim.IssueID, workstream)
	if _, err := runComponent("pr-publish", "./cmd/pr-publish", "--issue", claim.IssueID, "--title", "Worker: "+claim.Title, "--head", claim.Branch, "--base", "master", "--body-file", bodyPath); err != nil {
		emitWorkerObservability(claim.IssueID, "publish", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_, prURL = extractLinkage(claim.IssueID)
	emitWorkerObservability(claim.IssueID, "publish", "success", claim.Model, flowStartedAt, 0, claimFallback, false, evidenceContextLink, prURL)

	out, _ := json.MarshalIndent(map[string]any{
		"issue":  claim.IssueID,
		"branch": claim.Branch,
		"status": "review",
	}, "", "  ")
	fmt.Println(string(out))
}
