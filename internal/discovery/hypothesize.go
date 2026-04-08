package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Assumption is one hypothesis assumption with RAT (Riskiest Assumption Test) metadata.
type Assumption struct {
	Statement   string  `json:"statement"`
	RiskLevel   string  `json:"risk_level"`  // high|medium|low — impact if wrong
	Uncertainty string  `json:"uncertainty"` // high|medium|low — how unknown
	RATScore    float64 `json:"rat_score"`   // computed: risk_val × uncertainty_val (1–9)
	RATRank     int     `json:"rat_rank"`    // 1 = riskiest; assigned after sort
}

// HypothesisResult is the output of Phase 2 HYPOTHESIZE.
type HypothesisResult struct {
	WeBelieve    string       `json:"we_believe"`      // Strategyzer Test Card: belief statement
	ToVerify     string       `json:"to_verify"`       // cheapest test
	WeMeasure    string       `json:"we_measure"`      // key metric
	WeAreRightIf string       `json:"we_are_right_if"` // success criterion
	Assumptions  []Assumption `json:"assumptions"`     // RAT-ranked, index 0 = riskiest
	Requirements []string     `json:"requirements"`    // functional requirements
	RawIdea      string       `json:"raw_idea"`
}

const hypothesizeSystemPrompt = `You are a product hypothesis agent specializing in Strategyzer Test Cards and assumption mapping.
Respond ONLY with valid JSON — no markdown, no explanation.`

const hypothesizeUserPromptTpl = `Generate a product hypothesis for this problem using the Strategyzer Test Card format.

PROBLEM: %s
JOBS: %s
APPETITE: %s

QUALITY RULES:
- we_believe: name the SPECIFIC customer segment and their exact job (avoid generic "users need to"). Example of BAD: "users need a faster tool because it saves time". Example of GOOD: "solo founders need to validate market demand before writing code because they cannot afford 6-week discovery cycles".
- to_verify: must be the CHEAPEST possible test — often a 5-question customer interview, a one-page doc, or a smoke-test page — NOT a full prototype. A landing page is only appropriate if the core risk is demand, not usability.
- we_are_right_if: must include a NUMBER, a METRIC, and a TIME BOUND (e.g., "8 out of 10 interviewees rank this as their #1 friction point" or ">30%% click-through on smoke-test page within 7 days").
- assumptions: include 4–6 assumptions with genuinely VARIED risk/uncertainty combinations — NOT all high/high. At least one should be medium/low or low/medium to reflect known knowns.
- requirements: describe WHAT the product does for the user (functional), not HOW it is built (technical). Example of BAD: "use LLM API with streaming". Example of GOOD: "user can upload a transcript and receive a structured insight summary within 60 seconds".

Return JSON with this exact schema:
{"we_believe":"[specific segment] need to [specific job] because [specific reason]","to_verify":"cheapest test to validate the core assumption","we_measure":"the key metric","we_are_right_if":"measurable success criterion with number, metric, and time bound","assumptions":[{"statement":"assumption that must be true for the hypothesis to hold","risk_level":"high|medium|low","uncertainty":"high|medium|low"}],"requirements":["functional requirement 1","functional requirement 2"]}`

// Hypothesize generates a Strategyzer Test Card and RAT-ranked assumptions from a FrameResult.
func Hypothesize(ctx context.Context, c *LLMClient, frame *FrameResult) (*HypothesisResult, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultPlannerModel,
		Messages: []Message{
			{Role: "system", Content: hypothesizeSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(hypothesizeUserPromptTpl,
				frame.ProblemStatement, jobs, frame.Appetite)},
		},
		MaxTokens:   1500,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("hypothesize llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	// LLM returns assumptions without scores — parse raw first
	var raw struct {
		WeBelieve    string `json:"we_believe"`
		ToVerify     string `json:"to_verify"`
		WeMeasure    string `json:"we_measure"`
		WeAreRightIf string `json:"we_are_right_if"`
		Assumptions  []struct {
			Statement   string `json:"statement"`
			RiskLevel   string `json:"risk_level"`
			Uncertainty string `json:"uncertainty"`
		} `json:"assumptions"`
		Requirements []string `json:"requirements"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("hypothesize parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}

	// Compute RAT scores and rank
	assumptions := make([]Assumption, len(raw.Assumptions))
	for i, a := range raw.Assumptions {
		assumptions[i] = Assumption{
			Statement:   a.Statement,
			RiskLevel:   a.RiskLevel,
			Uncertainty: a.Uncertainty,
			RATScore:    ratScore(a.RiskLevel, a.Uncertainty),
		}
	}
	assumptions = computeRATRanks(assumptions)

	return &HypothesisResult{
		WeBelieve:    raw.WeBelieve,
		ToVerify:     raw.ToVerify,
		WeMeasure:    raw.WeMeasure,
		WeAreRightIf: raw.WeAreRightIf,
		Assumptions:  assumptions,
		Requirements: raw.Requirements,
		RawIdea:      frame.RawIdea,
	}, nil
}

// ratScore converts risk and uncertainty text levels to a numeric RAT score (1–9).
func ratScore(riskLevel, uncertainty string) float64 {
	val := map[string]float64{"high": 3, "medium": 2, "low": 1}
	r := val[riskLevel]
	u := val[uncertainty]
	if r == 0 {
		r = 1
	}
	if u == 0 {
		u = 1
	}
	return r * u
}

// computeRATRanks sorts assumptions by RAT score descending and assigns RATRank (1 = riskiest).
func computeRATRanks(assumptions []Assumption) []Assumption {
	sorted := make([]Assumption, len(assumptions))
	copy(sorted, assumptions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].RATScore > sorted[j].RATScore
	})
	for i := range sorted {
		sorted[i].RATRank = i + 1
	}
	return sorted
}
