package evaluator

import "sort"

// ScopeComponent defines a subsystem included in baseline evaluator runs.
type ScopeComponent struct {
	ID            string
	Description   string
	OwnerSignal   string
	EvidenceClass string
}

// OperabilityCheck defines one required happy-path prerequisite.
type OperabilityCheck struct {
	ID          string
	Description string
}

// OperabilityResult reports whether intake prerequisites are satisfied.
type OperabilityResult struct {
	Passed          bool
	MissingCheckIDs []string
}

var baselineScope = []ScopeComponent{
	{
		ID:            "issue-intake",
		Description:   "Issue scope, dependencies, and acceptance gates are captured before evaluator launch.",
		OwnerSignal:   "intake:issue-scoped",
		EvidenceClass: "intent-brief",
	},
	{
		ID:            "execution-flow",
		Description:   "Planner and execution path can produce deterministic patches for selected components.",
		OwnerSignal:   "execute:diff-prepared",
		EvidenceClass: "code-diff",
	},
	{
		ID:            "verification-stack",
		Description:   "Verification gates are runnable for evaluator-selected issue scope.",
		OwnerSignal:   "verify:tests-passed",
		EvidenceClass: "verification-report",
	},
	{
		ID:            "review-publish-link",
		Description:   "Review and publish handoff emits trace links and callback evidence.",
		OwnerSignal:   "publish:callback-published",
		EvidenceClass: "trace-link",
	},
}

var happyPathOperabilityChecks = []OperabilityCheck{
	{ID: "issue-selected", Description: "Target issue is selected and moved to in_progress."},
	{ID: "dependencies-clear", Description: "No blocking dependencies remain for the selected issue."},
	{ID: "scope-baseline-defined", Description: "Baseline evaluator scope components are defined."},
	{ID: "gate-command-declared", Description: "Happy-path operability gate command is declared (go test ./...)."},
	{ID: "callback-contract-available", Description: "PR callback publishing contract is available for evaluator publish phase."},
}

func BaselineScope() []ScopeComponent {
	out := make([]ScopeComponent, len(baselineScope))
	copy(out, baselineScope)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func HappyPathOperabilityChecks() []OperabilityCheck {
	out := make([]OperabilityCheck, len(happyPathOperabilityChecks))
	copy(out, happyPathOperabilityChecks)
	return out
}

func EvaluateHappyPathOperability(signals map[string]bool) OperabilityResult {
	missing := make([]string, 0, len(happyPathOperabilityChecks))
	for _, check := range happyPathOperabilityChecks {
		if !signals[check.ID] {
			missing = append(missing, check.ID)
		}
	}
	return OperabilityResult{Passed: len(missing) == 0, MissingCheckIDs: missing}
}
