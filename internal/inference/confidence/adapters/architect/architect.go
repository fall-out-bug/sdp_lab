// Package architect wraps architectural classification (internal/architect/
// classify Hypothesizer output) in a confidence.Checker. Profile: full set
// (constraint + nsample + self-check), N=3 fixed. UNSURE behavior:
// auto-retry with N=5 — this is an analysis path, not a production gate, so
// retry-once is preferable to human handoff.
package architect

import (
	"context"
	"fmt"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/constraint"
	"sdp_dev/internal/inference/confidence/nsample"
	"sdp_dev/internal/inference/confidence/selfcheck"
)

// Classification is one architectural verdict — a style hypothesis (layered,
// hexagonal, etc.) or a pattern claim (DDD aggregate, GoF observer, ...).
// We aggregate per-item confidence into a top-level Score via the composer.
type Classification struct {
	Items []ClassifiedItem `json:"items"`
}

type ClassifiedItem struct {
	Kind       string   `json:"kind"`       // "style" | "pattern"
	Name       string   `json:"name"`       // e.g. "layered", "ddd-aggregate"
	Confidence float64  `json:"confidence"` // [0, 1] per-item
	Evidence   []string `json:"evidence,omitempty"`
}

// Options configures an architect Checker. NSamplePrompt is the prompt to
// use for re-sampling (typically the original classification prompt).
type Options struct {
	Caller        confidence.LLMCaller
	NSamplePrompt string
	Parser        func(raw string) (Classification, error)
	Policy        *confidence.Policy
}

// New builds the full-set Checker for architect classification.
func New(opts Options) (*confidence.Checker[Classification], error) {
	if opts.Caller == nil {
		return nil, fmt.Errorf("architect: Caller is required")
	}
	if opts.Parser == nil {
		return nil, fmt.Errorf("architect: Parser is required")
	}

	cs, err := constraint.New[Classification](constraint.Options[Classification]{
		Invariants: []constraint.Invariant[Classification]{
			{
				Name: "non-empty-items",
				Check: func(c Classification) (bool, string) {
					if len(c.Items) == 0 {
						return false, "no items classified"
					}
					return true, ""
				},
			},
			{
				Name: "per-item-confidence-range",
				Check: func(c Classification) (bool, string) {
					for _, it := range c.Items {
						if it.Confidence < 0 || it.Confidence > 1 {
							return false, fmt.Sprintf("item %s confidence %v out of [0,1]", it.Name, it.Confidence)
						}
					}
					return true, ""
				},
			},
			{
				Name: "kind-known",
				Check: func(c Classification) (bool, string) {
					for _, it := range c.Items {
						switch it.Kind {
						case "style", "pattern":
							// ok
						default:
							return false, fmt.Sprintf("item %s has unknown kind %q", it.Name, it.Kind)
						}
					}
					return true, ""
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("architect: build constraint: %w", err)
	}

	sc, err := selfcheck.New[Classification](selfcheck.Options[Classification]{
		Mode: selfcheck.ModeFull,
	})
	if err != nil {
		return nil, fmt.Errorf("architect: build self-check: %w", err)
	}

	ns, err := nsample.New[Classification](nsample.Options[Classification]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		BasePrompt:   opts.NSamplePrompt,
		Parser:       opts.Parser,
		Agreement:    classificationAgreement,
	})
	if err != nil {
		return nil, fmt.Errorf("architect: build nsample: %w", err)
	}

	policy := canonicalPolicy()
	if opts.Policy != nil {
		policy = *opts.Policy
	}

	return confidence.NewChecker[Classification](opts.Caller,
		[]confidence.Strategy[Classification]{cs, sc, ns}, policy)
}

// Verify is a thin wrapper that runs the Checker on a parsed Classification.
// input must be the original classification prompt. Use AggregateScore to
// derive a per-item-confidence rollup score for telemetry; it is independent of
// the gating Score.
func Verify(ctx context.Context, checker *confidence.Checker[Classification], input string, parsed Classification, raw string) (confidence.Result[Classification], error) {
	return checker.Check(ctx, confidence.Request[Classification]{
		Input:  input,
		Answer: parsed,
		Raw:    raw,
	})
}

// AggregateScore returns the mean per-item Confidence — useful for telemetry
// independent of the composed (gating) Score.
func AggregateScore(c Classification) float64 {
	if len(c.Items) == 0 {
		return 0
	}
	var sum float64
	for _, it := range c.Items {
		sum += it.Confidence
	}
	return sum / float64(len(c.Items))
}

// canonicalPolicy: full set defaults, but UNSURE → retry-once (architect is
// analysis, not a gate).
func canonicalPolicy() confidence.Policy {
	p := confidence.DefaultPolicy()
	p.UnsureBehavior = confidence.UnsureRetryOnce
	return p
}

// classificationAgreement: top-1 style/pattern names match across samples.
// We do NOT compare full Items lists because evidence ordering varies;
// consensus on the dominant classification is the load-bearing signal.
func classificationAgreement(samples []Classification) float64 {
	n := len(samples)
	if n < 2 {
		if n == 1 {
			return 1
		}
		return 0
	}
	tops := make([]string, n)
	for i, s := range samples {
		tops[i] = top1Name(s)
	}
	pairs, matches := 0, 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs++
			if tops[i] != "" && tops[i] == tops[j] {
				matches++
			}
		}
	}
	if pairs == 0 {
		return 0
	}
	return float64(matches) / float64(pairs)
}

func top1Name(c Classification) string {
	if len(c.Items) == 0 {
		return ""
	}
	best := c.Items[0]
	for _, it := range c.Items[1:] {
		if it.Confidence > best.Confidence {
			best = it
		}
	}
	return best.Kind + ":" + best.Name
}
