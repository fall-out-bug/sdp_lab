package decompose_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mock CascadeInvoker ---

type mockCascade struct {
	provider string
	err      error
	calls    int
}

func (m *mockCascade) Invoke(ctx context.Context, fn func() (any, decompose.StageTrace, error)) (any, decompose.StageTrace, decompose.CascadeTrace, error) {
	m.calls++
	if m.err != nil {
		return nil, decompose.StageTrace{}, decompose.CascadeTrace{}, m.err
	}
	out, trace, err := fn()
	ct := decompose.CascadeTrace{Provider: m.provider, Attempts: 1}
	return out, trace, ct, err
}

// --- mock ConfidenceRunner ---

type mockConfidence struct {
	score   float64
	status  decompose.Status
	reasons []string
	err     error
	calls   int
}

func (m *mockConfidence) Run(_ context.Context, _, _ string, _ any) (decompose.ConfidenceCheckResult, error) {
	m.calls++
	if m.err != nil {
		return decompose.ConfidenceCheckResult{}, m.err
	}
	return decompose.ConfidenceCheckResult{
		Score:   m.score,
		Status:  m.status,
		Reasons: m.reasons,
	}, nil
}

// --- (a) Confidence-only stage ---

func TestIntegration_ConfidenceOnly(t *testing.T) {
	mc := &mockConfidence{score: 0.9, status: decompose.StatusOK}

	s := decompose.NewStage[string, string]("echo", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return in, decompose.StageTrace{TokensIn: 5}, nil
	})

	p := decompose.New[string]("conf-only")
	decompose.Then(p, s, decompose.StageConfig{Confidence: mc})

	res, err := p.Run(context.Background(), "hello")
	require.NoError(t, err)
	assert.Equal(t, "hello", res.Answer)
	assert.Equal(t, 1, mc.calls)
	// ConfidenceLog populated.
	require.NotNil(t, res.StageResults[0].Trace.ConfidenceLog)
	assert.InDelta(t, 0.9, res.StageResults[0].Trace.ConfidenceLog.Score, 0.001)
	assert.Equal(t, decompose.StatusOK, res.StageResults[0].Trace.ConfidenceLog.Status)
	// SubScore derived from confidence.
	assert.InDelta(t, 0.9, res.StageResults[0].SubScore, 0.001)
	// Cascade not used → CascadeLog nil.
	assert.Nil(t, res.StageResults[0].Trace.CascadeLog)
}

// Confidence UNSURE → stage StatusUnsure, pipeline StatusUnsure.
func TestIntegration_Confidence_Unsure(t *testing.T) {
	mc := &mockConfidence{score: 0.4, status: decompose.StatusUnsure}

	s := decompose.NewStage[string, string]("echo", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return in, decompose.StageTrace{}, nil
	})

	p := decompose.New[string]("conf-unsure")
	decompose.Then(p, s, decompose.StageConfig{Confidence: mc})

	res, err := p.Run(context.Background(), "x")
	require.NoError(t, err)
	assert.Equal(t, decompose.StatusUnsure, res.StageResults[0].Status)
	assert.Equal(t, decompose.StatusUnsure, res.Status)
}

// Confidence error → stage fails.
func TestIntegration_ConfidenceError(t *testing.T) {
	boom := errors.New("confidence failed")
	mc := &mockConfidence{err: boom}

	s := decompose.NewStage[string, string]("echo", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return in, decompose.StageTrace{}, nil
	})

	p := decompose.New[string]("conf-err")
	decompose.Then(p, s, decompose.StageConfig{Confidence: mc})

	_, err := p.Run(context.Background(), "x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confidence")
}

// --- (b) Cascade-only stage via mock ---

func TestIntegration_CascadeOnly(t *testing.T) {
	mc := &mockCascade{provider: "haiku"}

	s := decompose.NewStage[string, string]("echo", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return in + "-processed", decompose.StageTrace{}, nil
	})

	p := decompose.New[string]("casc-only")
	decompose.Then(p, s, decompose.StageConfig{Cascade: mc})

	res, err := p.Run(context.Background(), "input")
	require.NoError(t, err)
	assert.Equal(t, "input-processed", res.Answer)
	assert.Equal(t, 1, mc.calls)
	// CascadeLog populated.
	require.NotNil(t, res.StageResults[0].Trace.CascadeLog)
	assert.Equal(t, "haiku", res.StageResults[0].Trace.CascadeLog.Provider)
	// Confidence not used → ConfidenceLog nil.
	assert.Nil(t, res.StageResults[0].Trace.ConfidenceLog)
	// Default SubScore 1.0 (no confidence).
	assert.InDelta(t, 1.0, res.StageResults[0].SubScore, 0.001)
}

