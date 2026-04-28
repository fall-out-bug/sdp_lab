package policy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
)

// ExplainResult provides a human-readable explanation of a decision
type ExplainResult struct {
	Decision   string            `json:"decision"`   // allow, ask, deny
	Reason     string            `json:"reason"`     // Primary reason for the decision
	RuleID     string            `json:"rule_id"`    // Matched rule identifier
	Evidence   []EvidenceRef     `json:"evidence"`   // Supporting evidence references
	Context    map[string]string `json:"context"`    // Additional context
	Sanitized  bool              `json:"sanitized"`  // True if internal details are excluded
}

// EvidenceRef references a piece of evidence supporting the decision
type EvidenceRef struct {
	Type    string `json:"type"`    // attestation, discrepancy, boundary, etc.
	Path    string `json:"path"`    // Path to the evidence file
	Summary string `json:"summary"` // Brief summary of the evidence
}

// ExplainMode controls the level of detail in explanations
type ExplainMode int

const (
	// ExplainModeBasic provides a simple one-line explanation
	ExplainModeBasic ExplainMode = iota
	// ExplainModeDetailed provides full explanation with evidence references
	ExplainModeDetailed
	// ExplainModeInternal includes internal stack traces (not for user-facing output)
	ExplainModeInternal
)

// ExplainDecision explains an allow/ask/deny decision
func ExplainDecision(allowed bool, reasons []string, severityCount map[string]int, mode ExplainMode) ExplainResult {
	explanation := ExplainResult{
		Decision:  decisionString(allowed),
		Reason:    primaryReasonFromStrings(reasons),
		Context:   make(map[string]string),
		Sanitized: mode != ExplainModeInternal,
	}

	// Add evidence references
	if mode == ExplainModeDetailed || mode == ExplainModeInternal {
		explanation.Evidence = buildEvidenceRefsFromCount(severityCount)
	}

	// Add context
	for key, value := range severityCount {
		explanation.Context[key] = fmt.Sprintf("%d", value)
	}

	return explanation
}

// ExplainAttestation explains the rationale behind an attestation decision
func ExplainAttestation(stmt evidence.CodingWorkflowStatement, mode ExplainMode) ExplainResult {
	explanation := ExplainResult{
		Decision:  "allow",
		Reason:    "attestation is valid and compliant",
		Context:   make(map[string]string),
		Sanitized: mode != ExplainModeInternal,
	}

	// Check boundary compliance
	if !stmt.Predicate.Boundary.Compliance.OK {
		explanation.Decision = "deny"
		explanation.Reason = fmt.Sprintf("boundary violation: %s", stmt.Predicate.Boundary.Compliance.Reason)
		explanation.RuleID = "boundary-compliance"
	}

	// Check verification results
	if hasTestFailures(stmt.Predicate.Verification.Tests) {
		if explanation.Decision == "allow" {
			explanation.Decision = "ask"
		}
		explanation.Reason = "test failures detected - review required"
		explanation.RuleID = "test-verification"
	}

	// Add context
	explanation.Context["run_id"] = stmt.Predicate.Provenance.RunID
	explanation.Context["feature_id"] = stmt.Predicate.Intent.IssueID
	explanation.Context["branch"] = stmt.Predicate.Execution.Branch
	explanation.Context["orchestrator"] = stmt.Predicate.Provenance.Orchestrator

	return explanation
}

// ExplainDiscrepancy explains a discrepancy report
func ExplainDiscrepancy(report evidence.DiscrepancyReport, mode ExplainMode) ExplainResult {
	explanation := ExplainResult{
		Decision:  "allow",
		Reason:    report.Summary,
		Context:   make(map[string]string),
		Sanitized: mode != ExplainModeInternal,
	}

	if !report.OK {
		explanation.Decision = "deny"
		explanation.Reason = fmt.Sprintf("critical or high discrepancies: %s", report.Summary)
		explanation.RuleID = "discrepancy-threshold"
	}

	explanation.Context["run_id"] = report.RunID
	if report.AgentFile != "" {
		explanation.Context["agent_attestation"] = report.AgentFile
	}
	if report.CIFile != "" {
		explanation.Context["ci_attestation"] = report.CIFile
	}

	// Add evidence references for each discrepancy
	for _, d := range report.Discrepancies {
		explanation.Evidence = append(explanation.Evidence, EvidenceRef{
			Type:    string(d.Type),
			Summary: d.Description,
		})
	}

	return explanation
}

