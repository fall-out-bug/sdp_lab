package pipeline

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/llm"
	"sdp_dev/internal/policy"
	"sdp_dev/internal/quality"
)

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
	decision := policy.Decide(policy.DecisionRequest{IssueID: issueID, Title: task.Issue.Title})
	riskClass := decision.RiskClass
	if riskClass == "" {
		riskClass = "medium"
	}
	boundary, _ := llm.LoadBoundary(workDir, role)

	// 2. Evidence collection (init + first update)
	if _, err := quality.CollectEvidence(quality.EvidenceConfig{
		WorkDir:      workDir,
		IssueID:      issueID,
		Branch:       branch,
		RiskClass:    riskClass,
		Model:        model,
		Role:         role,
		Boundary:     boundary,
		ChangedFiles: changed,
		ModelUsed:    model,
		TestsPassed:  false,
	}); err != nil {
		return err
	}

	// 3. Trace emission (before tests so we have run file)
	tracer := agent.NewTraceEmitter(b, task.ProjectID, runID, "swarm-orchestrator", role, workDir)
	_ = tracer.BeginTrace(issueID)
	_ = tracer.EmitPhase("execute", "ok", fmt.Sprintf("%d files changed", len(changed)))

	// 4. Tests
	testsPassed, _ := quality.RunTests(workDir)
	if err := quality.UpdateEvidence(workDir, issueID, agent.CollectResult{
		ChangedFiles: changed,
		ModelUsed:    model,
		TestsPassed:  testsPassed,
	}); err != nil {
		return fmt.Errorf("evidence update: %w", err)
	}
	_ = tracer.EmitPhase("verify", statusStr(testsPassed), "")

	if !testsPassed {
		return fmt.Errorf("go test ./... failed")
	}

	// 5. Provenance signing
	tracePath := filepath.Join(workDir, ".sdp", "runs", runID+".json")
	evidencePath := filepath.Join(workDir, ".sdp", "evidence", issueID+".json")
	signed, err := quality.SignProvenance(quality.ProvenanceConfig{
		AgentID:      "swarm-orchestrator",
		Role:         role,
		IssueID:      issueID,
		ArtifactID:   runID + ":strict-evidence",
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
	if err := quality.RunPRGate(issueID, workDir); err != nil {
		return err
	}

	// 7. FSM transition to review
	if err := quality.TransitionFSM(issueID, "review", workDir); err != nil {
		return err
	}

	// 8. Publish provenance to NATS
	if b != nil {
		subject := "sdp.artifact." + task.ProjectID + "." + runID + "." + role
		_ = b.Publish(subject, signed)
	}

	// 9. Commit, push, PR
	_, err = quality.CommitAndPublish(quality.PublishConfig{
		WorkDir:     workDir,
		IssueID:    issueID,
		Title:      task.Issue.Title,
		Changed:    changed,
		BaseBranch: quality.BaseBranch(),
	})
	return err
}

func statusStr(ok bool) string {
	if ok {
		return "ok"
	}
	return "failed"
}
