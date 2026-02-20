package artifact

import "sort"

// Class captures intake metadata for artifacts produced on the swarm bus.
type Class struct {
	ID                   string
	Description          string
	RetentionDays        int
	Phases               []string
	RequiredProvenance   []string
	IntegrityCriticality string
}

// PhaseRequirement captures artifact and provenance requirements per swarm phase.
type PhaseRequirement struct {
	Phase                    string
	RequiredClassIDs         []string
	AdditionalProvenanceKeys []string
}

// GateSignal captures a normalized gate outcome token.
type GateSignal struct {
	ID          string
	Phase       string
	Description string
}

// TransitionPrerequisite captures evidence required before moving between phases.
type TransitionPrerequisite struct {
	FromPhase                string
	ToPhase                  string
	RequiredGateSignals      []string
	RequiredArtifactClassIDs []string
	RequiredProvenanceKeys   []string
}

var baseProvenanceKeys = []string{
	"run_id",
	"orchestrator",
	"runtime",
	"model",
	"gate_results",
	"phase",
	"role",
	"captured_at",
	"source_issue_id",
	"artifact_id",
	"hash",
	"hash_prev",
}

var classes = []Class{
	{
		ID:            "intent-brief",
		Description:   "Input framing, acceptance gates, and risk posture for the selected issue.",
		RetentionDays: 365,
		Phases:        []string{"intake", "plan"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"trigger",
		),
		IntegrityCriticality: "high",
	},
	{
		ID:            "execution-plan",
		Description:   "Workstream plan and ordering rationale used for deterministic execution.",
		RetentionDays: 365,
		Phases:        []string{"plan"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"depends_on",
		),
		IntegrityCriticality: "high",
	},
	{
		ID:            "code-diff",
		Description:   "Patch set representing implementation changes.",
		RetentionDays: 1095,
		Phases:        []string{"execute"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"paths_touched",
			"test_targets",
		),
		IntegrityCriticality: "critical",
	},
	{
		ID:            "verification-report",
		Description:   "Machine-readable results from test, lint, and contract gates.",
		RetentionDays: 1095,
		Phases:        []string{"verify"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"gate_name",
			"gate_status",
		),
		IntegrityCriticality: "critical",
	},
	{
		ID:            "review-verdict",
		Description:   "Review outcome and risk signoff for publication decision.",
		RetentionDays: 1095,
		Phases:        []string{"review", "publish"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"reviewer_role",
			"decision",
		),
		IntegrityCriticality: "critical",
	},
	{
		ID:            "trace-link",
		Description:   "Trace pointers to commits, PR URLs, and external evidence handles.",
		RetentionDays: 1825,
		Phases:        []string{"verify", "publish"},
		RequiredProvenance: appendCopy(baseProvenanceKeys,
			"commit_ids",
			"pr_url",
		),
		IntegrityCriticality: "critical",
	},
}

var phaseRequirements = []PhaseRequirement{
	{Phase: "intake", RequiredClassIDs: []string{"intent-brief"}, AdditionalProvenanceKeys: []string{"trigger"}},
	{Phase: "plan", RequiredClassIDs: []string{"intent-brief", "execution-plan"}, AdditionalProvenanceKeys: []string{"depends_on"}},
	{Phase: "execute", RequiredClassIDs: []string{"code-diff"}, AdditionalProvenanceKeys: []string{"paths_touched", "test_targets"}},
	{Phase: "verify", RequiredClassIDs: []string{"verification-report", "trace-link"}, AdditionalProvenanceKeys: []string{"gate_name", "gate_status"}},
	{Phase: "review", RequiredClassIDs: []string{"review-verdict"}, AdditionalProvenanceKeys: []string{"reviewer_role", "decision"}},
	{Phase: "publish", RequiredClassIDs: []string{"review-verdict", "trace-link"}, AdditionalProvenanceKeys: []string{"decision", "pr_url"}},
}