// FormatExplanation formats an explanation for display
func FormatExplanation(explanation ExplainResult) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Decision: %s\n", explanation.Decision))
	sb.WriteString(fmt.Sprintf("Reason: %s\n", explanation.Reason))

	if explanation.RuleID != "" {
		sb.WriteString(fmt.Sprintf("Rule: %s\n", explanation.RuleID))
	}

	if len(explanation.Evidence) > 0 {
		sb.WriteString("\nEvidence:\n")
		for i, ref := range explanation.Evidence {
			sb.WriteString(fmt.Sprintf("  %d. %s", i+1, ref.Type))
			if ref.Summary != "" {
				sb.WriteString(fmt.Sprintf(": %s", ref.Summary))
			}
			if ref.Path != "" {
				sb.WriteString(fmt.Sprintf(" (%s)", ref.Path))
			}
			sb.WriteString("\n")
		}
	}

	if len(explanation.Context) > 0 {
		sb.WriteString("\nContext:\n")
		for key, value := range explanation.Context {
			sb.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	return sb.String()
}

// ExplainToJSON converts an explanation to JSON
func ExplainToJSON(explanation ExplainResult) (string, error) {
	data, err := json.MarshalIndent(explanation, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal explanation: %w", err)
	}
	return string(data), nil
}

// ParseExplainFromJSON parses an explanation from JSON
func ParseExplainFromJSON(data []byte) (ExplainResult, error) {
	var explanation ExplainResult
	if err := json.Unmarshal(data, &explanation); err != nil {
		return ExplainResult{}, fmt.Errorf("parse explanation: %w", err)
	}
	return explanation, nil
}

// Helper functions

func decisionString(allowed bool) string {
	if allowed {
		return "allow"
	}
	return "deny"
}

func primaryReasonFromStrings(reasons []string) string {
	if len(reasons) == 0 {
		return "all checks passed"
	}
	return reasons[0]
}

func buildEvidenceRefsFromCount(severityCount map[string]int) []EvidenceRef {
	var refs []EvidenceRef

	// Add severity counts as evidence
	if severityCount != nil {
		for severity, count := range severityCount {
			if count > 0 {
				refs = append(refs, EvidenceRef{
					Type:    "discrepancy-severity",
					Summary: fmt.Sprintf("%d %s discrepancies", count, severity),
				})
			}
		}
	}

	return refs
}

func hasTestFailures(tests []evidence.GateResult) bool {
	for _, test := range tests {
		if strings.HasPrefix(strings.ToLower(test.Status), "fail") {
			return true
		}
	}
	return false
}

// Decision represents a policy decision with explanation
type Decision struct {
	Allowed     bool         `json:"allowed"`
	Explanation ExplainResult `json:"explanation"`
	Timestamp   string       `json:"timestamp,omitempty"`
}

// NewDecision creates a new decision with explanation
func NewDecision(allowed bool, explanation ExplainResult) Decision {
	return Decision{
		Allowed:     allowed,
		Explanation: explanation,
	}
}

// Format formats the decision for display
func (d Decision) Format() string {
	var sb strings.Builder

	status := "✓ ALLOW"
	if !d.Allowed {
		status = "✗ DENY"
	}

	sb.WriteString(fmt.Sprintf("%s\n", status))
	sb.WriteString(FormatExplanation(d.Explanation))

	return sb.String()
}

// ToJSON converts the decision to JSON
func (d Decision) ToJSON() (string, error) {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal decision: %w", err)
	}
	return string(data), nil
}
