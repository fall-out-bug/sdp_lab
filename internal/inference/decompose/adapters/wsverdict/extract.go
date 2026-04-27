package wsverdict

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sdp_dev/internal/inference/decompose"
)

const modelHaiku = "anthropic/claude-haiku-4-5"

// newExtractStage returns Stage[Diff, ExtractOut] that calls Haiku with a JSON
// prompt, parses the response, and returns structured extraction output.
func newExtractStage(client LLMCaller) decompose.Stage[Diff, ExtractOut] {
	return decompose.NewStage[Diff, ExtractOut]("extract", func(ctx context.Context, d Diff) (ExtractOut, decompose.StageTrace, error) {
		start := time.Now()
		prompt := extractPrompt(d)
		text, tokIn, tokOut, cost, err := client.Call(ctx, prompt, CallOptions{
			Model:     modelHaiku,
			MaxTokens: 512,
		})
		trace := decompose.StageTrace{
			LatencyMs:   time.Since(start).Milliseconds(),
			TokensIn:    tokIn,
			TokensOut:   tokOut,
			CostUSD:     cost,
			RawResponse: text,
		}
		if err != nil {
			return ExtractOut{}, trace, fmt.Errorf("extract LLM call: %w", err)
		}
		out, err := parseJSON[ExtractOut](text)
		if err != nil {
			return ExtractOut{}, trace, fmt.Errorf("extract parse: %w", err)
		}
		return out, trace, nil
	})
}

func extractPrompt(d Diff) string {
	return fmt.Sprintf(`You are a code change analyzer. Analyze the following workstream diff and return ONLY valid JSON matching this schema:
{
  "changed_files": ["list of changed file paths"],
  "modules": ["list of high-level modules/packages affected"],
  "change_type": "feat|fix|refactor|docs|test|chore",
  "summary": "one sentence description of the change"
}

Workstream: %s
Context: %s

Diff:
%s

Return ONLY the JSON object, no markdown, no explanation.`, d.WSID, d.Context, d.DiffText)
}

func parseJSON[T any](text string) (T, error) {
	var out T
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return out, fmt.Errorf("json parse: %w (raw: %.100s)", err, text)
	}
	return out, nil
}
