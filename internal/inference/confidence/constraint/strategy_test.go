package constraint_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/constraint"
)

func TestNewRejectsNilInvariantCheck(t *testing.T) {
	_, err := constraint.New[string](constraint.Options[string]{
		Invariants: []constraint.Invariant[string]{{Name: "x", Check: nil}},
	})
	if err == nil {
		t.Fatal("expected error for nil Invariant.Check")
	}
}

func TestNewDefaultName(t *testing.T) {
	s, err := constraint.New[string](constraint.Options[string]{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Name() != "constraint" {
		t.Errorf("default Name = %q, want 'constraint'", s.Name())
	}
}

func TestNewCustomName(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{Name: "custom"})
	if s.Name() != "custom" {
		t.Errorf("Name = %q, want 'custom'", s.Name())
	}
}

func TestSchemaOnlySuccess(t *testing.T) {
	calls := 0
	s, _ := constraint.New[string](constraint.Options[string]{
		SchemaValidator: func(raw string) error {
			calls++
			if raw == "{\"ok\":true}" {
				return nil
			}
			return errors.New("bad schema")
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Raw: "{\"ok\":true}", Answer: "ok"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if calls != 1 {
		t.Errorf("schema validator called %d times, want 1", calls)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false")
	}
	if out.Tokens != (confidence.TokenUsage{}) {
		t.Errorf("Tokens = %+v, want zero (no LLM calls)", out.Tokens)
	}
	if !strings.Contains(out.Reason, "schema") {
		t.Errorf("Reason = %q, want mention of schema", out.Reason)
	}
}

func TestSchemaFailHardFails(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{
		SchemaValidator: func(string) error { return errors.New("missing required field 'kind'") },
		Invariants: []constraint.Invariant[string]{
			{Name: "should-not-run", Check: func(string) (bool, string) {
				t.Errorf("invariant should not run on schema fail")
				return true, ""
			}},
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Raw: "{}"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.0 {
		t.Errorf("SubScore = %v, want 0.0", out.SubScore)
	}
	if !out.HardFail {
		t.Errorf("HardFail = false, want true")
	}
	if !strings.Contains(out.Reason, "schema invalid") {
		t.Errorf("Reason = %q, want 'schema invalid'", out.Reason)
	}
	log, ok := out.Log.(constraint.Log)
	if !ok {
		t.Fatalf("Log type = %T, want constraint.Log", out.Log)
	}
	if !strings.Contains(log.SchemaError, "kind") {
		t.Errorf("Log.SchemaError = %q, want to mention 'kind'", log.SchemaError)
	}
}

func TestAllInvariantsPass(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{
		Invariants: []constraint.Invariant[string]{
			{Name: "non-empty", Check: func(a string) (bool, string) {
				if a == "" {
					return false, "empty"
				}
				return true, ""
			}},
			{Name: "ascii", Check: func(string) (bool, string) { return true, "" }},
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "hello"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false")
	}
}

func TestPartialInvariantFailures(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{
		Invariants: []constraint.Invariant[string]{
			{Name: "a", Check: func(string) (bool, string) { return true, "" }},
			{Name: "b", Check: func(string) (bool, string) { return false, "out of range" }},
			{Name: "c", Check: func(string) (bool, string) { return true, "" }},
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "x"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := 1.0 - 1.0/3.0
	if diff := out.SubScore - want; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("SubScore = %v, want ~%v", out.SubScore, want)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false (invariants don't hard-fail)")
	}
	if !strings.Contains(out.Reason, "1/3 invariants failed") {
		t.Errorf("Reason = %q, want '1/3 invariants failed'", out.Reason)
	}
	if !strings.Contains(out.Reason, "b") || !strings.Contains(out.Reason, "out of range") {
		t.Errorf("Reason = %q, want mention of failing invariant 'b' and detail", out.Reason)
	}
	log := out.Log.(constraint.Log)
	if len(log.Failures) != 1 {
		t.Errorf("Log.Failures len = %d, want 1", len(log.Failures))
	}
	if log.Failures[0].Name != "b" || log.Failures[0].Detail != "out of range" {
		t.Errorf("Log.Failures[0] = %+v", log.Failures[0])
	}
}

func TestAllInvariantsFail(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{
		Invariants: []constraint.Invariant[string]{
			{Name: "a", Check: func(string) (bool, string) { return false, "x" }},
			{Name: "b", Check: func(string) (bool, string) { return false, "y" }},
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Request: confidence.Request[string]{Answer: "z"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.0 {
		t.Errorf("SubScore = %v, want 0.0", out.SubScore)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false (invariants don't hard-fail)")
	}
}

func TestContextCancel(t *testing.T) {
	s, _ := constraint.New[string](constraint.Options[string]{
		Invariants: []constraint.Invariant[string]{
			{Name: "a", Check: func(string) (bool, string) { return true, "" }},
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Run(ctx, confidence.StrategyInput[string]{Request: confidence.Request[string]{Answer: "x"}})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestNoChecksAtAllReturnsNeutralPass(t *testing.T) {
	// No schema, no invariants: technically nothing to validate. Return
	// SubScore=1.0 (nothing failed) with a clear reason — caller's
	// responsibility to wire up at least one check.
	s, _ := constraint.New[string](constraint.Options[string]{})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Request: confidence.Request[string]{Answer: "x"}})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
}

// Verify Strategy[T] satisfaction for a struct type T.
type structAns struct {
	Kind string
	N    int
}

func TestGenericStructAnswer(t *testing.T) {
	s, _ := constraint.New[structAns](constraint.Options[structAns]{
		Invariants: []constraint.Invariant[structAns]{
			{Name: "kind-set", Check: func(a structAns) (bool, string) {
				if a.Kind == "" {
					return false, "empty Kind"
				}
				return true, ""
			}},
			{Name: "n-positive", Check: func(a structAns) (bool, string) {
				if a.N <= 0 {
					return false, "N must be > 0"
				}
				return true, ""
			}},
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[structAns]{
		Request: confidence.Request[structAns]{Answer: structAns{Kind: "feat", N: 3}},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
}

// Compile-time assertion: constraint.Strategy[T] satisfies confidence.Strategy[T].
var _ confidence.Strategy[string] = (*constraint.Strategy[string])(nil)
