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

// runStage applies a per-stage timeout and failure policy around runCore.
// runCore does the full integration stack: cascade → confidence → stitcher.
func (p *Pipeline[Final]) runStage(ctx context.Context, s anyStage, cfg StageConfig, in any) (StageResult, any, error) {
	stageCtx := ctx
	var cancel context.CancelFunc
	if cfg.Timeout > 0 {
		stageCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	out, trace, err := p.runAttempt(stageCtx, s, cfg, in)
	if err == nil {
		sr := stageResultFromCore(s.stageName(), out, trace, nil)
		return sr, out, nil
	}

	// First attempt failed — apply failure policy.
	switch cfg.OnFailure {
	case RetryOnce:
		out2, trace2, err2 := p.runAttempt(stageCtx, s, cfg, in)
		combined := combineTraces(trace, trace2)
		if err2 == nil {
			sr := stageResultFromCore(s.stageName(), out2, combined, nil)
			sr.Trace.Attempts = 2
			return sr, out2, nil
		}
		sr := stageResultFromCore(s.stageName(), nil, combined, err2)
		sr.Trace.Attempts = 2
		return sr, nil, err2

	case Fallback:
		sr := stageResultFromCore(s.stageName(), cfg.FallbackOut, trace, err)
		sr.Status = StatusFail
		sr.SubScore = 0.0
		sr.Out = cfg.FallbackOut
		return sr, cfg.FallbackOut, nil

	default: // Abort
		sr := stageResultFromCore(s.stageName(), nil, trace, err)
		return sr, nil, err
	}
}

// runAttempt executes one attempt through the integration stack and measures wall-clock latency.
func (p *Pipeline[Final]) runAttempt(ctx context.Context, s anyStage, cfg StageConfig, in any) (any, StageTrace, error) {
	start := time.Now()
	out, trace, err := runCore(ctx, s, cfg, in)
	trace.LatencyMs = time.Since(start).Milliseconds()
	if trace.Attempts == 0 {
		trace.Attempts = 1
	}
	return out, trace, err
}

func combineTraces(a, b StageTrace) StageTrace {
	combined := StageTrace{
		LatencyMs:     a.LatencyMs + b.LatencyMs,
		TokensIn:      a.TokensIn + b.TokensIn,
		TokensOut:     a.TokensOut + b.TokensOut,
		CostUSD:       a.CostUSD + b.CostUSD,
		Attempts:      a.Attempts + b.Attempts,
		ConfidenceLog: b.ConfidenceLog,
		CascadeLog:    b.CascadeLog,
	}
	return combined
}

func collectStatuses(results []StageResult) []Status {
	out := make([]Status, len(results))
	for i, r := range results {
		out[i] = r.Status
	}
	return out
}
