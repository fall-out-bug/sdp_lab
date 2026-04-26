package confidence

import (
	"errors"
	"fmt"
	"math"
)

// UnsureBehavior tells the Checker what to do when a Result lands in the
// UNSURE band. Exact routing is up to the call-site (the Checker just
// reports the decision); the values are stable strings so they survive
// JSON serialization into evidence.
type UnsureBehavior string

const (
	// UnsureRetryOnce — re-run Check once with extra samples / higher N
	// before reporting. Good for analytics paths where retry cost is small.
	UnsureRetryOnce UnsureBehavior = "retry_once"
	// UnsureHumanHandoff — escalate via bd human <id> for manual review.
	// Good for production-blocking decisions like ws-verdict.
	UnsureHumanHandoff UnsureBehavior = "human_handoff"
	// UnsureConservativeFallback — drop the Answer and emit a safe default
	// (decided by the call-site). Good for hot paths like dispatch where
	// blocking on UNSURE is worse than degraded routing.
	UnsureConservativeFallback UnsureBehavior = "conservative_fallback"
)

// Valid reports whether b is one of the defined UnsureBehavior constants.
func (b UnsureBehavior) Valid() bool {
	switch b {
	case UnsureRetryOnce, UnsureHumanHandoff, UnsureConservativeFallback:
		return true
	default:
		return false
	}
}

// Policy is the per-call-site configuration for a Checker. It is value-typed
// so call-sites can derive variants without aliasing (e.g. lite mode for
// dispatch vs full mode for ws-verdict).
type Policy struct {
	// OKThreshold — score >= OKThreshold maps to StatusOK. Default 0.8.
	OKThreshold float64
	// FailThreshold — score < FailThreshold maps to StatusFail. Scores in
	// [FailThreshold, OKThreshold) map to StatusUnsure. Default 0.5.
	FailThreshold float64
	// Weights map strategy name to its weight in the composed score. Only
	// strategies present at Compose time contribute; missing strategies do
	// not penalize the score. Unknown strategy names are ignored.
	Weights map[string]float64
	// UnsureBehavior tells the Checker how to route an UNSURE result.
	UnsureBehavior UnsureBehavior
	// MaxLatencyMs is a soft budget — strategies that exceed it short-
	// circuit with a neutral subscore. 0 means no limit.
	MaxLatencyMs int64
	// MaxCostUSD is a hard budget — exceeding it returns an error from
	// Check. 0 means no limit.
	MaxCostUSD float64
}

// DefaultPolicy returns the canonical starting point: OK >= 0.8, FAIL < 0.5,
// weights self_check=0.4 / consensus=0.4 / constraint=0.2, retry-once on
// UNSURE. These are the values defended in the F144 design doc; tune per
// call-site rather than mutating this constructor.
func DefaultPolicy() Policy {
	return Policy{
		OKThreshold:   0.8,
		FailThreshold: 0.5,
		Weights: map[string]float64{
			"self_check": 0.4,
			"consensus":  0.4,
			"constraint": 0.2,
		},
		UnsureBehavior: UnsureRetryOnce,
	}
}

// Validate reports whether p is internally consistent. It does not consult
// any strategies — those are validated at Checker construction time.
func (p Policy) Validate() error {
	for name, v := range map[string]float64{
		"OKThreshold":   p.OKThreshold,
		"FailThreshold": p.FailThreshold,
	} {
		if v < 0 || v > 1 || math.IsNaN(v) {
			return fmt.Errorf("threshold %s out of range [0,1]: %v", name, v)
		}
	}
	if p.OKThreshold <= p.FailThreshold {
		return fmt.Errorf("OKThreshold (%v) must be strictly greater than FailThreshold (%v)",
			p.OKThreshold, p.FailThreshold)
	}
	if len(p.Weights) == 0 {
		return errors.New("weights must contain at least one entry")
	}
	var sum float64
	for name, w := range p.Weights {
		if w < 0 || math.IsNaN(w) || math.IsInf(w, 0) {
			return fmt.Errorf("weight for %q is invalid: %v", name, w)
		}
		sum += w
	}
	if sum == 0 {
		return errors.New("at least one weight must be positive")
	}
	if !p.UnsureBehavior.Valid() {
		return fmt.Errorf("UnsureBehavior %q is not recognized", p.UnsureBehavior)
	}
	return nil
}

// StatusFor maps a composed score in [0, 1] to a Status using p's thresholds.
// Out-of-range inputs are clamped to FAIL on the low side and OK on the high
// side; callers are expected to feed in values produced by Compose, but this
// keeps the function total.
func (p Policy) StatusFor(score float64) Status {
	switch {
	case math.IsNaN(score) || score < p.FailThreshold:
		return StatusFail
	case score >= p.OKThreshold:
		return StatusOK
	default:
		return StatusUnsure
	}
}

// Compose aggregates per-strategy subscores into a single score in [0, 1]
// using a weighted average over the strategies actually present in subs.
// Missing strategies (e.g. lite-mode skipping consensus) do not penalize the
// score; unknown strategy names (not in Weights) are ignored. An empty input
// or all-out-of-range subscores returns an error.
func (p Policy) Compose(subs map[string]float64) (float64, error) {
	if len(subs) == 0 {
		return 0, errors.New("no subscores provided")
	}
	var (
		weighted float64
		sumW     float64
	)
	for name, sub := range subs {
		w, known := p.Weights[name]
		if !known || w == 0 {
			continue
		}
		if sub < 0 || sub > 1 || math.IsNaN(sub) || math.IsInf(sub, 0) {
			return 0, fmt.Errorf("subscore for %q out of range [0,1]: %v", name, sub)
		}
		weighted += w * sub
		sumW += w
	}
	if sumW == 0 {
		return 0, errors.New("no subscores from known weighted strategies")
	}
	return weighted / sumW, nil
}
