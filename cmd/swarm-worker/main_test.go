package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"sdp_dev/internal/observability"
)

func TestResolveWorkstream(t *testing.T) {
	tests := []struct {
		labels []string
		want   string
	}{
		{[]string{"workstream:policy-slugify-trim"}, "policy-slugify-trim"},
		{[]string{"workstream:generic"}, "generic"},
		{[]string{"workstream:builder"}, "builder"},
		{[]string{"workstream:oneshot-swarm-orchestrator"}, "oneshot-swarm-orchestrator"},
		{[]string{"workstream:handoff-validation"}, "handoff-validation"},
		{[]string{"workstream:self-improvement"}, "self-improvement"},
		{[]string{"workstream:evaluator-recommendation"}, "evaluator-recommendation"},
		{[]string{"workstream:telegram-ingress-intake"}, "telegram-ingress-intake"},
		{[]string{"workstream:planner-boundary-decomposition"}, "planner-boundary-decomposition"},
		{[]string{}, ""},
		{[]string{"autonomy"}, ""},
	}
	for _, tt := range tests {
		got := resolveWorkstream(tt.labels)
		if got != tt.want {
			t.Errorf("resolveWorkstream(%v) = %q, want %q", tt.labels, got, tt.want)
		}
	}
}

func TestCommitBodyForWorkstream(t *testing.T) {
	tests := []struct {
		workstream string
		want       string
	}{
		{"policy-slugify-trim", "Fix slugify truncation and add regression coverage."},
		{"handoff-validation", "Add handoff validation timestamp for adapter checklist run."},
		{"generic", "Builder workstream: LLM-backed implementation via opencode run."},
		{"builder", "Builder workstream: LLM-backed implementation via opencode run."},
		{"unknown", "Implement workstream changes with regression coverage."},
	}
	for _, tt := range tests {
		got := commitBodyForWorkstream(tt.workstream)
		if got != tt.want {
			t.Errorf("commitBodyForWorkstream(%q) = %q, want %q", tt.workstream, got, tt.want)
		}
	}
}

func TestHasLabel(t *testing.T) {
	if !hasLabel([]string{"autonomy", "strict-evidence"}, "autonomy") {
		t.Error("hasLabel(autonomy) should be true")
	}
	if hasLabel([]string{"autonomy"}, "strict-evidence") {
		t.Error("hasLabel(strict-evidence) should be false")
	}
}

func TestParseClaim(t *testing.T) {
	valid := []byte(`{"issue_id":"sdp_dev-4pg","title":"x","model":"glm-5","branch":"feat/sdp_dev-4pg"}`)
	claim, err := parseClaim(valid)
	if err != nil {
		t.Fatalf("parseClaim: %v", err)
	}
	if claim.IssueID != "sdp_dev-4pg" || claim.Branch != "feat/sdp_dev-4pg" {
		t.Fatalf("unexpected claim: %+v", claim)
	}

	noise := []byte(`some output\n` + string(valid))
	claim2, err := parseClaim(noise)
	if err != nil {
		t.Fatalf("parseClaim with noise: %v", err)
	}
	if claim2.IssueID != "sdp_dev-4pg" {
		t.Fatalf("unexpected claim: %+v", claim2)
	}

	invalid := []byte(`{"issue_id":"","branch":"x"}`)
	_, err = parseClaim(invalid)
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}

	badJSON := []byte(`not json`)
	_, err = parseClaim(badJSON)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestToStringSlice(t *testing.T) {
	if got := toStringSlice([]any{"a", "b"}); len(got) != 2 || got[0] != "a" {
		t.Fatalf("toStringSlice([]any): %v", got)
	}
	if got := toStringSlice([]string{"x"}); len(got) != 1 || got[0] != "x" {
		t.Fatalf("toStringSlice([]string): %v", got)
	}
	if got := toStringSlice(123); got != nil {
		t.Fatalf("toStringSlice(123): %v", got)
	}
}

func TestHasPrefixAny(t *testing.T) {
	if !hasPrefixAny("internal/policy/foo.go", []string{"internal/", "cmd/"}) {
		t.Error("hasPrefixAny should match internal/")
	}
	if hasPrefixAny("docs/foo.md", []string{"internal/", "cmd/"}) {
		t.Error("hasPrefixAny should not match")
	}
}

func TestApplyBuilderWorkstream(t *testing.T) {
	dir := t.TempDir()
	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `workstreams:
  - label: workstream:builder
    path_prefixes:
      - internal/
      - cmd/
`
	if err := os.WriteFile(filepath.Join(specsDir, "workstream-config.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := applyBuilderWorkstream(dir, "test-1", issueDetail{ID: "test-1", Title: "T", SpecID: "spec", Description: "d", AcceptanceCriteria: "ac"}, "glm-4.7")
	// Outcome depends on opencode availability; we only verify no panic
	if err != nil {
		t.Logf("applyBuilderWorkstream (opencode may be unavailable): %v", err)
	}
}

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
