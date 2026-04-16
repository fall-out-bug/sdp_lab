package metrics

import (
	"testing"
	"time"
)

func TestAnalyzeWasteNil(t *testing.T) {
	if AnalyzeWaste(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeWasteEmpty(t *testing.T) {
	if AnalyzeWaste(&GitData{}) != nil {
		t.Fatal("expected nil for empty commits")
	}
}

func TestAnalyzeWasteChurnRatio(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Subject: "add feature",
				Date:    time.Now(),
				Files: []FileChange{
					{Path: "a.go", Added: 100, Deleted: 0},
					{Path: "b.go", Added: 50, Deleted: 0},
				},
			},
			{
				Subject: "refactor",
				Date:    time.Now(),
				Files: []FileChange{
					{Path: "a.go", Added: 80, Deleted: 90},
					{Path: "b.go", Added: 20, Deleted: 30},
				},
			},
		},
	}
	w := AnalyzeWaste(data)
	// Churn: min(180, 90) for a.go = 90, min(70, 30) for b.go = 30 → total churn = 120
	// Total added = 100+50+80+20 = 250
	// ChurnRatio = 120/250
	if w.ChurnRatio != 120.0/250.0 {
		t.Fatalf("expected churn_ratio=%.4f got %.4f", 120.0/250.0, w.ChurnRatio)
	}
}

func TestAnalyzeWasteChurnTopFiles(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{
				Subject: "c1",
				Date:    time.Now(),
				Files: []FileChange{
					{Path: "high.go", Added: 200, Deleted: 150},
					{Path: "low.go", Added: 10, Deleted: 5},
				},
			},
		},
	}
	w := AnalyzeWaste(data)
	if len(w.ChurnFilesTop) < 1 {
		t.Fatal("expected at least 1 churn file")
	}
	if w.ChurnFilesTop[0].Path != "high.go" {
		t.Fatalf("expected top churn file 'high.go' got %q", w.ChurnFilesTop[0].Path)
	}
}

func TestAnalyzeWasteRevertCount(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "Revert bad commit", Date: time.Now()},
			{Subject: "normal commit", Date: time.Now()},
			{Subject: "Rollback migration", Date: time.Now()},
			{Subject: "feat: add thing", Date: time.Now()},
		},
	}
	w := AnalyzeWaste(data)
	if w.RevertCount != 2 {
		t.Fatalf("expected 2 reverts got %d", w.RevertCount)
	}
	if w.RevertRate != 0.5 {
		t.Fatalf("expected revert_rate=0.5 got %.2f", w.RevertRate)
	}
}

func TestAnalyzeWasteAbandonedBranches(t *testing.T) {
	old := time.Now().AddDate(0, 0, -60)
	recent := time.Now().AddDate(0, 0, -5)
	data := &GitData{
		Commits: []RawCommit{{Subject: "x", Date: time.Now(), Files: []FileChange{{Path: "f.go", Added: 1}}}},
		Branches: []BranchInfo{
			{Name: "feature/old", LastCommit: &old},
			{Name: "feature/recent", LastCommit: &recent},
			{Name: "feature/ancient", LastCommit: &old},
		},
	}
	w := AnalyzeWaste(data)
	if w.AbandonedBranches != 2 {
		t.Fatalf("expected 2 abandoned branches got %d", w.AbandonedBranches)
	}
}

// WS-13: AbandonedLinesEst must be populated
func TestAnalyzeWasteAbandonedLinesEst(t *testing.T) {
	old := time.Now().AddDate(0, 0, -60)
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "x", Date: time.Now(), Files: []FileChange{{Path: "f.go", Added: 1}}},
			{Subject: "y", Date: old, Files: []FileChange{{Path: "f.go", Added: 100, Deleted: 10}}},
		},
		Branches: []BranchInfo{
			{Name: "feature/old", LastCommit: &old},
		},
	}
	w := AnalyzeWaste(data)
	if w.AbandonedLinesEst <= 0 {
		t.Errorf("AbandonedLinesEst = %d, want > 0 when abandoned branches exist", w.AbandonedLinesEst)
	}
}
