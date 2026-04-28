package wsverdict

import (
	"context"
	"fmt"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
)

// MonolithicRunner wraps a single-shot pipeline that emits the same
// FinalVerdict shape as the decomposed runner, enabling fair A/B comparison.
// Result.StageResults has exactly one entry with Name=="monolithic".
type MonolithicRunner struct {
	pipeline *decompose.Pipeline[FinalVerdict]
}

// NewMonolithicRunner creates a single-shot Sonnet pipeline for ws-verdict.
// The monolithic runner uses the same model as the decomposed classify stage
// for fair A/B: Sonnet for the full prompt.
func NewMonolithicRunner(client LLMCaller) *MonolithicRunner {
	stage := decompose.NewStage[Diff, FinalVerdict]("monolithic", func(ctx context.Context, d Diff) (FinalVerdict, decompose.StageTrace, error) {
		start := time.Now()
		prompt := monolithicPrompt(d)
		text, tokIn, tokOut, cost, err := client.Call(ctx, prompt, CallOptions{
			Model:     modelSonnet,
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
			return FinalVerdict{}, trace, fmt.Errorf("monolithic LLM call: %w", err)
		}
		out, err := parseJSON[FinalVerdict](text)
		if err != nil {
			return FinalVerdict{}, trace, fmt.Errorf("monolithic parse: %w", err)
		}
		if !isAllowed(out.Verdict, allowedVerdicts) {
			return FinalVerdict{}, trace, fmt.Errorf("monolithic: invalid verdict %q", out.Verdict)
		}
		if out.Score < 0 || out.Score > 1 {
			return FinalVerdict{}, trace, fmt.Errorf("monolithic: score %v out of range [0, 1]", out.Score)
		}
		return out, trace, nil
	})

	p := decompose.New[FinalVerdict]("ws-verdict-monolithic")
	decompose.Then(p, stage, decompose.StageConfig{
		OnFailure: decompose.Abort,
	})

	return &MonolithicRunner{pipeline: p}
}

// Run executes the single-shot pipeline on the given diff.
func (r *MonolithicRunner) Run(ctx context.Context, diff Diff) (decompose.Result[FinalVerdict], error) {
	result, err := r.pipeline.Run(ctx, diff)
	if err != nil {
		return result, err
	}
	// Verify exactly one synthetic stage named "monolithic".
	if len(result.StageResults) != 1 || result.StageResults[0].Name != "monolithic" {
		return result, fmt.Errorf("monolithic: unexpected stage results structure")
	}
	return result, nil
}
