package decompose_test

import (
	"context"
	"errors"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/decompose"
)

// confidentOut implements Confider for use in tests.
type confidentOut struct {
	val        string
	confidence float64
	status     decompose.Status
}

func (c confidentOut) Confidence() float64          { return c.confidence }
func (c confidentOut) ConfStatus() decompose.Status { return c.status }

// dumbOut does NOT implement Confider.
type dumbOut struct{ val string }

// makeCountingMicro returns a micro stage that always returns out with zero error,
// incrementing *calls on each Run.
func makeCountingMicro(out confidentOut, calls *int) decompose.Stage[string, confidentOut] {
	return decompose.NewStage[string, confidentOut]("micro", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		*calls++
		return out, decompose.StageTrace{TokensIn: 10, TokensOut: 5}, nil
	})
}

// makeCountingLLM returns a stage that increments *calls and returns llmOut.
func makeCountingLLM(out confidentOut, tokenIn int, calls *int) decompose.Stage[string, confidentOut] {
	return decompose.NewStage[string, confidentOut]("llm", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		*calls++
		return out, decompose.StageTrace{TokensIn: tokenIn, TokensOut: 20}, nil
	})
}

// Test 1: micro confident — llm NOT called.
func TestWithEscalation_MicroConfident(t *testing.T) {
	microCalls, llmCalls := 0, 0
	microOut := confidentOut{val: "micro-answer", confidence: 0.95, status: decompose.StatusOK}
	llmOut := confidentOut{val: "llm-answer", confidence: 1.0, status: decompose.StatusOK}

	micro := makeCountingMicro(microOut, &microCalls)
	llm := makeCountingLLM(llmOut, 100, &llmCalls)

	cfg := decompose.EscalationConfig{
		ConfidenceThreshold: 0.85,
		RecordSkippedTrace:  true,
	}
	stage := decompose.WithEscalation(micro, llm, cfg)

	out, trace, err := stage.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalls != 0 {
		t.Errorf("llm should not be called, got %d calls", llmCalls)
	}
	if microCalls != 1 {
		t.Errorf("expected 1 micro call, got %d", microCalls)
	}
	if out.val != "micro-answer" {
		t.Errorf("expected micro-answer, got %q", out.val)
	}
	if trace.Attempts != 1 {
		t.Errorf("expected Attempts=1, got %d", trace.Attempts)
	}
	if trace.TokensIn != 0 {
		t.Errorf("expected TokensIn=0 (skipped llm), got %d", trace.TokensIn)
	}
}

// Test 2: micro unsure (low confidence) — llm IS called.
func TestWithEscalation_MicroLowConfidence(t *testing.T) {
	microCalls, llmCalls := 0, 0
	microOut := confidentOut{val: "micro-uncertain", confidence: 0.60, status: decompose.StatusOK}
	llmOut := confidentOut{val: "llm-answer", confidence: 1.0, status: decompose.StatusOK}
	const llmTokensIn = 200

	micro := makeCountingMicro(microOut, &microCalls)
	llm := makeCountingLLM(llmOut, llmTokensIn, &llmCalls)

	cfg := decompose.EscalationConfig{ConfidenceThreshold: 0.85}
	stage := decompose.WithEscalation(micro, llm, cfg)

	out, trace, err := stage.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalls != 1 {
		t.Errorf("expected 1 llm call, got %d", llmCalls)
	}
	if out.val != "llm-answer" {
		t.Errorf("expected llm-answer, got %q", out.val)
	}
	if trace.Attempts != 2 {
		t.Errorf("expected Attempts=2, got %d", trace.Attempts)
	}
	if trace.TokensIn != llmTokensIn {
		t.Errorf("expected TokensIn=%d, got %d", llmTokensIn, trace.TokensIn)
	}
}

// Test 3: micro status unsure — escalation to llm.
func TestWithEscalation_MicroStatusUnsure(t *testing.T) {
	microCalls, llmCalls := 0, 0
	// Status=Unsure even though confidence is high — should still escalate.
	microOut := confidentOut{val: "micro-unsure", confidence: 0.99, status: decompose.StatusUnsure}
	llmOut := confidentOut{val: "llm-answer", confidence: 1.0, status: decompose.StatusOK}

	micro := makeCountingMicro(microOut, &microCalls)
	llm := makeCountingLLM(llmOut, 150, &llmCalls)

	cfg := decompose.EscalationConfig{ConfidenceThreshold: 0.85}
	stage := decompose.WithEscalation(micro, llm, cfg)

	out, trace, err := stage.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalls != 1 {
		t.Errorf("expected 1 llm call, got %d", llmCalls)
	}
	if out.val != "llm-answer" {
		t.Errorf("expected llm-answer, got %q", out.val)
	}
	if trace.Attempts != 2 {
		t.Errorf("expected Attempts=2, got %d", trace.Attempts)
	}
	_ = microCalls
}

