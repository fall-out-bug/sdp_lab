package selfcheck_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence/selfcheck"
)

// fakeCaller returns a canned response or error.
type fakeCaller struct {
	resp   string
	usage  confidence.TokenUsage
	err    error
	calls  int
	lastT  float64
	lastMx int
}

func (c *fakeCaller) Call(_ context.Context, _ string, opts confidence.CallOptions) (string, confidence.TokenUsage, error) {
	c.calls++
	c.lastT = opts.Temperature
	c.lastMx = opts.MaxTokens
	if c.err != nil {
		return "", confidence.TokenUsage{}, c.err
	}
	return c.resp, c.usage, nil
}

func TestModeValid(t *testing.T) {
	if !selfcheck.ModeFull.Valid() {
		t.Errorf("ModeFull.Valid() = false")
	}
	if !selfcheck.ModeLite.Valid() {
		t.Errorf("ModeLite.Valid() = false")
	}
	if selfcheck.Mode("garbage").Valid() {
		t.Errorf("garbage Mode.Valid() = true")
	}
}

func TestNewDefaultsToFull(t *testing.T) {
	s, err := selfcheck.New[string](selfcheck.Options[string]{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Name() != "self_check" {
		t.Errorf("default Name = %q", s.Name())
	}
}

func TestNewRejectsBadMode(t *testing.T) {
	_, err := selfcheck.New[string](selfcheck.Options[string]{Mode: "bogus"})
	if err == nil {
		t.Errorf("expected error for bad Mode")
	}
}

func TestFullModeAgree(t *testing.T) {
	caller := &fakeCaller{
		resp:  `{"verdict":"agree","confidence":0.92,"issues":[]}`,
		usage: confidence.TokenUsage{In: 100, Out: 30},
	}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Input: "q", Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.92 {
		t.Errorf("SubScore = %v, want 0.92", out.SubScore)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false")
	}
	if !strings.Contains(out.Reason, "agree") {
		t.Errorf("Reason = %q, want mention of agree", out.Reason)
	}
	if out.Tokens.In != 100 || out.Tokens.Out != 30 {
		t.Errorf("Tokens = %+v", out.Tokens)
	}
	if caller.lastT != 0.0 {
		t.Errorf("critic temperature = %v, want 0.0", caller.lastT)
	}
}

func TestFullModeDisagreeInverts(t *testing.T) {
	caller := &fakeCaller{resp: `{"verdict":"disagree","confidence":0.8,"issues":["x"]}`}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := 1.0 - 0.8
	if diff := out.SubScore - want; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("SubScore = %v, want %v", out.SubScore, want)
	}
}

func TestFullModeUnsureNeutral(t *testing.T) {
	caller := &fakeCaller{resp: `{"verdict":"unsure","confidence":0.5,"issues":[]}`}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.5 {
		t.Errorf("SubScore = %v, want 0.5", out.SubScore)
	}
}

func TestFullModeMalformedJSONNeutral(t *testing.T) {
	caller := &fakeCaller{resp: `not-a-json-object`}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.5 {
		t.Errorf("SubScore = %v, want 0.5 (neutral on bad critic)", out.SubScore)
	}
	if out.HardFail {
		t.Errorf("HardFail = true, want false")
	}
	if !strings.Contains(out.Reason, "malformed") {
		t.Errorf("Reason = %q, want 'malformed'", out.Reason)
	}
	log := out.Log.(selfcheck.Log)
	if log.RawCritic != "not-a-json-object" {
		t.Errorf("Log.RawCritic = %q", log.RawCritic)
	}
}

func TestFullModeUnknownVerdictNeutral(t *testing.T) {
	caller := &fakeCaller{resp: `{"verdict":"perhaps","confidence":1.0}`}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.5 {
		t.Errorf("SubScore = %v, want 0.5 on unknown verdict", out.SubScore)
	}
}

func TestFullModeCallerError(t *testing.T) {
	caller := &fakeCaller{err: errors.New("upstream boom")}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if err == nil || !strings.Contains(err.Error(), "upstream boom") {
		t.Errorf("err = %v, want wrapped 'upstream boom'", err)
	}
}

func TestFullModeWithoutCaller(t *testing.T) {
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
	})
	if err == nil {
		t.Errorf("expected error: full mode requires Caller")
	}
}

func TestLiteModeExtractor(t *testing.T) {
	extracted := false
	s, _ := selfcheck.New[string](selfcheck.Options[string]{
		Mode: selfcheck.ModeLite,
		LiteScoreExtractor: func(raw string) (float64, bool) {
			extracted = true
			if strings.Contains(raw, "self_score: 0.85") {
				return 0.85, true
			}
			return 0, false
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "answer text\nself_score: 0.85"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !extracted {
		t.Errorf("LiteScoreExtractor not called")
	}
	if out.SubScore != 0.85 {
		t.Errorf("SubScore = %v, want 0.85", out.SubScore)
	}
	if out.Tokens != (confidence.TokenUsage{}) {
		t.Errorf("Tokens = %+v, want zero (no LLM call in lite)", out.Tokens)
	}
}

func TestLiteModeAnnotationMissing(t *testing.T) {
	s, _ := selfcheck.New[string](selfcheck.Options[string]{
		Mode: selfcheck.ModeLite,
		LiteScoreExtractor: func(string) (float64, bool) {
			return 0, false
		},
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "no annotation"},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.5 {
		t.Errorf("SubScore = %v, want 0.5 on missing annotation", out.SubScore)
	}
	if !strings.Contains(out.Reason, "no self_score") {
		t.Errorf("Reason = %q, want 'no self_score'", out.Reason)
	}
}

func TestLiteModeWithoutExtractor(t *testing.T) {
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeLite})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "x"},
	})
	if err == nil {
		t.Errorf("expected error: lite mode requires LiteScoreExtractor")
	}
}

func TestContextCancel(t *testing.T) {
	caller := &fakeCaller{resp: `{"verdict":"agree","confidence":1.0}`}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{Mode: selfcheck.ModeFull})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Run(ctx, confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Answer: "a", Raw: "a"},
		Caller:  caller,
	})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestCustomTemplateUsed(t *testing.T) {
	caller := &fakeCaller{resp: `{"verdict":"agree","confidence":1.0}`}
	captured := ""
	tmpl := func(input, raw string) string {
		captured = "INPUT=" + input + "|RAW=" + raw
		return captured
	}
	s, _ := selfcheck.New[string](selfcheck.Options[string]{
		Mode:                 selfcheck.ModeFull,
		CriticPromptTemplate: tmpl,
	})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Request: confidence.Request[string]{Input: "q", Answer: "x", Raw: "y"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if captured != "INPUT=q|RAW=y" {
		t.Errorf("custom template not invoked correctly: %q", captured)
	}
}

func TestGenericStructAnswer(t *testing.T) {
	type ans struct{ Kind string }
	caller := &fakeCaller{resp: `{"verdict":"agree","confidence":1.0}`}
	s, _ := selfcheck.New[ans](selfcheck.Options[ans]{Mode: selfcheck.ModeFull})
	out, err := s.Run(context.Background(), confidence.StrategyInput[ans]{
		Request: confidence.Request[ans]{Answer: ans{Kind: "x"}, Raw: "x"},
		Caller:  caller,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
}

// Compile-time satisfaction.
var _ confidence.Strategy[string] = (*selfcheck.Strategy[string])(nil)
