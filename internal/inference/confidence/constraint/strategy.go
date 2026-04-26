// Package constraint provides a confidence.Strategy[T] that validates an
// inference result against a schema and a list of invariants. It performs no
// LLM calls — it is the cheapest of the F144 strategies and runs first in
// most call-site stacks. A schema-validation failure is hard-fail
// (Status=FAIL is forced regardless of the composed score) because semantic
// confidence has no meaning on broken format. Invariant failures contribute
// linearly to SubScore but never hard-fail.
package constraint

import (
	"context"
	"fmt"
	"strings"

	"sdp_dev/internal/inference/confidence"
)

// Invariant is one named predicate over the parsed answer.
//
// Check returns (ok, detail). On ok=false, detail is a short human-readable
// explanation that surfaces in the StrategyOutput.Reason and Log.
type Invariant[T any] struct {
	Name  string
	Check func(T) (ok bool, detail string)
}

// Options configures a constraint Strategy.
type Options[T any] struct {
	// SchemaValidator validates the raw text returned by the primary call
	// (typically JSON-Schema). A non-nil error from this callback is a
	// hard fail — invariants are not consulted.
	SchemaValidator func(raw string) error
	// Invariants are checked after a successful schema validation. Each
	// failure subtracts proportionally from SubScore.
	Invariants []Invariant[T]
	// Name overrides the strategy name (default "constraint"). Use this
	// when wiring two constraint strategies into one Checker — Policy
	// requires unique strategy names.
	Name string
}

// Strategy implements confidence.Strategy[T] for format + invariant checks.
type Strategy[T any] struct {
	schema     func(string) error
	invariants []Invariant[T]
	name       string
}

// New constructs a constraint Strategy. Returns an error if any invariant
// has a nil Check (caller bug — better to surface early than silently pass).
func New[T any](opts Options[T]) (*Strategy[T], error) {
	for i, inv := range opts.Invariants {
		if inv.Check == nil {
			return nil, fmt.Errorf("constraint: invariants[%d] (%q) has nil Check", i, inv.Name)
		}
	}
	name := opts.Name
	if name == "" {
		name = "constraint"
	}
	return &Strategy[T]{
		schema:     opts.SchemaValidator,
		invariants: append([]Invariant[T]{}, opts.Invariants...),
		name:       name,
	}, nil
}

// Failure records one invariant violation for trace output.
type Failure struct {
	Name   string
	Detail string
}

// Log is the per-run diagnostic blob exposed via StrategyOutput.Log.
type Log struct {
	SchemaError string
	Failures    []Failure
}

// Name reports the strategy's registered name.
func (s *Strategy[T]) Name() string { return s.name }

// Run validates the inference result.
func (s *Strategy[T]) Run(ctx context.Context, in confidence.StrategyInput[T]) (confidence.StrategyOutput, error) {
	if err := ctx.Err(); err != nil {
		return confidence.StrategyOutput{}, err
	}

	// Step 1: schema. Hard-fail short-circuits invariant checks — we don't
	// want to spend time validating fields on malformed input.
	if s.schema != nil {
		if err := s.schema(in.Request.Raw); err != nil {
			return confidence.StrategyOutput{
				SubScore: 0,
				HardFail: true,
				Reason:   fmt.Sprintf("schema invalid: %s", err.Error()),
				Log:      Log{SchemaError: err.Error()},
			}, nil
		}
	}

	// Step 2: invariants. Linear scoring over (passed / total).
	total := len(s.invariants)
	if total == 0 {
		reason := "no checks configured"
		if s.schema != nil {
			reason = "schema valid"
		}
		return confidence.StrategyOutput{SubScore: 1, Reason: reason, Log: Log{}}, nil
	}

	failures := make([]Failure, 0, total)
	for _, inv := range s.invariants {
		if err := ctx.Err(); err != nil {
			return confidence.StrategyOutput{}, err
		}
		ok, detail := inv.Check(in.Request.Answer)
		if !ok {
			failures = append(failures, Failure{Name: inv.Name, Detail: detail})
		}
	}
	failed := len(failures)
	score := 1.0 - float64(failed)/float64(total)
	reason := fmt.Sprintf("%d/%d invariants failed: %s", failed, total, formatFailures(failures))
	if failed == 0 {
		if s.schema != nil {
			reason = fmt.Sprintf("all checks passed (schema + %d invariants)", total)
		} else {
			reason = fmt.Sprintf("all %d invariants passed", total)
		}
	}
	return confidence.StrategyOutput{
		SubScore: score,
		Reason:   reason,
		Log:      Log{Failures: failures},
	}, nil
}

func formatFailures(fs []Failure) string {
	if len(fs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		parts = append(parts, fmt.Sprintf("%s(%s)", f.Name, f.Detail))
	}
	return strings.Join(parts, "; ")
}
