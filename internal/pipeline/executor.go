package pipeline

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/llm"
	"sdp_dev/internal/policy"
)

// baseBranch returns the PR base branch: SDP_REPO_BRANCH env or "master" (matches swarm-worker, swarm-reviewer).
func baseBranch() string {
	if b := os.Getenv("SDP_REPO_BRANCH"); b != "" {
		return b
	}
	return "master"
}

// ExecuteTask runs opencode with the SDP agent and the full quality pipeline.
func ExecuteTask(ctx context.Context, b bus.Bus, task federation.FederatedTask) error {
	role := resolveRole(task)
	model := resolveModel(task)
	runID := task.ProjectID + "-" + task.Issue.ID

	// 1. Dispatch: run opencode with existing SDP agent
	execRes, err := dispatchOpencode(ctx, task, role, model)
	if err != nil {
		return fmt.Errorf("dispatch: %w", err)
	}

	// 2. Quality pipeline (post-dispatch)
	return runQualityPipeline(ctx, b, task, role, model, runID, execRes)
}

func resolveRole(task federation.FederatedTask) string {
	for _, l := range task.Issue.Labels {
		if strings.HasPrefix(l, "workstream:") {
			return strings.TrimPrefix(l, "workstream:")
		}
	}
	return "builder"
}

func resolveModel(task federation.FederatedTask) string {
	for _, l := range task.Issue.Labels {
		if strings.HasPrefix(l, "model:") {
			return strings.TrimPrefix(l, "model:")
		}
	}
	return "glm-4.7"
}

func dispatchOpencode(ctx context.Context, task federation.FederatedTask, role, model string) (llm.ExecuteResult, error) {
	boundary, err := llm.LoadBoundary(task.Workspace, role)
	if err != nil {
		return llm.ExecuteResult{}, err
	}

	req := llm.ExecuteRequest{
		IssueID:            task.Issue.ID,
		Title:              task.Issue.Title,
		Description:        task.Issue.Description,
		AcceptanceCriteria: task.Issue.AcceptanceCriteria,
		SpecID:             task.Issue.SpecID,
		Model:              model,
		WorkDir:             task.Workspace,
		Boundary:            boundary,
	}
	return llm.Execute(ctx, req)
}

