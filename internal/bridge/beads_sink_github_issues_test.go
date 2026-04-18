package bridge

import (
	"context"
	"strings"
	"testing"
)

func TestSyncGitHubIssuesCreatesBeadsTasks(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  42,
			Title:   "F077: Fix broken CI pipeline",
			Body:    "The CI pipeline fails on protocol check step.",
			State:   "open",
			HTMLURL: "https://github.com/org/repo/issues/42",
			Labels: []GitHubLabel{
				{Name: "bug"},
			},
			User: &GitHubUser{Login: "contributor"},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Processed != 1 {
		t.Fatalf("expected 1 processed, got %d", stats.Processed)
	}
	if stats.Created != 1 {
		t.Fatalf("expected 1 created, got %d", stats.Created)
	}
}

func TestSyncGitHubIssuesExtractsFeatureID(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  10,
			Title:   "Fix auth module",
			HTMLURL: "https://github.com/org/repo/issues/10",
			State:   "open",
			Labels: []GitHubLabel{
				{Name: "feature/F077"},
			},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Created != 1 {
		t.Fatalf("expected 1 created, got %+v", stats)
	}
}

func TestSyncGitHubIssuesExtractsWSIDFromLabel(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  15,
			Title:   "Improve logging",
			HTMLURL: "https://github.com/org/repo/issues/15",
			State:   "open",
			Labels: []GitHubLabel{
				{Name: "00-077-02"},
			},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Created != 1 {
		t.Fatalf("expected 1 created, got %+v", stats)
	}
}

func TestSyncGitHubIssuesDeduplicatesAcrossRuns(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  99,
			Title:   "Fix build",
			HTMLURL: "https://github.com/org/repo/issues/99",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}

	stats1 := sink.GetStats()
	if stats1.Created != 1 {
		t.Fatalf("expected 1 created on first run, got %d", stats1.Created)
	}

	err = sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}

	stats2 := sink.GetStats()
	if stats2.Created != 1 {
		t.Fatalf("expected no additional creates on second run (dedup), got %d", stats2.Created)
	}
	if stats2.Skipped < 1 {
		t.Fatalf("expected at least 1 skipped on second run, got %d", stats2.Skipped)
	}
}

