package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sdp_dev/internal/inference/decompose/adapters/wsverdict"
	"sdp_dev/internal/inference/replayutil"
	"sdp_dev/internal/llmclient"
)

// dryRunClient returns deterministic LLM responses based on fixture golden status.
// Used when no API key is available or --dry-run is set.
type dryRunClient struct {
	// Map fixture WSID → golden verdict for deterministic responses.
	goldenByWS map[string]string
	callN      int
}

func newDryRunClient(fixtures []replayutil.Fixture) *dryRunClient {
	m := map[string]string{}
	for _, f := range fixtures {
		wsID, _ := f.Data["ws_id"].(string)
		if wsID != "" {
			m[wsID] = f.GoldenStatus
		}
	}
	return &dryRunClient{goldenByWS: m}
}

func (d *dryRunClient) Call(_ context.Context, prompt string, opts wsverdict.CallOptions) (string, int, int, float64, error) {
	d.callN++
	tokIn := len(prompt) / 4 // rough BPE estimate
	tokOut := 50

	// Detect which stage we're in based on prompt content.
	if strings.Contains(prompt, "changed_files") && strings.Contains(prompt, "JSON") {
		// Extract stage: return synthetic ExtractOut.
		out := wsverdict.ExtractOut{
			ChangedFiles: []string{"internal/foo/bar.go"},
			Modules:      []string{"foo"},
			ChangeType:   "feat",
			Summary:      "dry-run extraction",
		}
		data, _ := json.Marshal(out)
		return string(data), tokIn, tokOut, costFor(opts.Model, tokIn, tokOut), nil
	}
	if strings.Contains(prompt, "passed, partial, failed") || strings.Contains(prompt, "passed\n\nReturn ONLY") {
		// Classify or monolithic stage: look up golden.
		verdict := d.verdictForPrompt(prompt)
		if strings.Contains(prompt, "verdict_report") || strings.Contains(prompt, "verdict report") || strings.Contains(prompt, "score") {
			// Monolithic or aggregate: return FinalVerdict JSON.
			fv := wsverdict.FinalVerdict{
				Verdict: verdict,
				Score:   scoreFor(verdict),
				Summary: "dry-run verdict for " + verdict,
			}
			data, _ := json.Marshal(fv)
			return string(data), tokIn, tokOut, costFor(opts.Model, tokIn, tokOut), nil
		}
		return verdict, tokIn, 1, costFor(opts.Model, tokIn, 1), nil
	}
	// Aggregate stage.
	fv := wsverdict.FinalVerdict{
		Verdict: "passed",
		Score:   0.85,
		Summary: "dry-run aggregate",
	}
	data, _ := json.Marshal(fv)
	return string(data), tokIn, tokOut, costFor(opts.Model, tokIn, tokOut), nil
}

func (d *dryRunClient) verdictForPrompt(_ string) string {
	// Simple heuristic: cycle through known golden verdicts.
	keys := make([]string, 0, len(d.goldenByWS))
	for k := range d.goldenByWS {
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return "passed"
	}
	idx := d.callN % len(keys)
	return d.goldenByWS[keys[idx]]
}

func scoreFor(verdict string) float64 {
	switch verdict {
	case "passed":
		return 0.95
	case "partial":
		return 0.60
	default:
		return 0.20
	}
}

// costFor returns USD cost estimate for a model call (rough).
func costFor(model string, tokIn, tokOut int) float64 {
	if strings.Contains(model, "haiku") {
		return float64(tokIn)*0.00000025 + float64(tokOut)*0.00000125
	}
	// Sonnet
	return float64(tokIn)*0.000003 + float64(tokOut)*0.000015
}

// openRouterClient wraps llmclient.Client to satisfy wsverdict.LLMCaller.
type openRouterClient struct {
	inner *llmclient.Client
}

func newOpenRouterClient(apiKey string) *openRouterClient {
	return &openRouterClient{
		inner: llmclient.New(apiKey, "https://openrouter.ai/api/v1"),
	}
}

func (c *openRouterClient) Call(ctx context.Context, prompt string, opts wsverdict.CallOptions) (string, int, int, float64, error) {
	resp, err := c.inner.Chat(ctx, llmclient.ChatRequest{
		Model: opts.Model,
		Messages: []llmclient.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   opts.MaxTokens,
		Temperature: opts.Temperature,
	})
	if err != nil {
		return "", 0, 0, 0, fmt.Errorf("openrouter: %w", err)
	}
	return resp.Content, resp.InputTokens, resp.OutputTokens, resp.CostUSD, nil
}
