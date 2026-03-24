package omoclient

import (
	"time"
)

// Finding represents a single finding from evidence collection
type Finding struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Source   string `json:"source,omitempty"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// EvidenceEnvelope contains evidence from an OmO serve execution
type EvidenceEnvelope struct {
	SessionID  string     `json:"session_id"`
	DispatchID string     `json:"dispatch_id,omitempty"`
	FeatureID  string     `json:"feature_id,omitempty"`
	Phase      string     `json:"phase,omitempty"`
	Agent      string     `json:"agent,omitempty"`
	ToolCalls  []OmOEvent `json:"tool_calls,omitempty"`
	Verdict    string     `json:"verdict"`
	Findings   []Finding  `json:"findings,omitempty"`
	TokenUsage TokenUsage `json:"token_usage,omitempty"`
	DurationMs int64      `json:"duration_ms,omitempty"`
	RawEvents  []OmOEvent `json:"raw_events,omitempty"`
}

// EvidenceBuilder accumulates evidence and builds envelopes
type EvidenceBuilder struct {
	envelope  EvidenceEnvelope
	startTime time.Time
}

// NewEvidenceBuilder creates a new evidence builder
func NewEvidenceBuilder() *EvidenceBuilder {
	return &EvidenceBuilder{
		envelope: EvidenceEnvelope{
			Findings:  []Finding{},
			ToolCalls: []OmOEvent{},
			RawEvents: []OmOEvent{},
		},
		startTime: time.Now(),
	}
}

// SetSessionID sets the session ID
func (b *EvidenceBuilder) SetSessionID(id string) *EvidenceBuilder {
	b.envelope.SessionID = id
	return b
}

// SetDispatchID sets the dispatch ID
func (b *EvidenceBuilder) SetDispatchID(id string) *EvidenceBuilder {
	b.envelope.DispatchID = id
	return b
}

// SetFeatureID sets the feature ID
func (b *EvidenceBuilder) SetFeatureID(id string) *EvidenceBuilder {
	b.envelope.FeatureID = id
	return b
}

// SetPhase sets the phase
func (b *EvidenceBuilder) SetPhase(phase string) *EvidenceBuilder {
	b.envelope.Phase = phase
	return b
}

// SetAgent sets the agent
func (b *EvidenceBuilder) SetAgent(agent string) *EvidenceBuilder {
	b.envelope.Agent = agent
	return b
}

// RecordEvent records an event
func (b *EvidenceBuilder) RecordEvent(event OmOEvent) *EvidenceBuilder {
	b.envelope.RawEvents = append(b.envelope.RawEvents, event)
	if event.Class == EventToolStarted || event.Class == EventToolCompleted {
		b.envelope.ToolCalls = append(b.envelope.ToolCalls, event)
	}
	return b
}

// SetVerdict sets the verdict
func (b *EvidenceBuilder) SetVerdict(verdict string) *EvidenceBuilder {
	b.envelope.Verdict = verdict
	return b
}

// AddFinding adds a finding
func (b *EvidenceBuilder) AddFinding(severity, message, source string) *EvidenceBuilder {
	b.envelope.Findings = append(b.envelope.Findings, Finding{
		Severity: severity,
		Message:  message,
		Source:   source,
	})
	return b
}

// SetTokenUsage sets token usage
func (b *EvidenceBuilder) SetTokenUsage(prompt, completion, total int) *EvidenceBuilder {
	b.envelope.TokenUsage = TokenUsage{
		PromptTokens:     prompt,
		CompletionTokens: completion,
		TotalTokens:      total,
	}
	return b
}

// Build finalizes the envelope
func (b *EvidenceBuilder) Build() EvidenceEnvelope {
	b.envelope.DurationMs = time.Since(b.startTime).Milliseconds()
	return b.envelope
}

// ClassifyOutcome determines verdict based on exit code and events
func ClassifyOutcome(exitCode int, events []OmOEvent) string {
	for _, event := range events {
		if event.Class == EventCompletionFailed {
			return "qa:fail"
		}
	}

	if exitCode == 0 {
		return "qa:pass"
	}

	if exitCode == 130 {
		return "cancelled"
	}

	return "qa:fail"
}
