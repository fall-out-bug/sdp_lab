package sdk

import (
	"testing"
	"time"
)

func TestNewValidator(t *testing.T) {
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}
	if validator == nil {
		t.Fatal("NewValidator() returned nil validator")
	}
}

func TestValidateOrchestrationEvent(t *testing.T) {
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	tests := []struct {
		name    string
		event   *OrchestrationEvent
		wantErr bool
	}{
		{
			name: "valid event",
			event: &OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Source: EventSource{
					System:    "omo",
					Component: "agent",
				},
				EventType: "task.completed",
				Payload: map[string]interface{}{
					"task_id": "task-123",
				},
			},
			wantErr: false,
		},
		{
			name: "missing spec version",
			event: &OrchestrationEvent{
				EventID:   "550e8400-e29b-41d4-a716-446655440000",
				Timestamp: time.Now(),
				Source: EventSource{
					System:    "omo",
					Component: "agent",
				},
				EventType: "task.completed",
			},
			wantErr: true,
		},
		{
			name: "missing event id",
			event: &OrchestrationEvent{
				SpecVersion: "v1.0",
				Timestamp:   time.Now(),
				Source: EventSource{
					System:    "omo",
					Component: "agent",
				},
				EventType: "task.completed",
			},
			wantErr: true,
		},
		{
			name: "missing event type",
			event: &OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Source: EventSource{
					System:    "omo",
					Component: "agent",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateOrchestrationEvent(tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrchestrationEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateRuntimeDecision(t *testing.T) {
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	tests := []struct {
		name     string
		decision *RuntimeDecision
		wantErr  bool
	}{
		{
			name: "valid allow decision",
			decision: &RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "660e8400-e29b-41d4-a716-446655440001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: DecisionReason{
					Code:    "within_scope",
					Message: "Task is within declared scope",
				},
				Context: DecisionContext{
					WorkstreamID: "00-061-01",
					FeatureID:    "F061",
				},
			},
			wantErr: false,
		},
		{
			name: "valid ask decision",
			decision: &RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "660e8400-e29b-41d4-a716-446655440002",
				Timestamp:    time.Now(),
				DecisionType: "security.approval",
				Decision:     "ask",
				Reason: DecisionReason{
					Code:    "sensitive_path",
					Message: "Access to sensitive file requires approval",
				},
				Context: DecisionContext{
					WorkstreamID: "00-061-02",
					FeatureID:    "F061",
				},
			},
			wantErr: false,
		},
		{
			name: "valid deny decision",
			decision: &RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "660e8400-e29b-41d4-a716-446655440003",
				Timestamp:    time.Now(),
				DecisionType: "resource.limit",
				Decision:     "deny",
				Reason: DecisionReason{
					Code:    "resource_exceeded",
					Message: "Resource limit exceeded",
				},
				Context: DecisionContext{
					WorkstreamID: "00-068-01",
					FeatureID:    "F068",
				},
			},
			wantErr: false,
		},
		{
			name: "missing spec version",
			decision: &RuntimeDecision{
				DecisionID:   "660e8400-e29b-41d4-a716-446655440001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: DecisionReason{
					Code:    "within_scope",
					Message: "Task is within declared scope",
				},
			},
			wantErr: true,
		},
		{
			name: "missing decision id",
			decision: &RuntimeDecision{
				SpecVersion:  "v1.0",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: DecisionReason{
					Code:    "within_scope",
					Message: "Task is within declared scope",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid decision value",
			decision: &RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "660e8400-e29b-41d4-a716-446655440001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "invalid",
				Reason: DecisionReason{
					Code:    "within_scope",
					Message: "Task is within declared scope",
				},
			},
			wantErr: true,
		},
		{
			name: "missing decision type",
			decision: &RuntimeDecision{
				SpecVersion: "v1.0",
				DecisionID:  "660e8400-e29b-41d4-a716-446655440001",
				Timestamp:   time.Now(),
				Decision:    "allow",
				Reason: DecisionReason{
					Code:    "within_scope",
					Message: "Task is within declared scope",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRuntimeDecision(tt.decision)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRuntimeDecision() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatorWithNilInputs(t *testing.T) {
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() failed: %v", err)
	}

	t.Run("nil event", func(t *testing.T) {
		err := validator.ValidateOrchestrationEvent(nil)
		if err == nil {
			t.Error("ValidateOrchestrationEvent(nil) should return error")
		}
	})

	t.Run("nil decision", func(t *testing.T) {
		err := validator.ValidateRuntimeDecision(nil)
		if err == nil {
			t.Error("ValidateRuntimeDecision(nil) should return error")
		}
	})
}
