// internal/discovery/clarify.go
package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ClarificationType is the category of a clarification request (from DeerFlow INSPIRE).
type ClarificationType string

const (
	ClarifyMissingInfo          ClarificationType = "missing_info"
	ClarifyAmbiguousRequirement ClarificationType = "ambiguous_requirement"
	ClarifyApproachChoice       ClarificationType = "approach_choice"
	ClarifyRiskConfirmation     ClarificationType = "risk_confirmation"
)

// ClarificationRequest is a typed question for the human before hypothesis generation.
type ClarificationRequest struct {
	Type     ClarificationType `json:"type"`
	Question string            `json:"question"`
	Context  string            `json:"context"`           // why this question matters
	Options  []string          `json:"options,omitempty"` // only for approach_choice
}

const clarifySystemPrompt = `You are a product discovery agent that identifies gaps in problem framing.
Respond ONLY with valid JSON — no markdown, no explanation.`

const clarifyUserPromptTpl = `Identify 2–3 clarifying questions that would materially improve the product hypothesis for this problem.

PROBLEM: %s
JOBS: %s

Use these types — each question MUST use a DIFFERENT type. Do NOT make all questions missing_info:
- missing_info: key data or context we don't have (e.g., who exactly is the customer, what volume, what budget)
- ambiguous_requirement: something in the problem statement that could mean two incompatible things
- approach_choice: a fork where choosing A vs B changes the core design significantly (always provide 2–3 concrete options)
- risk_confirmation: a high-stakes assumption where being wrong would invalidate the entire concept

DIVERSITY RULE: If you have 3 questions, use 3 different types. If 2 questions, use 2 different types.

Return JSON:
{"clarifications":[{"type":"missing_info|ambiguous_requirement|approach_choice|risk_confirmation","question":"specific, answerable question","context":"why this matters for the hypothesis","options":["option A","option B"]}]}`

// GenerateClarifications produces typed clarification questions for the human before hypothesis generation.
func GenerateClarifications(ctx context.Context, c *LLMClient, frame *FrameResult) ([]ClarificationRequest, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultPlannerModel,
		Messages: []Message{
			{Role: "system", Content: clarifySystemPrompt},
			{Role: "user", Content: fmt.Sprintf(clarifyUserPromptTpl,
				frame.ProblemStatement, jobs)},
		},
		MaxTokens:   600,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("clarify llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)

	var raw struct {
		Clarifications []ClarificationRequest `json:"clarifications"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("clarify parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	return raw.Clarifications, nil
}
