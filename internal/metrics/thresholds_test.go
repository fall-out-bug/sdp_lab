package metrics

import (
	"testing"
	"time"
)

// ── Stabilization tests ──

func TestAnalyzeStabilizationNil(t *testing.T) {
	if AnalyzeStabilization(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeStabilizationEmpty(t *testing.T) {
	s := AnalyzeStabilization(&GitData{})
	if s == nil {
		t.Fatal("expected non-nil for empty data")
	}
}

func TestAnalyzeStabilizationMultiplePatches(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: now.Add(-60 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.1", Date: now.Add(-50 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.2", Date: now.Add(-40 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.3", Date: now.Add(-30 * 24 * time.Hour), IsSemver: true},
		},
		Commits: []RawCommit{
			// Fixes after v1.0.0 → 2 fixes → not stable
			{Subject: "fix: bug1", Date: now.Add(-58 * 24 * time.Hour)},
			{Subject: "fix: bug2", Date: now.Add(-57 * 24 * time.Hour)},
			// Fixes after v1.0.1 → 0 fixes → stable at patch 2
			// No fixes after v1.0.2 or v1.0.3
		},
	}
	s := AnalyzeStabilization(data)
	if len(s.Releases) != 1 {
		t.Fatalf("expected 1 release line got %d", len(s.Releases))
	}
	ri := s.Releases[0]
	if ri.Base != "1.0" {
		t.Fatalf("expected base 1.0 got %s", ri.Base)
	}
	if ri.PatchesTotal != 4 {
		t.Fatalf("expected 4 patches got %d", ri.PatchesTotal)
	}
	if ri.StabilizedAtPatch != 2 {
		t.Fatalf("expected stabilized at patch 2 got %d", ri.StabilizedAtPatch)
	}
}

func TestAnalyzeStabilizationTrend(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: now.Add(-120 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.1", Date: now.Add(-110 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.2", Date: now.Add(-100 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.3", Date: now.Add(-90 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.0.4", Date: now.Add(-80 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.1.0", Date: now.Add(-70 * 24 * time.Hour), IsSemver: true},
			{Tag: "v1.1.1", Date: now.Add(-60 * 24 * time.Hour), IsSemver: true},
		},
		Commits: []RawCommit{},
	}
	s := AnalyzeStabilization(data)
	if s.Trend == "unknown" && len(s.Releases) >= 2 {
		t.Fatalf("expected a trend with %d release lines got %q", len(s.Releases), s.Trend)
	}
}

// ── Traffic-light threshold tests ──

func TestRateTicketLinkedRatio(t *testing.T) {
	tests := []struct {
		val  float64
		want TrafficLight
	}{
		{0.8, Green},
		{0.7, Yellow},  // >0.7 is green, 0.7 is not >0.7
		{0.5, Yellow},
		{0.3, Red},
	}
	for _, tt := range tests {
		got := RateTicketLinkedRatio(tt.val)
		if got != tt.want {
			t.Errorf("RateTicketLinkedRatio(%.1f) = %s, want %s", tt.val, got, tt.want)
		}
	}
}

func TestRateFixToFeature(t *testing.T) {
	tests := []struct {
		val  float64
		want TrafficLight
	}{
		{0.1, Green},
		{0.4, Yellow},
		{0.6, Red},
	}
	for _, tt := range tests {
		got := RateFixToFeature(tt.val)
		if got != tt.want {
			t.Errorf("RateFixToFeature(%.1f) = %s, want %s", tt.val, got, tt.want)
		}
	}
}

func TestRateChurnRatio(t *testing.T) {
	if RateChurnRatio(0.1) != Green {
		t.Error("0.1 should be green")
	}
	if RateChurnRatio(0.2) != Yellow {
		t.Error("0.2 should be yellow")
	}
	if RateChurnRatio(0.3) != Red {
		t.Error("0.3 should be red")
	}
}

func TestRateRevertRate(t *testing.T) {
	if RateRevertRate(0.005) != Green {
		t.Error("0.005 should be green")
	}
	if RateRevertRate(0.02) != Yellow {
		t.Error("0.02 should be yellow")
	}
	if RateRevertRate(0.04) != Red {
		t.Error("0.04 should be red")
	}
}

func TestRateBusFactor(t *testing.T) {
	if RateBusFactor(4) != Green {
		t.Error("4 should be green")
	}
	if RateBusFactor(2) != Yellow {
		t.Error("2 should be yellow")
	}
	if RateBusFactor(1) != Red {
		t.Error("1 should be red")
	}
}

func TestRateShotgunRatio(t *testing.T) {
	if RateShotgunRatio(0.01) != Green {
		t.Error("0.01 should be green")
	}
	if RateShotgunRatio(0.03) != Yellow {
		t.Error("0.03 should be yellow")
	}
	if RateShotgunRatio(0.06) != Red {
		t.Error("0.06 should be red")
	}
}

func TestRateConventionalCommitsRatio(t *testing.T) {
		tests := []struct {
			val  float64
			want TrafficLight
		}{
			{0.8, Green},
			{0.7, Yellow}, // >0.7 is green, 0.7 is not >0.7
			{0.5, Yellow},
			{0.3, Red},
		}
		for _, tt := range tests {
			got := RateConventionalCommitsRatio(tt.val)
			if got != tt.want {
				t.Errorf("RateConventionalCommitsRatio(%.1f) = %s, want %s", tt.val, got, tt.want)
			}
		}
	}

	func TestSemverBase(t *testing.T) {
	tests := []struct {
		tag, want string
	}{
		{"v1.18.3", "1.18"},
		{"v2.0.0", "2.0"},
		{"1.5.2", "1.5"},
		{"not-semver", ""},
	}
	for _, tt := range tests {
		got := semverBase(tt.tag)
		if got != tt.want {
			t.Errorf("semverBase(%q) = %q, want %q", tt.tag, got, tt.want)
		}
	}
}
