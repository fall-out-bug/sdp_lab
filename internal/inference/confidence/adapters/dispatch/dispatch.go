// Package dispatch wraps task-routing classifications (internal/dispatch
// classify) in a confidence.Checker — but in lite profile because dispatch
// is on the hot path. Strategies: constraint + selfcheck lite (single-pass
// self_score annotation extraction). N-sample is intentionally disabled.
// UNSURE behavior: conservative fallback (caller-defined safe default).
package dispatch

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/constraint"
	"sdp_dev/internal/inference/confidence/selfcheck"
)

// Decision is one routing decision: which harness/agent should handle a task.
type Decision struct {
	Harness    string `json:"harness"`     // "claude-code" | "opencode" | "codex" | "cursor"
	Agent      string `json:"agent"`       // role name; arbitrary
	Confidence float64 `json:"confidence"` // self-reported [0, 1]
	Rationale  string `json:"rationale,omitempty"`
}

// Options configures a dispatch Checker.
type Options struct {
	AllowedHarnesses []string
	AllowedAgents    []string // optional; empty = no constraint on agent
	Policy           *confidence.Policy
}

// New builds the lite-profile Checker. Note: no Caller dependency — selfcheck
// lite mode reads self_score from the primary raw text via the extractor;
// it does not make additional LLM calls.
func New(opts Options) (*confidence.Checker[Decision], error) {
	if len(opts.AllowedHarnesses) == 0 {
		return nil, errors.New("dispatch: AllowedHarnesses must be non-empty")
	}

	cs, err := constraint.New[Decision](constraint.Options[Decision]{
		Invariants: []constraint.Invariant[Decision]{
			{
				Name: "harness-allowed",
				Check: func(d Decision) (bool, string) {
					for _, h := range opts.AllowedHarnesses {
						if d.Harness == h {
							return true, ""
						}
					}
					return false, fmt.Sprintf("harness %q not in allowed list", d.Harness)
				},
			},
			{
				Name: "agent-allowed",
				Check: func(d Decision) (bool, string) {
					if len(opts.AllowedAgents) == 0 {
						return true, ""
					}
					for _, a := range opts.AllowedAgents {
						if d.Agent == a {
							return true, ""
						}
					}
					return false, fmt.Sprintf("agent %q not in allowed list", d.Agent)
				},
			},
			{
				Name: "self-confidence-range",
				Check: func(d Decision) (bool, string) {
					if d.Confidence < 0 || d.Confidence > 1 {
						return false, fmt.Sprintf("self confidence %v out of [0,1]", d.Confidence)
					}
					return true, ""
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatch: build constraint: %w", err)
	}

	sc, err := selfcheck.New[Decision](selfcheck.Options[Decision]{
		Mode: selfcheck.ModeLite,
		LiteScoreExtractor: func(raw string) (float64, bool) {
			// Look for a `self_score: X` annotation in raw text.
			// We accept either a JSON-style "self_score":X or the loose
			// `self_score: X` annotation we instruct the model to add.
			m := selfScoreAnnotation.FindStringSubmatch(raw)
			if len(m) < 2 {
				return 0, false
			}
			var f float64
			if _, err := fmt.Sscanf(m[1], "%f", &f); err != nil {
				return 0, false
			}
			return f, true
		},
	})
	if err != nil {
		return nil, fmt.Errorf("dispatch: build self-check: %w", err)
	}

	policy := canonicalPolicy()
	if opts.Policy != nil {
		policy = *opts.Policy
	}

	return confidence.NewChecker[Decision](nil,
		[]confidence.Strategy[Decision]{cs, sc}, policy)
}

// Verify runs the Checker on a parsed Decision.
func Verify(ctx context.Context, checker *confidence.Checker[Decision], parsed Decision, raw string) (confidence.Result[Decision], error) {
	return checker.Check(ctx, confidence.Request[Decision]{
		Input:  raw,
		Answer: parsed,
		Raw:    raw,
	})
}

// canonicalPolicy: lite weights without consensus (n-sample disabled).
// UnsureBehavior is conservative fallback — dispatch is hot, blocking on
// human review would stall every routing decision.
func canonicalPolicy() confidence.Policy {
	p := confidence.DefaultPolicy()
	p.Weights = map[string]float64{
		"constraint": 0.6, // structural correctness dominates lite signal
		"self_check": 0.4,
	}
	p.UnsureBehavior = confidence.UnsureConservativeFallback
	return p
}

// selfScoreAnnotation matches `self_score: 0.85` or `"self_score": 0.85`.
var selfScoreAnnotation = regexp.MustCompile(`(?i)"?self_score"?\s*[:=]\s*([0-9]*\.?[0-9]+)`)
