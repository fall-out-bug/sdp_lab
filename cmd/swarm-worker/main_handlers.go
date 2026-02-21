package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"sdp_dev/internal/llm"
)

func applyEvaluatorRecommendationWorkstream(repo string, issueID string, detail issueDetail) []string {
	path := filepath.Join(repo, "docs", "EVALUATOR_RECOMMENDATIONS_LOG.md")
	content := fmt.Sprintf("\n## %s\n- %s\n", detail.ID, detail.Title)
	if b, err := os.ReadFile(path); err == nil {
		content = string(b) + content
	} else {
		content = "# Evaluator Recommendations Log\n" + content
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(content), 0o644)
	return []string{path}
}

func applySelfImprovementWorkstream(repo string, issueID string, detail issueDetail) []string {
	path := filepath.Join(repo, "docs", "SELF_IMPROVEMENT_LOG.md")
	content := fmt.Sprintf("\n## %s\n- %s\n- spec: %s\n", detail.ID, detail.Title, detail.SpecID)
	if b, err := os.ReadFile(path); err == nil {
		content = string(b) + content
	} else {
		content = "# Self-Improvement Log\n" + content
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte(content), 0o644)
	return []string{path}
}

func applyBuilderWorkstream(repo string, issueID string, detail issueDetail, model string) ([]string, error) {
	lastBuilderResult = nil
	boundary, err := llm.LoadBoundary(repo, "builder")
	if err != nil {
		return nil, fmt.Errorf("load boundary: %w", err)
	}
	req := llm.ExecuteRequest{
		IssueID:            issueID,
		Title:              detail.Title,
		Description:        detail.Description,
		AcceptanceCriteria: detail.AcceptanceCriteria,
		SpecID:             detail.SpecID,
		Model:              model,
		WorkDir:             repo,
		Boundary:           boundary,
	}
	res, err := llm.Execute(context.Background(), req)
	if err != nil {
		if res.BoundaryViolation != nil {
			resetCmd := exec.Command("git", "reset", "--hard", "HEAD")
			resetCmd.Dir = repo
			_ = resetCmd.Run()
			note := fmt.Sprintf("worker: boundary violation: %s", res.BoundaryViolation.Error())
			bdCmd := exec.Command("bd", "update", issueID, "--append-notes", note)
			bdCmd.Dir = repo
			_, _ = bdCmd.CombinedOutput()
			blockCmd := exec.Command("bd", "update", issueID, "--status", "blocked")
			blockCmd.Dir = repo
			_, _ = blockCmd.CombinedOutput()
		}
		return nil, err
	}
	lastBuilderResult = &res
	return res.ChangedFiles, nil
}

var lastBuilderResult *llm.ExecuteResult

func appendHandoffValidationTimestamp(repo string) error {
	path := filepath.Join(repo, "docs", "AGENT_HANDOFF.md")
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ts := time.Now().UTC().Format(time.RFC3339)
	line := "\n\n## Validation Run\n\n- " + ts + " (workstream:handoff-validation)\n"
	return os.WriteFile(path, append(b, []byte(line)...), 0o644)
}
