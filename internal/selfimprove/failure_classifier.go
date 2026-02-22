package selfimprove

import (
	"strings"
)

// FailureClass matches docs/RETRY_ESCALATION_POLICY.md
type FailureClass string

const (
	ClassTransient         FailureClass = "transient"
	ClassToolFlake         FailureClass = "tool_flake"
	ClassVerificationFail  FailureClass = "verification_fail"
	ClassPolicyConflict    FailureClass = "policy_conflict"
	ClassSecuritySensitive FailureClass = "security_sensitive"
	ClassUnknown           FailureClass = "unknown"
)

// ClassifiedFailure holds a classified failure with context.
type ClassifiedFailure struct {
	IssueID   string
	RunID     string
	Class     FailureClass
	Message   string
	Phase     string
	RetryCount int
}

// FailureClassifier classifies failures per retry policy.
type FailureClassifier struct{}

// NewFailureClassifier returns a new classifier.
func NewFailureClassifier() *FailureClassifier {
	return &FailureClassifier{}
}

// ClassifyRun classifies a run doc's terminal state.
func (c *FailureClassifier) ClassifyRun(doc RunDoc) *ClassifiedFailure {
	if doc.LastState == "ok" {
		return nil
	}
	msg := ""
	if len(doc.Events) > 0 {
		msg = doc.Events[len(doc.Events)-1].Message
	}
	class := c.classifyMessage(msg)
	return &ClassifiedFailure{
		IssueID: doc.IssueID,
		RunID:   doc.RunID,
		Class:   class,
		Message: msg,
		Phase:   doc.LastPhase,
	}
}

// ClassifyTelemetry classifies a telemetry record.
func (c *FailureClassifier) ClassifyTelemetry(rec TelemetryRecord) *ClassifiedFailure {
	if rec.Status != "failed" && rec.Status != "escalated" {
		return nil
	}
	class := ClassUnknown
	if rec.Escalated {
		class = ClassPolicyConflict
	}
	if rec.RetryCount >= 3 {
		class = ClassTransient
	}
	return &ClassifiedFailure{
		IssueID:    rec.IssueID,
		Class:      class,
		Phase:      rec.Phase,
		RetryCount: rec.RetryCount,
	}
}

func (c *FailureClassifier) classifyMessage(msg string) FailureClass {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "429") || strings.Contains(lower, "5xx"):
		return ClassTransient
	case strings.Contains(lower, "infra") || strings.Contains(lower, "flak"):
		return ClassToolFlake
	case strings.Contains(lower, "verification") || strings.Contains(lower, "go test"):
		return ClassVerificationFail
	case strings.Contains(lower, "policy") || strings.Contains(lower, "allowlist"):
		return ClassPolicyConflict
	case strings.Contains(lower, "security") || strings.Contains(lower, "auth"):
		return ClassSecuritySensitive
	default:
		return ClassUnknown
	}
}
