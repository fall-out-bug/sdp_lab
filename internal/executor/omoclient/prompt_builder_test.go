package omoclient

import (
	"strings"
	"testing"
)

func TestGovernancePromptBuilder_ScopeOnly(t *testing.T) {
	b := NewGovernancePromptBuilder()
	envelope := TaskEnvelope{
		TaskID:    "F082",
		Phase:     "build",
		Objective: "Implement auth middleware",
		ScopeIn:   []string{"internal/auth/**", "cmd/server/**"},
		ScopeOut:  []string{"deploy/**", "docs/**"},
	}

	prompt := b.Build(envelope)

	if !strings.Contains(prompt, "F082") {
		t.Error("missing task ID")
	}
	if !strings.Contains(prompt, "internal/auth/**") {
		t.Error("missing scope in")
	}
	if !strings.Contains(prompt, "deploy/**") {
		t.Error("missing scope out")
	}
	if !strings.Contains(prompt, "Scope violations will be detected") {
		t.Error("missing scope warning")
	}
	if !strings.Contains(prompt, "## Task") {
		t.Error("missing task section")
	}
}

func TestGovernancePromptBuilder_Constraints(t *testing.T) {
	b := NewGovernancePromptBuilder()
	envelope := TaskEnvelope{
		TaskID:      "F083",
		Phase:       "review",
		Objective:   "Review auth module",
		Constraints: []string{"No new dependencies", "Maintain backward compat"},
	}

	prompt := b.Build(envelope)

	if !strings.Contains(prompt, "No new dependencies") {
		t.Error("missing constraint")
	}
	if !strings.Contains(prompt, "Maintain backward compat") {
		t.Error("missing constraint")
	}
}

func TestGovernancePromptBuilder_GovernanceRules(t *testing.T) {
	b := NewGovernancePromptBuilder()
	envelope := TaskEnvelope{
		TaskID:    "F084",
		Phase:     "build",
		Objective: "Add caching",
		Governance: &GovernanceConfig{
			MaxToolCalls:     50,
			MustCiteEvidence: true,
			MustReportOOS:    true,
		},
	}

	prompt := b.Build(envelope)

	if !strings.Contains(prompt, "50 tool calls") {
		t.Error("missing max tool calls")
	}
	if !strings.Contains(prompt, "Tool output IS evidence") {
		t.Error("missing evidence rule")
	}
	if !strings.Contains(prompt, "STOP and report it") {
		t.Error("missing OOS report rule")
	}
}

func TestGovernancePromptBuilder_FullPrompt(t *testing.T) {
	b := NewGovernancePromptBuilder()
	envelope := TaskEnvelope{
		TaskID:    "F082",
		Phase:     "build",
		Objective: "Implement auth",
		ScopeIn:   []string{"internal/auth/**"},
	}
	userPrompt := "Add JWT validation to the middleware."

	full := b.BuildFullPrompt(envelope, userPrompt)

	if !strings.Contains(full, "SDP Governance Rules") {
		t.Error("missing governance header")
	}
	if !strings.Contains(full, "Add JWT validation") {
		t.Error("missing user prompt")
	}
	// Governance should come before user prompt
	govIdx := strings.Index(full, "SDP Governance Rules")
	userIdx := strings.Index(full, "Add JWT validation")
	if govIdx > userIdx {
		t.Error("governance should precede user prompt")
	}
}

func TestGovernancePromptBuilder_EvidencePrompt(t *testing.T) {
	b := NewGovernancePromptBuilder()
	ev := b.BuildEvidencePrompt()

	if !strings.Contains(ev, "Files Changed") {
		t.Error("missing files changed section")
	}
	if !strings.Contains(ev, "Evidence") {
		t.Error("missing evidence section")
	}
	if !strings.Contains(ev, "Scope Check") {
		t.Error("missing scope check section")
	}
	if !strings.Contains(ev, "Findings") {
		t.Error("missing findings section")
	}
}

func TestGovernancePromptBuilder_EmptyEnvelope(t *testing.T) {
	b := NewGovernancePromptBuilder()
	envelope := TaskEnvelope{
		TaskID:    "F099",
		Phase:     "exploration",
		Objective: "Explore codebase",
	}

	prompt := b.Build(envelope)

	if !strings.Contains(prompt, "F099") {
		t.Error("missing task ID")
	}
	// Should not contain scope section
	if strings.Contains(prompt, "Files you MAY modify") {
		t.Error("should not have scope section for empty scope")
	}
}
