package confidence_test

import (
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/inference/confidence"
)

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name string
		s    confidence.Status
		want string
	}{
		{"OK", confidence.StatusOK, "ok"},
		{"UNSURE", confidence.StatusUnsure, "unsure"},
		{"FAIL", confidence.StatusFail, "fail"},
	}
	for _, tt := range tests {
		if string(tt.s) != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, string(tt.s), tt.want)
		}
	}
}

func TestStatusValid(t *testing.T) {
	valid := []confidence.Status{
		confidence.StatusOK,
		confidence.StatusUnsure,
		confidence.StatusFail,
	}
	for _, s := range valid {
		if !s.Valid() {
			t.Errorf("Status(%q).Valid() = false, want true", s)
		}
	}

	if confidence.Status("garbage").Valid() {
		t.Errorf("Status(garbage).Valid() = true, want false")
	}
	if confidence.Status("").Valid() {
		t.Errorf("Status(empty).Valid() = true, want false")
	}
}

func TestResultZeroValue(t *testing.T) {
	var r confidence.Result[string]
	if r.Status != "" {
		t.Errorf("zero Result.Status = %q, want empty", r.Status)
	}
	if r.Score != 0 {
		t.Errorf("zero Result.Score = %v, want 0", r.Score)
	}
	if r.Attempts != 0 {
		t.Errorf("zero Result.Attempts = %v, want 0", r.Attempts)
	}
}

func TestResultGenericsPreserveAnswer(t *testing.T) {
	type customAnswer struct {
		Tag   string
		Value int
	}
	r := confidence.Result[customAnswer]{
		Answer:    customAnswer{Tag: "x", Value: 42},
		Status:    confidence.StatusOK,
		Score:     0.95,
		SubScores: map[string]float64{"self_check": 0.9, "consensus": 1.0, "constraint": 1.0},
		Reasons:   []string{"all strategies agree"},
		Attempts:  1,
	}
	if r.Answer.Tag != "x" || r.Answer.Value != 42 {
		t.Errorf("Answer not preserved: %+v", r.Answer)
	}
	if r.Status != confidence.StatusOK {
		t.Errorf("Status = %q, want %q", r.Status, confidence.StatusOK)
	}
	if r.Score != 0.95 {
		t.Errorf("Score = %v, want 0.95", r.Score)
	}
	if got := r.SubScores["self_check"]; got != 0.9 {
		t.Errorf("SubScores[self_check] = %v, want 0.9", got)
	}
}

func TestTraceFields(t *testing.T) {
	tr := confidence.Trace{
		LatencyMs: 100,
		TokensIn:  200,
		TokensOut: 50,
		CostUSD:   0.001,
	}
	if tr.LatencyMs != 100 {
		t.Errorf("LatencyMs = %d, want 100", tr.LatencyMs)
	}
	if tr.TokensIn != 200 {
		t.Errorf("TokensIn = %d, want 200", tr.TokensIn)
	}
	if tr.TokensOut != 50 {
		t.Errorf("TokensOut = %d, want 50", tr.TokensOut)
	}
	if tr.CostUSD != 0.001 {
		t.Errorf("CostUSD = %v, want 0.001", tr.CostUSD)
	}
}

func TestTraceTotalTokens(t *testing.T) {
	tr := confidence.Trace{TokensIn: 200, TokensOut: 50}
	if got := tr.TotalTokens(); got != 250 {
		t.Errorf("TotalTokens() = %d, want 250", got)
	}
}
