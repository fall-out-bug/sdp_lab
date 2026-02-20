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

func appendCopy(base []string, items ...string) []string {
	out := make([]string, 0, len(base)+len(items))
	out = append(out, base...)
	out = append(out, items...)
	return out
}
