package omoclient

import (
	"fmt"
	"strings"
)

// GovernancePromptBuilder converts a TaskEnvelope into a governance preamble
// that is prepended to the agent prompt. This ensures OmO agents follow
// SDP discipline without any runtime changes to the agent itself.
type GovernancePromptBuilder struct{}

// NewGovernancePromptBuilder creates a new prompt builder.
func NewGovernancePromptBuilder() *GovernancePromptBuilder {
	return &GovernancePromptBuilder{}
}

// Build creates a governance preamble from the task envelope.
// The preamble is prepended to the user's prompt before sending to OmO.
func (b *GovernancePromptBuilder) Build(envelope TaskEnvelope) string {
	var sb strings.Builder

	// Header
	sb.WriteString("## SDP Governance Rules\n\n")
	sb.WriteString("You are executing a governed task. The following rules MUST be followed.\n\n")

	// Task identity
	fmt.Fprintf(&sb, "**Task:** %s (phase: %s)\n\n", envelope.TaskID, envelope.Phase)
	fmt.Fprintf(&sb, "**Objective:** %s\n\n", envelope.Objective)

	// Scope
	if len(envelope.ScopeIn) > 0 || len(envelope.ScopeOut) > 0 {
		sb.WriteString("### Scope\n\n")

		if len(envelope.ScopeIn) > 0 {
			sb.WriteString("**Files you MAY modify:**\n")
			for _, f := range envelope.ScopeIn {
fmt.Fprintf(&sb, "- `%s`\n", f)			}
			sb.WriteString("\n")
		}

		if len(envelope.ScopeOut) > 0 {
			sb.WriteString("**Files you MUST NOT modify (out of scope):**\n")
			for _, f := range envelope.ScopeOut {
fmt.Fprintf(&sb, "- `%s`\n", f)			}
			sb.WriteString("\n")
		}

		sb.WriteString("**Scope violations will be detected and blocked.** Stay within scope.\n\n")
	}

	// Constraints
	if len(envelope.Constraints) > 0 {
		sb.WriteString("### Constraints\n\n")
		for _, c := range envelope.Constraints {
fmt.Fprintf(&sb, "- %s\n", c)		}
		sb.WriteString("\n")
	}

	// Governance rules
	if envelope.Governance != nil {
		g := envelope.Governance
		sb.WriteString("### Execution Rules\n\n")

		if g.MaxToolCalls > 0 {
fmt.Fprintf(&sb, "- Maximum %d tool calls per task\n", g.MaxToolCalls)		}

		if g.MustCiteEvidence {
			sb.WriteString("- Every claim MUST be supported by evidence (test output, file content, command output)\n")
			sb.WriteString("- \"I think it works\" is NOT evidence. Tool output IS evidence.\n")
		}

		if g.MustReportOOS {
			sb.WriteString("- If you need to modify files outside the allowed scope, STOP and report it\n")
			sb.WriteString("- Do NOT modify out-of-scope files silently\n")
		}

		sb.WriteString("\n")
	}

	// Separator
	sb.WriteString("---\n\n")
	sb.WriteString("## Task\n\n")

	return sb.String()
}

// BuildFullPrompt combines governance preamble with the user prompt.
func (b *GovernancePromptBuilder) BuildFullPrompt(envelope TaskEnvelope, userPrompt string) string {
	return b.Build(envelope) + userPrompt
}

// BuildEvidencePrompt creates instructions for the agent to report findings
// in a parseable format. Appended to the end of the prompt.
func (b *GovernancePromptBuilder) BuildEvidencePrompt() string {
	return `

---

## Required Report

At the end of your work, provide a structured report in this exact format:

### Files Changed
- List each file you modified

### Evidence
- For each change: what tool output confirms it works

### Scope Check
- List any files you considered modifying but did NOT (and why)

### Findings
- Any issues, risks, or concerns discovered during execution`
}
