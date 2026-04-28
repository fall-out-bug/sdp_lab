// Package selfcheck provides a confidence.Strategy[T] that asks the model to
// review its own answer through a critic prompt with role-swap, or — in lite
// mode — extracts a self-score annotation from the primary call's raw text.
//
// Two modes:
//
//   - ModeFull (default): one extra LLM round-trip. The critic receives input
//     and the candidate raw answer, and is asked to return a strict JSON
//     verdict {agree, disagree, unsure} with a confidence in [0,1]. Strong
//     signal, but doubles round-trip count.
//   - ModeLite: no extra LLM calls. A caller-provided extractor reads a
//     self_score annotation from the primary raw text. Weaker signal, but
//     free. Suitable for hot paths like dispatch classify.
//
// Bad critic output (malformed JSON, unknown verdict) degrades to a neutral
// 0.5 — never hard-fails. We don't want a flaky critic to override a valid
// answer; that's what constraint and N-sample are for.
package selfcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
)

// Mode selects the self-check protocol.
type Mode string

const (
	ModeFull Mode = "full"
	ModeLite Mode = "lite"
)

// Valid reports whether m is one of the defined Mode constants.
func (m Mode) Valid() bool {
	switch m {
	case ModeFull, ModeLite:
		return true
	default:
		return false
	}
}

// Options configures a selfcheck Strategy.
type Options[T any] struct {
	// Mode selects full critic-prompt or lite annotation-only mode. Empty
	// defaults to ModeFull.
	Mode Mode
	// CriticPromptTemplate builds the critic prompt from input and raw.
	// If nil in ModeFull, defaultCriticPrompt is used.
	CriticPromptTemplate func(input, raw string) string
	// LiteScoreExtractor parses a self-score annotation from primary raw
	// text. Required in ModeLite. Returning ok=false signals "annotation
	// not found" → SubScore=0.5 with descriptive Reason.
	LiteScoreExtractor func(raw string) (score float64, ok bool)
	// Name overrides the default strategy name "self_check".
	Name string
}

// Strategy implements confidence.Strategy[T] for self-check.
type Strategy[T any] struct {
	mode      Mode
	template  func(input, raw string) string
	extractor func(string) (float64, bool)
	name      string
}

// New constructs a Strategy. Returns an error if Mode is set to an unknown
// value. Mode-specific required fields (Caller for full, Extractor for lite)
// are validated at Run time so the same constructor surface works for both
// modes.
func New[T any](opts Options[T]) (*Strategy[T], error) {
	mode := opts.Mode
	if mode == "" {
		mode = ModeFull
	}
	if !mode.Valid() {
		return nil, fmt.Errorf("selfcheck: Mode %q not recognized", opts.Mode)
	}
	tmpl := opts.CriticPromptTemplate
	if tmpl == nil {
		tmpl = defaultCriticPrompt
	}
	name := opts.Name
	if name == "" {
		name = "self_check"
	}
	return &Strategy[T]{
		mode:      mode,
		template:  tmpl,
		extractor: opts.LiteScoreExtractor,
		name:      name,
	}, nil
}

// Log captures per-run evidence: which mode ran, what the critic returned
// (raw + parsed), and any issues the critic listed.
type Log struct {
	Mode             string
	Verdict          string
	CriticConfidence float64
	Issues           []string
	RawCritic        string
}

// Name reports the strategy's registered name.
func (s *Strategy[T]) Name() string { return s.name }

// Run executes self-check in the configured mode.
func (s *Strategy[T]) Run(ctx context.Context, in confidence.StrategyInput[T]) (confidence.StrategyOutput, error) {
	if err := ctx.Err(); err != nil {
		return confidence.StrategyOutput{}, err
	}
	switch s.mode {
	case ModeLite:
		return s.runLite(in)
	default:
		return s.runFull(ctx, in)
	}
}

