package dispatch

import (
	"strings"
	"testing"
	"time"
)

func TestBenchScore_FixedWeights(t *testing.T) {
	// Perfect run: 10/10 tests, 2m duration, 5 commits — expect > 0.85
	perfect := BenchResult{
		TestsTotal:  10,
		TestsPassed: 10,
		Duration:    2 * time.Minute,
		Commits:     5,
	}
	score := BenchScore(perfect)
	if score <= 0.85 {
		t.Errorf("perfect run score = %.4f, want > 0.85", score)
	}

	// No tests: 0/0 tests, 3m duration, 2 commits — expect between 0.15 and 0.45
	noTests := BenchResult{
		TestsTotal:  0,
		TestsPassed: 0,
		Duration:    3 * time.Minute,
		Commits:     2,
	}
	scoreNoTests := BenchScore(noTests)
	if scoreNoTests < 0.15 || scoreNoTests > 0.45 {
		t.Errorf("no-tests run score = %.4f, want between 0.15 and 0.45", scoreNoTests)
	}

	// Slow run: 8/10 tests, 14m duration, 1 commit — expect between 0.35 and 0.55
	slow := BenchResult{
		TestsTotal:  10,
		TestsPassed: 8,
		Duration:    14 * time.Minute,
		Commits:     1,
	}
	scoreSlow := BenchScore(slow)
	if scoreSlow < 0.35 || scoreSlow > 0.55 {
		t.Errorf("slow run score = %.4f, want between 0.35 and 0.55", scoreSlow)
	}
}

func TestRankBenchResults(t *testing.T) {
	claude := BenchResult{
		Harness:     "claude",
		TestsTotal:  10,
		TestsPassed: 10,
		Duration:    3 * time.Minute,
		Commits:     4,
	}
	codex := BenchResult{
		Harness:     "codex",
		TestsTotal:  10,
		TestsPassed: 6,
		Duration:    5 * time.Minute,
		Commits:     2,
	}
	cursor := BenchResult{
		Harness:     "cursor",
		TestsTotal:  10,
		TestsPassed: 3,
		Duration:    8 * time.Minute,
		Commits:     1,
	}

	results := []BenchResult{codex, cursor, claude}
	ranked := RankBenchResults(results)

	if len(ranked) != 3 {
		t.Fatalf("expected 3 ranked results, got %d", len(ranked))
	}
	if ranked[0].Harness != "claude" {
		t.Errorf("expected claude to win, got %s", ranked[0].Harness)
	}
	if ranked[2].Harness != "cursor" {
		t.Errorf("expected cursor to be last, got %s", ranked[2].Harness)
	}

	// Input slice must not be mutated
	if results[0].Harness != "codex" {
		t.Error("RankBenchResults mutated input slice")
	}
}

func TestFormatCompareTable(t *testing.T) {
	results := []BenchResult{
		{Harness: "claude", TestsTotal: 10, TestsPassed: 10, Duration: 2 * time.Minute, Commits: 5},
		{Harness: "codex", TestsTotal: 10, TestsPassed: 5, Duration: 6 * time.Minute, Commits: 2},
	}
	table := FormatCompareTable(results)
	if !strings.Contains(table, "claude") {
		t.Error("table missing claude harness name")
	}
	if !strings.Contains(table, "WINNER") {
		t.Error("table missing WINNER marker")
	}
}
