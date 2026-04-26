package confidence_test

import (
	"math"
	"strings"
	"testing"

	"sdp_dev/internal/inference/confidence"
)

func TestDefaultPolicyDefaults(t *testing.T) {
	p := confidence.DefaultPolicy()

	if p.OKThreshold != 0.8 {
		t.Errorf("OKThreshold = %v, want 0.8", p.OKThreshold)
	}
	if p.FailThreshold != 0.5 {
		t.Errorf("FailThreshold = %v, want 0.5", p.FailThreshold)
	}
	if got := p.Weights["self_check"]; got != 0.4 {
		t.Errorf("Weights[self_check] = %v, want 0.4", got)
	}
	if got := p.Weights["consensus"]; got != 0.4 {
		t.Errorf("Weights[consensus] = %v, want 0.4", got)
	}
	if got := p.Weights["constraint"]; got != 0.2 {
		t.Errorf("Weights[constraint] = %v, want 0.2", got)
	}
	if p.UnsureBehavior != confidence.UnsureRetryOnce {
		t.Errorf("UnsureBehavior = %v, want UnsureRetryOnce", p.UnsureBehavior)
	}
	if err := p.Validate(); err != nil {
		t.Errorf("DefaultPolicy invalid: %v", err)
	}
}

func TestPolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*confidence.Policy)
		wantErr string
	}{
		{
			name:    "valid default",
			mutate:  func(*confidence.Policy) {},
			wantErr: "",
		},
		{
			name:    "ok < fail",
			mutate:  func(p *confidence.Policy) { p.OKThreshold = 0.3; p.FailThreshold = 0.7 },
			wantErr: "OKThreshold",
		},
		{
			name:    "ok = fail",
			mutate:  func(p *confidence.Policy) { p.OKThreshold = 0.5; p.FailThreshold = 0.5 },
			wantErr: "OKThreshold",
		},
		{
			name:    "negative threshold",
			mutate:  func(p *confidence.Policy) { p.FailThreshold = -0.1 },
			wantErr: "threshold",
		},
		{
			name:    "threshold > 1",
			mutate:  func(p *confidence.Policy) { p.OKThreshold = 1.1 },
			wantErr: "threshold",
		},
		{
			name:    "empty weights",
			mutate:  func(p *confidence.Policy) { p.Weights = nil },
			wantErr: "weights",
		},
		{
			name:    "negative weight",
			mutate:  func(p *confidence.Policy) { p.Weights["self_check"] = -0.1 },
			wantErr: "weight",
		},
		{
			name:    "all weights zero",
			mutate:  func(p *confidence.Policy) { p.Weights = map[string]float64{"x": 0, "y": 0} },
			wantErr: "weight",
		},
		{
			name:    "unknown unsure behavior",
			mutate:  func(p *confidence.Policy) { p.UnsureBehavior = "garbage" },
			wantErr: "UnsureBehavior",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := confidence.DefaultPolicy()
			tt.mutate(&p)
			err := p.Validate()
			switch {
			case tt.wantErr == "" && err != nil:
				t.Errorf("expected nil error, got %v", err)
			case tt.wantErr != "" && err == nil:
				t.Errorf("expected error containing %q, got nil", tt.wantErr)
			case tt.wantErr != "" && !strings.Contains(err.Error(), tt.wantErr):
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestPolicyStatusFor(t *testing.T) {
	p := confidence.DefaultPolicy() // 0.8 / 0.5
	tests := []struct {
		score float64
		want  confidence.Status
	}{
		{1.0, confidence.StatusOK},
		{0.81, confidence.StatusOK},
		{0.8, confidence.StatusOK},
		{0.79, confidence.StatusUnsure},
		{0.5, confidence.StatusUnsure},
		{0.499, confidence.StatusFail},
		{0.0, confidence.StatusFail},
	}
	for _, tt := range tests {
		if got := p.StatusFor(tt.score); got != tt.want {
			t.Errorf("StatusFor(%v) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestComposeWeightedAverage(t *testing.T) {
	p := confidence.DefaultPolicy() // weights: self_check=0.4, consensus=0.4, constraint=0.2
	subs := map[string]float64{
		"self_check": 1.0,
		"consensus":  1.0,
		"constraint": 1.0,
	}
	got, err := p.Compose(subs)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("all-1.0 score = %v, want 1.0", got)
	}

	subs = map[string]float64{
		"self_check": 0.5,
		"consensus":  0.5,
		"constraint": 0.5,
	}
	got, err = p.Compose(subs)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if math.Abs(got-0.5) > 1e-9 {
		t.Errorf("all-0.5 score = %v, want 0.5", got)
	}

	// Mixed: 0.4*1.0 + 0.4*0.5 + 0.2*0.0 = 0.6
	subs = map[string]float64{
		"self_check": 1.0,
		"consensus":  0.5,
		"constraint": 0.0,
	}
	got, err = p.Compose(subs)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if math.Abs(got-0.6) > 1e-9 {
		t.Errorf("mixed score = %v, want 0.6", got)
	}
}

func TestComposeIgnoresUnknownStrategies(t *testing.T) {
	p := confidence.DefaultPolicy()
	subs := map[string]float64{
		"self_check": 1.0,
		"consensus":  1.0,
		"constraint": 1.0,
		"phantom":    0.0, // unknown — must be ignored, no panic
	}
	got, err := p.Compose(subs)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("score with unknown strategy = %v, want 1.0", got)
	}
}

func TestComposeMissingStrategyIsNeutral(t *testing.T) {
	p := confidence.DefaultPolicy()
	// Only self_check provided. Composer must use only the present
	// strategies' weights so a missing strategy doesn't drag score to 0.
	// Expected: score = self_check / (self_check_weight / sum_weights_present)
	// = 1.0 (since only self_check is present and its weight is 0.4 of 0.4)
	subs := map[string]float64{"self_check": 1.0}
	got, err := p.Compose(subs)
	if err != nil {
		t.Fatalf("Compose: %v", err)
	}
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("self_check-only score = %v, want 1.0", got)
	}
}

func TestComposeEmptyReturnsError(t *testing.T) {
	p := confidence.DefaultPolicy()
	if _, err := p.Compose(map[string]float64{}); err == nil {
		t.Errorf("expected error for empty subscores, got nil")
	}
	if _, err := p.Compose(nil); err == nil {
		t.Errorf("expected error for nil subscores, got nil")
	}
}

func TestComposeRejectsOutOfRange(t *testing.T) {
	p := confidence.DefaultPolicy()
	for _, bad := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		subs := map[string]float64{"self_check": bad}
		if _, err := p.Compose(subs); err == nil {
			t.Errorf("Compose(%v) expected error, got nil", bad)
		}
	}
}

func TestUnsureBehaviorConstants(t *testing.T) {
	cases := []struct {
		b    confidence.UnsureBehavior
		want string
	}{
		{confidence.UnsureRetryOnce, "retry_once"},
		{confidence.UnsureHumanHandoff, "human_handoff"},
		{confidence.UnsureConservativeFallback, "conservative_fallback"},
	}
	for _, c := range cases {
		if string(c.b) != c.want {
			t.Errorf("UnsureBehavior = %q, want %q", string(c.b), c.want)
		}
	}
}
