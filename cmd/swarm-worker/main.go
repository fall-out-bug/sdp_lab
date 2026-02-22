package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"sdp_dev/internal/observability"
	"sdp_dev/internal/safeid"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

func projectFromIssueID(issueID string) string {
	if idx := strings.Index(issueID, "-"); idx > 0 {
		return issueID[:idx]
	}
	return ""
}

func main() {
	_, _ = observability.SetupTracing("swarm-worker")

	// Optional /metrics when METRICS_ADDR set (e.g. in K8s job)
	if addr := os.Getenv("METRICS_ADDR"); addr != "" {
		go func() {
			_ = observability.ServeMetrics(context.Background(), addr)
		}()
	}

	flowStartedAt := time.Now()
	ctx := context.Background()
	ctx, span := otel.Tracer("swarm-worker").Start(ctx, "WorkerFlow")
	defer span.End()

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

	_, execSpan := otel.Tracer("swarm-worker").Start(ctx, "execute")
	execSpan.SetAttributes(attribute.String("issue", claim.IssueID), attribute.String("workstream", workstream))
	changedFiles := applyWorkstreamFlow(workstream, claim.IssueID, issue, claim.Model)
	execSpan.End()

	_, verifySpan := otel.Tracer("swarm-worker").Start(ctx, "verify")
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
	verifySpan.End()

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
	commitMsg := fmt.Sprintf("feat(%s): %s", projectFromIssueID(claim.IssueID), claim.IssueID)
	if _, err := run("git", "commit", "--no-verify", "-m", commitMsg, "-m", commitBody); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if _, err := run("git", "push", "-u", "origin", claim.Branch); err != nil {
		emitWorkerObservability(claim.IssueID, "execute", "failed", claim.Model, flowStartedAt, 0, claimFallback, true, evidenceContextLink, prURL)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_, publishSpan := otel.Tracer("swarm-worker").Start(ctx, "publish")
	defer publishSpan.End()
	publishSpan.SetAttributes(attribute.String("issue", claim.IssueID), attribute.String("branch", claim.Branch))
	bodyPath := writePRBody(claim.IssueID, workstream)
	projectID := projectFromIssueID(claim.IssueID)
	prArgs := []string{"--issue", claim.IssueID, "--title", "Worker: " + claim.Title, "--head", claim.Branch, "--base", "master", "--body-file", bodyPath}
	if projectID != "" {
		prArgs = append(prArgs, "--project", projectID)
	}
	if _, err := runComponent("pr-publish", "./cmd/pr-publish", prArgs...); err != nil {
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
