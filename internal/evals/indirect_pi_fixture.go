package evals

// IndirectPICase represents one indirect prompt-injection demo case.
// It is defensive-only: no real exploit payloads, no live service targets.
type IndirectPICase struct {
	// CaseID is a unique identifier (e.g., F165-VEC-001).
	CaseID string `yaml:"case_id" json:"case_id"`

	// Vector names the SDP-native attack surface.
	// Values: beads_issue, workstream_markdown, evidence_finding, cross_agent_handoff, mcp_resource.
	Vector string `yaml:"vector" json:"vector"`

	// TrustedOperatorRequest is the operator action being performed.
	TrustedOperatorRequest string `yaml:"trusted_operator_request" json:"trusted_operator_request"`

	// TrustedStateSnapshot is pre-ingestion trusted state for validation.
	TrustedStateSnapshot TrustedStateSnapshot `yaml:"trusted_state_snapshot" json:"trusted_state_snapshot"`

	// UntrustedArtifact is the sanitized fixture payload.
	UntrustedArtifact string `yaml:"untrusted_artifact" json:"untrusted_artifact"`

	// ExpectedUnsafeResult is what the naive runner produces.
	ExpectedUnsafeResult UnsafeResult `yaml:"expected_unsafe_result" json:"expected_unsafe_result"`

	// ExpectedDefendedResult is what the defended runner produces.
	ExpectedDefendedResult DefendedResult `yaml:"expected_defended_result" json:"expected_defended_result"`

	// EvidenceExpectation describes what evidence is required.
	EvidenceExpectation string `yaml:"evidence_expectation" json:"evidence_expectation"`

	// ResidualRiskCategory classifies residual risk for this case.
	// Values: none, unsupported_surface, partial_coverage, not_tested.
	ResidualRiskCategory string `yaml:"residual_risk_category" json:"residual_risk_category"`
}

// TrustedStateSnapshot carries pre-ingestion trusted fields.
type TrustedStateSnapshot struct {
	// Beads fields (populated for beads_issue vector).
	BeadsIssueID     string   `yaml:"beads_issue_id,omitempty" json:"beads_issue_id,omitempty"`
	BeadsStatus      string   `yaml:"beads_status,omitempty" json:"beads_status,omitempty"`
	BeadsPriority    int      `yaml:"beads_priority,omitempty" json:"beads_priority,omitempty"`
	BeadsLabels      []string `yaml:"beads_labels,omitempty" json:"beads_labels,omitempty"`

	// Workstream fields (populated for workstream_markdown vector).
	WorkstreamID    string   `yaml:"workstream_id,omitempty" json:"workstream_id,omitempty"`
	WorkstreamScope []string `yaml:"workstream_scope,omitempty" json:"workstream_scope,omitempty"`

	// Evidence fields (populated for evidence_finding vector).
	ToolExitCode    int    `yaml:"tool_exit_code,omitempty" json:"tool_exit_code,omitempty"`
	EvidenceRef     string `yaml:"evidence_ref,omitempty" json:"evidence_ref,omitempty"`

	// TrustedNarrative is operator-authored summary of known-good state.
	TrustedNarrative string `yaml:"trusted_narrative,omitempty" json:"trusted_narrative,omitempty"`
}

// UnsafeResult describes expected naive-runner failure.
type UnsafeResult struct {
	// UnsafeAction is the action the naive runner would take.
	UnsafeAction string `yaml:"unsafe_action" json:"unsafe_action"`

	// UnsafeClaim is the false assertion the naive runner would emit.
	UnsafeClaim string `yaml:"unsafe_claim" json:"unsafe_claim"`
}

// DefendedResult describes expected defended-runner outcome.
type DefendedResult struct {
	// Verdict is the demo verdict: blocked, clean, residual_risk.
	Verdict string `yaml:"verdict" json:"verdict"`

	// BlockedReason is from the closed F165 set; empty when verdict is clean.
	BlockedReason string `yaml:"blocked_reason,omitempty" json:"blocked_reason,omitempty"`

	// TrustedEvidenceRef references deterministic evidence.
	TrustedEvidenceRef string `yaml:"trusted_evidence_ref,omitempty" json:"trusted_evidence_ref,omitempty"`
}

// BlockedReason values (closed set for F165).
const (
	BlockedReasonUntrustedCompletionClaim        = "untrusted_completion_claim"
	BlockedReasonScopePolicyConflict             = "scope_policy_conflict"
	BlockedReasonEvidenceSourceMismatch          = "evidence_source_mismatch"
	BlockedReasonWriteWithoutTrustedAuthorization = "write_without_trusted_authorization"
	BlockedReasonParseError                       = "parse_error"
	BlockedReasonPolicyConflict                   = "policy_conflict"
	BlockedReasonUnsupportedResidualRisk          = "unsupported_residual_risk"
)

// ResidualRiskCategory values (closed set for F165).
const (
	ResidualRiskNone               = "none"
	ResidualRiskUnsupportedSurface = "unsupported_surface"
	ResidualRiskPartialCoverage    = "partial_coverage"
	ResidualRiskNotTested          = "not_tested"
)

// ValidVectors is the closed set of F165 attack vectors.
var ValidVectors = []string{
	"beads_issue",
	"workstream_markdown",
	"evidence_finding",
	"cross_agent_handoff",
	"mcp_resource",
}

// ValidVerdicts is the closed set of F165 demo verdicts.
var ValidVerdicts = []string{
	"blocked",
	"clean",
	"residual_risk",
}

// IsValidBlockedReason reports whether r is in the closed F165 set.
func IsValidBlockedReason(r string) bool {
	switch r {
	case BlockedReasonUntrustedCompletionClaim,
		BlockedReasonScopePolicyConflict,
		BlockedReasonEvidenceSourceMismatch,
		BlockedReasonWriteWithoutTrustedAuthorization,
		BlockedReasonParseError,
		BlockedReasonPolicyConflict,
		BlockedReasonUnsupportedResidualRisk:
		return true
	}
	return false
}

// IsValidResidualRiskCategory reports whether c is in the closed F165 set.
func IsValidResidualRiskCategory(c string) bool {
	switch c {
	case ResidualRiskNone,
		ResidualRiskUnsupportedSurface,
		ResidualRiskPartialCoverage,
		ResidualRiskNotTested:
		return true
	}
	return false
}

// IsValidVector reports whether v is a recognized F165 vector.
func IsValidVector(v string) bool {
	for _, valid := range ValidVectors {
		if v == valid {
			return true
		}
	}
	return false
}

// IsValidVerdict reports whether v is a recognized F165 demo verdict.
func IsValidVerdict(v string) bool {
	for _, valid := range ValidVerdicts {
		if v == valid {
			return true
		}
	}
	return false
}
