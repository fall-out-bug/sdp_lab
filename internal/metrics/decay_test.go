package metrics

import (
	"testing"
	"time"
)

func TestAnalyzeDecayNil(t *testing.T) {
	if AnalyzeDecay(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeDecayEmpty(t *testing.T) {
	if AnalyzeDecay(&GitData{}) != nil {
		t.Fatal("expected nil for empty commits")
	}
}

func TestAnalyzeDecayShotgunSurgery(t *testing.T) {
	now := time.Now()
	// Shotgun: 5+ files, 3+ dirs, avg lines <= 20
	files := []FileChange{
		{Path: "a/f1.go", Added: 5, Deleted: 3},
		{Path: "b/f2.go", Added: 4, Deleted: 2},
		{Path: "c/f3.go", Added: 6, Deleted: 1},
		{Path: "d/f4.go", Added: 3, Deleted: 4},
		{Path: "e/f5.go", Added: 7, Deleted: 2},
	}
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "refactor: touch many files", Date: now, Files: files},
			{Subject: "feat: normal commit", Date: now, Files: []FileChange{{Path: "main.go", Added: 50}}},
		},
	}
	d := AnalyzeDecay(data)
	if d.ShotgunCommits != 1 {
		t.Fatalf("expected 1 shotgun commit got %d", d.ShotgunCommits)
	}
	if d.ShotgunSurgeryRatio != 0.5 {
		t.Fatalf("expected shotgun_ratio=0.5 got %.2f", d.ShotgunSurgeryRatio)
	}
}

func TestAnalyzeDecayShotgunBelowThreshold(t *testing.T) {
	now := time.Now()
	// Only 2 files → not shotgun
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "small change", Date: now, Files: []FileChange{
				{Path: "a/f1.go", Added: 5},
				{Path: "b/f2.go", Added: 3},
			}},
		},
	}
	d := AnalyzeDecay(data)
	if d.ShotgunCommits != 0 {
		t.Fatalf("expected 0 shotgun commits got %d", d.ShotgunCommits)
	}
}

func TestAnalyzeDecayFixRecurrence(t *testing.T) {
	now := time.Now()
	// 6 fix commits to same file → should appear in fix_recurrence
	var commits []RawCommit
	for i := 0; i < 6; i++ {
		commits = append(commits, RawCommit{
			Subject: "fix: bug in handler",
			Date:    now.Add(time.Duration(i) * time.Hour),
			Files:   []FileChange{{Path: "handler.go", Added: 5}},
		})
	}
	data := &GitData{Commits: commits}
	d := AnalyzeDecay(data)
	if len(d.FixRecurrence) != 1 {
		t.Fatalf("expected 1 fix recurrence got %d", len(d.FixRecurrence))
	}
	fr := d.FixRecurrence[0]
	if fr.Path != "handler.go" {
		t.Fatalf("expected handler.go got %s", fr.Path)
	}
	if fr.FixCount != 6 {
		t.Fatalf("expected 6 fixes got %d", fr.FixCount)
	}
	if fr.FixDensity != 1.0 {
		t.Fatalf("expected density 1.0 got %.2f", fr.FixDensity)
	}
}

func TestAnalyzeDecayFixRecurrenceBelowThreshold(t *testing.T) {
	now := time.Now()
	// Only 4 fix commits → below threshold of 5
	var commits []RawCommit
	for i := 0; i < 4; i++ {
		commits = append(commits, RawCommit{
			Subject: "fix: minor",
			Date:    now,
			Files:   []FileChange{{Path: "util.go", Added: 2}},
		})
	}
	data := &GitData{Commits: commits}
	d := AnalyzeDecay(data)
	if len(d.FixRecurrence) != 0 {
		t.Fatalf("expected 0 fix recurrence got %d", len(d.FixRecurrence))
	}
}

func TestAnalyzeDecayMonotonicGrowth(t *testing.T) {
	// File growing for 8+ months without refactoring
	start := time.Now().AddDate(0, -8, 0)
	commits := []RawCommit{
		{Subject: "add feature", Date: start, Files: []FileChange{{Path: "big.go", Added: 100, Deleted: 0}}},
		{Subject: "add more", Date: start.AddDate(0, 3, 0), Files: []FileChange{{Path: "big.go", Added: 50, Deleted: 5}}},
		{Subject: "add even more", Date: start.AddDate(0, 6, 0), Files: []FileChange{{Path: "big.go", Added: 80, Deleted: 2}}},
		{Subject: "latest", Date: time.Now(), Files: []FileChange{{Path: "big.go", Added: 30, Deleted: 1}}},
	}
	data := &GitData{Commits: commits}
	d := AnalyzeDecay(data)
	if len(d.MonotonicGrowthFiles) != 1 {
		t.Fatalf("expected 1 monotonic file got %d", len(d.MonotonicGrowthFiles))
	}
	mf := d.MonotonicGrowthFiles[0]
	if mf.Path != "big.go" {
		t.Fatalf("expected big.go got %s", mf.Path)
	}
	if !mf.ZeroRefactor {
		t.Fatal("expected zero_refactor=true")
	}
}

func TestAnalyzeDecayMonotonicWithRefactor(t *testing.T) {
	// File with refactoring event → should NOT be flagged as monotonic
	start := time.Now().AddDate(0, -8, 0)
	commits := []RawCommit{
		{Subject: "add feature", Date: start, Files: []FileChange{{Path: "refactored.go", Added: 100, Deleted: 0}}},
		{Subject: "refactor", Date: start.AddDate(0, 4, 0), Files: []FileChange{{Path: "refactored.go", Added: 40, Deleted: 35}}},
		{Subject: "latest", Date: time.Now(), Files: []FileChange{{Path: "refactored.go", Added: 50, Deleted: 5}}},
	}
	data := &GitData{Commits: commits}
	d := AnalyzeDecay(data)
	for _, mf := range d.MonotonicGrowthFiles {
		if mf.Path == "refactored.go" {
			t.Fatal("refactored.go should not be flagged as monotonic")
		}
	}
}

func TestDirOf(t *testing.T) {
	tests := []struct {
		path, want string
	}{
		{"a/b.go", "a"},
		{"b.go", "."},
		{"x/y/z.go", "x"},
	}
	for _, tt := range tests {
		got := dirOf(tt.path)
		if got != tt.want {
			t.Errorf("dirOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}
