package harness

import "time"

type TaskContract struct {
	Version            string                `json:"version"`
	RunID              string                `json:"run_id"`
	CreatedAt          string                `json:"created_at"`
	Objective          string                `json:"objective"`
	AcceptanceCriteria []AcceptanceCriterion `json:"acceptance_criteria"`
	RequiredMetrics    []RequiredMetric      `json:"required_metrics"`
	RequiredEvidence   []string              `json:"required_evidence"`
	QualityGates       QualityGates          `json:"quality_gates"`
	Constraints        Constraints           `json:"constraints"`
	ChangeRequests     []ChangeRequest       `json:"change_requests,omitempty"`
}

type AcceptanceCriterion struct {
	ID        string `json:"id"`
	Statement string `json:"statement"`
	Priority  string `json:"priority"`
}

type RequiredMetric struct {
	Name      string `json:"name"`
	Target    any    `json:"target"`
	Direction string `json:"direction"`
}

type QualityGates struct {
	Build     bool `json:"build"`
	Test      bool `json:"test"`
	Lint      bool `json:"lint"`
	Typecheck bool `json:"typecheck"`
}

type Constraints struct {
	AllowScopeReduction  bool   `json:"allow_scope_reduction"`
	AllowMetricReduction bool   `json:"allow_metric_reduction"`
	SecurityPolicy       string `json:"security_policy,omitempty"`
	PerformanceBudget    string `json:"performance_budget,omitempty"`
}

type ChangeRequest struct {
	ID         string `json:"id"`
	Reason     string `json:"reason"`
	ApprovedBy string `json:"approved_by"`
	ApprovedAt string `json:"approved_at"`
}

type TaskSnapshot struct {
	RunID               string            `json:"run_id,omitempty"`
	Phase               string            `json:"phase,omitempty"`
	AcceptanceCriteria  []CriterionStatus `json:"acceptance_criteria"`
	Metrics             []MetricSnapshot  `json:"metrics"`
	Evidence            []string          `json:"evidence"`
	QualityResults      map[string]bool   `json:"quality_results"`
	ProcessReport       ProcessReport     `json:"process_report"`
	Claims              []Claim           `json:"claims,omitempty"`
	RequirementsHash    string            `json:"requirements_hash,omitempty"`
	MetricsHash         string            `json:"metrics_hash,omitempty"`
	ContractHash        string            `json:"contract_hash,omitempty"`
	AdditionalTelemetry map[string]string `json:"additional_telemetry,omitempty"`
}

type CriterionStatus struct {
	ID     string `json:"id"`
	Status string `json:"status,omitempty"`
}

type MetricSnapshot struct {
	Name   string `json:"name"`
	Target any    `json:"target,omitempty"`
	Value  any    `json:"value,omitempty"`
}

type ProcessReport struct {
	ContractCoverageSummary bool `json:"contract_coverage_summary"`
	GateResults             bool `json:"gate_results"`
	EvidenceIndex           bool `json:"evidence_index"`
	DecisionLog             bool `json:"decision_log"`
}

type Claim struct {
	ID           string   `json:"id"`
	Statement    string   `json:"statement"`
	EvidenceRefs []string `json:"evidence_refs"`
}

type GateID string

const (
	GateRequirementIntegrity GateID = "requirement_integrity"
	GateEvidence             GateID = "evidence"
	GateMetricParity         GateID = "metric_parity"
	GateQuality              GateID = "quality"
	GateProcess              GateID = "process"
)

type GateStatus string

const (
	GatePass  GateStatus = "pass"
	GateWarn  GateStatus = "warn"
	GateBlock GateStatus = "block"
)

type DriftType string

const (
	DriftACDrop            DriftType = "ac_drop"
	DriftMetricDrop        DriftType = "metric_drop"
	DriftScopeWeaken       DriftType = "scope_weaken"
	DriftUnsupportedClaim  DriftType = "unsupported_claim"
	DriftMissingEvidence   DriftType = "missing_evidence"
	DriftQualityGateFail   DriftType = "quality_gate_fail"
	DriftProcessIncomplete DriftType = "process_incomplete"
)

type Violation struct {
	Type     DriftType `json:"type"`
	Field    string    `json:"field"`
	Message  string    `json:"message"`
	Expected string    `json:"expected,omitempty"`
	Actual   string    `json:"actual,omitempty"`
}

type GateResult struct {
	GateID     GateID      `json:"gate_id"`
	Status     GateStatus  `json:"status"`
	Violations []Violation `json:"violations,omitempty"`
}

type ComplianceReport struct {
	RunID           string       `json:"run_id,omitempty"`
	ContractVersion string       `json:"contract_version"`
	Phase           string       `json:"phase,omitempty"`
	GeneratedAt     string       `json:"generated_at"`
	Blocked         bool         `json:"blocked"`
	GateResults     []GateResult `json:"gate_results"`
}

type ClarificationChange struct {
	ID                       string                `json:"id,omitempty"`
	Reason                   string                `json:"reason,omitempty"`
	PolicySensitive          bool                  `json:"policy_sensitive,omitempty"`
	AddAcceptanceCriteria    []AcceptanceCriterion `json:"add_acceptance_criteria,omitempty"`
	RemoveAcceptanceCriteria []string              `json:"remove_acceptance_criteria,omitempty"`
	AddMetrics               []RequiredMetric      `json:"add_metrics,omitempty"`
	RemoveMetrics            []string              `json:"remove_metrics,omitempty"`
	AddEvidence              []string              `json:"add_evidence,omitempty"`
	RemoveEvidence           []string              `json:"remove_evidence,omitempty"`
	EnableQualityGates       []string              `json:"enable_quality_gates,omitempty"`
	DisableQualityGates      []string              `json:"disable_quality_gates,omitempty"`
}

type ClarificationClass string

const (
	ClarificationNoImpact        ClarificationClass = "no_impact"
	ClarificationAdditive        ClarificationClass = "additive"
	ClarificationReductive       ClarificationClass = "reductive"
	ClarificationPolicySensitive ClarificationClass = "policy_sensitive"
)

type ClarificationDecision struct {
	Classification   ClarificationClass `json:"classification"`
	RequiresApproval bool               `json:"requires_approval"`
	Blocking         bool               `json:"blocking"`
	Reasons          []string           `json:"reasons,omitempty"`
}

func newReport(contract *TaskContract, snapshot *TaskSnapshot) ComplianceReport {
	runID := contract.RunID
	phase := ""
	if snapshot != nil {
		if snapshot.RunID != "" {
			runID = snapshot.RunID
		}
		phase = snapshot.Phase
	}
	return ComplianceReport{
		RunID:           runID,
		ContractVersion: contract.Version,
		Phase:           phase,
		GeneratedAt:     time.Now().UTC().Format(time.RFC3339),
		GateResults:     make([]GateResult, 0, 5),
	}
}
