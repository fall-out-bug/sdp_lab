package roles

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/llm"
)

// ReviewerStrategy performs persona-based review.
type ReviewerStrategy struct {
	PersonaID string // e.g. "security", "dx", "correctness"
}

// Execute runs review and returns a verdict.
func (r *ReviewerStrategy) Execute(ctx context.Context, input TaskInput) (TaskResult, error) {
	issue := input.FederatedTask.Issue
	workDir := input.FederatedTask.Workspace

	model := "glm-5"
	if input.Ctx != nil && input.Ctx.Policy != nil {
		model = input.Ctx.Policy.SelectModelSimple()
	}

	evidencePath := filepath.Join(workDir, ".sdp", "evidence", issue.ID+".json")
	evidenceContent, _ := os.ReadFile(evidencePath)

	prompt := r.buildReviewPrompt(issue, string(evidenceContent), r.PersonaID)
	boundary := llm.BoundarySpec{
		AllowedPathPrefixes:   []string{".sdp/", "docs/"},
		ControlPathPrefixes:   llm.DefaultControlPaths,
		ForbiddenPathPrefixes: llm.DefaultForbiddenPaths,
	}
	req := llm.ExecuteRequest{
		IssueID:    issue.ID,
		Title:      "Review: " + issue.Title,
		Description: prompt,
		Model:      model,
		WorkDir:    workDir,
		Boundary:   boundary,
	}
	res, err := llm.Execute(ctx, req)
	if err != nil {
		return TaskResult{Err: err, Verdict: "reject"}, err
	}

	verdict := parseVerdictFromOutput(res.Stdout + res.Stderr)
	if verdict == "" {
		verdict = "needs_changes"
	}
	return TaskResult{
		Verdict:  verdict,
		Summary:  res.Stdout,
		Comments: extractComments(res.Stdout),
	}, nil
}

func (r *ReviewerStrategy) buildReviewPrompt(issue beads.Issue, evidence string, persona string) string {
	var b strings.Builder
	b.WriteString("You are a " + persona + " reviewer. Review this task and evidence.\n\n")
	b.WriteString("## Task\n")
	b.WriteString("ID: " + issue.ID + "\n")
	b.WriteString("Title: " + issue.Title + "\n")
	if issue.Description != "" {
		b.WriteString("Description: " + issue.Description + "\n")
	}
	b.WriteString("\n## Evidence\n")
	if evidence != "" {
		b.WriteString(evidence)
	} else {
		b.WriteString("(no evidence file found)\n")
	}
	b.WriteString("\n## Output\n")
	b.WriteString("Respond with JSON: {\"verdict\": \"approve\"|\"needs_changes\"|\"reject\", \"comments\": [\"...\"]}\n")
	return b.String()
}

func parseVerdictFromOutput(out string) string {
	var m struct {
		Verdict string `json:"verdict"`
	}
	// Find JSON in output
	start := strings.Index(out, "{")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(out, "}")
	if end <= start {
		return ""
	}
	if err := json.Unmarshal([]byte(out[start:end+1]), &m); err != nil {
		return ""
	}
	switch m.Verdict {
	case "approve", "needs_changes", "reject":
		return m.Verdict
	}
	return ""
}

func extractComments(out string) []string {
	var m struct {
		Comments []string `json:"comments"`
	}
	start := strings.Index(out, "{")
	if start < 0 {
		return nil
	}
	end := strings.LastIndex(out, "}")
	if end <= start {
		return nil
	}
	if err := json.Unmarshal([]byte(out[start:end+1]), &m); err != nil {
		return nil
	}
	return m.Comments
}
