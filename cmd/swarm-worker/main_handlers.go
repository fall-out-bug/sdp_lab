package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
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

func applyGenericWorkstream(repo string, issueID string, detail issueDetail) []string {
	path := filepath.Join(repo, "docs", "GENERIC_TASK_PLACEHOLDER.md")
	content := fmt.Sprintf("# Generic Task Placeholder: %s\n\n- spec_id: %s\n- description: %s\n\n## Acceptance\n\n%s\n\n*Full LLM-based implementation requires opencode-implement or similar tool.*\n",
		issueID, detail.SpecID, detail.Description, detail.AcceptanceCriteria)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil
	}
	return []string{path}
}

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
