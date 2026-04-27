package decompose

import (
	"context"
	"fmt"

	"sdp_dev/internal/inference/confidence"
)

// ConfidenceCheckResult is the outcome of a ConfidenceRunner.Run call.
type ConfidenceCheckResult struct {
	Score   float64
	Status  Status
	Reasons []string
}

// ConfidenceRunner is a type-erased wrapper around confidence.Checker[T].
// Use NewConfidenceRunner[T] to create one from a typed Checker.
type ConfidenceRunner interface {
	// Run executes the confidence check on answer (typed as any) and returns
	// the aggregate score, status, reasons, and any error.
	// input is the prompt/context string; raw is the unparsed LLM response.
	Run(ctx context.Context, input, raw string, answer any) (ConfidenceCheckResult, error)
}

// typedConfidenceRunner adapts confidence.Checker[T] to ConfidenceRunner.
type typedConfidenceRunner[T any] struct {
	checker *confidence.Checker[T]
}

// NewConfidenceRunner wraps a typed confidence.Checker[T] as a ConfidenceRunner
// that can be stored in StageConfig without knowing T at the pipeline level.
func NewConfidenceRunner[T any](checker *confidence.Checker[T]) ConfidenceRunner {
	return &typedConfidenceRunner[T]{checker: checker}
}

func (r *typedConfidenceRunner[T]) Run(ctx context.Context, input, raw string, answer any) (ConfidenceCheckResult, error) {
	typed, ok := answer.(T)
	if !ok {
		var zero T
		return ConfidenceCheckResult{}, fmt.Errorf("confidence runner: expected %T, got %T", zero, answer)
	}
	result, err := r.checker.Check(ctx, confidence.Request[T]{
		Input:  input,
		Answer: typed,
		Raw:    raw,
	})
	if err != nil {
		return ConfidenceCheckResult{}, err
	}
	return ConfidenceCheckResult{
		Score:   result.Score,
		Status:  mapConfidenceStatus(result.Status),
		Reasons: result.Reasons,
	}, nil
}

func mapConfidenceStatus(s confidence.Status) Status {
	switch s {
	case confidence.StatusOK:
		return StatusOK
	case confidence.StatusUnsure:
		return StatusUnsure
	default:
		return StatusFail
	}
}

// runCore executes a single stage with the full integration stack:
// cascade → confidence → stitcher.
// It returns the raw output, the populated StageTrace, and any error.
// Failure policy application is handled by the caller (pipeline.go runStage).
func runCore(ctx context.Context, s anyStage, cfg StageConfig, in any) (any, StageTrace, error) {
	var out any
	var trace StageTrace
	var err error

	// 1. Cascade wraps the raw LLM call thunk (F145 — swappable interface).
	if cfg.Cascade != nil {
		var cascadeTrace CascadeTrace
		fn := func() (any, StageTrace, error) {
			return s.runAny(ctx, in)
		}
		out, trace, cascadeTrace, err = cfg.Cascade.Invoke(ctx, fn)
		trace.CascadeLog = &cascadeTrace
	} else {
		out, trace, err = s.runAny(ctx, in)
	}
	if err != nil {
		return nil, trace, err
	}

	// 2. Confidence check (F144) on the stage output.
	if cfg.Confidence != nil {
		co, cerr := cfg.Confidence.Run(ctx, "", trace.RawResponse, out)
		if cerr != nil {
			return nil, trace, fmt.Errorf("confidence: %w", cerr)
		}
		trace.ConfidenceLog = &ConfidenceLog{
			Score:   co.Score,
			Status:  co.Status,
			Reasons: co.Reasons,
		}
	}

	// 3. Stitcher validates the output format.
	if cfg.Stitcher != nil {
		if serr := cfg.Stitcher.Validate(out); serr != nil {
			return nil, trace, fmt.Errorf("stitcher %s: %w", cfg.Stitcher.Name(), serr)
		}
	}

	return out, trace, nil
}

// stageResultFromCore converts the core execution output into a StageResult,
// applying confidence score/status if available.
func stageResultFromCore(name string, out any, trace StageTrace, err error) StageResult {
	sr := StageResult{
		Name:  name,
		Out:   out,
		Trace: trace,
		Err:   err,
	}
	if err != nil {
		sr.Status = StatusFail
		sr.SubScore = 0.0
		return sr
	}
	if trace.ConfidenceLog != nil {
		sr.Status = trace.ConfidenceLog.Status
		sr.SubScore = trace.ConfidenceLog.Score
	} else {
		sr.Status = StatusOK
		sr.SubScore = 1.0
	}
	return sr
}
