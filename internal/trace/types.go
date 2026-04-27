package trace

import (
	"encoding/json"
	"time"
)

// SpanKind represents the type of span
type SpanKind string

const (
	SpanKindExecuteTool       SpanKind = "execute_tool"
	SpanKindInvokeAgent       SpanKind = "invoke_agent"
	SpanKindDeliveryLoopPhase SpanKind = "delivery_loop_phase"
	SpanKindSDPBeadEvent      SpanKind = "sdp_bead_event"
)

// ConsentLevel represents the telemetry consent level
type ConsentLevel string

const (
	ConsentLevelNone     ConsentLevel = "none"
	ConsentLevelMetadata ConsentLevel = "metadata"
	ConsentLevelFindings ConsentLevel = "findings"
	ConsentLevelContent  ConsentLevel = "content"
)

// IsValid checks if the consent level is valid
func (c ConsentLevel) IsValid() bool {
	switch c {
	case ConsentLevelNone, ConsentLevelMetadata, ConsentLevelFindings, ConsentLevelContent:
		return true
	default:
		return false
	}
}

// AllowsExport returns true if the consent level permits any outbound export.
// Only metadata, findings, and content levels allow export; "none" blocks it.
func (c ConsentLevel) AllowsExport() bool {
	switch c {
	case ConsentLevelMetadata, ConsentLevelFindings, ConsentLevelContent:
		return true
	default:
		return false
	}
}

// Span represents a telemetry span
type Span struct {
	TraceID    string            `json:"trace_id"`
	SpanID     string            `json:"span_id"`
	ParentID   string            `json:"parent_id,omitempty"`
	SpanKind   SpanKind          `json:"span_kind"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time,omitempty"`
	DurationMs int64             `json:"duration_ms,omitempty"`
	Attributes map[string]string `json:"attributes"`
	Events     []SpanEvent       `json:"events,omitempty"`
	Status     SpanStatus        `json:"status"`
}

// SpanStatus represents the span status
type SpanStatus string

const (
	SpanStatusOK    SpanStatus = "ok"
	SpanStatusError SpanStatus = "error"
)

// SpanEvent represents an event within a span
type SpanEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	Name        string            `json:"name"`
	Attributes  map[string]string `json:"attributes,omitempty"`
}

// SpanHandle represents an active span in the daemon registry
type SpanHandle struct {
	Span        *Span
	TraceID     string
	ToolCallID  string
	StartTime   time.Time
}

// DaemonRequest represents a request to the daemon
type DaemonRequest struct {
	Command    string            `json:"command"` // span-start, span-end, event, shutdown
	SpanID     string            `json:"span_id,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
	Event      SpanEvent         `json:"event,omitempty"`
}

// DaemonResponse represents a response from the daemon
type DaemonResponse struct {
	Success bool   `json:"success"`
	SpanID  string `json:"span_id,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TraceParent represents W3C Trace Context
type TraceParent struct {
	Version    byte
	TraceID    string
	ParentID   string
	Flags      byte
}

// ParseTraceParent parses a TRACEPARENT header string
func ParseTraceParent(tp string) (*TraceParent, error) {
	// Format: version-traceid-parentid-flags
	// Example: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
	// For MVP, we'll generate new trace IDs rather than parse
	return nil, nil
}

// String formats the TraceParent as a W3C header
func (tp *TraceParent) String() string {
	// For MVP, generate simplified format
	return ""
}

// AttributeFilter filters attributes based on consent level
type AttributeFilter struct {
	allowlist   map[string]map[string]bool // span_kind -> attr_name -> allowed
	consentLevel ConsentLevel
}

// NewAttributeFilter creates a new attribute filter
func NewAttributeFilter(schema Schema, consentLevel ConsentLevel) *AttributeFilter {
	filter := &AttributeFilter{
		allowlist:   make(map[string]map[string]bool),
		consentLevel: consentLevel,
	}

	// Build allowlist from schema
	for spanKind, spanDef := range schema.SpanKinds {
		filter.allowlist[spanKind] = make(map[string]bool)
		for attrName, attrDef := range spanDef.AllowedAttributes {
			// Check if attribute requires higher consent level
			if attrDef.ConsentLevel != "" {
				attrConsent := ConsentLevel(attrDef.ConsentLevel)
				if !consentSufficient(consentLevel, attrConsent) {
					continue // Skip this attribute
				}
			}
			filter.allowlist[spanKind][attrName] = true
		}
	}

	return filter
}

// consentSufficient checks if current consent level is sufficient for required level
func consentSufficient(current, required ConsentLevel) bool {
	levels := map[ConsentLevel]int{
		ConsentLevelNone:     0,
		ConsentLevelMetadata: 1,
		ConsentLevelFindings: 2,
		ConsentLevelContent:  3,
	}
	return levels[current] >= levels[required]
}

// Filter returns only allowed attributes for the given span kind
func (af *AttributeFilter) Filter(spanKind string, attrs map[string]string) map[string]string {
	filtered := make(map[string]string)
	allowed := af.allowlist[spanKind]

	for key, value := range attrs {
		if allowed[key] {
			filtered[key] = value
		}
	}

	return filtered
}

// IsAllowed checks if an attribute is allowed for the given span kind
func (af *AttributeFilter) IsAllowed(spanKind, attrName string) bool {
	allowed := af.allowlist[spanKind]
	return allowed[attrName]
}

// Schema represents the telemetry schema
type Schema struct {
	SpanKinds    map[string]SpanKindDef    `json:"span_kinds"`
	ConsentLevels map[string]ConsentLevelDef `json:"consent_levels"`
	SamplingPolicy SamplingPolicyDef        `json:"sampling_policy"`
}

// SpanKindDef defines a span kind
type SpanKindDef struct {
	AllowedAttributes map[string]AttributeDef `json:"allowed_attributes"`
}

// AttributeDef defines an attribute
type AttributeDef struct {
	Type         string `json:"type"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	ConsentLevel string `json:"consent_level,omitempty"`
}

// ConsentLevelDef defines a consent level
type ConsentLevelDef struct {
	Description      string   `json:"description"`
	ExampleAttributes []string `json:"example_attributes,omitempty"`
}

// SamplingPolicyDef defines the sampling policy
type SamplingPolicyDef struct {
	HeadBased  HeadBasedDef  `json:"head_based"`
	TailBased  TailBasedDef  `json:"tail_based"`
	HashBased  HashBasedDef  `json:"hash_based"`
}

// HeadBasedDef defines head-based sampling
type HeadBasedDef struct {
	DefaultRate float64 `json:"default_rate"`
}

// TailBasedDef defines tail-based sampling
type TailBasedDef struct {
	DropRules []DropRuleDef `json:"drop_rules,omitempty"`
}

// DropRuleDef defines a drop rule
type DropRuleDef struct {
	ToolName    string `json:"tool_name"`
	MaxDurationMs int   `json:"max_duration_ms"`
	RequireError bool  `json:"require_error"`
}

// HashBasedDef defines hash-based sampling
type HashBasedDef struct {
	ThresholdSpans int     `json:"threshold_spans"`
	SampleRate     float64 `json:"sample_rate"`
}

// LoadSchema loads the telemetry schema from JSON
func LoadSchema(data []byte) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// SessionMetadata represents session metadata persisted to .sdp/traces/current.env
type SessionMetadata struct {
	SessionID   string `json:"session_id"`
	TraceID     string `json:"trace_id"`
	StartTime   string `json:"start_time"`
	EpicBeadID  string `json:"epic_bead_id"`
	Harness     string `json:"harness"`
}
