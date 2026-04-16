package metrics

import (
	"strings"
	"testing"
)

// ── F121-3bb: bufio.Scanner buffer handling ─────────────────────────

func TestParseOneCommitLongNumstatLine(t *testing.T) {
	// Simulate a numstat line longer than 64KB (default bufio.Scanner limit)
	longPath := strings.Repeat("a", 100*1024) // 100KB path
	block := "abc123def456abc123def456abc123def456abc1\nAUTHOR:Alice\nDATE:2026-04-01T10:00:00Z\nSUBJECT:test\nBODY:\nNUMSTAT\n1\t2\t" + longPath + "\n"

	c, ok, _ := parseOneCommit(block)
	if !ok {
		t.Fatal("expected commit to parse despite long numstat line")
	}
	if len(c.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(c.Files))
	}
	if c.Files[0].Path != longPath {
		t.Errorf("path truncated or wrong: got len=%d, want len=%d", len(c.Files[0].Path), len(longPath))
	}
}

func TestParseOneCommitMultipleLongLines(t *testing.T) {
	// Multiple long numstat lines
	var numstatLines string
	for i := 0; i < 5; i++ {
		longPath := strings.Repeat("x", 80*1024)
		numstatLines += "1\t0\t" + longPath + "\n"
	}
	block := "abc123def456abc123def456abc123def456abc1\nAUTHOR:Bob\nDATE:2026-04-01T10:00:00Z\nSUBJECT:big\nBODY:\nNUMSTAT\n" + numstatLines

	c, ok, _ := parseOneCommit(block)
	if !ok {
		t.Fatal("expected commit to parse")
	}
	if len(c.Files) != 5 {
		t.Fatalf("expected 5 files, got %d", len(c.Files))
	}
}

// ── WS-12: parseTags with dates (for-each-ref format) ──────────────

func TestParseTagsWithDates(t *testing.T) {
	raw := "v1.0.0 2026-04-01T10:00:00+00:00\nv1.1.0 2026-04-02T15:30:00+00:00\n"
	tags := parseTags(raw)
	if len(tags) != 2 {
		t.Fatalf("tags = %d, want 2", len(tags))
	}
	if tags[0].Tag != "v1.0.0" {
		t.Errorf("tags[0].Tag = %q, want v1.0.0", tags[0].Tag)
	}
	if tags[0].Date.IsZero() {
		t.Error("tags[0].Date is zero — parseTags must extract dates")
	}
	if tags[0].Date.Month() != 4 || tags[0].Date.Day() != 1 {
		t.Errorf("tags[0].Date = %v, want April 1", tags[0].Date)
	}
	if !tags[0].IsSemver {
		t.Error("tags[0].IsSemver = false, want true")
	}
	if tags[1].Tag != "v1.1.0" {
		t.Errorf("tags[1].Tag = %q, want v1.1.0", tags[1].Tag)
	}
	if tags[1].Date.IsZero() {
		t.Error("tags[1].Date is zero")
	}
}

func TestParseTagsNameOnly(t *testing.T) {
	// Legacy format: name only (no date)
	raw := "v1.0.0\nv1.1.0\n"
	tags := parseTags(raw)
	if len(tags) != 2 {
		t.Fatalf("tags = %d, want 2", len(tags))
	}
	if tags[0].Tag != "v1.0.0" {
		t.Errorf("tags[0].Tag = %q, want v1.0.0", tags[0].Tag)
	}
	if !tags[0].IsSemver {
		t.Error("tags[0].IsSemver = false, want true")
	}
	// Date will be zero for legacy format — acceptable fallback
}

// WS-13: parseCommits returns warning count for truncated data
func TestParseCommitsWarningCount(t *testing.T) {
	// Normal parse should return 0 warnings
	raw := gitDelim + "abc123def456abc123def456abc123def456abc1\nAUTHOR:Alice\nDATE:2026-04-01T10:00:00Z\nSUBJECT:test\nBODY:\nNUMSTAT\n"
	commits, warnings := parseCommits(raw)
	if warnings > 0 {
		t.Errorf("warnings = %d, want 0 for valid input", warnings)
	}
	if len(commits) != 1 {
		t.Fatalf("commits = %d, want 1", len(commits))
	}
}

// Multi-line body: BODY: prefix followed by continuation lines
func TestParseOneCommitMultiLineBody(t *testing.T) {
	block := "abc123def456abc123def456abc123def456abc1\nAUTHOR:Alice\nDATE:2026-04-01T10:00:00Z\nSUBJECT:test\nBODY:\nline one\nline two\nline three\nNUMSTAT\n"
	c, ok, _ := parseOneCommit(block)
	if !ok {
		t.Fatal("expected commit to parse")
	}
	want := "line one\nline two\nline three"
	if c.Body != want {
		t.Errorf("Body = %q, want %q", c.Body, want)
	}
}

func TestParseOneCommitBodyWithTicketRef(t *testing.T) {
	block := "abc123def456abc123def456abc123def456abc1\nAUTHOR:Bob\nDATE:2026-04-01T10:00:00Z\nSUBJECT:feat: add feature\nBODY:\nThis adds the new feature.\nFixes #42\nSee PROJ-123 for details.\nNUMSTAT\n1\t2\tmain.go\n"
	c, ok, _ := parseOneCommit(block)
	if !ok {
		t.Fatal("expected commit to parse")
	}
	if !strings.Contains(c.Body, "Fixes #42") {
		t.Errorf("Body should contain 'Fixes #42', got %q", c.Body)
	}
	if !strings.Contains(c.Body, "PROJ-123") {
		t.Errorf("Body should contain 'PROJ-123', got %q", c.Body)
	}
}
