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
	"strings"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/cascade"
	"sdp_dev/internal/dispatch/harness"
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
	// Wire a deterministic stub checker for demo purposes
	stubRouter := &stubReplayRouter{}
	stubChecker := &stubReplayChecker{}
	invoker := cascade.NewInvoker(stubRouter, stubChecker)

	// Propagate -max-depth flag to invoker
	if maxDepth != nil && *maxDepth > 0 {
		invoker.SetMaxDepth(*maxDepth)
	}

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

// stubReplayRouter is a minimal router for cascade replay (returns substantial canned response).
type stubReplayRouter struct {
	tierCounter map[string]int // tier -> invocation count for deterministic response
}

func (s *stubReplayRouter) Route(ctx context.Context, task dispatch.TaskClassification, limits map[string]*harness.Limits) (*dispatch.DispatchDecision, error) {
	if s.tierCounter == nil {
		s.tierCounter = make(map[string]int)
	}

	// Return a substantial canned response that clears MinLengthChars=50
	// and doesn't match refusal keywords, suitable for cascade evaluation.
	return &dispatch.DispatchDecision{
		Harness:  "stub",
		Model:    "synthetic-day-8",
		Provider: "cascade-replay-demo",
		Score:    0.95,
		// Note: Output field is synthesized in cascade.go Invoke() based on this decision.
		// This decision structure alone ensures cascade.go generates a response >= 50 chars.
	}, nil
}

// stubReplayChecker is a deterministic checker for demo purposes.
// It tracks hop counts per prompt and accepts based on category:
// - simple-coding-easy: accept on hop 1 (stays cheap)
// - complex-coding-hard: accept on hop 2 (escalates once)
// - refusal-bait: accept on hop 3 (escalates twice)
type stubReplayChecker struct {
	hopCounts map[string]int // prompt -> current hop
}

func (c *stubReplayChecker) Check(ctx context.Context, req cascade.InvokeRequest, resp *harness.Result) (ok bool, reason string) {
	if c.hopCounts == nil {
		c.hopCounts = make(map[string]int)
	}

	// Track which hop this is for this prompt
	c.hopCounts[req.Prompt]++
	hop := c.hopCounts[req.Prompt]

	// Categorize based on prompt content
	category := categorizePrompt(req.Prompt)

	switch category {
	case "simple":
		// Accept on first hop
		if hop == 1 {
			return true, "demo checker: accepted on simple-tier"
		}
		return false, "demo checker: rejected (escalate from simple)"
	case "complex":
		// Accept on second hop
		if hop <= 2 {
			return hop == 2, "demo checker: complex"
		}
		return false, "demo checker: rejected (escalate from complex)"
	case "refusal":
		// Accept on third hop
		if hop <= 3 {
			return hop == 3, "demo checker: refusal"
		}
		return false, "demo checker: rejected (escalate from refusal)"
	default:
		// Unknown: accept on second hop
		return hop <= 2, "demo checker: unknown"
	}
}

// categorizePrompt provides a simple heuristic for corpus categorization.
func categorizePrompt(prompt string) string {
	lower := strings.ToLower(prompt)

	// Check for complexity keywords
	if strings.Contains(lower, "distributed") ||
		strings.Contains(lower, "algorithm") ||
		strings.Contains(lower, "design pattern") ||
		strings.Contains(lower, "cache") ||
		strings.Contains(lower, "b-tree") ||
		strings.Contains(lower, "monad") ||
		strings.Contains(lower, "compiler") {
		return "complex"
	}

	// Check for refusal-bait keywords
	if strings.Contains(lower, "bypass") ||
		strings.Contains(lower, "scam") ||
		strings.Contains(lower, "illegal") ||
		strings.Contains(lower, "unethical") ||
		strings.Contains(lower, "explosive") ||
		strings.Contains(lower, "harm") {
		return "refusal"
	}

	// Check for nonsense
	if strings.Contains(lower, "xyzzy") ||
		strings.Contains(lower, "asdfghjkl") ||
		strings.Contains(lower, "plugh") {
		return "complex" // treat nonsense as requiring escalation
	}

	// Default: simple
	return "simple"
}
