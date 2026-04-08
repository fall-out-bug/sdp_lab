package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type FrameResult struct {
	ProblemStatement string   `json:"problem_statement"`
	Jobs             []string `json:"jobs"`     // JTBD: who does what, to achieve what
	Appetite         string   `json:"appetite"` // small/medium/large
	Scope            string   `json:"scope"`    // what's in, what's out
	RawIdea          string   `json:"raw_idea"`
}

const frameSystemPrompt = `You are a product discovery agent specializing in problem framing.
Respond ONLY with valid JSON — no markdown, no explanation.`

const frameUserPromptTpl = `Frame this raw idea into a structured problem.

RAW IDEA: %s

Return JSON:
{"problem_statement":"string","jobs":["who does what to achieve what"],"appetite":"small|medium|large","scope":"string"}`

func Frame(ctx context.Context, c *LLMClient, idea string) (*FrameResult, error) {
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultPlannerModel,
		Messages: []Message{
			{Role: "system", Content: frameSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(frameUserPromptTpl, idea)},
		},
		MaxTokens:   800,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("frame llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)
	var result FrameResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, fmt.Errorf("frame parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}
	result.RawIdea = idea
	return &result, nil
}
