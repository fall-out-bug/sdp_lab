package metrics

import (
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
