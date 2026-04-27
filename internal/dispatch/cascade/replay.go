package cascade

import (
	"context"
	"fmt"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch"
)

// ReplayCase is a single test case in the cascade corpus.
type ReplayCase struct {
	ID           string                // unique case identifier
	Prompt       string                // the test prompt
	ExpectedTier dispatch.TierClass     // expected tier to use
	ExpectedHops int                    // expected cascade hops
}

// CaseResult is the outcome of a single replay case.
type CaseResult struct {
	CaseID   string                // from ReplayCase.ID
	TierUsed dispatch.TierClass     // tier that delivered the response
	Hops     int                    // cascade depth
	Cause    string                // "ok" | "max_depth" | "budget" | "checker_failed"
	Match    bool                  // true if TierUsed == ExpectedTier && Hops == ExpectedHops
}

// ReplayCorpus is a collection of test cases.
type ReplayCorpus struct {
	Cases []ReplayCase
}

// ReplayReport aggregates results from a cascade replay run.
type ReplayReport struct {
	TotalCases      int                  // total number of cases run
	TierUsedDist    map[string]int       // distribution of tier_used values (e.g. "TierLocal": 8)
	CauseDist       map[string]int       // distribution of cause values (e.g. "ok": 18)
	StayedCheapPct  float64              // percentage of cases that stayed on TierLocal/TierFast with hops==1
	EscalatedPct    float64              // percentage of cases with hops > 1
	Cases           []CaseResult         // per-case results
}

// ReplayRunner drives a CascadingInvoker over a corpus and produces aggregated metrics.
type ReplayRunner struct {
	invoker *CascadingInvoker
}

// NewReplayRunner creates a new ReplayRunner.
func NewReplayRunner(invoker *CascadingInvoker) *ReplayRunner {
	return &ReplayRunner{
		invoker: invoker,
	}
}

// Run executes the cascade replay over the given corpus and returns a ReplayReport.
func (r *ReplayRunner) Run(ctx context.Context, corpus ReplayCorpus) (*ReplayReport, error) {
	report := &ReplayReport{
		TotalCases:   len(corpus.Cases),
		TierUsedDist: make(map[string]int),
		CauseDist:    make(map[string]int),
		Cases:        make([]CaseResult, 0, len(corpus.Cases)),
	}

	// Handle empty corpus
	if len(corpus.Cases) == 0 {
		return report, nil
	}

	var stayedCheapCount, escalatedCount int

	for _, testCase := range corpus.Cases {
		// Create a request for this case
		req := InvokeRequest{
			Harness:   "test",
			Prompt:    testCase.Prompt,
			Agent:     "test",
			Worktree:  "",
			TaskFile:  "",
			Timeout:   0,
			StartTier: dispatch.TierLocal,
		}

		// Invoke cascade
		result, err := r.invoker.Invoke(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("invoke case %s: %w", testCase.ID, err)
		}

		// Record this case's result
		match := result.Tier == testCase.ExpectedTier && result.Hops == testCase.ExpectedHops
		caseResult := CaseResult{
			CaseID:   testCase.ID,
			TierUsed: result.Tier,
			Hops:     result.Hops,
			Cause:    result.Cause,
			Match:    match,
		}
		report.Cases = append(report.Cases, caseResult)

		// Update distributions
		tierStr := string(result.Tier)
		report.TierUsedDist[tierStr]++
		report.CauseDist[result.Cause]++

		// Check if stayed cheap: TierLocal or TierFast with hops==1
		if (result.Tier == dispatch.TierLocal || result.Tier == dispatch.TierFast) && result.Hops == 1 {
			stayedCheapCount++
		}

		// Check if escalated: hops > 1
		if result.Hops > 1 {
			escalatedCount++
		}
	}

	// Calculate percentages
	if report.TotalCases > 0 {
		report.StayedCheapPct = 100.0 * float64(stayedCheapCount) / float64(report.TotalCases)
		report.EscalatedPct = 100.0 * float64(escalatedCount) / float64(report.TotalCases)
	}

	return report, nil
}
