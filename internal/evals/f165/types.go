package f165

// Case represents one indirect prompt-injection demo case.
// It is defensive-only: no real exploit payloads, no live service targets.
type Case struct {
	CaseID                 string               `yaml:"case_id" json:"case_id"`
	Vector                 string               `yaml:"vector" json:"vector"`
	TrustedOperatorRequest string               `yaml:"trusted_operator_request" json:"trusted_operator_request"`
	TrustedStateSnapshot   TrustedStateSnapshot `yaml:"trusted_state_snapshot" json:"trusted_state_snapshot"`
	UntrustedArtifact      string               `yaml:"untrusted_artifact" json:"untrusted_artifact"`
	ExpectedUnsafeResult   UnsafeResult         `yaml:"expected_unsafe_result" json:"expected_unsafe_result"`
	ExpectedDefendedResult DefendedResult       `yaml:"expected_defended_result" json:"expected_defended_result"`
	EvidenceExpectation    string               `yaml:"evidence_expectation" json:"evidence_expectation"`
	ResidualRiskCategory   string               `yaml:"residual_risk_category" json:"residual_risk_category"`
}

type TrustedStateSnapshot struct {
	BeadsIssueID     string   `yaml:"beads_issue_id,omitempty" json:"beads_issue_id,omitempty"`
	BeadsStatus      string   `yaml:"beads_status,omitempty" json:"beads_status,omitempty"`
	BeadsPriority    int      `yaml:"beads_priority,omitempty" json:"beads_priority,omitempty"`
	BeadsLabels      []string `yaml:"beads_labels,omitempty" json:"beads_labels,omitempty"`
	WorkstreamID     string   `yaml:"workstream_id,omitempty" json:"workstream_id,omitempty"`
	WorkstreamScope  []string `yaml:"workstream_scope,omitempty" json:"workstream_scope,omitempty"`
	ToolExitCode     int      `yaml:"tool_exit_code,omitempty" json:"tool_exit_code,omitempty"`
	EvidenceRef      string   `yaml:"evidence_ref,omitempty" json:"evidence_ref,omitempty"`
	TrustedNarrative string   `yaml:"trusted_narrative,omitempty" json:"trusted_narrative,omitempty"`
}

type UnsafeResult struct {
	UnsafeAction string `yaml:"unsafe_action" json:"unsafe_action"`
	UnsafeClaim  string `yaml:"unsafe_claim" json:"unsafe_claim"`
}

type DefendedResult struct {
	Verdict            string `yaml:"verdict" json:"verdict"`
	BlockedReason      string `yaml:"blocked_reason,omitempty" json:"blocked_reason,omitempty"`
	TrustedEvidenceRef string `yaml:"trusted_evidence_ref,omitempty" json:"trusted_evidence_ref,omitempty"`
}

const (
	BlockedReasonUntrustedCompletionClaim         = "untrusted_completion_claim"
	BlockedReasonScopePolicyConflict              = "scope_policy_conflict"
	BlockedReasonEvidenceSourceMismatch           = "evidence_source_mismatch"
	BlockedReasonWriteWithoutTrustedAuthorization = "write_without_trusted_authorization"
	BlockedReasonParseError                       = "parse_error"
	BlockedReasonPolicyConflict                   = "policy_conflict"
	BlockedReasonUnsupportedResidualRisk          = "unsupported_residual_risk"
)

const (
	ResidualRiskNone               = "none"
	ResidualRiskUnsupportedSurface = "unsupported_surface"
	ResidualRiskPartialCoverage    = "partial_coverage"
	ResidualRiskNotTested          = "not_tested"
)

var validVectors = [...]string{"beads_issue", "workstream_markdown", "evidence_finding", "cross_agent_handoff", "mcp_resource"}
var validVerdicts = [...]string{"blocked", "clean", "residual_risk"}

func IsValidBlockedReason(r string) bool {
	switch r {
	case BlockedReasonUntrustedCompletionClaim, BlockedReasonScopePolicyConflict,
		BlockedReasonEvidenceSourceMismatch, BlockedReasonWriteWithoutTrustedAuthorization,
		BlockedReasonParseError, BlockedReasonPolicyConflict, BlockedReasonUnsupportedResidualRisk:
		return true
	}
	return false
}

func IsValidResidualRiskCategory(c string) bool {
	switch c {
	case ResidualRiskNone, ResidualRiskUnsupportedSurface, ResidualRiskPartialCoverage, ResidualRiskNotTested:
		return true
	}
	return false
}

func IsValidVector(v string) bool {
	for _, valid := range validVectors {
		if v == valid {
			return true
		}
	}
	return false
}

func IsValidVerdict(v string) bool {
	for _, valid := range validVerdicts {
		if v == valid {
			return true
		}
	}
	return false
}
