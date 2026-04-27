package wsverdict

import (
	"context"
	"strconv"
	"strings"

	"sdp_dev/internal/inference/decompose"
	wsverdict_micro "sdp_dev/internal/inference/microfirst/wsverdict"
)

const microConfidenceThreshold = 0.85

// DecomposedRunnerOptions controls micro-first behaviour for DecomposedRunner.
type DecomposedRunnerOptions struct {
	// MicroFirst enables the micro classifier pre-gate (default true via NewDecomposedRunner).
	// When true, confident PASS/FAIL results from the micro classifier skip the LLM pipeline.
	MicroFirst bool
	// MicroRules configures the micro classifier thresholds.
	MicroRules wsverdict_micro.RulesConfig
}

// DecomposedRunner wraps the 3-stage decomposed pipeline.
type DecomposedRunner struct {
	pipeline *decompose.Pipeline[FinalVerdict]
	opts     DecomposedRunnerOptions
}

// NewDecomposedRunner creates a 3-stage ws-verdict pipeline with MicroFirst=true (default on):
//   - Stage 1 (extract): Haiku, JSON output, RetryOnce
//   - Stage 2 (classify): Sonnet, Enum output, Abort
//   - Stage 3 (aggregate): Haiku, JSON→FinalVerdict, Fallback
func NewDecomposedRunner(client LLMCaller) *DecomposedRunner {
	return NewDecomposedRunnerWithOpts(client, DecomposedRunnerOptions{MicroFirst: true})
}

// NewDecomposedRunnerWithOpts creates a DecomposedRunner with optional micro-first pre-gate.
func NewDecomposedRunnerWithOpts(client LLMCaller, opts DecomposedRunnerOptions) *DecomposedRunner {
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

	return &DecomposedRunner{pipeline: p, opts: opts}
}

// Run executes the pipeline on the given diff.
// When MicroFirst is enabled, a heuristic micro classifier is run first.
// If it returns a confident PASS or FAIL (confidence ≥ 0.85), the LLM pipeline is skipped.
// UNSURE results fall through to the full LLM pipeline.
func (r *DecomposedRunner) Run(ctx context.Context, diff Diff) (decompose.Result[FinalVerdict], error) {
	if r.opts.MicroFirst {
		micro := wsverdict_micro.New(r.opts.MicroRules)
		input := diffToMicroInput(diff)
		microResult, trace, _ := micro.Run(ctx, input)

		if microResult.Confidence() >= microConfidenceThreshold {
			// Confident result — skip LLM pipeline entirely.
			verdict := string(microResult.Verdict)
			// Normalize micro verdict names to pipeline verdict names.
			switch verdict {
			case "pass":
				verdict = "passed"
			case "fail":
				verdict = "failed"
			}
			return decompose.Result[FinalVerdict]{
				Answer: FinalVerdict{
					Verdict: verdict,
					Score:   microResult.Confidence(),
					Summary: strings.Join(microResult.Reasons, "; "),
				},
				Status: microResult.ConfStatus(),
				Score:  microResult.Confidence(),
				StageResults: []decompose.StageResult{
					{
						Name:     micro.Name(),
						Status:   microResult.ConfStatus(),
						SubScore: microResult.Confidence(),
						Out:      microResult,
						Trace:    trace,
					},
				},
				Trace: decompose.AggregateTrace{
					LatencyMs: trace.LatencyMs,
					TokensIn:  trace.TokensIn,
					TokensOut: trace.TokensOut,
					CostUSD:   trace.CostUSD,
				},
				Reasons: microResult.Reasons,
			}, nil
		}
	}

	return r.pipeline.Run(ctx, diff)
}

// diffToMicroInput converts a Diff into a WsVerdictInput using best-effort heuristics.
// It parses the diff text for test signal lines:
//   - "FAIL\t" or "--- FAIL" → failed test count
//   - "--- SKIP" or "=== SKIP" → skipped test count
//   - "coverage: X%" → coverage percentage
//   - "build failed" or "ERRORS:" → errored count (prevents R2 PASS from firing)
func diffToMicroInput(diff Diff) wsverdict_micro.WsVerdictInput {
	var failed, skipped, errored int
	var coverage float64

	for _, line := range strings.Split(diff.DiffText, "\n") {
		lower := strings.ToLower(line)
		switch {
		case strings.Contains(line, "FAIL\t") || strings.HasPrefix(line, "--- FAIL"):
			failed++
		case strings.Contains(line, "--- SKIP") || strings.Contains(line, "=== SKIP"):
			skipped++
		case strings.Contains(lower, "build failed") || strings.Contains(lower, "[errors]"):
			errored++
		case strings.Contains(line, "coverage:") && strings.Contains(line, "%"):
			// Parse "coverage: 85.0% of statements"
			idx := strings.Index(line, "coverage:")
			if idx >= 0 {
				rest := strings.TrimSpace(line[idx+len("coverage:"):])
				pctIdx := strings.Index(rest, "%")
				if pctIdx > 0 {
					numStr := strings.TrimSpace(rest[:pctIdx])
					if v, err := strconv.ParseFloat(numStr, 64); err == nil {
						coverage = v
					}
				}
			}
		}
	}

	return wsverdict_micro.WsVerdictInput{
		Report: wsverdict_micro.TestReport{
			Failed:   failed,
			Skipped:  skipped,
			Errored:  errored,
			Coverage: coverage,
		},
		Guard: wsverdict_micro.GuardDiff{}, // no out-of-scope detection from diff text
	}
}
