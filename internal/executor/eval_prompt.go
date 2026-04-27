package executor

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/augmentation"
	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/prompt"
)

func BuildEvaluationPrompt(model string, card *control.FeatureCard, evidence *buildEvidence, changedFiles []string, diff string) string {
	if strings.TrimSpace(model) == "" {
		model = "default"
	}
	evidenceJSON, _ := json.MarshalIndent(evidence, "", "  ")
	base := fmt.Sprintf(`You are the SDP build evaluator.

Evaluate the completed build for card %s.
Model hint: %s.
Return JSON only with this exact shape:
{
  "verdict": "pass|fail|needs_review",
  "score": 0.0,
  "findings": ["specific issue"],
  "passed": {
    "tests_pass": true,
    "scope_adherence": true,
    "evidence_complete": true,
    "code_quality": true
  }
}

Rules:
- Be strict and evidence-based.
- Use only the supplied evidence, card scope, changed files, and git diff.
- Fail if tests/evidence clearly show failure.
- Fail if changed files violate scope constraints.
- Use needs_review for ambiguous or medium-confidence cases.
- Score must be between 0 and 1.
- Findings must be specific and concise.
- Respond in English.
- JSON only. No markdown fences.

Card:
- ID: %s
- Title: %s
- Objective: %s
- Scope In: %s
- Scope Out: %s
- Required Artifacts: %s

Build Evidence JSON:
%s

Changed Files:
%s

Relevant Diff (truncated if needed):
%s
`, card.ID, model, card.ID, card.Title, card.NormalizedIntent, joinQuoted(card.ScopeIn), joinQuoted(card.ScopeOut), joinQuoted(card.RequiredArtifacts), string(evidenceJSON), joinQuoted(changedFiles), diff)
	if packSection := prompt.ContextSegmentsSection("Pack Context", augmentation.MustResolveDefaultPromptContext("reviewer.pack")); packSection != "" {
		return packSection + "\n\n" + base
	}
	return base
}

func joinQuoted(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}
