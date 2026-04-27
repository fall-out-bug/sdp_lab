//go:build sdp_experimental

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose/adapters/wsverdict"
	"github.com/fall-out-bug/sdp_lab/internal/inference/replayutil"
	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

// dryRunClient returns deterministic LLM responses driven by the current
// fixture's golden verdict. Call SetFixture before each fixture run to
// ensure each fixture's calls reflect the correct expected output.
type dryRunClient struct {
	currentGolden string
}

func newDryRunClient(_ []replayutil.Fixture) *dryRunClient {
	return &dryRunClient{currentGolden: "passed"}
}

// SetFixture binds the client to a fixture's golden verdict for the duration
// of that fixture's pipeline run. Call before monoRunner.Run and decompRunner.Run.
func (d *dryRunClient) SetFixture(f replayutil.Fixture) {
	d.currentGolden = f.GoldenStatus
	if d.currentGolden == "" {
		d.currentGolden = "passed"
	}
}

func (d *dryRunClient) Call(_ context.Context, prompt string, opts wsverdict.CallOptions) (string, int, int, float64, error) {
	tokIn := len(prompt) / 4 // rough BPE estimate

	// Extract stage: prompt asks for changed_files JSON.
	if strings.Contains(prompt, "changed_files") && strings.Contains(prompt, "JSON") {
		out := wsverdict.ExtractOut{
			ChangedFiles: []string{"internal/foo/bar.go"},
			Modules:      []string{"foo"},
			ChangeType:   "feat",
			Summary:      "dry-run extraction",
		}
		data, _ := json.Marshal(out)
		return string(data), tokIn, 50, costFor(opts.Model, tokIn, 50), nil
	}

	// Classify stage: prompt asks for single verdict word.
	if strings.Contains(prompt, "passed, partial, failed") {
		return d.currentGolden, tokIn, 1, costFor(opts.Model, tokIn, 1), nil
	}

	// Aggregate stage: the classified verdict is already bound in d.currentGolden
	// (set via SetFixture before the pipeline run). Use it directly rather than
	// re-extracting from the prompt, which is fragile — the prompt template
	// contains literal "passed" in its instruction text regardless of the verdict.
	verdict := d.currentGolden
	fv := wsverdict.FinalVerdict{
		Verdict: verdict,
		Score:   scoreFor(verdict),
		Summary: "dry-run verdict for " + verdict,
	}
	data, _ := json.Marshal(fv)
	return string(data), tokIn, 50, costFor(opts.Model, tokIn, 50), nil
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