var gateSignals = []GateSignal{
	{ID: "intake:issue-scoped", Phase: "intake", Description: "Issue scope and acceptance criteria captured for execution."},
	{ID: "intake:risk-assessed", Phase: "intake", Description: "Risk posture and ownership lane declared."},
	{ID: "plan:dependencies-resolved", Phase: "plan", Description: "Execution order and blocking dependencies resolved."},
	{ID: "execute:diff-prepared", Phase: "execute", Description: "Patch set produced with touched paths and targets."},
	{ID: "verify:tests-passed", Phase: "verify", Description: "Required test targets passed for the issue scope."},
	{ID: "verify:boundary-ok", Phase: "verify", Description: "Boundary policy validation passed with no forbidden writes."},
	{ID: "verify:evidence-contract-pass", Phase: "verify", Description: "Strict evidence contract validation passed."},
	{ID: "review:decision-recorded", Phase: "review", Description: "Reviewer decision and rationale recorded."},
	{ID: "review:risk-signoff", Phase: "review", Description: "Risk signoff completed for publish eligibility."},
	{ID: "publish:pr-gate-pass", Phase: "publish", Description: "PR gate checks passed for publish transition."},
	{ID: "publish:callback-published", Phase: "publish", Description: "PR callback payload published to callback stream."},
}

var transitionPrerequisites = []TransitionPrerequisite{
	{
		FromPhase:                "intake",
		ToPhase:                  "plan",
		RequiredGateSignals:      []string{"intake:issue-scoped", "intake:risk-assessed"},
		RequiredArtifactClassIDs: []string{"intent-brief"},
		RequiredProvenanceKeys:   []string{"trigger"},
	},
	{
		FromPhase:                "plan",
		ToPhase:                  "execute",
		RequiredGateSignals:      []string{"plan:dependencies-resolved"},
		RequiredArtifactClassIDs: []string{"execution-plan"},
		RequiredProvenanceKeys:   []string{"depends_on"},
	},
	{
		FromPhase:                "execute",
		ToPhase:                  "verify",
		RequiredGateSignals:      []string{"execute:diff-prepared"},
		RequiredArtifactClassIDs: []string{"code-diff"},
		RequiredProvenanceKeys:   []string{"paths_touched", "test_targets"},
	},
	{
		FromPhase:                "verify",
		ToPhase:                  "review",
		RequiredGateSignals:      []string{"verify:tests-passed", "verify:boundary-ok", "verify:evidence-contract-pass"},
		RequiredArtifactClassIDs: []string{"verification-report", "trace-link"},
		RequiredProvenanceKeys:   []string{"gate_name", "gate_status"},
	},
	{
		FromPhase:                "review",
		ToPhase:                  "publish",
		RequiredGateSignals:      []string{"review:decision-recorded", "review:risk-signoff", "publish:pr-gate-pass", "publish:callback-published"},
		RequiredArtifactClassIDs: []string{"review-verdict", "trace-link"},
		RequiredProvenanceKeys:   []string{"reviewer_role", "decision", "pr_url"},
	},
}

func Classes() []Class {
	out := make([]Class, len(classes))
	copy(out, classes)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func PhaseRequirements() []PhaseRequirement {
	out := make([]PhaseRequirement, len(phaseRequirements))
	copy(out, phaseRequirements)
	sort.Slice(out, func(i, j int) bool { return out[i].Phase < out[j].Phase })
	return out
}

func BaseProvenanceKeys() []string {
	out := make([]string, len(baseProvenanceKeys))
	copy(out, baseProvenanceKeys)
	return out
}

func GateSignals() []GateSignal {
	out := make([]GateSignal, len(gateSignals))
	copy(out, gateSignals)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func TransitionPrerequisites() []TransitionPrerequisite {
	out := make([]TransitionPrerequisite, len(transitionPrerequisites))
	copy(out, transitionPrerequisites)
	sort.Slice(out, func(i, j int) bool {
		if out[i].FromPhase == out[j].FromPhase {
			return out[i].ToPhase < out[j].ToPhase
		}
		return out[i].FromPhase < out[j].FromPhase
	})
	return out
}

func appendCopy(base []string, items ...string) []string {
	out := make([]string, 0, len(base)+len(items))
	out = append(out, base...)
	out = append(out, items...)
	return out
}
