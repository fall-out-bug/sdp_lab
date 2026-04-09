package omoclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func TestClassifyError_Nil(t *testing.T) {
	result := ClassifyError(nil)
	if result != nil {
		t.Errorf("ClassifyError(nil) should return nil, got %v", result)
	}
}

func TestClassifyError_ContextCanceled(t *testing.T) {
	err := context.Canceled
	failure := ClassifyError(err)

	if failure.Kind != FailureRuntime {
		t.Errorf("Expected FailureRuntime, got %v", failure.Kind)
	}
	if failure.Code != "ctx_canceled" {
		t.Errorf("Expected code 'ctx_canceled', got %s", failure.Code)
	}
	if failure.Retryable {
		t.Error("ContextCanceled should not be retryable")
	}
}

func TestClassifyError_ContextDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	<-ctx.Done()

	err := ctx.Err()
	failure := ClassifyError(err)

	if failure.Kind != FailureTransport {
		t.Errorf("Expected FailureTransport, got %v", failure.Kind)
	}
	if failure.Code != "ctx_timeout" {
		t.Errorf("Expected code 'ctx_timeout', got %s", failure.Code)
	}
	if !failure.Retryable {
		t.Error("DeadlineExceeded should be retryable")
	}
	if !failure.Temporary {
		t.Error("DeadlineExceeded should be temporary")
	}
}

func TestClassifyError_NetError(t *testing.T) {
	err := &net.OpError{
		Op:  "dial",
		Net: "tcp",
		Err: &timeoutError{},
	}
	failure := ClassifyError(err)

	if failure.Kind != FailureTransport {
		t.Errorf("Expected FailureTransport, got %v", failure.Kind)
	}
	if failure.Code != "net_error" {
		t.Errorf("Expected code 'net_error', got %s", failure.Code)
	}
	if !failure.Retryable {
		t.Error("NetError should be retryable")
	}
	if !failure.Temporary {
		t.Error("NetError should be temporary")
	}
}

type timeoutError struct{}

func (e *timeoutError) Error() string   { return "timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return true }

func TestClassifyError_Timeout(t *testing.T) {
	err := fmt.Errorf("connection timeout after 30s")
	failure := ClassifyError(err)

	if failure.Kind != FailureTransport {
		t.Errorf("Expected FailureTransport, got %v", failure.Kind)
	}
	if failure.Code != "timeout" {
		t.Errorf("Expected code 'timeout', got %s", failure.Code)
	}
}

func TestClassifyError_Validation(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected FailureKind
	}{
		{"invalid json", errors.New("invalid json syntax"), FailureValidation},
		{"malformed", errors.New("malformed data"), FailureValidation},
		{"schema", errors.New("schema validation failed"), FailureValidation},
		{"unmarshal", errors.New("unmarshal error"), FailureValidation},
		{"marshal", errors.New("marshal error"), FailureValidation},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := ClassifyError(tt.err)
			if failure.Kind != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, failure.Kind)
			}
			if failure.Retryable {
				t.Error("Validation errors should not be retryable")
			}
		})
	}
}

func TestClassifyError_Persistence(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected FailureKind
	}{
		{"database", errors.New("database connection failed"), FailurePersistence},
		{"storage", errors.New("storage error"), FailurePersistence},
		{"file not found", errors.New("file not found"), FailurePersistence},
		{"permission", errors.New("permission denied"), FailurePersistence},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failure := ClassifyError(tt.err)
			if failure.Kind != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, failure.Kind)
			}
		})
	}
}

func TestClassifyError_Unknown(t *testing.T) {
	err := errors.New("some unknown error")
	failure := ClassifyError(err)

	if failure.Kind != FailureRuntime {
		t.Errorf("Expected FailureRuntime, got %v", failure.Kind)
	}
	if failure.Code != "runtime_error" {
		t.Errorf("Expected code 'runtime_error', got %s", failure.Code)
	}
}

func TestClassifyError_WrappedFailure(t *testing.T) {
	original := &Failure{
		Kind:      FailureGovernance,
		Code:      "policy_violation",
		Message:   "governance violation",
		Retryable: false,
	}
	err := fmt.Errorf("wrapped: %w", original)

	result := ClassifyError(err)
	if result != original {
		t.Errorf("Should unwrap and return original Failure, got %v", result)
	}
}

