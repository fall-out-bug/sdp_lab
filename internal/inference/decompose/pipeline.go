package decompose

import (
	"context"
	"fmt"
	"time"
)

// Pipeline executes a sequential chain of stages, collecting per-stage results
// into a Result[Final]. Final is the output type of the last stage.
type Pipeline[Final any] struct {
	name    string
	stages  []anyStage
	configs []StageConfig
}

// New creates a new, empty Pipeline named name. Add stages with Then.
func New[Final any](name string) *Pipeline[Final] {
	return &Pipeline[Final]{name: name}
}

// Then appends stage to the pipeline with the given StageConfig.
// Returns the same *Pipeline to allow chaining: p.Then(s1, c1).Then(s2, c2).
func Then[Final, In, Out any](p *Pipeline[Final], stage Stage[In, Out], cfg StageConfig) *Pipeline[Final] {
	p.stages = append(p.stages, wrapAny[In, Out](stage))
	p.configs = append(p.configs, cfg)
	return p
}

// Run executes the pipeline sequentially, starting with in as the first stage
// input. Each stage receives the previous stage's output as its input.
// Run honours context cancellation and per-stage Timeout in StageConfig.
func (p *Pipeline[Final]) Run(ctx context.Context, in any) (Result[Final], error) {
	var stageResults []StageResult
	var aggTrace AggregateTrace
	var reasons []string

	current := in
	for i, s := range p.stages {
		cfg := p.configs[i]
		sr, next, err := p.runStage(ctx, s, cfg, current)
		stageResults = append(stageResults, sr)
		aggTrace.add(sr.Trace)
		if sr.Err != nil {
			reasons = append(reasons, fmt.Sprintf("stage %s: %v", s.stageName(), sr.Err))
		}
		if err != nil {
			// Abort or unrecoverable error: return what we have so far.
			statuses := collectStatuses(stageResults)
			return Result[Final]{
				Status:       aggregateStatus(statuses),
				Score:        meanScore(stageResults),
				StageResults: stageResults,
				Trace:        aggTrace,
				Reasons:      reasons,
			}, err
		}
		current = next
	}

	// All stages succeeded (or Fallback handled failures).
	final, ok := current.(Final)
	if !ok {
		return Result[Final]{}, fmt.Errorf("decompose: last stage output type mismatch: %T", current)
	}

	statuses := collectStatuses(stageResults)
	return Result[Final]{
		Answer:       final,
		Status:       aggregateStatus(statuses),
		Score:        meanScore(stageResults),
		StageResults: stageResults,
		Trace:        aggTrace,
		Reasons:      reasons,
	}, nil
}

// runStage runs a single stage with the given config, applying failure policy.
// Returns the StageResult, the next input (stage output), and any terminal error.
func (p *Pipeline[Final]) runStage(ctx context.Context, s anyStage, cfg StageConfig, in any) (StageResult, any, error) {
	stageCtx := ctx
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		stageCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	start := time.Now()
	out, trace, err := s.runAny(stageCtx, in)
	trace.LatencyMs = time.Since(start).Milliseconds()
	trace.Attempts = 1

	if err == nil {
		return StageResult{
			Name:     s.stageName(),
			Status:   StatusOK,
			SubScore: 1.0,
			Out:      out,
			Trace:    trace,
		}, out, nil
	}

	// First attempt failed.
	switch cfg.OnFailure {
	case RetryOnce:
		start2 := time.Now()
		out2, trace2, err2 := s.runAny(stageCtx, in)
		trace2.LatencyMs = time.Since(start2).Milliseconds()
		trace2.Attempts = 1
		// Sum both attempt traces.
		combined := StageTrace{
			LatencyMs: trace.LatencyMs + trace2.LatencyMs,
			TokensIn:  trace.TokensIn + trace2.TokensIn,
			TokensOut: trace.TokensOut + trace2.TokensOut,
			CostUSD:   trace.CostUSD + trace2.CostUSD,
			Attempts:  2,
		}
		if err2 == nil {
			return StageResult{
				Name:     s.stageName(),
				Status:   StatusOK,
				SubScore: 1.0,
				Out:      out2,
				Trace:    combined,
			}, out2, nil
		}
		// Second failure → Abort.
		sr := StageResult{
			Name:     s.stageName(),
			Status:   StatusFail,
			SubScore: 0.0,
			Trace:    combined,
			Err:      err2,
		}
		return sr, nil, err2

	case Fallback:
		sr := StageResult{
			Name:     s.stageName(),
			Status:   StatusFail,
			SubScore: 0.0,
			Out:      cfg.FallbackOut,
			Trace:    trace,
			Err:      err,
		}
		return sr, cfg.FallbackOut, nil

	default: // Abort
		sr := StageResult{
			Name:     s.stageName(),
			Status:   StatusFail,
			SubScore: 0.0,
			Trace:    trace,
			Err:      err,
		}
		return sr, nil, err
	}
}

func collectStatuses(results []StageResult) []Status {
	out := make([]Status, len(results))
	for i, r := range results {
		out[i] = r.Status
	}
	return out
}
