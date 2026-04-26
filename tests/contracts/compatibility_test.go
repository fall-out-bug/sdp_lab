// Package contracts provides compatibility tests for contract schemas
package contracts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/adapters/sdk"

	"github.com/stretchr/testify/require"
)

// TestOrchestrationEventV1Compatibility validates that v1 orchestration events
// can be unmarshaled and validated against the current SDK.
func TestOrchestrationEventV1Compatibility(t *testing.T) {
	fixturePath := filepath.Join("fixtures", "orchestration-event-v1.json")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "Failed to read fixture: %s", fixturePath)

	var event sdk.OrchestrationEvent
	err = json.Unmarshal(data, &event)
	require.NoError(t, err, "Failed to unmarshal fixture")

	// Validate required fields
	require.Equal(t, "v1.0", event.SpecVersion)
	require.NotEmpty(t, event.EventID)
	require.NotEmpty(t, event.EventType)
	require.NotZero(t, event.Timestamp)
	require.NotEmpty(t, event.Source.System)
	require.NotEmpty(t, event.Source.Component)

	// Validate against schema
	validator, err := sdk.NewValidator()
	require.NoError(t, err)

	err = validator.ValidateOrchestrationEvent(&event)
	require.NoError(t, err, "OrchestrationEvent v1 fixture failed validation")
}

// TestRuntimeDecisionV1Compatibility validates that v1 runtime decisions
// can be unmarshaled and validated against the current SDK.
func TestRuntimeDecisionV1Compatibility(t *testing.T) {
	fixturePath := filepath.Join("fixtures", "runtime-decision-v1.json")
	data, err := os.ReadFile(fixturePath)
	require.NoError(t, err, "Failed to read fixture: %s", fixturePath)

	var decision sdk.RuntimeDecision
	err = json.Unmarshal(data, &decision)
	require.NoError(t, err, "Failed to unmarshal fixture")

	// Validate required fields
	require.Equal(t, "v1.0", decision.SpecVersion)
	require.NotEmpty(t, decision.DecisionID)
	require.NotEmpty(t, decision.DecisionType)
	require.NotEmpty(t, decision.Decision)
	require.NotEmpty(t, decision.Reason.Code)
	require.NotEmpty(t, decision.Reason.Message)

	// Validate decision value is one of allowed values
	require.Contains(t, []string{"allow", "ask", "deny"}, decision.Decision)

	// Validate against schema
	validator, err := sdk.NewValidator()
	require.NoError(t, err)

	err = validator.ValidateRuntimeDecision(&decision)
	require.NoError(t, err, "RuntimeDecision v1 fixture failed validation")
}

