package selfimprove

import (
	"testing"
)

func TestNewFailureClassifier(t *testing.T) {
	c := NewFailureClassifier()
	if c == nil {
		t.Fatal("NewFailureClassifier returned nil")
	}
}

func TestFailureClassifier_ClassifyRun(t *testing.T) {
	c := NewFailureClassifier()

	// OK run returns nil
	cf := c.ClassifyRun(RunDoc{LastState: "ok"})
	if cf != nil {
		t.Errorf("expected nil for ok run, got %+v", cf)
	}

	// Failed run
	cf = c.ClassifyRun(RunDoc{
		RunID: "r1", IssueID: "i1",
		LastPhase: "verify", LastState: "failed",
		Events: []RunEvent{{Message: "timeout after 30s"}},
	})
	if cf == nil {
		t.Fatal("expected ClassifiedFailure")
	}
	if cf.IssueID != "i1" || cf.Class != ClassTransient {
		t.Errorf("got %+v", cf)
	}

	// Message classification
	tests := []struct {
		msg  string
		want FailureClass
	}{
		{"429 rate limit", ClassTransient},
		{"5xx server error", ClassTransient},
		{"infra flaky", ClassToolFlake},
		{"go test failed", ClassVerificationFail},
		{"policy allowlist", ClassPolicyConflict},
		{"security auth failed", ClassSecuritySensitive},
		{"unknown error", ClassUnknown},
	}
	for _, tt := range tests {
		cf := c.ClassifyRun(RunDoc{
			LastState: "failed",
			Events:    []RunEvent{{Message: tt.msg}},
		})
		if cf == nil || cf.Class != tt.want {
			t.Errorf("msg %q: got %v, want %v", tt.msg, cf, tt.want)
		}
	}
}

func TestFailureClassifier_ClassifyTelemetry(t *testing.T) {
	c := NewFailureClassifier()

	// OK status returns nil
	cf := c.ClassifyTelemetry(TelemetryRecord{Status: "ok"})
	if cf != nil {
		t.Errorf("expected nil, got %+v", cf)
	}

	// Escalated
	cf = c.ClassifyTelemetry(TelemetryRecord{Status: "escalated", Escalated: true})
	if cf == nil || cf.Class != ClassPolicyConflict {
		t.Errorf("escalated: got %+v", cf)
	}

	// Retry count >= 3
	cf = c.ClassifyTelemetry(TelemetryRecord{Status: "failed", RetryCount: 3})
	if cf == nil || cf.Class != ClassTransient {
		t.Errorf("retry: got %+v", cf)
	}
}
