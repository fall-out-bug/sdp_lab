package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ScanResult struct {
	Items            []ScanItem `json:"items"`
	Whitespace       string     `json:"whitespace"`
	RecommendedStack []string   `json:"recommended_stack"`
	CostUSD          float64    `json:"cost_usd"`
}

func (r *ScanResult) Settled() []ScanItem {
	var out []ScanItem
	for _, item := range r.Items {
		if item.DepthFlag == nil || !item.DepthFlag.Flagged {
			out = append(out, item)
		}
	}
	return out
}

func (r *ScanResult) Flagged() []ScanItem {
	var out []ScanItem
	for _, item := range r.Items {
		if item.DepthFlag != nil && item.DepthFlag.Flagged {
			out = append(out, item)
		}
	}
	return out
}

const scanSystemPrompt = `You are a market intelligence agent. Analyze the problem and identify relevant tools, frameworks, and competitors.
Respond ONLY with valid JSON — no markdown, no explanation.`

const scanUserPromptTpl = `Scan the market for tools relevant to this problem.

PROBLEM: %s
JOBS: %s

Find 5–8 relevant tools/frameworks/products. For each, provide honest coverage metadata based on how well you actually know the tool.

Return JSON:
{"items":[{"name":"string","disposition":"ADOPT|EXTRACT|INSPIRE|MONITOR|IGNORE","covers_phases":["frame|hypothesize|scan|validate|experiment"],"key_strength":"string","key_gap":"string","stars":0,"primary_source_read":false,"architecture_reviewed":false,"desc_sentences":3,"source_count":1,"multi_source":false,"disposition_confidence":0.5}],"whitespace":"string describing the gap nobody fills","recommended_stack":["string"]}`

func Scan(ctx context.Context, c *LLMClient, frame *FrameResult) (*ScanResult, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultSynthModel,
		Messages: []Message{
			{Role: "system", Content: scanSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(scanUserPromptTpl, frame.ProblemStatement, jobs)},
		},
		MaxTokens:   2000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("scan llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	// strip markdown fences if model disobeyed
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) > 2 {
			content = strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	var raw struct {
		Items            []ScanItem `json:"items"`
		Whitespace       string     `json:"whitespace"`
		RecommendedStack []string   `json:"recommended_stack"`
	}
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("scan parse (finish=%s): %w\ncontent: %s",
			resp.FinishReason, err, content)
	}

	// Apply depth evaluation to each item
	for i := range raw.Items {
		score := CoverageScore(raw.Items[i])
		raw.Items[i].CoverageScore = score
		flag := EvalDepth(raw.Items[i])
		if flag.Flagged {
			raw.Items[i].DepthFlag = &flag
		}
	}

	return &ScanResult{
		Items:            raw.Items,
		Whitespace:       raw.Whitespace,
		RecommendedStack: raw.RecommendedStack,
		CostUSD:          resp.CostUSD,
	}, nil
}
