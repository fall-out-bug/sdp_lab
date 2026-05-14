package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyJSONUsesP0P1Counts(t *testing.T) {
	v := verdictDoc{
		Verdict:    "CHANGES_REQUESTED",
		P0Count:    intPtr(1),
		P1Count:    intPtr(2),
		ModelPanel: []modelDoc{{Status: "ok"}},
	}
	p := buildPolicy(v)
	if p.P0Findings != 1 || p.P1Findings != 2 {
		t.Fatalf("counts = P0 %d P1 %d, want 1/2", p.P0Findings, p.P1Findings)
	}
	if p.ApprovalCapable {
		t.Fatal("CHANGES_REQUESTED with blocking findings must not be approval-capable")
	}
}

func TestEscalatedRequiresOverrideForApproval(t *testing.T) {
	v := verdictDoc{
		Verdict:         "ESCALATED",
		EscalationIssue: "sdplab-1",
		ModelPanel:      []modelDoc{{Status: "failed", AssessmentState: "cannot_verify"}},
	}
	p := buildPolicy(v)
	if !p.ReviewEscalated || !p.ReviewCannotVerify {
		t.Fatalf("expected escalated/cannot_verify policy, got %+v", p)
	}
	if err := requireApprovalCapable(v, p); err == nil {
		t.Fatal("ESCALATED without maintainer override must fail approval")
	}
	v.OverrideReason = "Maintainer accepted provider outage after independent review panel."
	p = buildPolicy(v)
	if err := requireApprovalCapable(v, p); err != nil {
		t.Fatalf("ESCALATED with override should pass approval gate: %v", err)
	}
}

func TestApprovedRequiresCompleteAssessedModelPanel(t *testing.T) {
	v := verdictDoc{
		Verdict: "APPROVED",
		ModelPanel: []modelDoc{
			{Slot: "zai", Status: "ok", AssessmentState: "assessed"},
		},
	}
	p := buildPolicy(v)
	if p.ApprovalCapable {
		t.Fatal("truncated model panel must not be approval-capable")
	}
	if !p.ReviewCannotVerify {
		t.Fatalf("truncated model panel must be cannot_verify: %+v", p)
	}

	v.ModelPanel = []modelDoc{
		{Slot: "zai", Status: "ok", AssessmentState: "assessed"},
		{Slot: "kimi", Status: "ok", AssessmentState: "assessed"},
		{Slot: "minimax", Status: "ok", AssessmentState: "assessed"},
	}
	p = buildPolicy(v)
	if !p.ApprovalCapable {
		t.Fatalf("complete assessed panel should be approval-capable: %+v", p)
	}
}

func TestApprovedRequiresAssessmentState(t *testing.T) {
	v := verdictDoc{
		Verdict: "APPROVED",
		ModelPanel: []modelDoc{
			{Slot: "zai", Status: "ok"},
			{Slot: "kimi", Status: "ok", AssessmentState: "assessed"},
			{Slot: "minimax", Status: "ok", AssessmentState: "assessed"},
		},
	}
	p := buildPolicy(v)
	if p.ApprovalCapable {
		t.Fatal("missing assessment_state must not be approval-capable")
	}
}

func TestSchemaValidationRejectsMalformedVerdict(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "verdict.json")
	if err := os.WriteFile(path, []byte(`{"feature":"F168","verdict":"ESCALATED"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := run([]string{"--schema", filepath.Join(repoRoot(t), "schema", "review-verdict.schema.json"), path}, os.Stdout, os.Stderr); code == 0 {
		t.Fatal("malformed verdict should fail schema validation")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for dir := wd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	t.Fatal("repo root not found")
	return ""
}

func intPtr(v int) *int { return &v }
