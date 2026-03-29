// Package sdk provides adapter SDK interfaces for contract producers and consumers.
package sdk

import (
	"time"
)

// OrchestrationEvent represents a workflow orchestration event.
type OrchestrationEvent struct {
	SpecVersion string                 `json:"spec_version"`
	EventID     string                 `json:"event_id"`
	Timestamp   time.Time              `json:"timestamp"`
	Source      EventSource            `json:"source"`
	EventType   string                 `json:"event_type"`
	Payload     map[string]interface{} `json:"payload"`
	Metadata    *EventMetadata         `json:"metadata,omitempty"`
	Context     *ExecutionContext      `json:"context,omitempty"`
}

// EventSource identifies the source of an event.
type EventSource struct {
	System    string `json:"system"`
	Component string `json:"component"`
	Version   string `json:"version,omitempty"`
}

// EventMetadata contains event correlation and tracing information.
type EventMetadata struct {
	CorrelationID string            `json:"correlation_id,omitempty"`
	CausationID   string            `json:"causation_id,omitempty"`
	TraceContext  *TraceContext     `json:"trace_context,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// TraceContext contains distributed tracing information.
type TraceContext struct {
	Traceparent string `json:"traceparent,omitempty"`
	Tracestate  string `json:"tracestate,omitempty"`
}

// ExecutionContext provides execution context for an event.
type ExecutionContext struct {
	WorkstreamID string      `json:"workstream_id,omitempty"`
	FeatureID    string      `json:"feature_id,omitempty"`
	BeadsID      string      `json:"beads_id,omitempty"`
	SessionID    string      `json:"session_id,omitempty"`
	GitContext   *GitContext `json:"git_context,omitempty"`
}

// GitContext contains git repository information.
type GitContext struct {
	Branch    string `json:"branch,omitempty"`
	CommitSHA string `json:"commit_sha,omitempty"`
	RepoURL   string `json:"repo_url,omitempty"`
}

// RuntimeDecision represents a governance decision.
type RuntimeDecision struct {
	SpecVersion     string             `json:"spec_version"`
	DecisionID      string             `json:"decision_id"`
	Timestamp       time.Time          `json:"timestamp"`
	DecisionType    string             `json:"decision_type"`
	Decision        string             `json:"decision"` // allow, ask, deny
	Reason          DecisionReason     `json:"reason"`
	Context         DecisionContext    `json:"context"`
	PolicyReference *PolicyReference   `json:"policy_reference,omitempty"`
	Evidence        []DecisionEvidence `json:"evidence,omitempty"`
	Overrides       *DecisionOverride  `json:"overrides,omitempty"`
}

// DecisionReason explains why a decision was made.
type DecisionReason struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// DecisionContext provides context for a decision.
type DecisionContext struct {
	Request      DecisionRequest `json:"request"`
	Actor        *Actor          `json:"actor,omitempty"`
	Environment  string          `json:"environment,omitempty"`
	WorkstreamID string          `json:"workstream_id,omitempty"`
	FeatureID    string          `json:"feature_id,omitempty"`
	SessionID    string          `json:"session_id,omitempty"`
}

// DecisionRequest represents the original request that triggered a decision.
type DecisionRequest struct {
	Action     string                 `json:"action,omitempty"`
	Resource   string                 `json:"resource,omitempty"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// Actor represents who or what initiated a request.
type Actor struct {
	Type  string   `json:"type"` // agent, user, system
	ID    string   `json:"id"`
	Roles []string `json:"roles,omitempty"`
}

// PolicyReference references the policy that informed a decision.
type PolicyReference struct {
	PolicyID      string `json:"policy_id,omitempty"`
	PolicyVersion string `json:"policy_version,omitempty"`
	RuleName      string `json:"rule_name,omitempty"`
	OPADecisionID string `json:"opa_decision_id,omitempty"`
}

// DecisionEvidence provides evidence supporting a decision.
type DecisionEvidence struct {
	Type      string `json:"type"`
	Reference string `json:"reference"`
	Summary   string `json:"summary,omitempty"`
}

// DecisionOverride contains override information if a decision was overridden.
type DecisionOverride struct {
	Overridden      bool      `json:"overridden"`
	OverriddenBy    string    `json:"overridden_by,omitempty"`
	OverrideReason  string    `json:"override_reason,omitempty"`
	OverrideExpires time.Time `json:"override_expires,omitempty"`
}
