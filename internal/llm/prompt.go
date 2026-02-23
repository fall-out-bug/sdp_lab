package llm

import (
	"fmt"
	"strings"

	"sdp_dev/internal/prompt"
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
	ws := prompt.WorkstreamSpec{
		ID:                 issue.ID,
		Title:              issue.Title,
		Description:        issue.Description,
		AcceptanceCriteria: parseAcceptanceCriteria(issue.AcceptanceCriteria),
		SpecID:             issue.SpecID,
	}
	b.WriteString(prompt.TaskSection(ws))
	bound := prompt.BoundaryInput{
		AllowedPathPrefixes:   boundary.AllowedPathPrefixes,
		ForbiddenPathPrefixes: boundary.ForbiddenPathPrefixes,
		ControlPathPrefixes:   boundary.ControlPathPrefixes,
	}
	b.WriteString(prompt.BoundarySection(bound))
	return b.String()
}

func parseAcceptanceCriteria(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			if strings.HasPrefix(line, "- ") {
				line = strings.TrimPrefix(line, "- ")
			} else if strings.HasPrefix(line, "* ") {
				line = strings.TrimPrefix(line, "* ")
			}
			out = append(out, strings.TrimSpace(line))
		}
	}
	return out
}

// BuildPromptShort returns a one-line prompt for quick tasks (e.g. decomposition).
func BuildPromptShort(issue IssueInput) string {
	return fmt.Sprintf("Task %s: %s. %s", issue.ID, issue.Title, strings.TrimSpace(issue.Description))
}