func TestIsStrike(t *testing.T) {
	tests := []struct {
		name     string
		failure  *Failure
		expected bool
	}{
		{"transport transient", &Failure{Kind: FailureTransport, Retryable: true, Temporary: true}, false},
		{"transport non-retryable", &Failure{Kind: FailureTransport, Retryable: false, Temporary: false}, true},
		{"validation", &Failure{Kind: FailureValidation}, true},
		{"protocol", &Failure{Kind: FailureProtocol}, true},
		{"governance", &Failure{Kind: FailureGovernance}, true},
		{"runtime temporary", &Failure{Kind: FailureRuntime, Temporary: true}, false},
		{"runtime non-temporary", &Failure{Kind: FailureRuntime, Temporary: false}, true},
		{"persistence", &Failure{Kind: FailurePersistence}, true},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStrike(tt.failure)
			if result != tt.expected {
				t.Errorf("IsStrike(%v): expected %v, got %v", tt.name, tt.expected, result)
			}
		})
	}
}

func TestStrikeTracker_Record(t *testing.T) {
	policy := DefaultStrikePolicy()
	tracker := NewStrikeTracker(policy)

	tracker.Record(&Failure{Kind: FailureTransport, Retryable: true})
	tracker.Record(&Failure{Kind: FailureValidation})
	tracker.Record(&Failure{Kind: FailureGovernance})
	tracker.Record(&Failure{Kind: FailureRuntime, Temporary: true})

	transport, quality, policyStrikes := tracker.GetCounts()

	if transport != 1 {
		t.Errorf("Expected 1 transport retry, got %d", transport)
	}
	if quality != 1 {
		t.Errorf("Expected 1 quality strike, got %d", quality)
	}
	if policyStrikes != 1 {
		t.Errorf("Expected 1 policy strike, got %d", policyStrikes)
	}
}

func TestStrikeTracker_ShouldBlock(t *testing.T) {
	tests := []struct {
		name            string
		setupFunc       func(*StrikeTracker)
		expectedBlocked bool
	}{
		{
			name:            "no strikes",
			setupFunc:       func(st *StrikeTracker) {},
			expectedBlocked: false,
		},
		{
			name: "max transport retries",
			setupFunc: func(st *StrikeTracker) {
				st.policy.MaxTransportRetries = 3
				for i := 0; i < 3; i++ {
					st.Record(&Failure{Kind: FailureTransport, Retryable: true})
				}
			},
			expectedBlocked: true,
		},
		{
			name: "max quality strikes",
			setupFunc: func(st *StrikeTracker) {
				st.policy.MaxQualityStrikes = 2
				st.Record(&Failure{Kind: FailureValidation})
				st.Record(&Failure{Kind: FailureProtocol})
			},
			expectedBlocked: true,
		},
		{
			name: "any policy strike",
			setupFunc: func(st *StrikeTracker) {
				st.policy.MaxPolicyStrikes = 1
				st.Record(&Failure{Kind: FailureGovernance})
			},
			expectedBlocked: true,
		},
		{
			name: "below thresholds",
			setupFunc: func(st *StrikeTracker) {
				st.Record(&Failure{Kind: FailureTransport, Retryable: true})
				st.Record(&Failure{Kind: FailureValidation})
			},
			expectedBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := DefaultStrikePolicy()
			tracker := NewStrikeTracker(policy)
			tt.setupFunc(tracker)

			result := tracker.ShouldBlock()
			if result != tt.expectedBlocked {
				t.Errorf("Expected blocked=%v, got %v", tt.expectedBlocked, result)
			}
		})
	}
}

func TestStrikeTracker_Reset(t *testing.T) {
	policy := DefaultStrikePolicy()
	tracker := NewStrikeTracker(policy)

	tracker.Record(&Failure{Kind: FailureTransport, Retryable: true})
	tracker.Record(&Failure{Kind: FailureValidation})
	tracker.Record(&Failure{Kind: FailureGovernance})

	tracker.Reset()

	transport, quality, policyStrikes := tracker.GetCounts()

	if transport != 0 {
		t.Errorf("Expected 0 transport retries after reset, got %d", transport)
	}
	if quality != 0 {
		t.Errorf("Expected 0 quality strikes after reset, got %d", quality)
	}
	if policyStrikes != 0 {
		t.Errorf("Expected 0 policy strikes after reset, got %d", policyStrikes)
	}
}

