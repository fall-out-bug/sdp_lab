package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ExperimentFormat is the type of cheapest experiment to run.
type ExperimentFormat string

const (
	ExperimentSmokeTest        ExperimentFormat = "smoke_test"
	ExperimentLandingPage      ExperimentFormat = "landing_page"
	ExperimentCustomerInterview ExperimentFormat = "customer_interview"
	ExperimentWizardOfOz       ExperimentFormat = "wizard_of_oz"
)

// ExperimentBrief is the output of Phase 4b: the cheapest test design
// for resolving assumptions that desk research could not settle.
type ExperimentBrief struct {
	Format        ExperimentFormat `json:"format"`
	Objective     string           `json:"objective"`
	Hypothesis    string           `json:"hypothesis"`
	SuccessMetric string           `json:"success_metric"`
	TimeBoxDays   int              `json:"time_box_days"`
	SetupSteps    []string         `json:"setup_steps"`
	RequiredTools []string         `json:"required_tools"`
	RawClaim      string           `json:"raw_claim"`
	CostUSD       float64          `json:"cost_usd"`
}

const experimentSystemPrompt = `You are a lean experiment designer. Given unresolved product assumptions, design the single cheapest experiment that would produce a clear signal within 1–2 weeks.
Respond ONLY with valid JSON — no markdown, no explanation.`

const experimentUserPromptTpl = `Design the cheapest experiment to resolve these unvalidated product assumptions.

PROBLEM: %s
JOBS: %s

UNRESOLVED CLAIMS (insufficient_data):
%s

Choose ONE experiment format:
- "smoke_test": build a no-code landing page or waitlist page to measure demand (best for: demand/pricing signals)
- "landing_page": one-page value proposition site with CTA click measurement (best for: positioning/messaging)
- "customer_interview": 5 structured interviews with target customers (best for: job/pain validation)
- "wizard_of_oz": manually deliver the product behind the scenes to test willingness to pay/use (best for: trust/usage patterns)

Return JSON:
{"format":"smoke_test|landing_page|customer_interview|wizard_of_oz","objective":"one sentence — what will this experiment prove or disprove","hypothesis":"if [we do X], then [Y%% of target users] will [measurable action] within [N days]","success_metric":"specific number + metric + time bound","time_box_days":7,"setup_steps":["step 1","step 2","step 3"],"required_tools":["tool 1"],"raw_claim":"the primary assumption being tested"}`

// insufficientClaims returns claims with insufficient_data verdict from a ValidationResult.
func insufficientClaims(v *ValidationResult) []ClaimValidation {
	var out []ClaimValidation
	for _, c := range v.Claims {
		if c.Verdict == VerdictInsufficientData {
			out = append(out, c)
		}
	}
	return out
}

// renderInsufficientClaims formats insufficient_data claims for the experiment prompt.
func renderInsufficientClaims(claims []ClaimValidation) string {
	var b strings.Builder
	for _, c := range claims {
		fmt.Fprintf(&b, "Rank %d: %q (confidence: %.2f)\n  Notes: %s\n\n",
			c.RATRank, c.Claim, c.Confidence, c.Notes)
	}
	return strings.TrimSpace(b.String())
}

// GenerateExperiment designs the cheapest experiment to resolve insufficient_data assumptions.
// It is called only when validation.NeedsExperiment == true.
func GenerateExperiment(ctx context.Context, c *LLMClient, frame *FrameResult, val *ValidationResult) (*ExperimentBrief, error) {
	claims := insufficientClaims(val)
	if len(claims) == 0 {
		return nil, fmt.Errorf("GenerateExperiment called but no insufficient_data claims found")
	}

	jobs := strings.Join(frame.Jobs, "; ")
	rendered := renderInsufficientClaims(claims)

	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultReasonerModel,
		Messages: []Message{
			{Role: "system", Content: experimentSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(experimentUserPromptTpl,
				frame.ProblemStatement, jobs, rendered)},
		},
		MaxTokens:   1000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("experiment llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw ExperimentBrief
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("experiment parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	raw.CostUSD = resp.CostUSD
	return &raw, nil
}
