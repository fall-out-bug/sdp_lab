package discuss

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sdp_dev/internal/llm"
)

// OpenRouterAnalyzer uses the OpenRouter API to analyze feature ideas.
type OpenRouterAnalyzer struct {
	Client *llm.OpenRouterClient
	Model  string
}

// NewOpenRouterAnalyzer returns an analyzer using OPENROUTER_API_KEY.
func NewOpenRouterAnalyzer(model string) *OpenRouterAnalyzer {
	if model == "" {
		model = "anthropic/claude-sonnet-4"
	}
	return &OpenRouterAnalyzer{
		Client: llm.NewOpenRouterClient(),
		Model:  model,
	}
}

// Analyze calls the LLM to analyze the feature and returns structured analysis.
func (a *OpenRouterAnalyzer) Analyze(ctx context.Context, sess *Session) (*AnalysisResult, error) {
	systemPrompt := `You are a software architect. Analyze the feature idea and respond with a JSON object only (no markdown, no explanation).
The JSON must have exactly these keys:
- "scope": string, brief description of what the feature entails
- "risks": array of strings, potential risks or concerns
- "subtasks": array of objects, each with: "title", "description", "acceptance", and optionally "depends_on_index" (0-based index of prior subtask)
Create 2-8 subtasks. Each subtask should be independently implementable.`

	userPrompt := fmt.Sprintf("Feature: %s\n\nDescription: %s\n\nProvide analysis as JSON only.", sess.Title, sess.Description)

	content, err := a.Client.Complete(ctx, a.Model, systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSpace(content)
	}

	var raw struct {
		Scope    string            `json:"scope"`
		Risks    []string          `json:"risks"`
		Subtasks []SubtaskProposal `json:"subtasks"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("parse analysis JSON: %w", err)
	}

	return &AnalysisResult{
		Scope:      raw.Scope,
		Risks:      raw.Risks,
		Subtasks:   raw.Subtasks,
		ModelUsed:  a.Model,
		AnalyzedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}
