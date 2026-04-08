package discovery

import (
	"context"
	"encoding/json"
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

const validateClaimSystemPrompt = `You are an adversarial product analyst. Find evidence BOTH FOR and AGAINST the given assumption. Be intellectually honest — do not favour confirmation. Cite real studies or data where you can; flag all estimates clearly.
Respond ONLY with valid JSON — no markdown, no explanation.`

const validateClaimUserPromptTpl = `Evaluate this product assumption using desk research.

ASSUMPTION (RAT rank %d, score %.0f): %s
PROBLEM CONTEXT: %s

Find 3–4 pieces of evidence FOR this assumption and 3–4 pieces AGAINST it.
For each piece of evidence:
- direction: "for" or "against"
- statement: one specific, concrete finding (not vague)
- source_url: URL if you know a real one; empty string if not — do NOT fabricate URLs
- is_estimate: true if pattern-based reasoning; false only if citing a known study/report/dataset

After listing evidence, assess the verdict:
- "supported": preponderance of credible evidence supports the assumption
- "contradicted": preponderance of credible evidence refutes it
- "insufficient_data": evidence is mixed, weak, or mostly estimated

Return JSON:
{"evidence":[{"direction":"for|against","statement":"...","source_url":"","is_estimate":true}],"verdict":"supported|contradicted|insufficient_data","confidence":0.7,"notes":"one-sentence synthesis"}`

const validateSynthesisSystemPrompt = `You are a product strategist giving a final GO / PIVOT / KILL recommendation.
Respond ONLY with valid JSON — no markdown, no explanation.`

const validateSynthesisUserPromptTpl = `Based on validated product assumptions, give a final verdict.

PROBLEM: %s

VALIDATED CLAIMS:
%s

Rules:
- GO: majority of high-RAT claims are "supported" with reasonable confidence (>= 0.6)
- KILL: the rank-1 claim is "contradicted" with high confidence (>= 0.7), OR majority are contradicted
- PIVOT: mixed — some supported, some contradicted or insufficient_data; a pivot direction is visible

Return JSON:
{"final_verdict":"GO|PIVOT|KILL","verdict_reason":"2-3 sentences referencing specific claims","pivot_suggestion":"what to pivot to (only if PIVOT, else omit)","kill_reason":"why kill (only if KILL, else omit)"}`

const validateTopN = 3

// validateClaim runs one adversarial desk-research LLM call for a single assumption.
// Returns the ClaimValidation and the LLM cost in USD.
func validateClaim(ctx context.Context, c *LLMClient, problem string, a Assumption) (ClaimValidation, float64, error) {
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: validateClaimSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(validateClaimUserPromptTpl,
				a.RATRank, a.RATScore, a.Statement, problem)},
		},
		MaxTokens:   1500,
		Temperature: 0.1,
	})
	if err != nil {
		return ClaimValidation{}, 0, fmt.Errorf("validate claim llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw struct {
		Evidence   []Evidence   `json:"evidence"`
		Verdict    ClaimVerdict `json:"verdict"`
		Confidence float64      `json:"confidence"`
		Notes      string       `json:"notes"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return ClaimValidation{}, 0, fmt.Errorf("validate claim parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}

	return ClaimValidation{
		Claim:      a.Statement,
		RATRank:    a.RATRank,
		Evidence:   raw.Evidence,
		Verdict:    raw.Verdict,
		Confidence: raw.Confidence,
		Notes:      raw.Notes,
	}, resp.CostUSD, nil
}

// synthResult is a helper for unmarshalling the synthesis LLM response.
type synthResult struct {
	FinalVerdict    FinalVerdict `json:"final_verdict"`
	VerdictReason   string       `json:"verdict_reason"`
	PivotSuggestion string       `json:"pivot_suggestion"`
	KillReason      string       `json:"kill_reason"`
}

// validateSynthesis runs the synthesis LLM call to produce the overall GO/PIVOT/KILL verdict.
// Returns the synthResult and LLM cost in USD.
func validateSynthesis(ctx context.Context, c *LLMClient, problem string, claims []ClaimValidation) (synthResult, float64, error) {
	rendered := RenderClaimsForSynthesis(claims)
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: validateSynthesisSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(validateSynthesisUserPromptTpl, problem, rendered)},
		},
		MaxTokens:   800,
		Temperature: 0.1,
	})
	if err != nil {
		return synthResult{}, 0, fmt.Errorf("validate synthesis llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var sr synthResult
	if err := json.Unmarshal([]byte(content), &sr); err != nil {
		return synthResult{}, 0, fmt.Errorf("validate synthesis parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	return sr, resp.CostUSD, nil
}

// Validate performs adversarial desk-research validation of the top RAT assumptions
// and synthesises a GO / PIVOT / KILL verdict.
func Validate(ctx context.Context, c *LLMClient, frame *FrameResult, h *HypothesisResult) (*ValidationResult, error) {
	top := h.Assumptions
	if len(top) > validateTopN {
		top = top[:validateTopN]
	}

	var totalCost float64
	claims := make([]ClaimValidation, 0, len(top))

	for _, a := range top {
		cv, cost, err := validateClaim(ctx, c, frame.ProblemStatement, a)
		if err != nil {
			return nil, fmt.Errorf("validate claim rank %d: %w", a.RATRank, err)
		}
		claims = append(claims, cv)
		totalCost += cost
	}

	sr, cost, err := validateSynthesis(ctx, c, frame.ProblemStatement, claims)
	if err != nil {
		return nil, fmt.Errorf("validate synthesis: %w", err)
	}
	totalCost += cost

	return &ValidationResult{
		Claims:          claims,
		FinalVerdict:    sr.FinalVerdict,
		VerdictReason:   sr.VerdictReason,
		PivotSuggestion: sr.PivotSuggestion,
		KillReason:      sr.KillReason,
		NeedsExperiment: NeedsExperimentFromClaims(claims),
		CostUSD:         totalCost,
	}, nil
}
