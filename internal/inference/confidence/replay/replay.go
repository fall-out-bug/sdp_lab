// Package replay drives confidence checkers against fixture corpora and
// aggregates the results into evidence-bound metrics. The corpus structure
// is testdata/<call-site>/<category>/*.json where category is one of
// {correct, edge, adversarial}.
//
// Metrics tracked per category × call-site:
//   - rejection_rate (FAIL): adversarial → must be high (≥0.8); correct → must be low (≤0.02)
//   - retry_rate (UNSURE): % of UNSURE outcomes
//   - latency_ms p50 / p95
//   - tokens (in / out / total) summed per fixture, mean across fixtures
//
// The cost overhead is reported as an aggregate token count; converting to
// USD is left to telemetry pipelines that know the active provider's price.
package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"

	"sdp_dev/internal/inference/confidence"
)

// Category enumerates corpus partitions.
type Category string

const (
	Correct     Category = "correct"
	Edge        Category = "edge"
	Adversarial Category = "adversarial"
)

func AllCategories() []Category { return []Category{Correct, Edge, Adversarial} }

// FixtureResult is the per-fixture record emitted by Run.
type FixtureResult struct {
	Path      string             `json:"path"`
	Category  Category           `json:"category"`
	Status    confidence.Status  `json:"status"`
	Score     float64            `json:"score"`
	SubScores map[string]float64 `json:"sub_scores"`
	Reasons   []string           `json:"reasons"`
	LatencyMs int64              `json:"latency_ms"`
	TokensIn  int                `json:"tokens_in"`
	TokensOut int                `json:"tokens_out"`
	Err       string             `json:"err,omitempty"`
}

// CategoryMetrics summarizes one category for one call-site.
type CategoryMetrics struct {
	Category      Category `json:"category"`
	N             int      `json:"n"`
	OK            int      `json:"ok"`
	Unsure        int      `json:"unsure"`
	Fail          int      `json:"fail"`
	Errors        int      `json:"errors"`
	RejectionRate float64  `json:"rejection_rate"` // fail / n
	UnsureRate    float64  `json:"unsure_rate"`    // unsure / n
	OKRate        float64  `json:"ok_rate"`        // ok / n
	LatencyMsP50  int64    `json:"latency_ms_p50"`
	LatencyMsP95  int64    `json:"latency_ms_p95"`
	MeanTokensTot float64  `json:"mean_tokens_total"`
}

// CallSiteReport groups all categories for one call-site.
type CallSiteReport struct {
	CallSite   string            `json:"call_site"`
	Categories []CategoryMetrics `json:"categories"`
	Fixtures   []FixtureResult   `json:"fixtures,omitempty"`
}

// Runner runs a checker against a corpus directory.
type Runner[T any] struct {
	Checker   *confidence.Checker[T]
	CorpusDir string // typically internal/inference/confidence/testdata/<call-site>
	// Verify wraps Checker.Check; adapters provide their own (parses raw,
	// builds Request).
	Verify func(ctx context.Context, raw []byte) (confidence.Result[T], error)
}

// Run walks CorpusDir/<category>/*.json and produces a CallSiteReport.
func (r *Runner[T]) Run(ctx context.Context, callSite string) (CallSiteReport, error) {
	report := CallSiteReport{CallSite: callSite}
	for _, cat := range AllCategories() {
		dir := filepath.Join(r.CorpusDir, string(cat))
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return report, fmt.Errorf("read %s: %w", dir, err)
		}
		var (
			fixtures          []FixtureResult
			latencies         []int64
			tokenSum          float64
			ok, uns, fl, errs int
		)
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			path := filepath.Join(dir, e.Name())
			raw, rerr := os.ReadFile(path)
			if rerr != nil {
				return report, fmt.Errorf("read %s: %w", path, rerr)
			}
			start := time.Now()
			res, verr := r.Verify(ctx, raw)
			elapsed := time.Since(start).Milliseconds()
			fix := FixtureResult{
				Path:      path,
				Category:  cat,
				LatencyMs: elapsed,
			}
			if verr != nil {
				fix.Err = verr.Error()
				errs++
				fixtures = append(fixtures, fix)
				report.Fixtures = append(report.Fixtures, fix)
				latencies = append(latencies, elapsed)
				continue
			}
			fix.Status = res.Status
			fix.Score = res.Score
			fix.SubScores = res.SubScores
			fix.Reasons = res.Reasons
			fix.TokensIn = res.Trace.TokensIn
			fix.TokensOut = res.Trace.TokensOut
			fixtures = append(fixtures, fix)
			report.Fixtures = append(report.Fixtures, fix)
			latencies = append(latencies, elapsed)
			tokenSum += float64(res.Trace.TokensIn + res.Trace.TokensOut)
			switch res.Status {
			case confidence.StatusOK:
				ok++
			case confidence.StatusUnsure:
				uns++
			case confidence.StatusFail:
				fl++
			}
		}
		n := len(fixtures)
		var p50, p95 int64
		if n > 0 {
			p50 = percentile(latencies, 0.5)
			p95 = percentile(latencies, 0.95)
		}
		var meanTok float64
		if n > 0 {
			meanTok = tokenSum / float64(n)
		}
		report.Categories = append(report.Categories, CategoryMetrics{
			Category:      cat,
			N:             n,
			OK:            ok,
			Unsure:        uns,
			Fail:          fl,
			Errors:        errs,
			RejectionRate: ratio(fl, n),
			UnsureRate:    ratio(uns, n),
			OKRate:        ratio(ok, n),
			LatencyMsP50:  p50,
			LatencyMsP95:  p95,
			MeanTokensTot: meanTok,
		})
	}
	return report, nil
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

