package wsverdict

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"sdp_dev/internal/inference/decompose"
)

// newAggregateStage returns Stage[string, FinalVerdict] that calls Haiku with a
// TOON-formatted prompt and returns the FinalVerdict.
// For v1 the stage returns JSON; TOON output format is used in evidence logging.
func newAggregateStage(client LLMCaller) decompose.Stage[string, FinalVerdict] {
	return decompose.NewStage[string, FinalVerdict]("aggregate", func(ctx context.Context, verdict string) (FinalVerdict, decompose.StageTrace, error) {
		start := time.Now()
		prompt := aggregatePrompt(verdict)
		text, tokIn, tokOut, cost, err := client.Call(ctx, prompt, CallOptions{
			Model:     modelHaiku,
			MaxTokens: 256,
		})
		trace := decompose.StageTrace{
			LatencyMs: time.Since(start).Milliseconds(),
			TokensIn:  tokIn,
			TokensOut: tokOut,
			CostUSD:   cost,
		}
		if err != nil {
			return FinalVerdict{}, trace, fmt.Errorf("aggregate LLM call: %w", err)
		}
		out, err := parseJSON[FinalVerdict](text)
		if err != nil {
			return FinalVerdict{}, trace, fmt.Errorf("aggregate parse: %w", err)
		}
		if !isAllowed(out.Verdict, allowedVerdicts) {
			return FinalVerdict{}, trace, fmt.Errorf("aggregate: invalid verdict %q", out.Verdict)
		}
		return out, trace, nil
	})
}

func aggregatePrompt(verdict string) string {
	return fmt.Sprintf(`Given the workstream verdict classification "%s", produce a final verdict report as JSON:
{
  "verdict": "%s",
  "score": <float 0.0-1.0 reflecting confidence>,
  "summary": "<one sentence rationale>",
  "blocking_gates": ["<list only if verdict is partial or failed>"]
}

Return ONLY the JSON object. blocking_gates should be empty array for "passed".`, verdict, verdict)
}

// toonFinalVerdictColumns defines the TOON evidence columns for FinalVerdict.
var toonFinalVerdictColumns = []decompose.TOONColumn{
	{Name: "verdict", Type: "string"},
	{Name: "score", Type: "float"},
	{Name: "summary", Type: "string"},
}

// MarshalFinalVerdictTOON serializes a FinalVerdict as a TOON table row for evidence logging.
func MarshalFinalVerdictTOON(fv FinalVerdict) (string, error) {
	s := decompose.NewTOONStitcher("final_verdict", toonFinalVerdictColumns)
	rows := []map[string]any{
		{"verdict": fv.Verdict, "score": fv.Score, "summary": fv.Summary},
	}
	return s.Marshal(rows)
}

// MarshalEvidenceJSON marshals a FinalVerdict to indented JSON evidence.
func MarshalEvidenceJSON(fv FinalVerdict) (string, error) {
	data, err := json.MarshalIndent(fv, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
