package nsample_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence/nsample"
)

type fakeCaller struct {
	mu             sync.Mutex
	samplesByTemp  map[float64]string
	tokens         confidence.TokenUsage
	delay          time.Duration
	errOnTemp      map[float64]error
	startTimes     []time.Time
	prompts        []string
	calls          int32
	respondInOrder []string
}

func (c *fakeCaller) Call(ctx context.Context, prompt string, opts confidence.CallOptions) (string, confidence.TokenUsage, error) {
	c.mu.Lock()
	c.startTimes = append(c.startTimes, time.Now())
	c.prompts = append(c.prompts, prompt)
	idx := int(atomic.AddInt32(&c.calls, 1)) - 1
	c.mu.Unlock()

	if c.delay > 0 {
		select {
		case <-time.After(c.delay):
		case <-ctx.Done():
			return "", confidence.TokenUsage{}, ctx.Err()
		}
	}

	if err, ok := c.errOnTemp[opts.Temperature]; ok {
		return "", confidence.TokenUsage{}, err
	}
	if c.samplesByTemp != nil {
		raw, ok := c.samplesByTemp[opts.Temperature]
		if !ok {
			return "", confidence.TokenUsage{}, fmt.Errorf("no canned sample for temp %v", opts.Temperature)
		}
		return raw, c.tokens, nil
	}
	if idx < len(c.respondInOrder) {
		return c.respondInOrder[idx], c.tokens, nil
	}
	return "", confidence.TokenUsage{}, fmt.Errorf("no sample at idx %d", idx)
}

func passParser(raw string) (string, error) {
	if raw == "" {
		return "", errors.New("empty")
	}
	return raw, nil
}

func selectiveParser(raw string) (string, error) {
	if strings.HasPrefix(raw, "BAD") {
		return "", fmt.Errorf("bad: %s", raw)
	}
	return raw, nil
}

func equalityAgreement(samples []string) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return 1
	}
	pairs, matches := 0, 0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			pairs++
			if samples[i] == samples[j] {
				matches++
			}
		}
	}
	return float64(matches) / float64(pairs)
}

func TestNewRejectsEmptyTemperatures(t *testing.T) {
	_, err := nsample.New[string](nsample.Options[string]{Parser: passParser, Agreement: equalityAgreement})
	if err == nil {
		t.Errorf("expected error for empty Temperatures")
	}
}

func TestNewRejectsNilParser(t *testing.T) {
	_, err := nsample.New[string](nsample.Options[string]{Temperatures: []float64{0.0}, Agreement: equalityAgreement})
	if err == nil {
		t.Errorf("expected error for nil Parser")
	}
}

func TestNewRejectsNilAgreement(t *testing.T) {
	_, err := nsample.New[string](nsample.Options[string]{Temperatures: []float64{0.0}, Parser: passParser})
	if err == nil {
		t.Errorf("expected error for nil Agreement")
	}
}

func TestNewDefaults(t *testing.T) {
	s, err := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Name() != "consensus" {
		t.Errorf("default Name = %q", s.Name())
	}
}

func TestRunRequiresCaller(t *testing.T) {
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{})
	if err == nil {
		t.Errorf("expected error when Caller nil")
	}
}

func TestRunIdenticalSamplesFullAgreement(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "A", 0.7: "A"}, tokens: confidence.TokenUsage{In: 10, Out: 5}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
	if out.Tokens.In != 30 || out.Tokens.Out != 15 {
		t.Errorf("Tokens = %+v, want In=30 Out=15", out.Tokens)
	}
}

func TestRunUsesRequestInputWhenBasePromptEmpty(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "A"}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{
		Caller: caller,
		Request: confidence.Request[string]{
			Input: "current request prompt",
		},
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, prompt := range caller.prompts {
		if prompt != "current request prompt" {
			t.Fatalf("prompt = %q, want current request prompt", prompt)
		}
	}
}

func TestRunAllDifferentZero(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "B", 0.7: "C"}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.0 {
		t.Errorf("SubScore = %v, want 0.0", out.SubScore)
	}
}

func TestRunPartialAgreement(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "A", 0.7: "B"}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := 1.0 / 3.0
	if diff := out.SubScore - want; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("SubScore = %v, want ~%v", out.SubScore, want)
	}
}