func runQualityPipeline(ctx context.Context, b bus.Bus, task federation.FederatedTask,
	role, model, runID string, execRes llm.ExecuteResult) error {

	workDir := task.Workspace
	issueID := task.Issue.ID
	branch := "worker/" + issueID

	// 0. Ensure we're on the worker branch (create if needed)
	checkoutCmd := exec.Command("git", "checkout", "-b", branch)
	checkoutCmd.Dir = workDir
	_ = checkoutCmd.Run() // ignore error if branch exists
	checkoutCmd2 := exec.Command("git", "checkout", branch)
	checkoutCmd2.Dir = workDir
	_ = checkoutCmd2.Run()

	// 1. Boundary validation (already done in Execute; reset if violation)
	if execRes.BoundaryViolation != nil {
		resetCmd := exec.Command("git", "reset", "--hard", "HEAD")
		resetCmd.Dir = workDir
		_ = resetCmd.Run()
		return execRes.BoundaryViolation
	}

	changed := execRes.ChangedFiles

	// 2. Evidence collection
	evidence := agent.NewEvidenceCollector(workDir)
	decision := policy.Decide(policy.DecisionRequest{IssueID: issueID, Title: task.Issue.Title})
	riskClass := decision.RiskClass
	if riskClass == "" {
		riskClass = "medium"
	}
	boundary, _ := llm.LoadBoundary(workDir, role)
	if _, err := evidence.Initialize(issueID, branch, riskClass, model, role, boundary); err != nil {
		return fmt.Errorf("evidence init: %w", err)
	}
	_ = evidence.UpdateExecution(issueID, agent.CollectResult{
		ChangedFiles:      changed,
		ModelUsed:         model,
		BoundaryViolation: nil,
		TestsPassed:       false, // updated below
	})

	// 3. Trace emission (before tests so we have run file)
	tracer := agent.NewTraceEmitter(b, task.ProjectID, runID, "swarm-orchestrator", role, workDir)
	_ = tracer.BeginTrace(issueID)
	_ = tracer.EmitPhase("execute", "ok", fmt.Sprintf("%d files changed", len(changed)))

	// 4. Tests
	testCmd := exec.Command("go", "test", "./...")
	testCmd.Dir = workDir
	testsPassed := testCmd.Run() == nil

	_ = evidence.UpdateExecution(issueID, agent.CollectResult{
		ChangedFiles: changed,
		ModelUsed:    model,
		TestsPassed:  testsPassed,
	})
	_ = tracer.EmitPhase("verify", statusStr(testsPassed), "")

	if !testsPassed {
		return fmt.Errorf("go test ./... failed")
	}

	// 5. Provenance signing
	signer := agent.NewProvenanceSigner("swarm-orchestrator", role)
	tracePath := filepath.Join(workDir, ".sdp", "runs", runID+".json")
	evidencePath := filepath.Join(workDir, ".sdp", "evidence", issueID+".json")
	signed, err := signer.Sign(agent.SignInput{
		IssueID:      issueID,
		ArtifactID:   runID + ":strict-evidence",
		ArtifactClass: "artifact",
		Phase:        "completed",
		Payload:      map[string]any{"changed_files": changed},
		ModelUsed:    model,
		TraceLink:    tracePath,
		EvidenceLink: evidencePath,
	})
	if err != nil {
		return fmt.Errorf("provenance sign: %w", err)
	}

	// 6. PR gate
	prGate := exec.Command("pr-gate", "--prepublish", "--issue", issueID)
	prGate.Dir = workDir
	if out, err := prGate.CombinedOutput(); err != nil {
		return fmt.Errorf("pr-gate: %w: %s", err, string(out))
	}

	// 7. FSM transition to review
	fsmCmd := exec.Command("beads-fsm", "--issue", issueID, "--to", "review", "--apply")
	fsmCmd.Dir = workDir
	if out, err := fsmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("beads-fsm: %w: %s", err, string(out))
	}
	_ = fsmCmd

	// 8. Publish provenance to NATS
	if b != nil {
		subject := "sdp.artifact." + task.ProjectID + "." + runID + "." + role
		_ = b.Publish(subject, signed)
	}

	// 9. Commit, push, PR
	return commitAndPublish(workDir, issueID, task.Issue.Title, changed)
}

func statusStr(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}

func commitAndPublish(workDir, issueID, title string, changed []string) error {
	// Stage changed files + beads metadata
	args := append([]string{"add"}, changed...)
	args = append(args, ".beads/issues.jsonl", ".beads/metadata.json")
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add: %w: %s", err, string(out))
	}

	commitCmd := exec.Command("git", "commit", "-m", "worker: implement "+issueID, "-m", "SDP swarm quality pipeline")
	commitCmd.Dir = workDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit: %w: %s", err, string(out))
	}

	pushCmd := exec.Command("git", "push", "-u", "origin", "worker/"+issueID)
	pushCmd.Dir = workDir
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push: %w: %s", err, string(out))
	}

	// Write PR body and publish
	bodyPath := filepath.Join(workDir, ".sdp", "pr-body-"+issueID+".md")
	_ = os.MkdirAll(filepath.Dir(bodyPath), 0o755)
	body := "## Summary\n\n- SDP swarm pipeline execution for " + issueID + "\n- " + title + "\n"
	_ = os.WriteFile(bodyPath, []byte(body), 0o644)

	prTitle := "Worker: " + title
	if prTitle == "Worker: " {
		prTitle = "Worker: " + issueID
	}
	prCmd := exec.Command("pr-publish", "--issue", issueID, "--title", prTitle, "--head", "worker/"+issueID, "--base", baseBranch(), "--body-file", bodyPath)
	prCmd.Dir = workDir
	if out, err := prCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pr-publish: %w: %s", err, string(out))
	}

	return nil
}
