package policy

import (
	"context"

	"sdp_dev/internal/evidence"
)

// ContractVersion is the semantic version of the sdp-policy-core API contract
const ContractVersion = "v1.0.0"

// Evaluator defines the core policy evaluation interface
type Evaluator interface {
	// Evaluate processes evidence inputs and produces a policy decision
	// EvidenceInput wraps attestation data, discrepancy reports, and gate configuration
	Evaluate(ctx context.Context, evidence EvidenceInput) (Decision, error)
}

// Explainer provides human-readable explanations for policy decisions
type Explainer interface {
	// Explain generates a detailed explanation of a policy decision
	// allowed: whether the decision was to allow
	// reasons: list of reasons that contributed to the decision
	// severityCount: map of severity levels to discrepancy counts
	// mode: level of detail (Basic/Detailed/Internal)
	Explain(ctx context.Context, allowed bool, reasons []string, severityCount map[string]int, mode ExplainMode) (ExplainResult, error)
}

// RuleStore manages policy rule persistence and retrieval
type RuleStore interface {
	// LoadRules retrieves all rules from the store
	LoadRules(ctx context.Context) ([]Rule, error)
	// SaveRule persists a single rule to the store
	SaveRule(ctx context.Context, rule Rule) error
}

// EvidenceInput wraps all inputs required for policy evaluation
type EvidenceInput struct {
	// Attestation is the signed coding workflow statement
	Attestation []byte
	// DiscrepancyReport contains the discrepancy analysis results
	DiscrepancyReport evidence.DiscrepancyReport
	// GateConfig configures the evidence gate thresholds and requirements
	GateConfig GateConfig
}

// GateConfig configures the evidence gate evaluation
type GateConfig struct {
	// RequireSignedAttestation indicates if a signature is required
	RequireSignedAttestation bool
	// Thresholds defines the maximum allowed discrepancies per severity
	Thresholds DiscrepancyThresholds
}

// DiscrepancyThresholds defines the maximum allowed counts for each severity level
type DiscrepancyThresholds struct {
	Critical int
	High     int
	Medium   int
	Low      int
}

// Rule represents a single policy rule
type Rule struct {
	ID        string // Unique identifier for the rule
	Type      string // Rule type (e.g., "boundary", "verification", "discrepancy")
	Severity  string // Severity level (e.g., "critical", "high", "medium", "low")
	Condition string // Condition expression to evaluate
	Enabled   bool   // Whether the rule is currently active
}

// EvaluationContract documents the v1 evaluation semantics
type EvaluationContract struct {
	// Version is the contract version this struct documents
	Version string
	// DeterministicOrdering guarantees rules are applied in a predictable order
	DeterministicOrdering bool // Rules applied by severity descending
	// ShortCircuitDeny indicates evaluation stops on first deny
	ShortCircuitDeny bool // Stop immediately when a deny decision is reached
	// OverrideHooksEnabled indicates whether post-evaluation hooks are executed
	OverrideHooksEnabled bool // Hooks run after evaluation completes
	// AuditTrailEnabled indicates whether all decisions are logged
	AuditTrailEnabled bool // All decisions logged for compliance
}

// DefaultEvaluationContract returns the standard v1 evaluation contract
func DefaultEvaluationContract() EvaluationContract {
	return EvaluationContract{
		Version:               ContractVersion,
		DeterministicOrdering: true,
		ShortCircuitDeny:      true,
		OverrideHooksEnabled:  true,
		AuditTrailEnabled:     true,
	}
}

// OverrideHook is a function that can modify a decision after evaluation
// This enables post-evaluation adjustments with audit trail requirements
type OverrideHook func(ctx context.Context, decision Decision) (Decision, error)

// EvaluateEvidenceGate is the exported wrapper for internal evidence gate evaluation
// It evaluates attestation and discrepancy report against gate configuration
func EvaluateEvidenceGate(ctx context.Context, input EvidenceInput) (Decision, error) {
	config := evidenceGateConfig{
		RequireSignedAttestation: input.GateConfig.RequireSignedAttestation,
		Thresholds: discrepancyThresholds{
			Critical: input.GateConfig.Thresholds.Critical,
			High:     input.GateConfig.Thresholds.High,
			Medium:   input.GateConfig.Thresholds.Medium,
			Low:      input.GateConfig.Thresholds.Low,
		},
	}

	result := evaluateEvidenceGate(
		config,
		input.Attestation,
		input.DiscrepancyReport,
		defaultVerifyAttestation,
	)

	explanation := ExplainDecision(result.Allowed, result.Reasons, result.SeverityCount, ExplainModeDetailed)
	return NewDecision(result.Allowed, explanation), nil
}