func (s *Strategy[T]) runLite(in confidence.StrategyInput[T]) (confidence.StrategyOutput, error) {
	if s.extractor == nil {
		return confidence.StrategyOutput{}, errors.New("selfcheck: lite mode requires LiteScoreExtractor")
	}
	score, ok := s.extractor(in.Request.Raw)
	if !ok {
		return confidence.StrategyOutput{
			SubScore: 0.5,
			Reason:   "lite mode: no self_score annotation found",
			Log:      Log{Mode: string(ModeLite)},
		}, nil
	}
	if score < 0 || score > 1 {
		// Out-of-range extractor output is a contract violation; clamp to
		// neutral and record so the caller can fix their extractor.
		return confidence.StrategyOutput{
			SubScore: 0.5,
			Reason:   fmt.Sprintf("lite mode: extractor returned out-of-range score %v", score),
			Log:      Log{Mode: string(ModeLite)},
		}, nil
	}
	return confidence.StrategyOutput{
		SubScore: score,
		Reason:   fmt.Sprintf("lite mode: self_score=%.2f", score),
		Log:      Log{Mode: string(ModeLite)},
	}, nil
}

// criticVerdict is the wire format we expect the critic to return.
type criticVerdict struct {
	Verdict    string   `json:"verdict"`
	Confidence float64  `json:"confidence"`
	Issues     []string `json:"issues"`
}

func (s *Strategy[T]) runFull(ctx context.Context, in confidence.StrategyInput[T]) (confidence.StrategyOutput, error) {
	if in.Caller == nil {
		return confidence.StrategyOutput{}, errors.New("selfcheck: full mode requires StrategyInput.Caller")
	}
	prompt := s.template(in.Request.Input, in.Request.Raw)
	resp, usage, err := in.Caller.Call(ctx, prompt, confidence.CallOptions{
		Temperature: 0.0, // critic is deterministic
		MaxTokens:   256,
	})
	if err != nil {
		return confidence.StrategyOutput{}, fmt.Errorf("selfcheck: critic call failed: %w", err)
	}

	var v criticVerdict
	if jerr := json.Unmarshal([]byte(resp), &v); jerr != nil {
		return confidence.StrategyOutput{
			SubScore: 0.5,
			Reason:   "critic returned malformed JSON",
			Tokens:   usage,
			Log:      Log{Mode: string(ModeFull), RawCritic: resp},
		}, nil
	}

	conf := clampConfidence(v.Confidence)
	var (
		score  float64
		reason string
	)
	switch v.Verdict {
	case "agree":
		score = conf
		reason = fmt.Sprintf("critic agreed (conf %.2f)", conf)
	case "disagree":
		score = 1 - conf
		reason = fmt.Sprintf("critic disagreed (conf %.2f)", conf)
		if len(v.Issues) > 0 {
			reason = fmt.Sprintf("%s: %s", reason, v.Issues[0])
		}
	case "unsure":
		score = 0.5
		reason = "critic unsure"
	default:
		// Unknown verdict: degrade to neutral, don't fail.
		score = 0.5
		reason = fmt.Sprintf("critic returned unknown verdict %q", v.Verdict)
	}

	return confidence.StrategyOutput{
		SubScore: score,
		Reason:   reason,
		Tokens:   usage,
		Log: Log{
			Mode:             string(ModeFull),
			Verdict:          v.Verdict,
			CriticConfidence: conf,
			Issues:           v.Issues,
			RawCritic:        resp,
		},
	}, nil
}

func clampConfidence(c float64) float64 {
	switch {
	case c < 0:
		return 0
	case c > 1:
		return 1
	default:
		return c
	}
}

func defaultCriticPrompt(input, raw string) string {
	return `You are a strict reviewer. Verify whether the candidate answer is correct given the input.

Input:
` + input + `

Candidate answer:
` + raw + `

Return STRICT JSON only, no prose, no fencing:
{"verdict": "agree" | "disagree" | "unsure", "confidence": 0.0-1.0, "issues": ["..."]}

- "agree" with high confidence: answer is correct and complete.
- "disagree" with high confidence: answer is wrong; list issues.
- "unsure": evidence is mixed.
`
}