// TestOrchestrationEventRoundTrip verifies JSON marshaling/unmarshaling preserves data.
func TestOrchestrationEventRoundTrip(t *testing.T) {
	original := &sdk.OrchestrationEvent{
		SpecVersion: "v1.0",
		EventID:     "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Source: sdk.EventSource{
			System:    "omo",
			Component: "agent",
			Version:   "1.0.0",
		},
		EventType: "task.completed",
		Payload: map[string]interface{}{
			"task_id":       "task-123",
			"feature_id":    "F068",
			"workstream_id": "00-068-02",
			"status":        "success",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var unmarshaled sdk.OrchestrationEvent
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	require.Equal(t, original.SpecVersion, unmarshaled.SpecVersion)
	require.Equal(t, original.EventID, unmarshaled.EventID)
	require.Equal(t, original.EventType, unmarshaled.EventType)
	require.Equal(t, original.Source.System, unmarshaled.Source.System)
	require.Equal(t, original.Source.Component, unmarshaled.Source.Component)
}

// TestRuntimeDecisionRoundTrip verifies JSON marshaling/unmarshaling preserves data.
func TestRuntimeDecisionRoundTrip(t *testing.T) {
	original := &sdk.RuntimeDecision{
		SpecVersion:  "v1.0",
		DecisionID:   "660e8400-e29b-41d4-a716-446655440001",
		Timestamp:    time.Date(2024, 1, 15, 10, 31, 0, 0, time.UTC),
		DecisionType: "scope.boundary",
		Decision:     "allow",
		Reason: sdk.DecisionReason{
			Code:    "WITHIN_SCOPE",
			Message: "Task is within declared scope boundaries",
		},
		Context: sdk.DecisionContext{
			Environment:  "development",
			WorkstreamID: "00-068-02",
			FeatureID:    "F068",
			SessionID:    "ses-abc123",
		},
	}

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var unmarshaled sdk.RuntimeDecision
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	require.Equal(t, original.SpecVersion, unmarshaled.SpecVersion)
	require.Equal(t, original.DecisionID, unmarshaled.DecisionID)
	require.Equal(t, original.DecisionType, unmarshaled.DecisionType)
	require.Equal(t, original.Decision, unmarshaled.Decision)
	require.Equal(t, original.Reason.Code, unmarshaled.Reason.Code)
	require.Equal(t, original.Reason.Message, unmarshaled.Reason.Message)
}

// AdapterMatrixTestCase represents a test case for adapter compatibility.
type AdapterMatrixTestCase struct {
	Name    string
	System  string
	Event   *sdk.OrchestrationEvent
	IsValid bool
}

// TestAdapterMatrix validates that OMO, Beads, and Gas Town adapters
// can produce valid contract payloads.
func TestAdapterMatrix(t *testing.T) {
	validator, err := sdk.NewValidator()
	require.NoError(t, err)

	testCases := []AdapterMatrixTestCase{
		{
			Name:   "OMO task.completed",
			System: "omo",
			Event: &sdk.OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "omo-event-001",
				Timestamp:   time.Now(),
				Source: sdk.EventSource{
					System:    "omo",
					Component: "worker",
					Version:   "1.0.0",
				},
				EventType: "task.completed",
				Payload: map[string]interface{}{
					"workstream_id": "00-068-03",
					"feature_id":    "F068",
				},
			},
			IsValid: true,
		},
		{
			Name:   "Beads issue.created",
			System: "beads",
			Event: &sdk.OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "beads-event-001",
				Timestamp:   time.Now(),
				Source: sdk.EventSource{
					System:    "beads",
					Component: "bridge",
					Version:   "1.0.0",
				},
				EventType: "issue.created",
				Payload: map[string]interface{}{
					"issue_id": "sdplab-fpb",
					"title":    "F068-03 contract-tests",
					"priority": "P1",
				},
			},
			IsValid: true,
		},
		{
			Name:   "Gas Town build.completed",
			System: "gas-town",
			Event: &sdk.OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "gt-event-001",
				Timestamp:   time.Now(),
				Source: sdk.EventSource{
					System:    "gas-town",
					Component: "builder",
					Version:   "1.0.0",
				},
				EventType: "build.completed",
				Payload: map[string]interface{}{
					"build_id": "build-123",
					"status":   "success",
					"duration": "45s",
				},
			},
			IsValid: true,
		},
		{
			Name:   "Invalid missing spec_version",
			System: "omo",
			Event: &sdk.OrchestrationEvent{
				EventID:   "invalid-event-001",
				Timestamp: time.Now(),
				Source: sdk.EventSource{
					System:    "omo",
					Component: "worker",
				},
				EventType: "task.completed",
			},
			IsValid: false,
		},
		{
			Name:   "Invalid missing event_type",
			System: "beads",
			Event: &sdk.OrchestrationEvent{
				SpecVersion: "v1.0",
				EventID:     "invalid-event-002",
				Timestamp:   time.Now(),
				Source: sdk.EventSource{
					System:    "beads",
					Component: "bridge",
				},
			},
			IsValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := validator.ValidateOrchestrationEvent(tc.Event)
			if tc.IsValid {
				require.NoError(t, err, "%s: expected valid event", tc.Name)
			} else {
				require.Error(t, err, "%s: expected invalid event", tc.Name)
			}
		})
	}
}

