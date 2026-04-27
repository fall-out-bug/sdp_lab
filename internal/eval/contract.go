package eval

import "time"

const ContractVersion = "v1.0.0"

// CaseRunnerV1 defines the interface for running a single evaluation case.
// Cases are idempotent: same transcript + patterns = same result.
type CaseRunnerV1 interface {
	RunCase(c *Case, projectRoot string) Result
}

// SuiteRunnerV1 defines the interface for running a suite of evaluation cases.
type SuiteRunnerV1 interface {
	Run(projectRoot, casesDir, skill string) ([]Result, error)
}

// CaseLoaderV1 defines the interface for loading evaluation cases from disk.
type CaseLoaderV1 interface {
	LoadCases(casesDir, skill string) ([]Case, error)
}

// BaselineComparator defines the interface for comparing current results against a baseline.
type BaselineComparator interface {
	Compare(current, baseline []Result) BaselineComparison
}

// Scoreboard defines the interface for recording and retrieving evaluation history.
type Scoreboard interface {
	Record(runID string, results []Result) ScoreboardEntry
	History(limit int) []ScoreboardEntry
}

// BaselineComparison captures the delta between current and baseline evaluation results.
type BaselineComparison struct {
	Regressions  int                // Cases that passed baseline but failed current
	Improvements int                // Cases that failed baseline but passed current
	Unchanged    int                // Cases with same pass/fail status
	Details      []ComparisonDetail // Per-case breakdown
}

// ComparisonDetail shows the status change for a single case.
type ComparisonDetail struct {
	Case         string // Case name
	CurrentPass  bool   // Current run pass status
	BaselinePass bool   // Baseline run pass status
	Delta        string // Description of change (e.g., "PASS → FAIL")
}

// ScoreboardEntry records a single evaluation run's metrics.
type ScoreboardEntry struct {
	RunID       string    // Unique identifier for this run
	Timestamp   time.Time // When the run was recorded
	TotalCases  int       // Total number of cases in the run
	PassedCases int       // Number of cases that passed
	PassRate    float64   // Percentage of passed cases (0.0-100.0)
	Regressions int       // Number of regressions vs baseline (if available)
}

// MismatchMetric quantifies evidence-mismatch rate in governance decisions.
// This replaces the hallucination rate metric per IIP council decision.
type MismatchMetric struct {
	TotalDecisions        int     // Total governance decisions evaluated
	EvidenceMismatchCount int     // Decisions with mismatched evidence
	MismatchRate          float64 // Evidence-mismatch rate (0.0-1.0)
}

// EvaluationContract documents the core evaluation semantics.
// Cases are idempotent: given the same transcript and pattern set,
// the result is deterministic and reproducible.
//
// Verdict semantics:
//   - verdict=PASS means the case expects no violations:
//     agent output must contain no forbidden patterns and all required patterns.
//   - verdict=FAIL means the case expects violations:
//     the evaluation passes if we correctly detect the expected violations.
type EvaluationContract struct {
	// Documented for v1 API contract
	_ struct{} // enforce struct usage
}

// MismatchMetricContract documents the evidence-mismatch metric semantics.
// This metric replaces the hallucination rate per IIP council decision.
// It measures the accuracy of governance decisions by comparing
// decision outcomes against evidence alignment.
//
// Mismatch rate = (mismatched decisions) / (total decisions)
// Lower rates indicate better governance decision accuracy.
type MismatchMetricContract struct {
	// Documented for v1 API contract
	_ struct{} // enforce struct usage
}
