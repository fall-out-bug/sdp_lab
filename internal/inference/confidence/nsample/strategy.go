// Package nsample implements a redundancy-based confidence.Strategy[T]: it
// re-queries the LLM N times with different temperatures and scores
// confidence by mean pairwise agreement over the parsed answers.
//
// The Agreement function is supplied by the caller — this package is
// intentionally agnostic to whether agreement means exact equality, structural
// match, semantic similarity, or anything else. Pick what fits T.
//
// Cost: N× baseline tokens. Default N=3 ([0.0, 0.3, 0.7]) is the canonical
// starting point from the F144 design; N=2 ([0.2, 0.5]) is the cheap variant
// for hot paths.
package nsample

import (
	"context"
	"errors"
	"fmt"

	"golang.org/x/sync/errgroup"

	"sdp_dev/internal/inference/confidence"
)

const (
	defaultName        = "consensus"
	defaultMaxTokens   = 1024
	rawTextTraceMaxLen = 256
)

// Options configures an nsample Strategy.
type Options[T any] struct {
	// Temperatures is the jitter set; len(Temperatures) is N. Must be
	// non-empty.
	Temperatures []float64
	// BasePrompt is the prompt to re-sample. Adapters typically pass
	// Request.Input here.
	BasePrompt string
	// Parser parses each sample's raw text into T. Sample-level parse
	// failures are tolerated; majority parse-failure forces SubScore=0.
	Parser func(raw string) (T, error)
	// Agreement returns mean pairwise agreement in [0, 1] over the
	// successfully parsed samples. Caller-defined metric.
	Agreement func(samples []T) float64
	// MaxTokens caps each call's response. 0 → defaultMaxTokens.
	MaxTokens int
	// Name overrides "consensus".
	Name string
}

// SampleTrace records one sample's outcome for diagnostics.
type SampleTrace struct {
	Temperature float64
	RawText     string // truncated to rawTextTraceMaxLen
	ParseErr    string // empty on success
}

// Log is the per-run diagnostic blob exposed via StrategyOutput.Log.
type Log struct {
	Samples       []SampleTrace
	Agreement     float64
	ParseFailures int
}

// Strategy implements confidence.Strategy[T] via N-sample redundancy.
type Strategy[T any] struct {
	temps     []float64
	prompt    string
	parse     func(string) (T, error)
	agreement func([]T) float64
	maxTokens int
	name      string
}

// New constructs an nsample Strategy. Returns an error if Temperatures is
// empty or any required callback is nil.
func New[T any](opts Options[T]) (*Strategy[T], error) {
	if len(opts.Temperatures) == 0 {
		return nil, errors.New("nsample: Temperatures must be non-empty")
	}
	if opts.Parser == nil {
		return nil, errors.New("nsample: Parser is required")
	}
	if opts.Agreement == nil {
		return nil, errors.New("nsample: Agreement is required")
	}
	maxTok := opts.MaxTokens
	if maxTok <= 0 {
		maxTok = defaultMaxTokens
	}
	name := opts.Name
	if name == "" {
		name = defaultName
	}
	temps := make([]float64, len(opts.Temperatures))
	copy(temps, opts.Temperatures)
	return &Strategy[T]{
		temps:     temps,
		prompt:    opts.BasePrompt,
		parse:     opts.Parser,
		agreement: opts.Agreement,
		maxTokens: maxTok,
		name:      name,
	}, nil
}

// Name reports the registered strategy name (default "consensus").
func (s *Strategy[T]) Name() string { return s.name }

// Run launches len(Temperatures) parallel calls and aggregates agreement.
func (s *Strategy[T]) Run(ctx context.Context, in confidence.StrategyInput[T]) (confidence.StrategyOutput, error) {
	if in.Caller == nil {
		return confidence.StrategyOutput{}, errors.New("nsample: StrategyInput.Caller is required")
	}
	if err := ctx.Err(); err != nil {
		return confidence.StrategyOutput{}, err
	}

	n := len(s.temps)
	rawTexts := make([]string, n)
	usages := make([]confidence.TokenUsage, n)

	g, gctx := errgroup.WithContext(ctx)
	for i, t := range s.temps {
		i, t := i, t
		g.Go(func() error {
			raw, usage, err := in.Caller.Call(gctx, s.prompt, confidence.CallOptions{
				Temperature: t,
				MaxTokens:   s.maxTokens,
			})
			if err != nil {
				return fmt.Errorf("sample %d (temp=%v): %w", i, t, err)
			}
			rawTexts[i] = raw
			usages[i] = usage
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return confidence.StrategyOutput{}, err
	}

	var tokens confidence.TokenUsage
	for _, u := range usages {
		tokens = tokens.Add(u)
	}

	parsed := make([]T, 0, n)
	traces := make([]SampleTrace, n)
	parseFails := 0
	for i, raw := range rawTexts {
		traces[i].Temperature = s.temps[i]
		traces[i].RawText = truncate(raw, rawTextTraceMaxLen)
		v, perr := s.parse(raw)
		if perr != nil {
			traces[i].ParseErr = perr.Error()
			parseFails++
			continue
		}
		parsed = append(parsed, v)
	}

	if parseFails*2 > n {
		return confidence.StrategyOutput{
			SubScore: 0,
			Reason:   fmt.Sprintf("majority of samples failed to parse (%d/%d)", parseFails, n),
			Tokens:   tokens,
			Log:      Log{Samples: traces, ParseFailures: parseFails},
		}, nil
	}

	var score float64
	if len(parsed) > 0 {
		score = s.agreement(parsed)
	}

	reason := fmt.Sprintf("N=%d samples, agreement=%.2f", n, score)
	if parseFails > 0 {
		reason = fmt.Sprintf("%s (parse failures: %d)", reason, parseFails)
	}

	return confidence.StrategyOutput{
		SubScore: score,
		Reason:   reason,
		Tokens:   tokens,
		Log:      Log{Samples: traces, Agreement: score, ParseFailures: parseFails},
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
