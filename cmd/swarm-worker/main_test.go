package main

import (
	"testing"
	"time"

	"sdp_dev/internal/observability"
)

func TestEvaluateOneShotVerificationPassesWithTests(t *testing.T) {
	result, err := evaluateOneShotVerification([]string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}, true)
	if err != nil {
		t.Fatalf("evaluate oneshot verification: %v", err)
	}
	if !result.Report.OK {
		t.Fatalf("expected report OK, got %+v", result.Report)
	}
	if len(result.FailedTaskIDs) != 0 {
		t.Fatalf("expected no failed tasks, got %#v", result.FailedTaskIDs)
	}
	if result.RecoveryPlan != nil {
		t.Fatalf("expected no recovery plan, got %+v", *result.RecoveryPlan)
	}
}

func TestEvaluateOneShotVerificationBuildFailureCreatesRecovery(t *testing.T) {
	result, err := evaluateOneShotVerification([]string{"internal/oneshot/manifest.go", "internal/oneshot/manifest_test.go"}, false)
	if err != nil {
		t.Fatalf("evaluate oneshot verification: %v", err)
	}
	if result.Report.OK {
		t.Fatalf("expected report failure on failed tests, got %+v", result.Report)
	}
	if len(result.FailedTaskIDs) == 0 {
		t.Fatal("expected failed task ids")
	}
	if result.RecoveryPlan == nil {
		t.Fatal("expected recovery plan")
	}
	if len(result.RecoveryPlan.RequeueTaskIDs) == 0 {
		t.Fatalf("expected non-empty requeue tasks, got %+v", result.RecoveryPlan)
	}
}

func TestApplyOneShotVerificationWritesMachineReadableSections(t *testing.T) {
	payload := map[string]any{"verification": map[string]any{}}
	runPacket := map[string]any{}
	note, err := applyOneShotVerification(payload, runPacket, []string{"internal/oneshot/manifest.go"}, true)
	if err != nil {
		t.Fatalf("apply oneshot verification: %v", err)
	}
	if note == "" {
		t.Fatal("expected non-empty machine-readable note")
	}

	verification, ok := payload["verification"].(map[string]any)
	if !ok {
		t.Fatal("missing verification section")
	}
	ones, ok := verification["oneshot"].(map[string]any)
	if !ok {
		t.Fatal("missing verification.oneshot section")
	}
	if _, ok := ones["report"]; !ok {
		t.Fatal("missing oneshot report")
	}
	if _, ok := runPacket["oneshot_verification"]; !ok {
		t.Fatal("missing run packet oneshot_verification section")
	}
}

func TestBuildWorkerObservabilityRecordsValidatorCompatible(t *testing.T) {
	records := buildWorkerObservabilityRecords(
		"sdp_dev-2aq.20.2",
		"verify",
		"fallback",
		"glm-4.7",
		1,
		true,
		false,
		".sdp/evidence/sdp_dev-2aq.20.2.json",
		"https://example.invalid/org/repo/pull/50",
		230*time.Millisecond,
	)
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	event, ok := records[0]["event"].(map[string]any)
	if !ok {
		t.Fatalf("missing event payload: %#v", records[0])
	}
	if errs := observability.ValidateUnifiedMetricsTraceEvent(event); len(errs) != 0 {
		t.Fatalf("event failed schema validation: %v", errs)
	}
	resilience, _ := event["resilience"].(map[string]any)
	if fallback, _ := resilience["fallback_used"].(bool); !fallback {
		t.Fatal("expected fallback_used=true")
	}
}