// Test 4: micro out not Confider — always escalates.
func TestWithEscalation_MicroNotConfider(t *testing.T) {
	llmCalls := 0

	micro := decompose.NewStage[string, dumbOut]("micro", func(_ context.Context, _ string) (dumbOut, decompose.StageTrace, error) {
		return dumbOut{val: "micro-dumb"}, decompose.StageTrace{}, nil
	})
	llm := decompose.NewStage[string, dumbOut]("llm", func(_ context.Context, _ string) (dumbOut, decompose.StageTrace, error) {
		llmCalls++
		return dumbOut{val: "llm-smart"}, decompose.StageTrace{TokensIn: 50, TokensOut: 30}, nil
	})

	cfg := decompose.EscalationConfig{ConfidenceThreshold: 0.85}
	stage := decompose.WithEscalation(micro, llm, cfg)

	out, trace, err := stage.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalls != 1 {
		t.Errorf("expected 1 llm call, got %d", llmCalls)
	}
	if out.val != "llm-smart" {
		t.Errorf("expected llm-smart, got %q", out.val)
	}
	if trace.Attempts != 2 {
		t.Errorf("expected Attempts=2, got %d", trace.Attempts)
	}
}

// Test 5: micro error + EscalateOnError=true — llm called.
func TestWithEscalation_MicroError_EscalateOnError(t *testing.T) {
	llmCalls := 0
	microErr := errors.New("micro failed")

	micro := decompose.NewStage[string, confidentOut]("micro", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		return confidentOut{}, decompose.StageTrace{}, microErr
	})
	llmOut := confidentOut{val: "llm-fallback", confidence: 1.0, status: decompose.StatusOK}
	llm := decompose.NewStage[string, confidentOut]("llm", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		llmCalls++
		return llmOut, decompose.StageTrace{TokensIn: 75, TokensOut: 40}, nil
	})

	cfg := decompose.EscalationConfig{
		ConfidenceThreshold: 0.85,
		EscalateOnError:     true,
	}
	stage := decompose.WithEscalation(micro, llm, cfg)

	out, trace, err := stage.Run(context.Background(), "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if llmCalls != 1 {
		t.Errorf("expected 1 llm call, got %d", llmCalls)
	}
	if out.val != "llm-fallback" {
		t.Errorf("expected llm-fallback, got %q", out.val)
	}
	if trace.Attempts != 2 {
		t.Errorf("expected Attempts=2, got %d", trace.Attempts)
	}
}

// Test 6: micro error + EscalateOnError=false — error propagated, llm NOT called.
func TestWithEscalation_MicroError_PropagateError(t *testing.T) {
	llmCalls := 0
	microErr := errors.New("micro hard failure")

	micro := decompose.NewStage[string, confidentOut]("micro", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		return confidentOut{}, decompose.StageTrace{}, microErr
	})
	llm := decompose.NewStage[string, confidentOut]("llm", func(_ context.Context, _ string) (confidentOut, decompose.StageTrace, error) {
		llmCalls++
		return confidentOut{}, decompose.StageTrace{}, nil
	})

	cfg := decompose.EscalationConfig{
		ConfidenceThreshold: 0.85,
		EscalateOnError:     false,
	}
	stage := decompose.WithEscalation(micro, llm, cfg)

	_, _, err := stage.Run(context.Background(), "input")
	if err == nil {
		t.Fatal("expected error to be propagated, got nil")
	}
	if !errors.Is(err, microErr) {
		t.Errorf("expected microErr, got %v", err)
	}
	if llmCalls != 0 {
		t.Errorf("llm should not be called, got %d calls", llmCalls)
	}
}

// TestWithEscalation_StageName verifies the composed stage name includes "+escalation".
func TestWithEscalation_StageName(t *testing.T) {
	micro := decompose.NewStage[string, dumbOut]("my-micro", func(_ context.Context, _ string) (dumbOut, decompose.StageTrace, error) {
		return dumbOut{}, decompose.StageTrace{}, nil
	})
	llm := decompose.NewStage[string, dumbOut]("my-llm", func(_ context.Context, _ string) (dumbOut, decompose.StageTrace, error) {
		return dumbOut{}, decompose.StageTrace{}, nil
	})
	cfg := decompose.EscalationConfig{}
	stage := decompose.WithEscalation(micro, llm, cfg)
	if stage.Name() != "my-micro+escalation" {
		t.Errorf("unexpected stage name: %q", stage.Name())
	}
}
