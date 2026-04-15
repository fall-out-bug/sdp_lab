package metrics

import (
	"context"
	"testing"
	"time"
)

func TestAnalyzeGitFlowNil(t *testing.T) {
	if AnalyzeGitFlow(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeGitFlowTrunkBased(t *testing.T) {
	now := time.Now()
	shortAgo := now.Add(-2 * 24 * time.Hour)
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "merge feature-a", Date: now},
			{Subject: "merge feature-b", Date: now.Add(-time.Hour)},
		},
		MergeCount: 50,
		Branches: []BranchInfo{
			{Name: "main"},
			{Name: "feature/a", LastCommit: &shortAgo},
			{Name: "feature/b", LastCommit: &shortAgo},
		},
	}
	gf := AnalyzeGitFlow(data)
	if gf.DetectedModel != "trunk-based" {
		t.Fatalf("expected trunk-based got %q", gf.DetectedModel)
	}
	if gf.MergeFrequencyPerWeek <= 0 {
		t.Fatal("expected positive merge frequency")
	}
}

func TestAnalyzeGitFlowGitFlow(t *testing.T) {
	now := time.Now()
	longAgo := now.Add(-30 * 24 * time.Hour)
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "merge", Date: now},
			{Subject: "merge", Date: now.Add(-24 * time.Hour)},
		},
		MergeCount: 5,
		Branches: []BranchInfo{
			{Name: "main"},
			{Name: "develop", LastCommit: &longAgo},
			{Name: "release/1.0", LastCommit: &longAgo},
			{Name: "hotfix/urgent", LastCommit: &longAgo},
		},
	}
	gf := AnalyzeGitFlow(data)
	if gf.DetectedModel != "gitflow" {
		t.Fatalf("expected gitflow got %q (score evidence: %v)", gf.DetectedModel, gf.Evidence)
	}
}

func TestAnalyzeGitFlowMergeFrequency(t *testing.T) {
	now := time.Now()
	weekAgo := now.Add(-7 * 24 * time.Hour)
	data := &GitData{
		Commits:    []RawCommit{{Subject: "c1", Date: now}, {Subject: "c2", Date: weekAgo}},
		MergeCount: 21,
		Branches:   []BranchInfo{{Name: "main"}},
	}
	gf := AnalyzeGitFlow(data)
	if gf.MergeFrequencyPerWeek != 21.0 {
		t.Fatalf("expected 21.0 merges/week got %.1f", gf.MergeFrequencyPerWeek)
	}
}

func TestAnalyzeGitFlowEvidence(t *testing.T) {
	now := time.Now()
	shortAgo := now.Add(-1 * 24 * time.Hour)
	data := &GitData{
		Commits:    []RawCommit{{Subject: "c1", Date: now}},
		MergeCount: 1,
		Branches: []BranchInfo{
			{Name: "feature/a", LastCommit: &shortAgo},
		},
	}
	gf := AnalyzeGitFlow(data)
	if len(gf.Evidence) == 0 {
		t.Fatal("expected some evidence")
	}
}

func TestAnalyzeGitFlowBranchLifetimes(t *testing.T) {
	now := time.Now()
	threeDaysAgo := now.Add(-3 * 24 * time.Hour)
	data := &GitData{
		Commits:    []RawCommit{{Subject: "c1", Date: now}},
		MergeCount: 1,
		Branches: []BranchInfo{
			{Name: "feature/a", LastCommit: &threeDaysAgo},
			{Name: "feature/b", LastCommit: &threeDaysAgo},
		},
	}
	gf := AnalyzeGitFlow(data)
	if gf.BranchLifetimeMedianH <= 0 {
		t.Fatal("expected positive median lifetime")
	}
}

// ── WS-07: P95 Percentile Fix ──────────────────────────────────────

func TestPercentileCorrect(t *testing.T) {
	tests := []struct {
		name string
		vals []float64
		p    float64
		want float64
	}{
		{
			name: "10 values P95 returns max",
			vals: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 100},
			p:    95,
			want: 100,
		},
		{
			name: "20 values P95 returns 19th",
			vals: []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 200},
			p:    95,
			want: 200,
		},
		{
			name: "100 values P95 returns 95th",
			vals: func() []float64 {
				v := make([]float64, 100)
				for i := range v {
					v[i] = float64(i + 1)
				}
				return v
			}(),
			p:    95,
			want: 96,
		},
		{
			name: "single value P95 returns that value",
			vals: []float64{42},
			p:    95,
			want: 42,
		},
		{
			name: "empty returns 0",
			vals: []float64{},
			p:    95,
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := percentile(tt.vals, tt.p)
			if got != tt.want {
				t.Errorf("percentile(%v, %v) = %v, want %v", tt.vals, tt.p, got, tt.want)
			}
		})
	}
}

// ── WS-07: Release Window Boundary Fix ────────────────────────────

func TestReleaseQualityNoDoubleCountAtBoundary(t *testing.T) {
	// Tag v1.0.0 at day 0, v1.1.0 at day 10.
	// Commit at exactly day 10 (same time as v1.1.0 tag).
	// This commit should appear in v1.1.0's window only, NOT in v1.0.0's window.
	tagDate1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	tagDate2 := time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC)
	boundaryCommit := time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC)

	data := &GitData{
		Commits: []RawCommit{
			{Subject: "fix: boundary bug", Date: boundaryCommit},
		},
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: tagDate1, IsSemver: true},
			{Tag: "v1.1.0", Date: tagDate2, IsSemver: true},
		},
	}
	rq := AnalyzeReleaseQuality(data)
	if rq == nil {
		t.Fatal("expected non-nil result")
	}
	if len(rq.Releases) < 2 {
		t.Fatalf("expected >= 2 releases, got %d", len(rq.Releases))
	}
	// v1.0.0 release should NOT count the boundary commit
	if rq.Releases[0].Fixes7d > 0 {
		t.Errorf("v1.0.0 Fixes7d = %d, want 0 (boundary commit should not be double-counted)", rq.Releases[0].Fixes7d)
	}
	// v1.1.0 release SHOULD count the boundary commit
	if rq.Releases[1].Fixes7d != 1 {
		t.Errorf("v1.1.0 Fixes7d = %d, want 1 (boundary commit belongs to v1.1.0 window)", rq.Releases[1].Fixes7d)
	}
}

// ── WS-07: Error Propagation ──────────────────────────────────────

func TestCollectPropagatesGitErrors(t *testing.T) {
	// Use a valid git repo path but with a cancelled context to force git errors
	dir := createTempGitRepo(t)
	commitFile(t, dir, "main.go", "package main\n", "2026-04-01T10:00:00Z", "initial")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := CollectWithContext(ctx, dir)
	if err == nil {
		t.Error("expected error when git commands fail due to cancelled context")
	}
}

func TestCollectNonGitDirPropagatesError(t *testing.T) {
	dir := t.TempDir()
	// Not a git repo — should fail fast
	_, err := Collect(dir)
	if err == nil {
		t.Error("expected error for non-git directory")
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		input []float64
		want  float64
	}{
		{[]float64{1, 2, 3}, 2},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{5}, 5},
		{[]float64{}, 0},
	}
	for _, tt := range tests {
		got := median(tt.input)
		if got != tt.want {
			t.Errorf("median(%v) = %.1f, want %.1f", tt.input, got, tt.want)
		}
	}
}
