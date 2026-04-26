package confidence

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Checker orchestrates a set of Strategy[T] under a Policy and produces a
// Result[T]. It is value-typed and safe for concurrent use as long as the
// underlying strategies are.
type Checker[T any] struct {
	caller     LLMCaller
	strategies []Strategy[T]
	policy     Policy
}

// NewChecker constructs a Checker with the given caller (may be nil if no
// strategy needs LLM calls), strategies, and policy. It validates the policy
// and rejects empty strategy lists.
func NewChecker[T any](caller LLMCaller, strategies []Strategy[T], policy Policy) (*Checker[T], error) {
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("policy invalid: %w", err)
	}
	if len(strategies) == 0 {
		return nil, errors.New("at least one strategy required")
	}
	return &Checker[T]{caller: caller, strategies: strategies, policy: policy}, nil
}

// Check runs every strategy against req and composes the results into a
// Result[T]. Strategy errors abort the check and bubble up wrapped. A
// hard-fail from any strategy forces Status=FAIL but Score is still computed
// from subscores so callers can see how close the result was.
func (c *Checker[T]) Check(ctx context.Context, req Request[T]) (Result[T], error) {
	if err := ctx.Err(); err != nil {
		return Result[T]{}, err
	}

	start := time.Now()
	subs := make(map[string]float64, len(c.strategies))
	reasons := make([]string, 0, len(c.strategies))
	var (
		tokens   TokenUsage
		hardFail bool
	)

	in := StrategyInput[T]{Request: req, Caller: c.caller}
	for _, s := range c.strategies {
		if err := ctx.Err(); err != nil {
			return Result[T]{}, err
		}
		if c.policy.MaxLatencyMs > 0 && time.Since(start).Milliseconds() > c.policy.MaxLatencyMs {
			// Soft latency budget exhausted: record a neutral subscore so
			// later strategies don't drag the composed score to zero just
			// for being skipped, and surface the skip in Reasons.
			subs[s.Name()] = 0.5
			reasons = append(reasons, fmt.Sprintf("%s: skipped: budget", s.Name()))
			continue
		}
		out, err := s.Run(ctx, in)
		if err != nil {
			return Result[T]{}, fmt.Errorf("strategy %q: %w", s.Name(), err)
		}
		subs[s.Name()] = out.SubScore
		if out.Reason != "" {
			reasons = append(reasons, fmt.Sprintf("%s: %s", s.Name(), out.Reason))
		}
		tokens = tokens.Add(out.Tokens)
		if out.HardFail {
			hardFail = true
		}
	}

	score, err := c.policy.Compose(subs)
	if err != nil {
		return Result[T]{}, fmt.Errorf("compose: %w", err)
	}

	status := c.policy.StatusFor(score)
	if hardFail {
		status = StatusFail
	}

	return Result[T]{
		Answer:    req.Answer,
		Status:    status,
		Score:     score,
		SubScores: subs,
		Reasons:   reasons,
		Trace: Trace{
			LatencyMs: time.Since(start).Milliseconds(),
			TokensIn:  tokens.In,
			TokensOut: tokens.Out,
		},
		Attempts: 1,
	}, nil
}
