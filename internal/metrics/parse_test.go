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

	c, ok := parseOneCommit(block)
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

	c, ok := parseOneCommit(block)
	if !ok {
		t.Fatal("expected commit to parse")
	}
	if len(c.Files) != 5 {
		t.Fatalf("expected 5 files, got %d", len(c.Files))
	}
}