// Cascade error propagates.
func TestIntegration_CascadeError(t *testing.T) {
	boom := errors.New("cascade failed")
	mc := &mockCascade{err: boom}

	s := decompose.NewStage[string, string]("echo", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return in, decompose.StageTrace{}, nil
	})

	p := decompose.New[string]("casc-err")
	decompose.Then(p, s, decompose.StageConfig{Cascade: mc})

	_, err := p.Run(context.Background(), "x")
	require.ErrorIs(t, err, boom)
}

// --- (c) Both cascade + confidence: assert call order via side effects ---

func TestIntegration_CascadeAndConfidence(t *testing.T) {
	callLog := []string{}

	innerCascade := &mockCascade{provider: "sonnet"}
	cascadeLogging := &loggingCascade{inner: innerCascade, log: &callLog, tag: "cascade"}

	s := decompose.NewStage[string, string]("stage", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		callLog = append(callLog, "stage")
		return "result", decompose.StageTrace{}, nil
	})

	innerConf := &mockConfidence{score: 1.0, status: decompose.StatusOK}
	confLogging := &loggingConfidence{inner: innerConf, log: &callLog, tag: "confidence"}

	p := decompose.New[string]("both")
	decompose.Then(p, s, decompose.StageConfig{
		Cascade:    cascadeLogging,
		Confidence: confLogging,
	})

	res, err := p.Run(context.Background(), "x")
	require.NoError(t, err)
	assert.Equal(t, "result", res.Answer)

	// Call order: cascade wraps fn (stage runs inside cascade's fn call),
	// then confidence runs after stage completes.
	require.Len(t, callLog, 3, "expected 3 logged calls: cascade, stage, confidence")
	assert.Equal(t, "cascade", callLog[0])
	assert.Equal(t, "stage", callLog[1])
	assert.Equal(t, "confidence", callLog[2])

	// Both logs populated.
	assert.NotNil(t, res.StageResults[0].Trace.CascadeLog)
	assert.NotNil(t, res.StageResults[0].Trace.ConfidenceLog)
}

// --- (d) Stitcher validation blocks invalid output ---

func TestIntegration_StitcherRejectsInvalid(t *testing.T) {
	s := decompose.NewStage[string, string]("enum-stage", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return "invalid-value", decompose.StageTrace{}, nil
	})

	stitcher := decompose.NewEnumStitcher("verdict", []string{"pass", "fail"})

	p := decompose.New[string]("stitcher-reject")
	decompose.Then(p, s, decompose.StageConfig{Stitcher: stitcher})

	_, err := p.Run(context.Background(), "start")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stitcher")
}

// Stitcher passes valid output.
func TestIntegration_StitcherAcceptsValid(t *testing.T) {
	s := decompose.NewStage[string, string]("enum-stage", func(_ context.Context, in string) (string, decompose.StageTrace, error) {
		return "pass", decompose.StageTrace{}, nil
	})

	stitcher := decompose.NewEnumStitcher("verdict", []string{"pass", "fail"})

	p := decompose.New[string]("stitcher-accept")
	decompose.Then(p, s, decompose.StageConfig{Stitcher: stitcher})

	res, err := p.Run(context.Background(), "start")
	require.NoError(t, err)
	assert.Equal(t, "pass", res.Answer)
}

// --- logging helpers ---

type loggingCascade struct {
	inner *mockCascade
	log   *[]string
	tag   string
}

func (l *loggingCascade) Invoke(ctx context.Context, fn func() (any, decompose.StageTrace, error)) (any, decompose.StageTrace, decompose.CascadeTrace, error) {
	*l.log = append(*l.log, l.tag)
	return l.inner.Invoke(ctx, fn)
}

type loggingConfidence struct {
	inner *mockConfidence
	log   *[]string
	tag   string
}

func (l *loggingConfidence) Run(ctx context.Context, input, raw string, answer any) (decompose.ConfidenceCheckResult, error) {
	*l.log = append(*l.log, l.tag)
	return l.inner.Run(ctx, input, raw, answer)
}