func TestDefaultStrikePolicy(t *testing.T) {
	policy := DefaultStrikePolicy()

	if policy.MaxTransportRetries != 5 {
		t.Errorf("Expected MaxTransportRetries=5, got %d", policy.MaxTransportRetries)
	}
	if policy.MaxQualityStrikes != 3 {
		t.Errorf("Expected MaxQualityStrikes=3, got %d", policy.MaxQualityStrikes)
	}
	if policy.MaxPolicyStrikes != 1 {
		t.Errorf("Expected MaxPolicyStrikes=1, got %d", policy.MaxPolicyStrikes)
	}
}

func TestDefaultGovernanceConfig(t *testing.T) {
	config := DefaultGovernanceConfig()

	if config.ConstitutionVersion != "1.0" {
		t.Errorf("Expected ConstitutionVersion=1.0, got %s", config.ConstitutionVersion)
	}
	if config.MaxToolCalls != 100 {
		t.Errorf("Expected MaxToolCalls=100, got %d", config.MaxToolCalls)
	}
	if !config.MustCiteEvidence {
		t.Error("Expected MustCiteEvidence=true")
	}
	if !config.MustReportOOS {
		t.Error("Expected MustReportOOS=true")
	}
}

func TestGovernanceWrapper_PreCall_MissingFields(t *testing.T) {
	tests := []struct {
		name      string
		envelope  TaskEnvelope
		expectErr bool
	}{
		{
			name:      "missing task_id",
			envelope:  TaskEnvelope{Phase: "test", EntryAgent: "agent", ScopeIn: []string{"*"}},
			expectErr: true,
		},
		{
			name:      "missing phase defaults to build",
			envelope:  TaskEnvelope{TaskID: "1", EntryAgent: "agent", ScopeIn: []string{"*"}},
			expectErr: false,
		},
		{
			name:      "missing entry_agent",
			envelope:  TaskEnvelope{TaskID: "1", Phase: "test", ScopeIn: []string{"*"}},
			expectErr: true,
		},
		{
			name:      "missing scope defaults to wildcard",
			envelope:  TaskEnvelope{TaskID: "1", Phase: "test", EntryAgent: "agent"},
			expectErr: false,
		},
		{
			name: "valid envelope",
			envelope: TaskEnvelope{
				TaskID:     "1",
				Phase:      "test",
				EntryAgent: "agent",
				ScopeIn:    []string{"*"},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
						client := NewClient("http://test")
			gw := NewGovernanceWrapper(client, DefaultStrikePolicy(), true)

			err := gw.PreCall(context.Background(), tt.envelope)

			if tt.expectErr && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestGovernanceWrapper_PreCall_ScopeConflict(t *testing.T) {
	envelope := TaskEnvelope{
		TaskID:     "1",
		Phase:      "test",
		EntryAgent: "agent",
		ScopeIn:    []string{"src/*.go"},
		ScopeOut:   []string{"src/*.go"},
	}

		client := NewClient("http://test")
	gw := NewGovernanceWrapper(client, DefaultStrikePolicy(), true)

	err := gw.PreCall(context.Background(), envelope)
	if err == nil {
		t.Error("Expected error for scope conflict")
	}
}

func TestGovernanceWrapper_PostCall(t *testing.T) {
	tests := []struct {
		name      string
		verdict   string
		toolCalls int
		findings  int
		oosClean  bool
		mustCite  bool
		expected  Outcome
	}{
		{
			name:      "successful pass",
			verdict:   "qa:pass",
			toolCalls: 5,
			findings:  1,
			oosClean:  true,
			mustCite:  false,
			expected:  OutcomeSucceeded,
		},
		{
			name:     "successful fail",
			verdict:  "qa:fail",
			oosClean: true,
			expected: OutcomeFailed,
		},
		{
			name:     "cancelled",
			verdict:  "cancelled",
			oosClean: true,
			expected: OutcomeCanceled,
		},
		{
			name:     "cancelled",
			verdict:  "cancelled",
			expected: OutcomeCanceled,
		},
		{
			name:     "out of scope",
			verdict:  "qa:pass",
			oosClean: false,
			expected: OutcomeOutOfScope,
		},
		{
			name:      "incomplete - missing evidence",
			verdict:   "qa:pass",
			toolCalls: 0,
			findings:  0,
			mustCite:  true,
			oosClean:  true,
			expected:  OutcomeIncomplete,
		},
		{
			name:     "unknown verdict",
			verdict:  "unknown",
			oosClean: true,
			expected: OutcomeIncomplete,
		},
		{
			name:     "unknown verdict",
			verdict:  "unknown",
			expected: OutcomeIncomplete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
						client := NewClient("http://test")
			gw := NewGovernanceWrapper(client, DefaultStrikePolicy(), true)

			envelope := TaskEnvelope{
				TaskID:     "1",
				Phase:      "test",
				EntryAgent: "agent",
				ScopeIn:    []string{"*"},
				Governance: &GovernanceConfig{
					MustCiteEvidence: tt.mustCite,
				},
			}

			evidence := EvidenceEnvelope{
				SessionID: "test",
				Verdict:   tt.verdict,
				ToolCalls: make([]OmOEvent, tt.toolCalls),
				Findings:  make([]Finding, tt.findings),
			}

			oos := OutOfScopeReport{
				Clean: tt.oosClean,
			}

			outcome, err := gw.PostCall(context.Background(), envelope, evidence, oos)

			if outcome != tt.expected {
				t.Errorf("Expected outcome %s, got %s", tt.expected, outcome)
			}
			if tt.expected != OutcomeIncomplete && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestGovernanceWrapper_Escalation(t *testing.T) {
		client := NewClient("http://test")
	policy := &StrikePolicy{MaxTransportRetries: 2, MaxQualityStrikes: 1}
	gw := NewGovernanceWrapper(client, policy, true)

	gw.RecordFailure(&Failure{Kind: FailureTransport, Retryable: true})
	gw.RecordFailure(&Failure{Kind: FailureTransport, Retryable: true})
	gw.RecordFailure(&Failure{Kind: FailureValidation})

	envelope := TaskEnvelope{
		TaskID:     "1",
		Phase:      "test",
		EntryAgent: "agent",
		ScopeIn:    []string{"*"},
	}
	evidence := EvidenceEnvelope{SessionID: "test", Verdict: "qa:pass"}
	oos := OutOfScopeReport{Clean: true}

	outcome, err := gw.PostCall(context.Background(), envelope, evidence, oos)

	if outcome != OutcomeEscalated {
		t.Errorf("Expected OutcomeEscalated, got %s", outcome)
	}
	if err == nil {
		t.Error("Expected error for escalation")
	}
}

func TestGovernanceWrapper_Disabled(t *testing.T) {
		client := NewClient("http://test")
	gw := NewGovernanceWrapper(client, nil, false)

	if gw.IsEnabled() {
		t.Error("Governance should be disabled")
	}

	gw.RecordFailure(errors.New("test error"))
	transport, quality, policy := gw.GetStrikeCounts()

	if transport != 0 || quality != 0 || policy != 0 {
		t.Error("Disabled governance should not track strikes")
	}

	if gw.ShouldBlock() {
		t.Error("Disabled governance should not block")
	}
}

func TestGovernanceWrapper_Client(t *testing.T) {
		client := NewClient("http://test")
	gw := NewGovernanceWrapper(client, DefaultStrikePolicy(), true)

	if gw.Client() != client {
		t.Error("Client() should return underlying client")
	}
}

func TestOutcome_String(t *testing.T) {
	tests := []struct {
		outcome  Outcome
		expected string
	}{
		{OutcomeSucceeded, "succeeded"},
		{OutcomeIncomplete, "incomplete"},
		{OutcomeOutOfScope, "out_of_scope"},
		{OutcomeFailed, "failed"},
		{OutcomeCanceled, "canceled"},
		{OutcomeEscalated, "escalated"},
	}

	for _, tt := range tests {
		t.Run(string(tt.outcome), func(t *testing.T) {
			result := string(tt.outcome)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFailure_Error(t *testing.T) {
	tests := []struct {
		name     string
		failure  *Failure
		contains string
	}{
		{"with cause", &Failure{Message: "test error", Cause: errors.New("underlying")}, "underlying"},
		{"without cause", &Failure{Message: "test error"}, "test error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.failure.Error()
			if !containsString(err, tt.contains) {
				t.Errorf("Error message should contain '%s', got: %s", tt.contains, err)
			}
		})
	}
}

func TestFailure_Unwrap(t *testing.T) {
	underlying := errors.New("underlying")
	failure := &Failure{Message: "test", Cause: underlying}

	unwrapped := failure.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap should return underlying error, got %v", unwrapped)
	}
}

func TestFailureKind_String(t *testing.T) {
	tests := []struct {
		kind     FailureKind
		expected string
	}{
		{FailureTransport, "transport"},
		{FailureProtocol, "protocol"},
		{FailureRuntime, "runtime"},
		{FailureGovernance, "governance"},
		{FailureValidation, "validation"},
		{FailurePersistence, "persistence"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.kind.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsInMiddle(s, substr)))
}

func containsInMiddle(s, substr string) bool {
	for i := 1; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
