// sdp-cascade-replay drives the F145 CascadingInvoker against a corpus of test
// prompts and generates a report with tier_used and hops metrics.
//
// Usage:
//
//	sdp-cascade-replay -corpus testdata/cascade_corpus.json [--threshold 0.7] [--max-depth 3]
//
// The report includes:
//   - Tier distribution: count and percentage of each tier (local, fast, balanced, strong)
//   - Cause distribution: count of ok, checker_failed, max_depth, budget
//   - Cascade efficiency: % stayed-cheap (TierLocal/TierFast + hops==1), % escalated (hops > 1)
//
// Exit codes:
//
//	0  successful run
//	1  cascade or report failures (e.g., checker rejections)
//	2  internal error (corpus missing, malformed JSON, invalid parameters)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/cascade"
)

// CorpusCase represents a single case in the cascade corpus JSON.
type CorpusCase struct {
	ID           string `json:"id"`
	Prompt       string `json:"prompt"`
	ExpectedTier string `json:"expected_tier"`
	ExpectedHops int    `json:"expected_hops"`
	Category     string `json:"category,omitempty"`
}

func main() {
	corpusPath := flag.String("corpus", "testdata/cascade_corpus.json", "path to cascade corpus JSON")
	maxDepth := flag.Int("max-depth", 3, "maximum cascade depth")
	flag.Parse()

	ctx := context.Background()

	// Load corpus
	corpusData, err := os.ReadFile(*corpusPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading corpus: %v\n", err)
		os.Exit(2)
	}

	var cases []CorpusCase
	if err := json.Unmarshal(corpusData, &cases); err != nil {
		fmt.Fprintf(os.Stderr, "error parsing corpus JSON: %v\n", err)
		os.Exit(2)
	}

	// Convert to cascade.ReplayCorpus
	replayCases := make([]cascade.ReplayCase, len(cases))
	for i, c := range cases {
		replayCases[i] = cascade.ReplayCase{
			ID:           c.ID,
			Prompt:       c.Prompt,
			ExpectedTier: dispatch.TierClass(c.ExpectedTier),
			ExpectedHops: c.ExpectedHops,
		}
	}

	corpus := cascade.ReplayCorpus{Cases: replayCases}

	// Create invoker with stub router (no real provider invocation)
	// TODO (F145-14): wire in configurable max-depth once CascadingInvoker exposes setter
	invoker := cascade.NewInvoker(nil, nil)
	_ = maxDepth // unused for now

	runner := cascade.NewReplayRunner(invoker)

	// Run replay
	report, err := runner.Run(ctx, corpus)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error running cascade replay: %v\n", err)
		os.Exit(2)
	}

	// Render report
	renderReport(report, *corpusPath)
}

func renderReport(report *cascade.ReplayReport, corpusPath string) {
	fmt.Printf("Cascade Replay Report (corpus: %s, %d cases)\n\n", corpusPath, report.TotalCases)

	// Tier distribution
	fmt.Println("Tier distribution:")
	tiers := []string{"local", "fast", "balanced", "strong"}
	for _, tier := range tiers {
		count := report.TierUsedDist[tier]
		pct := 0.0
		if report.TotalCases > 0 {
			pct = 100.0 * float64(count) / float64(report.TotalCases)
		}
		fmt.Printf("  %s: %3d (%.1f%%)\n", tier, count, pct)
	}

	fmt.Println()

	// Cause distribution
	fmt.Println("Cause distribution:")
	causes := []string{"ok", "checker_failed", "max_depth", "budget"}
	for _, cause := range causes {
		count := report.CauseDist[cause]
		pct := 0.0
		if report.TotalCases > 0 {
			pct = 100.0 * float64(count) / float64(report.TotalCases)
		}
		fmt.Printf("  %s: %3d (%.1f%%)\n", cause, count, pct)
	}

	fmt.Println()

	// Cascade efficiency
	fmt.Println("Cascade efficiency:")
	fmt.Printf("  stayed-cheap:  %6.1f%%   (TierLocal/TierFast + hops==1)\n", report.StayedCheapPct)
	fmt.Printf("  escalated:     %6.1f%%   (hops > 1)\n", report.EscalatedPct)
}