func TestSyncGitHubIssuesEmptyList(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	err := sink.SyncGitHubIssues(ctx, "org/repo", nil)
	if err != nil {
		t.Fatalf("SyncGitHubIssues on nil slice returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Processed != 0 {
		t.Fatalf("expected 0 processed, got %d", stats.Processed)
	}
}

func TestExtractFeatureIDFromLabels(t *testing.T) {
	tests := []struct {
		name     string
		issue    GitHubIssue
		expected string
	}{
		{
			name: "feature/ prefix label",
			issue: GitHubIssue{Labels: []GitHubLabel{{Name: "feature/F100"}}},
			expected: "F100",
		},
		{
			name: "bare F### label",
			issue: GitHubIssue{Labels: []GitHubLabel{{Name: "F100"}}},
			expected: "F100",
		},
		{
			name: "feature ID in title",
			issue: GitHubIssue{Title: "F200: do something", Labels: nil},
			expected: "F200",
		},
		{
			name: "no feature ID",
			issue: GitHubIssue{Title: "random issue", Labels: []GitHubLabel{{Name: "bug"}}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeatureID(&tt.issue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractWSIDFromLabels(t *testing.T) {
	tests := []struct {
		name     string
		issue    GitHubIssue
		expected string
	}{
		{
			name: "workstream/ prefix label",
			issue: GitHubIssue{Labels: []GitHubLabel{{Name: "workstream/00-077-02"}}},
			expected: "00-077-02",
		},
		{
			name: "bare ws ID label",
			issue: GitHubIssue{Labels: []GitHubLabel{{Name: "00-077-02"}}},
			expected: "00-077-02",
		},
		{
			name: "ws ID in title",
			issue: GitHubIssue{Title: "00-077-03: sync daemon fix", Labels: nil},
			expected: "00-077-03",
		},
		{
			name: "no ws ID",
			issue: GitHubIssue{Title: "random issue", Labels: []GitHubLabel{{Name: "bug"}}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWSID(&tt.issue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestSyncGitHubIssuesFailurePath(t *testing.T) {
	// Use dryRun=false so that CreateTypedFinding attempts the real code path.
	// A pre-cancelled context causes bd create to fail, exercising the error path.
	sink := NewBeadsSink("sdplab", false, nil)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately so any exec.CommandContext will fail

	issues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
		{
			Number:  2,
			Title:   "Bug: another issue",
			HTMLURL: "https://github.com/org/repo/issues/2",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues should not return error even when individual issues fail: %v", err)
	}

	stats := sink.GetStats()
	if stats.Processed != 2 {
		t.Fatalf("expected 2 processed, got %d", stats.Processed)
	}
	if stats.Failed != 2 {
		t.Fatalf("expected 2 failed, got %d (stats: %+v)", stats.Failed, stats)
	}
	// No issues should be created since both failed
	if stats.Created != 0 {
		t.Fatalf("expected 0 created, got %d", stats.Created)
	}
}

func TestSyncGitHubIssuesSkipsInfoSeverity(t *testing.T) {
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Documentation update",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "documentation"}},
		},
		{
			Number:  2,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/2",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Processed != 2 {
		t.Fatalf("expected 2 processed, got %d", stats.Processed)
	}
	// The documentation issue should be skipped (info severity), only the bug should be created
	if stats.Skipped != 1 {
		t.Fatalf("expected 1 skipped (info severity), got %d", stats.Skipped)
	}
	if stats.Created != 1 {
		t.Fatalf("expected 1 created, got %d", stats.Created)
	}
}

func TestExtractSeverityFromLabels(t *testing.T) {
	tests := []struct {
		name     string
		labels   []GitHubLabel
		expected string
	}{
		{name: "bug label", labels: []GitHubLabel{{Name: "bug"}}, expected: "error"},
		{name: "critical label", labels: []GitHubLabel{{Name: "critical"}}, expected: "error"},
		{name: "enhancement label", labels: []GitHubLabel{{Name: "enhancement"}}, expected: "warning"},
		{name: "documentation label", labels: []GitHubLabel{{Name: "documentation"}}, expected: "info"},
		{name: "no relevant label", labels: []GitHubLabel{{Name: "question"}}, expected: "warning"},
		{name: "no labels", labels: nil, expected: "warning"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := GitHubIssue{Labels: tt.labels}
			got := extractSeverityFromLabels(&issue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestBuildGitHubIssueDescription(t *testing.T) {
	issue := GitHubIssue{
		Number:  42,
		Title:   "Test issue",
		Body:    "This is the body.",
		State:   "open",
		HTMLURL: "https://github.com/org/repo/issues/42",
		Labels:  []GitHubLabel{{Name: "bug"}, {Name: "F077"}},
		User:    &GitHubUser{Login: "testuser"},
		Assignees: []GitHubUser{{Login: "dev1"}, {Login: "dev2"}},
		Milestone: &GitHubMilestone{Title: "v1.0"},
	}

	desc := buildGitHubIssueDescription(&issue, "org/repo")

	checks := []string{
		"**GitHub Issue:** https://github.com/org/repo/issues/42",
		"**Repository:** org/repo",
		"**State:** open",
		"**Author:** @testuser",
		"**Assignees:** @dev1, @dev2",
		"**Milestone:** v1.0",
		"**Labels:** bug, F077",
		"This is the body.",
	}

	for _, check := range checks {
		if !strings.Contains(desc, check) {
			t.Fatalf("expected description to contain %q, got:\n%s", check, desc)
		}
	}
}

// --- Issue 3: Closed GitHub issues reconciliation ---

func TestSyncGitHubIssuesClosedIssueSkipsIfNoMatch(t *testing.T) {
	// When a closed GitHub issue has no corresponding Beads issue, it should
	// be skipped (not created just to be closed).
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Closed bug",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "closed",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("SyncGitHubIssues returned error: %v", err)
	}

	stats := sink.GetStats()
	if stats.Processed != 1 {
		t.Fatalf("expected 1 processed, got %d", stats.Processed)
	}
	if stats.Skipped != 1 {
		t.Fatalf("expected 1 skipped (no matching Beads issue), got %d", stats.Skipped)
	}
	if stats.Closed != 0 {
		t.Fatalf("expected 0 closed, got %d", stats.Closed)
	}
	if stats.Created != 0 {
		t.Fatalf("expected 0 created, got %d", stats.Created)
	}
}

func TestSyncGitHubIssuesClosedIssueClosesExistingBeads(t *testing.T) {
	// First sync an open issue to create a Beads item, then sync it as closed.
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	// First sync: open issue -> creates a Beads issue.
	err := sink.SyncGitHubIssues(ctx, "org/repo", issues)
	if err != nil {
		t.Fatalf("first sync returned error: %v", err)
	}
	stats1 := sink.GetStats()
	if stats1.Created != 1 {
		t.Fatalf("expected 1 created on first run, got %d", stats1.Created)
	}

	// Second sync: same issue now closed -> should close the Beads issue.
	closedIssues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "closed",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	err = sink.SyncGitHubIssues(ctx, "org/repo", closedIssues)
	if err != nil {
		t.Fatalf("second sync returned error: %v", err)
	}
	stats2 := sink.GetStats()
	if stats2.Closed != 1 {
		t.Fatalf("expected 1 closed on second run, got %d", stats2.Closed)
	}
}

func TestSyncGitHubIssuesClosedIssueAlreadyClosedSkips(t *testing.T) {
	// If the Beads issue is already closed, syncing a closed GitHub issue
	// should skip it (no double-close).
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	issues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "open",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}

	// Create the Beads issue.
	_ = sink.SyncGitHubIssues(ctx, "org/repo", issues)

	// Close it.
	closedIssues := []GitHubIssue{
		{
			Number:  1,
			Title:   "Bug: something broke",
			HTMLURL: "https://github.com/org/repo/issues/1",
			State:   "closed",
			Labels:  []GitHubLabel{{Name: "bug"}},
		},
	}
	_ = sink.SyncGitHubIssues(ctx, "org/repo", closedIssues)

	// Sync closed again -> should skip (already closed).
	statsBefore := sink.GetStats()
	_ = sink.SyncGitHubIssues(ctx, "org/repo", closedIssues)
	statsAfter := sink.GetStats()

	// Closed count should not increase on third sync.
	if statsAfter.Closed != statsBefore.Closed {
		t.Fatalf("expected closed to remain %d, got %d", statsBefore.Closed, statsAfter.Closed)
	}
}

func TestSyncGitHubIssuesClosedWithTaggedIssueIsReconciled(t *testing.T) {
	// When an issue was created with FeatureID and WSID labels, closing it
	// must still find the original Beads issue.  The closed-issue path must
	// extract the same FeatureID/WSID to compute the same hash.
	sink := NewBeadsSink("sdplab", true, nil)
	ctx := t.Context()

	openIssue := []GitHubIssue{
		{
			Number:  55,
			Title:   "F077: Fix reconciliation bug",
			HTMLURL: "https://github.com/org/repo/issues/55",
			State:   "open",
			Labels: []GitHubLabel{
				{Name: "bug"},
				{Name: "feature/F077"},
				{Name: "00-077-02"},
			},
		},
	}

	// Create the Beads issue with tags.
	err := sink.SyncGitHubIssues(ctx, "org/repo", openIssue)
	if err != nil {
		t.Fatalf("open sync returned error: %v", err)
	}
	stats1 := sink.GetStats()
	if stats1.Created != 1 {
		t.Fatalf("expected 1 created, got %d", stats1.Created)
	}

	// Close the same issue (same labels so tags are extracted).
	closedIssue := []GitHubIssue{
		{
			Number:  55,
			Title:   "F077: Fix reconciliation bug",
			HTMLURL: "https://github.com/org/repo/issues/55",
			State:   "closed",
			Labels: []GitHubLabel{
				{Name: "bug"},
				{Name: "feature/F077"},
				{Name: "00-077-02"},
			},
		},
	}

	err = sink.SyncGitHubIssues(ctx, "org/repo", closedIssue)
	if err != nil {
		t.Fatalf("closed sync returned error: %v", err)
	}
	stats2 := sink.GetStats()
	if stats2.Closed != 1 {
		t.Fatalf("expected 1 closed, got %d (stats: %+v)", stats2.Closed, stats2)
	}
}

// --- Issue 4: Case-insensitive feature/WSID extraction ---

func TestExtractFeatureIDCaseInsensitiveLabel(t *testing.T) {
	tests := []struct {
		name     string
		issue    GitHubIssue
		expected string
	}{
		{
			name:     "mixed case Feature/ prefix",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "Feature/F077"}}},
			expected: "F077",
		},
		{
			name:     "uppercase FEATURE/ prefix",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "FEATURE/F100"}}},
			expected: "F100",
		},
		{
			name:     "lowercase feature/ prefix",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "feature/F200"}}},
			expected: "F200",
		},
		{
			name:     "bare lowercase f077 label",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "f077"}}},
			expected: "F077",
		},
		{
			name:     "bare uppercase F077 label (existing)",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "F077"}}},
			expected: "F077",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeatureID(&tt.issue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractFeatureIDFromTextCaseInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		expected string
	}{
		{
			name:     "bare lowercase f077 in title",
			text:     "f077: fix something",
			expected: "F077",
		},
		{
			name:     "bare uppercase F077 in title",
			text:     "F077: fix something",
			expected: "F077",
		},
		{
			name:     "no feature ID",
			text:     "random issue title",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeatureIDFromText(tt.text)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestExtractWSIDCaseInsensitiveLabel(t *testing.T) {
	tests := []struct {
		name     string
		issue    GitHubIssue
		expected string
	}{
		{
			name:     "mixed case Workstream/ prefix",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "Workstream/00-077-02"}}},
			expected: "00-077-02",
		},
		{
			name:     "uppercase WORKSTREAM/ prefix",
			issue:    GitHubIssue{Labels: []GitHubLabel{{Name: "WORKSTREAM/00-077-03"}}},
			expected: "00-077-03",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractWSID(&tt.issue)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

// --- Issue 5: isWSID rejects dates ---

func TestIsWSIDRejectsDates(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "valid WSID", input: "00-077-01", expected: true},
		{name: "valid WSID 2", input: "12-345-67", expected: true},
		{name: "date rejected (4-2-2 digits)", input: "2026-04-18", expected: false},
		{name: "date rejected (4-2-2 other)", input: "2025-12-31", expected: false},
		{name: "partial 1 digit", input: "0-077-01", expected: false},
		{name: "partial 4 digits in middle", input: "00-0770-01", expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "random text", input: "hello-world", expected: false},
		{name: "2-2-2 digits", input: "00-77-01", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isWSID(tt.input)
			if got != tt.expected {
				t.Fatalf("isWSID(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestExtractWSIDFromTextRejectsDate(t *testing.T) {
	// A title containing a date should not extract the date as a WSID.
	got := extractWSIDFromText("Scheduled for 2026-04-18 deployment")
	if got != "" {
		t.Fatalf("expected no WSID extracted from date text, got %q", got)
	}

	// But a valid WSID in a title should still work.
	got = extractWSIDFromText("00-077-03: sync daemon fix")
	if got != "00-077-03" {
		t.Fatalf("expected WSID 00-077-03, got %q", got)
	}
}
