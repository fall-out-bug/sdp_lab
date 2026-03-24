package omoclient

import (
	"context"
	"fmt"
)

// Outcome represents the result of a governed task execution
type Outcome string

const (
	// OutcomeSucceeded indicates successful completion
	OutcomeSucceeded Outcome = "succeeded"
	// OutcomeIncomplete indicates partial completion or missing evidence
	OutcomeIncomplete Outcome = "incomplete"
	// OutcomeOutOfScope indicates scope violations
	OutcomeOutOfScope Outcome = "out_of_scope"
	// OutcomeFailed indicates general failure
	OutcomeFailed Outcome = "failed"
	// OutcomeCanceled indicates cancellation
	OutcomeCanceled Outcome = "canceled"
	// OutcomeEscalated indicates escalation due to strike thresholds
	OutcomeEscalated Outcome = "escalated"
)

// GovernanceConfig defines governance constraints for a task
type GovernanceConfig struct {
	ConstitutionVersion string `json:"constitution_version"`
	MaxToolCalls        int    `json:"max_tool_calls"`
	MustCiteEvidence    bool   `json:"must_cite_evidence"`
	MustReportOOS       bool   `json:"must_report_oos"`
}

// DefaultGovernanceConfig returns sensible default governance settings
func DefaultGovernanceConfig() *GovernanceConfig {
	return &GovernanceConfig{
		ConstitutionVersion: "1.0",
		MaxToolCalls:        100,
		MustCiteEvidence:    true,
		MustReportOOS:       true,
	}
}

// TaskEnvelope contains all metadata and constraints for a governed task
type TaskEnvelope struct {
	TaskID      string            `json:"task_id"`
	Phase       string            `json:"phase"`
	EntryAgent  string            `json:"entry_agent"`
	Objective   string            `json:"objective"`
	ScopeIn     []string          `json:"scope_in"`
	ScopeOut    []string          `json:"scope_out"`
	Constraints []string          `json:"constraints"`
	Inputs      map[string]any    `json:"inputs"`
	Governance  *GovernanceConfig `json:"governance"`
	Provenance  map[string]any    `json:"provenance"`
}

// GovernanceWrapper wraps an OmOServeClient with governance enforcement
type GovernanceWrapper struct {
	client        *OmOServeClient
	strikeTracker *StrikeTracker
	enabled       bool
}

// NewGovernanceWrapper creates a new governance wrapper
func NewGovernanceWrapper(client *OmOServeClient, policy *StrikePolicy, enabled bool) *GovernanceWrapper {
	var tracker *StrikeTracker
	if enabled {
		tracker = NewStrikeTracker(policy)
	}
	return &GovernanceWrapper{
		client:        client,
		strikeTracker: tracker,
		enabled:       enabled,
	}
}

// PreCall validates task constraints before execution
func (gw *GovernanceWrapper) PreCall(ctx context.Context, envelope TaskEnvelope) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Apply default governance config if not provided
	if envelope.Governance == nil {
		envelope.Governance = DefaultGovernanceConfig()
	}

	// Validate required fields
	if envelope.TaskID == "" {
		return fmt.Errorf("governance: task_id is required")
	}

	if envelope.Phase == "" {
		return fmt.Errorf("governance: phase is required")
	}

	if envelope.EntryAgent == "" {
		return fmt.Errorf("governance: entry_agent is required")
	}

	// Validate scope consistency
	if len(envelope.ScopeIn) == 0 && len(envelope.ScopeOut) == 0 {
		return fmt.Errorf("governance: at least one scope constraint is required")
	}

	// Validate scope_out doesn't conflict with scope_in (simple check: no exact matches)
	scopeInMap := make(map[string]bool)
	for _, s := range envelope.ScopeIn {
		scopeInMap[s] = true
	}
	for _, s := range envelope.ScopeOut {
		if scopeInMap[s] {
			return fmt.Errorf("governance: scope_out '%s' conflicts with scope_in", s)
		}
	}

	return nil
}

// PostCall evaluates execution results and determines the outcome
func (gw *GovernanceWrapper) PostCall(
	ctx context.Context,
	envelope TaskEnvelope,
	evidence EvidenceEnvelope,
	oos OutOfScopeReport,
) (Outcome, error) {
	if ctx.Err() != nil {
		return OutcomeCanceled, ctx.Err()
	}

	// Check for escalation first
	if gw.enabled && gw.strikeTracker != nil && gw.strikeTracker.ShouldBlock() {
		return OutcomeEscalated, fmt.Errorf("governance: execution blocked due to strike threshold exceeded")
	}

	// Check verdict first — cancelled/failed take priority over OOS
	switch evidence.Verdict {
	case "cancelled":
		return OutcomeCanceled, nil
	case "qa:fail":
		if !oos.Clean {
			return OutcomeOutOfScope, nil
		}
		return OutcomeFailed, nil
	case "qa:pass":
		// Check for out of scope violations even on pass
		if !oos.Clean {
			return OutcomeOutOfScope, nil
		}
		// Verify evidence completeness
		if envelope.Governance.MustCiteEvidence && len(evidence.ToolCalls) == 0 && len(evidence.Findings) == 0 {
			return OutcomeIncomplete, fmt.Errorf("governance: evidence citation required but none provided")
		}
		return OutcomeSucceeded, nil
	default:
		// Unknown verdict — always incomplete regardless of OOS
		// OOS for unknown verdicts is not actionable
		return OutcomeIncomplete, fmt.Errorf("governance: unknown verdict '%s'", evidence.Verdict)
	}
}

// RecordFailure records a failure with the strike tracker
func (gw *GovernanceWrapper) RecordFailure(err error) {
	if !gw.enabled || gw.strikeTracker == nil {
		return
	}

	failure := ClassifyError(err)
	gw.strikeTracker.Record(failure)
}

// ShouldBlock returns true if execution should be blocked
func (gw *GovernanceWrapper) ShouldBlock() bool {
	if !gw.enabled || gw.strikeTracker == nil {
		return false
	}
	return gw.strikeTracker.ShouldBlock()
}

// GetStrikeCounts returns current strike counts
func (gw *GovernanceWrapper) GetStrikeCounts() (transportRetries, qualityStrikes, policyStrikes int) {
	if !gw.enabled || gw.strikeTracker == nil {
		return 0, 0, 0
	}
	return gw.strikeTracker.GetCounts()
}

// ResetStrikes clears all strike counters
func (gw *GovernanceWrapper) ResetStrikes() {
	if gw.enabled && gw.strikeTracker != nil {
		gw.strikeTracker.Reset()
	}
}

// IsEnabled returns true if governance is enabled
func (gw *GovernanceWrapper) IsEnabled() bool {
	return gw.enabled
}

// Client returns the underlying OmOServeClient
func (gw *GovernanceWrapper) Client() *OmOServeClient {
	return gw.client
}