// TestBreakingChangeDetection verifies that breaking schema changes are detected.
// This test ensures that modifications to required fields or type changes fail validation.
func TestBreakingChangeDetection(t *testing.T) {
	validator, err := sdk.NewValidator()
	require.NoError(t, err)

	tests := []struct {
		name        string
		decision    *sdk.RuntimeDecision
		shouldError bool
		description string
	}{
		{
			name: "missing_required_spec_version",
			decision: &sdk.RuntimeDecision{
				DecisionID:   "test-001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: sdk.DecisionReason{
					Code:    "TEST",
					Message: "Test",
				},
			},
			shouldError: true,
			description: "Breaking: spec_version is required",
		},
		{
			name: "missing_required_decision_id",
			decision: &sdk.RuntimeDecision{
				SpecVersion:  "v1.0",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: sdk.DecisionReason{
					Code:    "TEST",
					Message: "Test",
				},
			},
			shouldError: true,
			description: "Breaking: decision_id is required",
		},
		{
			name: "missing_required_timestamp",
			decision: &sdk.RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "test-001",
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: sdk.DecisionReason{
					Code:    "TEST",
					Message: "Test",
				},
			},
			shouldError: true,
			description: "Breaking: timestamp is required",
		},
		{
			name: "invalid_decision_value",
			decision: &sdk.RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "test-001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "maybe", // Invalid - must be allow/ask/deny
				Reason: sdk.DecisionReason{
					Code:    "TEST",
					Message: "Test",
				},
			},
			shouldError: true,
			description: "Breaking: decision must be allow/ask/deny",
		},
		{
			name: "invalid_decision_type",
			decision: &sdk.RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "test-001",
				Timestamp:    time.Now(),
				DecisionType: "unknown.type", // Invalid type
				Decision:     "allow",
				Reason: sdk.DecisionReason{
					Code:    "TEST",
					Message: "Test",
				},
			},
			shouldError: true,
			description: "Breaking: decision_type must be valid enum value",
		},
		{
			name: "valid_complete_decision",
			decision: &sdk.RuntimeDecision{
				SpecVersion:  "v1.0",
				DecisionID:   "test-001",
				Timestamp:    time.Now(),
				DecisionType: "scope.boundary",
				Decision:     "allow",
				Reason: sdk.DecisionReason{
					Code:    "WITHIN_SCOPE",
					Message: "Valid decision",
				},
			},
			shouldError: false,
			description: "Non-breaking: valid decision passes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateRuntimeDecision(tt.decision)
			if tt.shouldError {
				require.Error(t, err, "%s should fail: %s", tt.name, tt.description)
				// Log the specific error for CI debugging
				t.Logf("Breaking change detected: %s -> %v", tt.description, err)
			} else {
				require.NoError(t, err, "%s should pass: %s", tt.name, tt.description)
			}
		})
	}
}

// TestContractVersionCompatibility verifies backward compatibility across versions.
// When v1.1 is introduced, this test should still pass with v1.0 fixtures.
func TestContractVersionCompatibility(t *testing.T) {
	// Load all v1 fixtures
	fixtureDir := filepath.Join("fixtures")
	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	validator, err := sdk.NewValidator()
	require.NoError(t, err)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join(fixtureDir, filename)
			data, err := os.ReadFile(path)
			require.NoError(t, err)

			// Detect contract type from filename
			if contains(filename, "orchestration-event") {
				var event sdk.OrchestrationEvent
				err = json.Unmarshal(data, &event)
				require.NoError(t, err)

				err = validator.ValidateOrchestrationEvent(&event)
				require.NoError(t, err, "Fixture %s failed validation", filename)

				t.Logf("✓ %s: spec_version=%s, event_type=%s",
					filename, event.SpecVersion, event.EventType)
			} else if contains(filename, "runtime-decision") {
				var decision sdk.RuntimeDecision
				err = json.Unmarshal(data, &decision)
				require.NoError(t, err)

				err = validator.ValidateRuntimeDecision(&decision)
				require.NoError(t, err, "Fixture %s failed validation", filename)

				t.Logf("✓ %s: spec_version=%s, decision=%s",
					filename, decision.SpecVersion, decision.Decision)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
