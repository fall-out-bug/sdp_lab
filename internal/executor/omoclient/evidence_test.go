package omoclient

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvidenceBuilder(t *testing.T) {
	builder := NewEvidenceBuilder()

	builder.
		SetSessionID("session-123").
		SetDispatchID("dispatch-456").
		SetFeatureID("F001").
		SetPhase("build").
		SetAgent("test-agent").
		RecordEvent(OmOEvent{Class: EventToolStarted, Data: "tool1"}).
		RecordEvent(OmOEvent{Class: EventToolCompleted, Data: "tool1"}).
		SetVerdict("qa:pass").
		AddFinding("warning", "test finding", "test").
		SetTokenUsage(100, 200, 300)

	time.Sleep(10 * time.Millisecond)
	envelope := builder.Build()

	if envelope.SessionID != "session-123" {
		t.Errorf("Expected session ID 'session-123', got '%s'", envelope.SessionID)
	}

	if envelope.DispatchID != "dispatch-456" {
		t.Errorf("Expected dispatch ID 'dispatch-456', got '%s'", envelope.DispatchID)
	}

	if envelope.FeatureID != "F001" {
		t.Errorf("Expected feature ID 'F001', got '%s'", envelope.FeatureID)
	}

	if envelope.Phase != "build" {
		t.Errorf("Expected phase 'build', got '%s'", envelope.Phase)
	}

	if envelope.Agent != "test-agent" {
		t.Errorf("Expected agent 'test-agent', got '%s'", envelope.Agent)
	}

	if len(envelope.ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls, got %d", len(envelope.ToolCalls))
	}

	if envelope.Verdict != "qa:pass" {
		t.Errorf("Expected verdict 'qa:pass', got '%s'", envelope.Verdict)
	}

	if len(envelope.Findings) != 1 {
		t.Errorf("Expected 1 finding, got %d", len(envelope.Findings))
	}

	if envelope.TokenUsage.PromptTokens != 100 {
		t.Errorf("Expected 100 prompt tokens, got %d", envelope.TokenUsage.PromptTokens)
	}

	if envelope.DurationMs <= 0 {
		t.Error("Expected duration > 0ms")
	}
}

func TestEvidenceBuilderEmpty(t *testing.T) {
	builder := NewEvidenceBuilder()
	envelope := builder.Build()

	if envelope.SessionID != "" {
		t.Errorf("Expected empty session ID, got '%s'", envelope.SessionID)
	}

	if envelope.Verdict != "" {
		t.Errorf("Expected empty verdict, got '%s'", envelope.Verdict)
	}
}

func TestClassifyOutcome(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		events   []OmOEvent
		expected string
	}{
		{
			name:     "successful completion",
			exitCode: 0,
			events:   []OmOEvent{{Class: EventCompletionSucceeded}},
			expected: "qa:pass",
		},
		{
			name:     "failed completion",
			exitCode: 0,
			events:   []OmOEvent{{Class: EventCompletionFailed}},
			expected: "qa:fail",
		},
		{
			name:     "non-zero exit code",
			exitCode: 1,
			events:   []OmOEvent{},
			expected: "qa:fail",
		},
		{
			name:     "cancelled",
			exitCode: 130,
			events:   []OmOEvent{},
			expected: "cancelled",
		},
		{
			name:     "other non-zero exit",
			exitCode: 2,
			events:   []OmOEvent{},
			expected: "qa:fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyOutcome(tt.exitCode, tt.events)
			if result != tt.expected {
				t.Errorf("ClassifyOutcome(exitCode=%d): expected %s, got %s", tt.exitCode, tt.expected, result)
			}
		})
	}
}

func TestEvidenceEnvelopeJSON(t *testing.T) {
	envelope := EvidenceEnvelope{
		SessionID:  "test-session",
		DispatchID: "test-dispatch",
		FeatureID:  "F001",
		Phase:      "build",
		Agent:      "test-agent",
		Verdict:    "qa:pass",
		Findings: []Finding{
			{Severity: "warning", Message: "test finding"},
		},
		TokenUsage: TokenUsage{
			PromptTokens:     100,
			CompletionTokens: 200,
			TotalTokens:      300,
		},
		DurationMs: 1000,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("Marshal envelope failed: %v", err)
	}

	var unmarshaled EvidenceEnvelope
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal envelope failed: %v", err)
	}

	if unmarshaled.SessionID != envelope.SessionID {
		t.Errorf("SessionID mismatch after marshal/unmarshal")
	}

	if unmarshaled.Verdict != envelope.Verdict {
		t.Errorf("Verdict mismatch after marshal/unmarshal")
	}
}

func TestEvidenceDuration(t *testing.T) {
	builder := NewEvidenceBuilder()
	time.Sleep(10 * time.Millisecond)
	envelope := builder.Build()

	if envelope.DurationMs <= 0 {
		t.Errorf("Expected duration > 0ms, got %dms", envelope.DurationMs)
	}
}

func TestFindings(t *testing.T) {
	builder := NewEvidenceBuilder()

	builder.AddFinding("error", "test error 1", "source1")
	builder.AddFinding("warning", "test warning", "source2")

	envelope := builder.Build()

	if len(envelope.Findings) != 2 {
		t.Fatalf("Expected 2 findings, got %d", len(envelope.Findings))
	}

	if envelope.Findings[0].Severity != "error" {
		t.Errorf("Expected severity 'error', got '%s'", envelope.Findings[0].Severity)
	}

	if envelope.Findings[1].Source != "source2" {
		t.Errorf("Expected source 'source2', got '%s'", envelope.Findings[1].Source)
	}
}
