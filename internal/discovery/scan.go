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

Find 5–8 relevant tools/frameworks/products. For each, provide honest coverage metadata.

IMPORTANT coverage field rules:
- primary_source_read: true ONLY if you have read their actual docs, README, or website (not just training memory). Default false.
- architecture_reviewed: true ONLY if you have reviewed their technical architecture docs. Default false.
- desc_sentences: count of sentences you can write from verified knowledge (not guesses).
- source_count: number of distinct sources you can cite for this tool.
- multi_source: true only if source_count >= 2 AND sources are from different domains.
- disposition_confidence: 0.0-1.0, how confident you are in the disposition given your actual knowledge depth.

Be conservative — most tools should have primary_source_read: false in this initial scan.

Return JSON:
{"items":[{"name":"string","disposition":"ADOPT|EXTRACT|INSPIRE|MONITOR|IGNORE","covers_phases":["frame|hypothesize|scan|validate|experiment"],"key_strength":"string","key_gap":"string","stars":0,"primary_source_read":false,"architecture_reviewed":false,"desc_sentences":3,"source_count":1,"multi_source":false,"disposition_confidence":0.5}],"whitespace":"string describing the gap nobody fills","recommended_stack":["string"]}`

// ApplyResolutions applies human checkpoint-C decisions to a ScanResult.
// "downgrade" -> change disposition to MONITOR and unflag the item.
// "proceed_provisional" -> leave disposition unchanged, keep DepthFlag.
// "deep_dive" -> treated as proceed_provisional for now (real deep dive is future work).
func ApplyResolutions(r *ScanResult, resolutions map[string]string) *ScanResult {
	if len(resolutions) == 0 {
		return r
	}
	updated := &ScanResult{
		Items:            make([]ScanItem, len(r.Items)),
		Whitespace:       r.Whitespace,
		RecommendedStack: r.RecommendedStack,
		CostUSD:          r.CostUSD,
	}
	copy(updated.Items, r.Items)
	for i, item := range updated.Items {
		res, ok := resolutions[item.Name]
		if !ok {
			continue
		}
		switch res {
		case "downgrade":
			updated.Items[i].Disposition = DispositionMonitor
			updated.Items[i].DepthFlag = nil // clear flag — decision made
		case "deep_dive":
			// Treated as proceed_provisional until deep dive is implemented.
			// Flag is preserved; user can re-run with real sources.
		}
		// proceed_provisional: no change to disposition or flag
	}
	return updated
}

func Scan(ctx context.Context, c *LLMClient, frame *FrameResult) (*ScanResult, error) {
	jobs := strings.Join(frame.Jobs, "; ")
	resp, err := c.Chat(ctx, ChatRequest{
		Model: DefaultSynthModel,
		Messages: []Message{
			{Role: "system", Content: scanSystemPrompt},
			{Role: "user", Content: fmt.Sprintf(scanUserPromptTpl, frame.ProblemStatement, jobs)},
		},
		MaxTokens:   8000,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("scan llm: %w", err)
	}
	content := strings.TrimSpace(resp.Content)
	content = stripMarkdownFences(content)
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
