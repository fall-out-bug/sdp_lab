package executor

import (
	"fmt"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/augmentation"
	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/prompt"
)

const planSystemPrompt = `You are an SDP implementation planner. Your job is to create a detailed implementation plan for a FeatureCard before execution.

Given the card intent, scope, risk level, phase, and project context:
1. Analyze the task requirements
2. Identify specific files that need to be created or modified
3. Determine what tests should be written
4. Assess risks and dependencies
5. Estimate the number of implementation steps

Rules:
- Be specific about files. Use exact file paths from the project structure.
- Consider existing patterns in the codebase when planning.
- Break down complex tasks into clear, actionable steps.
- Identify potential risks that could block or delay implementation.
- Recommend appropriate test coverage.

Output JSON only, no markdown:
{
  "approach": "Brief description of the implementation approach",
  "files_to_change": ["path/to/file1.go", "path/to/file2.go"],
  "tests_to_write": ["path/to/file1_test.go"],
  "risk_assessment": "Description of potential risks and mitigations",
  "estimated_steps": 5
}`

func BuildPlanPrompt(projectRoot string, card *control.FeatureCard) string {
	var sections []string
	sections = append(sections, planSystemPrompt)
	if packSection := prompt.ContextSegmentsSection("Pack Context", augmentation.MustResolveDefaultPromptContext("planner.pack")); packSection != "" {
		sections = append(sections, packSection)
	}

	sections = append(sections, "=== CARD DETAILS ===")
	sections = append(sections, fmt.Sprintf("Card ID: %s", card.ID))
	sections = append(sections, fmt.Sprintf("Title: %s", card.Title))

	if strings.TrimSpace(card.NormalizedIntent) != "" {
		sections = append(sections, fmt.Sprintf("Intent: %s", card.NormalizedIntent))
	} else if strings.TrimSpace(card.RawRequest) != "" {
		sections = append(sections, fmt.Sprintf("Intent: %s", card.RawRequest))
	}

	if card.TaskType != "" {
		sections = append(sections, fmt.Sprintf("Phase: %s", card.TaskType))
	}

	if card.RiskLevel != "" {
		sections = append(sections, fmt.Sprintf("Risk Level: %s", card.RiskLevel))
	}

	if len(card.ScopeIn) > 0 {
		sections = append(sections, fmt.Sprintf("Scope In: %s", strings.Join(card.ScopeIn, ", ")))
	}

	if len(card.ScopeOut) > 0 {
		sections = append(sections, fmt.Sprintf("Scope Out: %s", strings.Join(card.ScopeOut, ", ")))
	}

	if len(card.RequiredArtifacts) > 0 {
		sections = append(sections, fmt.Sprintf("Required Artifacts: %s", strings.Join(card.RequiredArtifacts, ", ")))
	}

	if len(card.RequiredChecks) > 0 {
		sections = append(sections, fmt.Sprintf("Required Checks: %s", strings.Join(card.RequiredChecks, ", ")))
	}

	sections = append(sections, "=== PROJECT CONTEXT ===")
	sections = append(sections, collectProjectContext(projectRoot))

	return strings.Join(sections, "\n\n")
}
