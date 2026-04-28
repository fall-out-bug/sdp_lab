package policy

import (
	"errors"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
)

func TestEvaluateEvidenceGate_RequiresSignature(t *testing.T) {
	config := defaultEvidenceGateConfig()
	result := evaluateEvidenceGate(config, []byte(`{"payload":"e30=","signatures":[]}`), evidence.DiscrepancyReport{}, func(_ []byte) error {
		return nil
	})

	if result.Allowed {
		t.Fatal("expected gate to reject unsigned attestation")
	}
	if !containsReason(result.Reasons, "not signed") {
		t.Fatalf("expected unsigned reason, got %v", result.Reasons)
	}
}

func TestEvaluateEvidenceGate_FailsOnThresholdBreach(t *testing.T) {
	config := defaultEvidenceGateConfig()
	config.Thresholds.High = 0
	report := evidence.DiscrepancyReport{
		Discrepancies: []evidence.Discrepancy{{Severity: "high"}},
	}

	result := evaluateEvidenceGate(config, []byte(`{"signatures":[{"sig":"abc"}]}`), report, func(_ []byte) error {
		return nil
	})

	if result.Allowed {
		t.Fatal("expected threshold breach to block gate")
	}
	if !containsReason(result.Reasons, "high discrepancies exceeded threshold") {
		t.Fatalf("expected threshold reason, got %v", result.Reasons)
	}
}

func TestEvaluateEvidenceGate_FailsOnVerificationError(t *testing.T) {
	config := defaultEvidenceGateConfig()
	result := evaluateEvidenceGate(config, []byte(`{"signatures":[{"sig":"abc"}]}`), evidence.DiscrepancyReport{}, func(_ []byte) error {
		return errors.New("verify failed")
	})

	if result.Allowed {
		t.Fatal("expected verification failure to block gate")
	}
	if !containsReason(result.Reasons, "verification failed") {
		t.Fatalf("expected verification reason, got %v", result.Reasons)
	}
}

func TestEvidenceGateResult_AuditFields(t *testing.T) {
	result := evidenceGateResult{
		Allowed:       true,
		Reasons:       []string{},
		SeverityCount: map[string]int{"critical": 0, "high": 0, "medium": 1, "low": 0},
		Config:        defaultEvidenceGateConfig(),
	}

	fields := result.AuditFields()
	if allowed, ok := fields["allowed"].(bool); !ok || !allowed {
		t.Fatalf("unexpected allowed field: %v", fields["allowed"])
	}
	counts, ok := fields["severity_count"].(map[string]int)
	if !ok {
		t.Fatalf("expected severity_count map, got %T", fields["severity_count"])
	}
	if counts["medium"] != 1 {
		t.Fatalf("expected medium count 1, got %d", counts["medium"])
	}
}

func containsReason(reasons []string, fragment string) bool {
	for _, r := range reasons {
		if strings.Contains(r, fragment) {
			return true
		}
	}
	return false
}
