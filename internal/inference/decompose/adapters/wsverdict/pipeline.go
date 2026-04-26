package wsverdict

import (
	"context"

	"sdp_dev/internal/inference/decompose"
)

// DecomposedRunner wraps the 3-stage decomposed pipeline.
type DecomposedRunner struct {
	pipeline *decompose.Pipeline[FinalVerdict]
}

// NewDecomposedRunner creates a 3-stage ws-verdict pipeline:
//   - Stage 1 (extract): Haiku, JSON output, RetryOnce
//   - Stage 2 (classify): Sonnet, Enum output, Abort
//   - Stage 3 (aggregate): Haiku, JSON→FinalVerdict, Fallback
func NewDecomposedRunner(client LLMCaller) *DecomposedRunner {
	p := decompose.New[FinalVerdict]("ws-verdict-decomposed")

	decompose.Then(p, newExtractStage(client), decompose.StageConfig{
		OnFailure: decompose.RetryOnce,
		Stitcher: decompose.MustNewJSONStitcherFromBytes("extract",
			[]byte(`{"type":"object","required":["changed_files","modules","change_type","summary"],"properties":{"changed_files":{"type":"array","items":{"type":"string"}},"modules":{"type":"array","items":{"type":"string"}},"change_type":{"type":"string"},"summary":{"type":"string"}}}`)),
	})

	decompose.Then(p, newClassifyStage(client), decompose.StageConfig{
		OnFailure: decompose.Abort,
		Stitcher:  decompose.NewEnumStitcher("classify", allowedVerdicts),
	})

	decompose.Then(p, newAggregateStage(client), decompose.StageConfig{
		OnFailure:   decompose.Fallback,
		FallbackOut: FinalVerdict{Verdict: "failed", Score: 0.0, Summary: "aggregate stage failed"},
	})

	return &DecomposedRunner{pipeline: p}
}

// Run executes the 3-stage pipeline on the given diff.
func (r *DecomposedRunner) Run(ctx context.Context, diff Diff) (decompose.Result[FinalVerdict], error) {
	return r.pipeline.Run(ctx, diff)
}
