package decompose_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"sdp_dev/internal/inference/decompose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

func okStage[In, Out any](name string, fn func(In) Out) decompose.Stage[In, Out] {
	return decompose.NewStage[In, Out](name, func(_ context.Context, in In) (Out, decompose.StageTrace, error) {
		return fn(in), decompose.StageTrace{}, nil
	})
}

func errStage[In, Out any](name string, err error) decompose.Stage[In, Out] {
	return decompose.NewStage[In, Out](name, func(_ context.Context, _ In) (Out, decompose.StageTrace, error) {
		var zero Out
		return zero, decompose.StageTrace{}, err
	})
}

type stepA struct{ Val int }
type stepB struct{ Val string }
type stepC struct{ Score float64 }

// --- tests ---

// (a) Happy path: 3 stages, all succeed.
func TestPipeline_HappyPath(t *testing.T) {
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{Val: "ok"} })
	s2 := okStage[stepB, stepC]("s2", func(b stepB) stepC { return stepC{Score: 0.9} })

	p := decompose.New[stepC]("happy")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{})

	res, err := p.Run(context.Background(), stepA{Val: 42})
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusOK, res.Status)
	assert.Equal(t, 0.9, res.Answer.Score)
	assert.Len(t, res.StageResults, 2)
	assert.Equal(t, decompose.StatusOK, res.StageResults[0].Status)
	assert.Equal(t, decompose.StatusOK, res.StageResults[1].Status)
	// Score = mean(1.0, 1.0)
	assert.InDelta(t, 1.0, res.Score, 0.001)
}

// (b) Stage 2 fails → Abort.
func TestPipeline_Abort(t *testing.T) {
	boom := errors.New("stage2 error")
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	s2 := errStage[stepB, stepC]("s2", boom)

	p := decompose.New[stepC]("abort")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{OnFailure: decompose.Abort})

	res, err := p.Run(context.Background(), stepA{})
	require.ErrorIs(t, err, boom)
	assert.Equal(t, decompose.StatusFail, res.Status)
	assert.Len(t, res.StageResults, 2)
	assert.Equal(t, decompose.StatusOK, res.StageResults[0].Status)
	assert.Equal(t, decompose.StatusFail, res.StageResults[1].Status)
}

// (c) Stage 2 fails → RetryOnce → recover on retry.
func TestPipeline_RetryOnce_Recover(t *testing.T) {
	calls := 0
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	s2 := decompose.NewStage[stepB, stepC]("s2", func(_ context.Context, b stepB) (stepC, decompose.StageTrace, error) {
		calls++
		if calls == 1 {
			return stepC{}, decompose.StageTrace{}, errors.New("transient")
		}
		return stepC{Score: 0.8}, decompose.StageTrace{}, nil
	})

	p := decompose.New[stepC]("retry-recover")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{OnFailure: decompose.RetryOnce})

	res, err := p.Run(context.Background(), stepA{})
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusOK, res.Status)
	assert.Equal(t, 2, calls)
	assert.Equal(t, 2, res.StageResults[1].Trace.Attempts)
	assert.InDelta(t, 0.8, res.Answer.Score, 0.001)
}

// RetryOnce: both attempts fail → behaves like Abort.
func TestPipeline_RetryOnce_BothFail(t *testing.T) {
	boom := errors.New("persistent")
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	s2 := errStage[stepB, stepC]("s2", boom)

	p := decompose.New[stepC]("retry-fail")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{OnFailure: decompose.RetryOnce})

	_, err := p.Run(context.Background(), stepA{})
	require.ErrorIs(t, err, boom)
}

// (d) Stage 2 fails → Fallback: pipeline continues with FallbackOut.
func TestPipeline_Fallback(t *testing.T) {
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	s2 := errStage[stepB, stepC]("s2", errors.New("flaky"))

	fallback := stepC{Score: -1}
	p := decompose.New[stepC]("fallback")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{OnFailure: decompose.Fallback, FallbackOut: fallback})

	res, err := p.Run(context.Background(), stepA{})
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusFail, res.Status) // stage failed → aggregate FAIL
	assert.InDelta(t, -1.0, res.Answer.Score, 0.001)
	assert.NotNil(t, res.StageResults[1].Err)
}

// (e) Per-stage timeout: stage exceeds deadline → DeadlineExceeded.
func TestPipeline_Timeout(t *testing.T) {
	slow := decompose.NewStage[stepA, stepB]("slow", func(ctx context.Context, _ stepA) (stepB, decompose.StageTrace, error) {
		select {
		case <-ctx.Done():
			return stepB{}, decompose.StageTrace{}, ctx.Err()
		case <-time.After(500 * time.Millisecond):
			return stepB{}, decompose.StageTrace{}, nil
		}
	})

	p := decompose.New[stepB]("timeout")
	decompose.Then(p, slow, decompose.StageConfig{Timeout: 10 * time.Millisecond})

	_, err := p.Run(context.Background(), stepA{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

// (f) Generic mismatch: stage output B but next stage expects C — caught at runtime.
func TestPipeline_TypeMismatch_Panic(t *testing.T) {
	// s1: stepA → stepB, s2 declared as stepA → stepC but receives stepB input.
	// The mismatch is caught in runAny when it tries to cast stepB to stepA.
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	// Intentionally wrong: s2 expects stepA input but s1 outputs stepB.
	s2 := okStage[stepA, stepC]("s2", func(a stepA) stepC { return stepC{} })

	p := decompose.New[stepC]("mismatch")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{})

	assert.Panics(t, func() {
		_, _ = p.Run(context.Background(), stepA{})
	})
}

// Score aggregation: mean of SubScores (default 1.0 for OK stages).
func TestPipeline_ScoreAggregation(t *testing.T) {
	s1 := okStage[stepA, stepB]("s1", func(a stepA) stepB { return stepB{} })
	s2 := okStage[stepB, stepC]("s2", func(b stepB) stepC { return stepC{} })

	p := decompose.New[stepC]("score")
	decompose.Then(p, s1, decompose.StageConfig{})
	decompose.Then(p, s2, decompose.StageConfig{})

	res, err := p.Run(context.Background(), stepA{})
	require.NoError(t, err)
	assert.InDelta(t, 1.0, res.Score, 0.001)
}

// AggregateTrace sums stage traces.
func TestPipeline_TraceAggregation(t *testing.T) {
	makeStage := func(name string, ms int64) decompose.Stage[stepA, stepA] {
		return decompose.NewStage[stepA, stepA](name, func(_ context.Context, a stepA) (stepA, decompose.StageTrace, error) {
			return a, decompose.StageTrace{LatencyMs: ms, TokensIn: 10, TokensOut: 5, CostUSD: 0.001}, nil
		})
	}
	p := decompose.New[stepA]("trace")
	decompose.Then(p, makeStage("s1", 10), decompose.StageConfig{})
	decompose.Then(p, makeStage("s2", 20), decompose.StageConfig{})

	res, err := p.Run(context.Background(), stepA{})
	require.NoError(t, err)
	// LatencyMs from runStage overrides trace.LatencyMs (real wall clock), so only
	// check token/cost sums which are additive from stage trace.
	assert.Equal(t, 20, res.Trace.TokensIn)
	assert.InDelta(t, 0.002, res.Trace.CostUSD, 0.0001)
}
