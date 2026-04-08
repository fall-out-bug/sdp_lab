package discovery

import (
	"context"
	"fmt"
	"strings"
)

// ClaimVerdict is the per-assumption verdict from desk research.
type ClaimVerdict string

const (
	VerdictSupported        ClaimVerdict = "supported"
	VerdictContradicted     ClaimVerdict = "contradicted"
	VerdictInsufficientData ClaimVerdict = "insufficient_data"
)

// FinalVerdict is the overall product GO / PIVOT / KILL recommendation.
type FinalVerdict string

const (
	VerdictGO    FinalVerdict = "GO"
	VerdictPIVOT FinalVerdict = "PIVOT"
	VerdictKILL  FinalVerdict = "KILL"
)

// Evidence is one piece of evidence for or against an assumption.
type Evidence struct {
	Direction  string `json:"direction"`   // "for" | "against"
	Statement  string `json:"statement"`
	SourceURL  string `json:"source_url"`
	IsEstimate bool   `json:"is_estimate"`
}

// ClaimValidation is the desk-research result for one assumption.
type ClaimValidation struct {
	Claim      string       `json:"claim"`
	RATRank    int          `json:"rat_rank"`
	Evidence   []Evidence   `json:"evidence"`
	Verdict    ClaimVerdict `json:"verdict"`
	Confidence float64      `json:"confidence"`
	Notes      string       `json:"notes"`
}

// ValidationResult is the output of Phase 4a VALIDATE.
type ValidationResult struct {
	Claims          []ClaimValidation `json:"claims"`
	FinalVerdict    FinalVerdict      `json:"final_verdict"`
	VerdictReason   string            `json:"verdict_reason"`
	PivotSuggestion string            `json:"pivot_suggestion,omitempty"`
	KillReason      string            `json:"kill_reason,omitempty"`
	NeedsExperiment bool              `json:"needs_experiment"`
	CostUSD         float64           `json:"cost_usd"`
}

// NeedsExperimentFromClaims returns true if any claim has insufficient_data verdict.
func NeedsExperimentFromClaims(claims []ClaimValidation) bool {
	for _, c := range claims {
		if c.Verdict == VerdictInsufficientData {
			return true
		}
	}
	return false
}

// RenderClaimsForSynthesis formats claims as readable text for the synthesis LLM prompt.
func RenderClaimsForSynthesis(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d: %q → %s (confidence: %.2f)\n",
			c.RATRank, c.Claim, strings.ToUpper(string(c.Verdict)), c.Confidence)
		if c.Notes != "" {
			fmt.Fprintf(&b, "  Notes: %s\n", c.Notes)
		}
		for _, e := range c.Evidence {
			fmt.Fprintf(&b, "  [%s] %s", strings.ToUpper(e.Direction), e.Statement)
			if e.IsEstimate {
				fmt.Fprintf(&b, " (estimate)")
			}
			fmt.Fprintf(&b, "\n")
		}
		fmt.Fprintf(&b, "\n")
	}
	return strings.TrimSpace(b.String())
}

// Validate performs adversarial desk-research validation of the top RAT assumptions.
// Body is implemented in Task 2.
func Validate(ctx context.Context, c *LLMClient, frame *FrameResult, h *HypothesisResult) (*ValidationResult, error) {
	_, _, _ = ctx, c, frame
	_ = h
	return nil, fmt.Errorf("validate: not implemented")
}
