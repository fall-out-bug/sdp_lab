package architect_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
	"testing"

	"sdp_dev/internal/inference/confidence"
	"sdp_dev/internal/inference/confidence/adapters/architect"
)

// fakeCaller cycles through a queue of responses. selfcheck call always
// returns first; subsequent N-sample calls return classifications.
type fakeCaller struct {
	calls    int32
	responses []string
}

func (f *fakeCaller) Call(_ context.Context, _ string, _ confidence.CallOptions) (string, confidence.TokenUsage, error) {
	idx := int(atomic.AddInt32(&f.calls, 1)) - 1
	if idx >= len(f.responses) {
		return "", confidence.TokenUsage{}, nil
	}
	return f.responses[idx], confidence.TokenUsage{In: 50, Out: 20}, nil
}

func parseClassification(raw string) (architect.Classification, error) {
	var c architect.Classification
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return architect.Classification{}, err
	}
	return c, nil
}

func TestNewRequiresCaller(t *testing.T) {
	_, err := architect.New(architect.Options{Parser: parseClassification})
	if err == nil {
		t.Errorf("expected error: Caller required")
	}
}

func TestNewRequiresParser(t *testing.T) {
	_, err := architect.New(architect.Options{Caller: &fakeCaller{}})
	if err == nil {
		t.Errorf("expected error: Parser required")
	}
}

func TestVerifyHappyPath(t *testing.T) {
	good := `{"items":[{"kind":"style","name":"layered","confidence":0.9}]}`
	caller := &fakeCaller{responses: []string{
		`{"verdict":"agree","confidence":0.95}`, // selfcheck critic
		good, good, good,                         // 3 nsample responses
	}}
	checker, err := architect.New(architect.Options{
		Caller:        caller,
		Parser:        parseClassification,
		NSamplePrompt: "classify",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	parsed, _ := parseClassification(good)
	res, err := architect.Verify(context.Background(), checker, parsed, good)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if res.Status != confidence.StatusOK {
		t.Errorf("Status = %q, want OK; reasons=%v", res.Status, res.Reasons)
	}
}

func TestVerifyEmptyItemsViolatesInvariant(t *testing.T) {
	empty := `{"items":[]}`
	caller := &fakeCaller{responses: []string{
		`{"verdict":"unsure","confidence":0.5}`,
		empty, empty, empty,
	}}
	checker, _ := architect.New(architect.Options{
		Caller:        caller,
		Parser:        parseClassification,
		NSamplePrompt: "classify",
	})
	parsed, _ := parseClassification(empty)
	res, err := architect.Verify(context.Background(), checker, parsed, empty)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "non-empty-items") {
		t.Errorf("Reasons missing non-empty-items: %v", res.Reasons)
	}
}

func TestVerifyDisagreementGivesLowerScore(t *testing.T) {
	a := `{"items":[{"kind":"style","name":"layered","confidence":0.9}]}`
	b := `{"items":[{"kind":"style","name":"hexagonal","confidence":0.9}]}`
	c := `{"items":[{"kind":"style","name":"event-driven","confidence":0.9}]}`
	caller := &fakeCaller{responses: []string{
		`{"verdict":"unsure","confidence":0.5}`,
		a, b, c,
	}}
	checker, _ := architect.New(architect.Options{
		Caller:        caller,
		Parser:        parseClassification,
		NSamplePrompt: "classify",
	})
	parsed, _ := parseClassification(a)
	res, err := architect.Verify(context.Background(), checker, parsed, a)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// Three samples all-different on top-1 → consensus subscore=0.
	if res.SubScores["consensus"] != 0 {
		t.Errorf("consensus subscore = %v, want 0", res.SubScores["consensus"])
	}
	if res.Status == confidence.StatusOK {
		t.Errorf("Status = OK on full disagreement; expected UNSURE/FAIL. Score=%v", res.Score)
	}
}

func TestAggregateScoreEmpty(t *testing.T) {
	if got := architect.AggregateScore(architect.Classification{}); got != 0 {
		t.Errorf("AggregateScore(empty) = %v, want 0", got)
	}
}

func TestAggregateScoreMean(t *testing.T) {
	c := architect.Classification{Items: []architect.ClassifiedItem{
		{Confidence: 0.6}, {Confidence: 0.8}, {Confidence: 1.0},
	}}
	got := architect.AggregateScore(c)
	want := 0.8
	if diff := got - want; diff < -1e-9 || diff > 1e-9 {
		t.Errorf("AggregateScore = %v, want %v", got, want)
	}
}

func TestVerifyKindUnknownInvariant(t *testing.T) {
	bad := `{"items":[{"kind":"unknown","name":"x","confidence":0.5}]}`
	caller := &fakeCaller{responses: []string{
		`{"verdict":"unsure","confidence":0.5}`,
		bad, bad, bad,
	}}
	checker, _ := architect.New(architect.Options{
		Caller: caller, Parser: parseClassification, NSamplePrompt: "x",
	})
	parsed, _ := parseClassification(bad)
	res, _ := architect.Verify(context.Background(), checker, parsed, bad)
	joined := strings.Join(res.Reasons, "|")
	if !strings.Contains(joined, "kind-known") {
		t.Errorf("Reasons missing kind-known invariant: %v", res.Reasons)
	}
}