func TestRunCallerErrorPropagates(t *testing.T) {
	caller := &fakeCaller{
		samplesByTemp: map[float64]string{0.0: "A", 0.3: "A"},
		errOnTemp:     map[float64]error{0.7: errors.New("upstream boom")},
	}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	_, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err == nil || !strings.Contains(err.Error(), "upstream boom") {
		t.Errorf("err = %v, want wrapped 'upstream boom'", err)
	}
}

func TestRunMajorityParseFailZero(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{
		0.0: "A", 0.3: "BAD1", 0.5: "BAD2", 0.7: "BAD3",
	}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.5, 0.7},
		Parser:       selectiveParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 0.0 {
		t.Errorf("SubScore = %v, want 0.0", out.SubScore)
	}
	if !strings.Contains(out.Reason, "majority") {
		t.Errorf("Reason = %q, want 'majority'", out.Reason)
	}
}

func TestRunHalfParseFailComputesOnRemainder(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{
		0.0: "A", 0.3: "A", 0.5: "BAD1", 0.7: "BAD2",
	}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.5, 0.7},
		Parser:       selectiveParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0 (two A's agree)", out.SubScore)
	}
	if !strings.Contains(out.Reason, "parse") {
		t.Errorf("Reason = %q, want 'parse failures'", out.Reason)
	}
}

func TestRunCtxCancelMidFlight(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "A", 0.7: "A"}, delay: 100 * time.Millisecond}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	_, err := s.Run(ctx, confidence.StrategyInput[string]{Caller: caller})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestRunGenericInt(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "1", 0.3: "1", 0.7: "1"}}
	parser := func(raw string) (int, error) {
		switch raw {
		case "1":
			return 1, nil
		case "2":
			return 2, nil
		}
		return 0, fmt.Errorf("bad")
	}
	agreement := func(s []int) float64 {
		if len(s) == 0 {
			return 0
		}
		first := s[0]
		all := true
		for _, v := range s {
			if v != first {
				all = false
				break
			}
		}
		if all {
			return 1
		}
		return 0
	}
	s, _ := nsample.New[int](nsample.Options[int]{
		Temperatures: []float64{0.0, 0.3, 0.7},
		Parser:       parser,
		Agreement:    agreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[int]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if out.SubScore != 1.0 {
		t.Errorf("SubScore = %v, want 1.0", out.SubScore)
	}
}

func TestRunParallelExecution(t *testing.T) {
	const n = 4
	const delay = 80 * time.Millisecond
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.2: "A", 0.4: "A", 0.6: "A"}, delay: delay}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.2, 0.4, 0.6},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	start := time.Now()
	if _, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	elapsed := time.Since(start)
	if elapsed > 2*delay {
		t.Errorf("elapsed=%v > sequential bound %v — calls not parallel", elapsed, n*delay)
	}
	caller.mu.Lock()
	defer caller.mu.Unlock()
	if len(caller.startTimes) != n {
		t.Fatalf("got %d start times, want %d", len(caller.startTimes), n)
	}
	first, last := caller.startTimes[0], caller.startTimes[0]
	for _, ts := range caller.startTimes {
		if ts.Before(first) {
			first = ts
		}
		if ts.After(last) {
			last = ts
		}
	}
	if last.Sub(first) > 50*time.Millisecond {
		t.Errorf("call start spread = %v, want < 50ms (parallel)", last.Sub(first))
	}
}

func TestRunLogShape(t *testing.T) {
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: "A", 0.3: "BAD"}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0, 0.3},
		Parser:       selectiveParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	log, ok := out.Log.(nsample.Log)
	if !ok {
		t.Fatalf("Log type = %T", out.Log)
	}
	if len(log.Samples) != 2 {
		t.Errorf("log.Samples len = %d, want 2", len(log.Samples))
	}
	if log.ParseFailures != 1 {
		t.Errorf("ParseFailures = %d, want 1", log.ParseFailures)
	}
}

func TestRunRawTextTruncated(t *testing.T) {
	long := strings.Repeat("x", 1024)
	caller := &fakeCaller{samplesByTemp: map[float64]string{0.0: long}}
	s, _ := nsample.New[string](nsample.Options[string]{
		Temperatures: []float64{0.0},
		Parser:       passParser,
		Agreement:    equalityAgreement,
	})
	out, err := s.Run(context.Background(), confidence.StrategyInput[string]{Caller: caller})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	log := out.Log.(nsample.Log)
	if len(log.Samples[0].RawText) > 300 {
		t.Errorf("RawText not truncated: len=%d", len(log.Samples[0].RawText))
	}
}

// Compile-time assertion.
var _ confidence.Strategy[string] = (*nsample.Strategy[string])(nil)
