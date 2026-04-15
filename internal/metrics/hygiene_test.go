package metrics

import (
	"testing"
	"time"
)

func TestAnalyzeHygieneNil(t *testing.T) {
	if AnalyzeHygiene(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeHygieneEmpty(t *testing.T) {
	if AnalyzeHygiene(&GitData{}) != nil {
		t.Fatal("expected nil for empty commits")
	}
}

func TestAnalyzeHygieneTicketLinked(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "fix(api): resolve login bug #42", Date: time.Now()},
			{Subject: "feat: add user model", Date: time.Now()},
			{Subject: "chore: update deps closes #100", Date: time.Now()},
		},
	}
	h := AnalyzeHygiene(data)
	if h.TicketLinkedRatio != 2.0/3.0 {
		t.Fatalf("expected ticket_linked_ratio=%.4f got %.4f", 2.0/3.0, h.TicketLinkedRatio)
	}
}

func TestAnalyzeHygieneConventionalCommits(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "feat(auth): add login", Date: time.Now()},
			{Subject: "fix: patch leak", Date: time.Now()},
			{Subject: "random message", Date: time.Now()},
		},
	}
	h := AnalyzeHygiene(data)
	if h.ConventionalCommitsRatio != 2.0/3.0 {
		t.Fatalf("expected conventional_commits_ratio=%.4f got %.4f", 2.0/3.0, h.ConventionalCommitsRatio)
	}
	if h.CommitTypeBreakdown["feat"] != 1 {
		t.Fatalf("expected 1 feat, got %d", h.CommitTypeBreakdown["feat"])
	}
	if h.CommitTypeBreakdown["fix"] != 1 {
		t.Fatalf("expected 1 fix, got %d", h.CommitTypeBreakdown["fix"])
	}
	if h.CommitTypeBreakdown["other"] != 1 {
		t.Fatalf("expected 1 other, got %d", h.CommitTypeBreakdown["other"])
	}
}

func TestAnalyzeHygieneFixToFeatureRatio(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "fix: patch", Date: time.Now()},
			{Subject: "fix: patch2", Date: time.Now()},
			{Subject: "feat: new", Date: time.Now()},
		},
	}
	h := AnalyzeHygiene(data)
	if h.FixToFeatureRatio != 2.0/3.0 {
		t.Fatalf("expected fix_to_feature=%.4f got %.4f", 2.0/3.0, h.FixToFeatureRatio)
	}
}

func TestAnalyzeHygieneMessageLength(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "short", Date: time.Now()},
			{Subject: "a longer commit message here", Date: time.Now()},
		},
	}
	h := AnalyzeHygiene(data)
	expected := float64(5+28) / 2
	if h.AvgMessageLength != expected {
		t.Fatalf("expected avg_msg_len=%.1f got %.1f", expected, h.AvgMessageLength)
	}
}

func TestAnalyzeHygieneAvgFilesPerCommit(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "a", Date: time.Now(), Files: []FileChange{{Path: "f1.go"}, {Path: "f2.go"}}},
			{Subject: "b", Date: time.Now(), Files: []FileChange{{Path: "f3.go"}}},
		},
	}
	h := AnalyzeHygiene(data)
	if h.AvgFilesPerCommit != 1.5 {
		t.Fatalf("expected avg_files=1.5 got %.1f", h.AvgFilesPerCommit)
	}
}

func TestAnalyzeHygieneMonorepoStyle(t *testing.T) {
	files := make([]FileChange, 15)
	for i := range files {
		files[i] = FileChange{Path: "f.go"}
	}
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "big", Date: time.Now(), Files: files},
			{Subject: "small", Date: time.Now(), Files: []FileChange{{Path: "f.go"}}},
		},
	}
	h := AnalyzeHygiene(data)
	if h.MonorepoStyleRatio != 0.5 {
		t.Fatalf("expected monorepo_style=0.5 got %.2f", h.MonorepoStyleRatio)
	}
}

func TestAnalyzeHygieneJiraPattern(t *testing.T) {
	data := &GitData{
		Commits: []RawCommit{
			{Subject: "PROJ-123: implement feature", Date: time.Now()},
		},
	}
	h := AnalyzeHygiene(data)
	if h.TicketLinkedRatio != 1.0 {
		t.Fatalf("expected ticket_linked=1.0 got %.2f", h.TicketLinkedRatio)
	}
	found := false
	for _, p := range h.TicketPatternsFound {
		if p == "JIRA-style" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected JIRA-style pattern to be found")
	}
}

func TestParseConventionalType(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"feat(auth): add login", "feat"},
		{"fix: patch bug", "fix"},
		{"chore(deps): update", "chore"},
		{"random message", ""},
		{"Feat: uppercase type", "feat"},
		{"refactor(api)!: breaking change", "refactor"},
	}
	for _, tt := range tests {
		got := parseConventionalType(tt.subject)
		if got != tt.want {
			t.Errorf("parseConventionalType(%q) = %q, want %q", tt.subject, got, tt.want)
		}
	}
}
