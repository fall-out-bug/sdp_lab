package llm

import (
	"fmt"
	"strings"
)

// IssueInput holds issue context for prompt construction.
type IssueInput struct {
	ID                 string
	Title              string
	Description        string
	AcceptanceCriteria string
	SpecID             string
}

// BuildPrompt constructs a structured prompt for opencode run.
func BuildPrompt(issue IssueInput, boundary BoundarySpec) string {
	var b strings.Builder
	b.WriteString("Implement the following task. Make code changes in the repository.\n\n")
	b.WriteString("## Task\n\n")
	b.WriteString("**ID:** " + issue.ID + "\n\n")
	b.WriteString("**Title:** " + issue.Title + "\n\n")
	if issue.Description != "" {
		b.WriteString("**Description:**\n")
		b.WriteString(issue.Description)
		b.WriteString("\n\n")
	}
	if issue.AcceptanceCriteria != "" {
		b.WriteString("**Acceptance Criteria:**\n")
		b.WriteString(issue.AcceptanceCriteria)
		b.WriteString("\n\n")
	}
	if issue.SpecID != "" {
		b.WriteString("**Spec ID:** " + issue.SpecID + "\n\n")
	}
	b.WriteString("## Constraints\n\n")
	b.WriteString("You may ONLY modify files under these path prefixes:\n")
	for _, p := range boundary.AllowedPathPrefixes {
		b.WriteString("- " + p + "\n")
	}
	b.WriteString("\nYou must NOT modify:\n")
	for _, p := range boundary.ForbiddenPathPrefixes {
		b.WriteString("- " + p + "\n")
	}
	for _, p := range boundary.ControlPathPrefixes {
		b.WriteString("- " + p + "\n")
	}
	b.WriteString("\nProduce working, testable code. Run `go test ./...` to verify.\n")
	return b.String()
}

// BuildPromptShort returns a one-line prompt for quick tasks (e.g. decomposition).
func BuildPromptShort(issue IssueInput) string {
	return fmt.Sprintf("Task %s: %s. %s", issue.ID, issue.Title, strings.TrimSpace(issue.Description))
}
