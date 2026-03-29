package dispatch

import (
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
)

// BenchResult holds the outcome metrics for a single benchmark run.
type BenchResult struct {
	Harness      string
	Provider     string
	Model        string
	Task         string
	TaskType     string
	Language     string
	Duration     time.Duration
	ExitCode     int
	TimedOut     bool
	FilesChanged int
	LinesAdded   int
	LinesRemoved int
	TestsTotal   int
	TestsPassed  int
	TestsFailed  int
	Commits      int
	PromptTokens int
	Cost         float64
	Timestamp    time.Time
}

const maxBenchMinutes = 15.0

// BenchScore computes a weighted score for a benchmark result.
//
// Formula: tests×0.5 + time×0.3 + commits×0.2
//
//   - testScore  = TestsPassed/TestsTotal (0 if TestsTotal == 0)
//   - timeScore  = 1.0 - min(Duration.Minutes()/15.0, 1.0)
//   - commitScore = min(Commits/5.0, 1.0)
func BenchScore(r BenchResult) float64 {
	var testScore float64
	if r.TestsTotal > 0 {
		testScore = float64(r.TestsPassed) / float64(r.TestsTotal)
	}

	timeScore := 1.0 - math.Min(r.Duration.Minutes()/maxBenchMinutes, 1.0)
	commitScore := math.Min(float64(r.Commits)/5.0, 1.0)

	return testScore*0.5 + timeScore*0.3 + commitScore*0.2
}

// RankBenchResults returns a copy of results sorted by BenchScore descending.
func RankBenchResults(results []BenchResult) []BenchResult {
	ranked := make([]BenchResult, len(results))
	copy(ranked, results)
	slices.SortFunc(ranked, func(a, b BenchResult) int {
		return cmp.Compare(BenchScore(b), BenchScore(a))
	})
	return ranked
}

// FormatCompareTable formats a comparison table for the given results with the
// top-scoring entry marked as WINNER.
func FormatCompareTable(results []BenchResult) string {
	if len(results) == 0 {
		return ""
	}

	ranked := RankBenchResults(results)
	winnerHarness := ranked[0].Harness

	var sb strings.Builder
	header := fmt.Sprintf("%-14s %-12s %-8s %6s %8s %7s %s\n",
		"HARNESS", "MODEL", "LANG", "SCORE", "DURATION", "COMMITS", "")
	sb.WriteString(header)
	sb.WriteString(strings.Repeat("-", len(header)) + "\n")

	for _, r := range ranked {
		marker := ""
		if r.Harness == winnerHarness {
			marker = "WINNER"
		}
		score := BenchScore(r)
		sb.WriteString(fmt.Sprintf("%-14s %-12s %-8s %6.3f %8s %7d %s\n",
			r.Harness,
			r.Model,
			r.Language,
			score,
			r.Duration.Round(time.Second).String(),
			r.Commits,
			marker,
		))
	}
	return sb.String()
}