// percentile returns the p-th percentile (p in [0,1]) via nearest-rank.
// Mutates a copy of the slice; safe to call repeatedly.
func percentile(xs []int64, p float64) int64 {
	if len(xs) == 0 {
		return 0
	}
	sorted := make([]int64, len(xs))
	copy(sorted, xs)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	if p <= 0 {
		return sorted[0]
	}
	if p >= 1 {
		return sorted[len(sorted)-1]
	}
	idx := int(math.Ceil(p*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// MarshalJSON renders a report as pretty JSON for evidence dumps.
func MarshalJSON(r CallSiteReport) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// RenderMarkdown produces a human-readable summary for docs/research/.
// The summary highlights acceptance metrics: adversarial rejection ≥ 0.8 and
// correct false-FAIL ≤ 0.02.
func RenderMarkdown(reports []CallSiteReport) string {
	var b []byte
	b = append(b, []byte("# F144 Confidence Replay Report\n\n")...)
	b = append(b, []byte("Generated automatically by `internal/inference/confidence/replay` against the fixture corpus under `internal/inference/confidence/testdata/`.\n\n")...)
	b = append(b, []byte("Acceptance gates (from F144 design §9):\n")...)
	b = append(b, []byte("- Adversarial rejection rate ≥ 0.80\n")...)
	b = append(b, []byte("- Correct false-FAIL rate ≤ 0.02\n\n")...)

	for _, r := range reports {
		b = append(b, []byte(fmt.Sprintf("## %s\n\n", r.CallSite))...)
		b = append(b, []byte("| Category | N | OK | UNSURE | FAIL | Errors | Rejection rate | Unsure rate | p50 ms | p95 ms | Mean tokens |\n")...)
		b = append(b, []byte("|---|---|---|---|---|---|---|---|---|---|---|\n")...)
		for _, c := range r.Categories {
			b = append(b, []byte(fmt.Sprintf("| %s | %d | %d | %d | %d | %d | %.2f | %.2f | %d | %d | %.0f |\n",
				c.Category, c.N, c.OK, c.Unsure, c.Fail, c.Errors, c.RejectionRate, c.UnsureRate,
				c.LatencyMsP50, c.LatencyMsP95, c.MeanTokensTot))...)
		}
		b = append(b, []byte("\nVerdict: ")...)
		b = append(b, []byte(verdictLine(r))...)
		b = append(b, []byte("\n\n")...)
	}
	return string(b)
}

func verdictLine(r CallSiteReport) string {
	var advRej, corrFalseFail float64
	var errors int
	for _, c := range r.Categories {
		errors += c.Errors
		switch c.Category {
		case Adversarial:
			advRej = c.RejectionRate
		case Correct:
			corrFalseFail = c.RejectionRate
		}
	}
	pass := errors == 0 && advRej >= 0.80 && corrFalseFail <= 0.02
	if pass {
		return fmt.Sprintf("**PASS** — adversarial rejection %.2f ≥ 0.80, correct false-FAIL %.2f ≤ 0.02", advRej, corrFalseFail)
	}
	if errors > 0 {
		return fmt.Sprintf("**FAIL** — %d fixture errors; adversarial rejection %.2f, correct false-FAIL %.2f", errors, advRej, corrFalseFail)
	}
	return fmt.Sprintf("**FAIL** — adversarial rejection %.2f, correct false-FAIL %.2f (gates 0.80 / 0.02)", advRej, corrFalseFail)
}
