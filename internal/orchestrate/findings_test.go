package orchestrate

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildReviewFailureFindingInput(t *testing.T) {
	cp := &Checkpoint{
		FeatureID: "F054",
		PRURL:     "https://github.com/org/repo/pull/123",
		Workstreams: []WSStatus{
			{ID: "00-054-01", Status: "done"},
			{ID: "00-054-03", Status: "done"},
		},
	}

	input := buildReviewFailureFindingInput(cp, "review output here", errors.New("review command failed"))

	if input.FeatureID != "F054" {
		t.Fatalf("expected feature id F054, got %q", input.FeatureID)
	}
	if input.WSID != "00-054-03" {
		t.Fatalf("expected last workstream id, got %q", input.WSID)
	}
	if !input.Blocking {
		t.Fatalf("expected blocking review finding")
	}
	if input.Title != "review not approved" {
		t.Fatalf("unexpected title %q", input.Title)
	}
	if input.Severity != "P1" || input.Priority != 1 {
		t.Fatalf("expected P1 priority mapping, got severity=%q priority=%d", input.Severity, input.Priority)
	}
	if input.PRURL != "https://github.com/org/repo/pull/123" {
		t.Fatalf("unexpected pr url %q", input.PRURL)
	}
	if input.Description != "review output here" {
		t.Fatalf("unexpected description %q", input.Description)
	}
	if input.DedupKey != "F054:review-not-approved" {
		t.Fatalf("unexpected dedup key %q", input.DedupKey)
	}
	if input.Summary == "" {
		t.Fatalf("expected non-empty summary")
	}
}

func TestCheckpointFindingWSID(t *testing.T) {
	cp := &Checkpoint{Workstreams: []WSStatus{{ID: "00-001-01"}, {ID: "00-001-04"}}}
	if got := checkpointFindingWSID(cp); got != "00-001-04" {
		t.Fatalf("expected last workstream id, got %q", got)
	}
	if got := checkpointFindingWSID(nil); got != "" {
		t.Fatalf("expected empty ws id for nil checkpoint, got %q", got)
	}
}

func TestBuildQAFailureFindingInput(t *testing.T) {
	cp := &Checkpoint{
		FeatureID: "F054",
		PRURL:     "https://github.com/org/repo/pull/123",
		Workstreams: []WSStatus{
			{ID: "00-054-03", Status: "done"},
		},
	}

	input := buildQAFailureFindingInput(cp, "qa output here", errors.New("qa command failed"))

	if input.FeatureID != "F054" || input.WSID != "00-054-03" {
		t.Fatalf("unexpected feature/workstream linkage: %+v", input)
	}
	if input.Title != "qa not passed" {
		t.Fatalf("unexpected title %q", input.Title)
	}
	if !input.Blocking || input.Priority != 1 || input.Severity != "P1" {
		t.Fatalf("unexpected qa blocking priority mapping: %+v", input)
	}
	if input.DedupKey != "F054:qa-not-passed" {
		t.Fatalf("unexpected dedup key %q", input.DedupKey)
	}
}

func TestBuildChangesRequestedReviewVerdict(t *testing.T) {
	cp := &Checkpoint{FeatureID: "F054", Review: &ReviewStatus{Iteration: 2}}
	v := buildChangesRequestedReviewVerdict(cp, "review failed", "sdplab-123")
	if v.Verdict != "CHANGES_REQUESTED" || v.Round != 2 {
		t.Fatalf("unexpected review verdict: %+v", v)
	}
	if len(v.FindingIDs) != 1 || v.FindingIDs[0] != "sdplab-123" {
		t.Fatalf("unexpected finding ids: %+v", v.FindingIDs)
	}
	if got := v.Reviewers["security"]; got.Verdict != "FAIL" || len(got.Findings) != 1 {
		t.Fatalf("unexpected reviewer entry: %+v", got)
	}
}

func TestWriteQAVerdict(t *testing.T) {
	dir := t.TempDir()
	cp := &Checkpoint{FeatureID: "F054", QA: &QAStatus{Iteration: 1}}
	path, err := WriteQAVerdict(dir, cp, buildPassedQAVerdict(cp, "qa passed", ".sdp/evidence/f054-uat.json"))
	if err != nil {
		t.Fatalf("WriteQAVerdict returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected verdict file to exist: %v", err)
	}
	if cp.QA == nil || cp.QA.VerdictFile != filepath.Join(dir, ".sdp", "qa_verdict.json") {
		t.Fatalf("unexpected qa checkpoint state: %+v", cp.QA)
	}
}

func TestBuildBlockedVerdictsCarryBlockingIDs(t *testing.T) {
	cp := &Checkpoint{FeatureID: "F090", Review: &ReviewStatus{Iteration: 1}, QA: &QAStatus{Iteration: 1}}
	review := buildBlockedReviewVerdict(cp, "blocked before review", []string{"sdplab-a", "sdplab-b"})
	if len(review.BlockingIDs) != 2 || review.Reviewers["qa"].Verdict != "BLOCKED" {
		t.Fatalf("unexpected blocked review verdict: %+v", review)
	}
	qa := buildBlockedQAVerdict(cp, "blocked before qa", ".sdp/evidence/uat.json", []string{"sdplab-c"})
	if len(qa.BlockingIDs) != 1 || qa.Verdict != "qa:fail" {
		t.Fatalf("unexpected blocked qa verdict: %+v", qa)
	}
}

func TestBuildOverrideReviewVerdict(t *testing.T) {
	cp := &Checkpoint{FeatureID: "F098", Review: &ReviewStatus{Iteration: 4}}
	v := buildOverrideReviewVerdict(cp, "P2 findings only", "doc-only findings, no code risk")
	if v.Verdict != "APPROVED" {
		t.Fatalf("expected APPROVED, got %q", v.Verdict)
	}
	if v.Round != 4 {
		t.Fatalf("expected round 4, got %d", v.Round)
	}
	if v.OverrideReason != "doc-only findings, no code risk" {
		t.Fatalf("unexpected override reason: %q", v.OverrideReason)
	}
	if v.Feature != "F098" {
		t.Fatalf("expected feature F098, got %q", v.Feature)
	}
	for _, role := range []string{"qa", "security", "devops", "sre", "techlead", "docs", "promptops"} {
		if v.Reviewers[role].Verdict != "PASS" {
			t.Fatalf("expected reviewer %s PASS, got %q", role, v.Reviewers[role].Verdict)
		}
	}
}

func TestBuildPartialReviewVerdict(t *testing.T) {
	cp := &Checkpoint{FeatureID: "F098", Review: &ReviewStatus{Iteration: 4}}
	roleFindings := map[string][]string{
		"security": {"sdplab-10"},
		"docs":     {"sdplab-11"},
	}
	v := buildPartialReviewVerdict(cp, "partial pass", []string{"security", "docs"}, roleFindings)
	if v.Verdict != "PARTIALLY_APPROVED" {
		t.Fatalf("expected PARTIALLY_APPROVED, got %q", v.Verdict)
	}
	if len(v.PartialFailingRoles) != 2 {
		t.Fatalf("expected 2 failing roles, got %d", len(v.PartialFailingRoles))
	}
	if v.Reviewers["security"].Verdict != "FAIL" {
		t.Fatalf("expected security FAIL, got %q", v.Reviewers["security"].Verdict)
	}
	if len(v.Reviewers["security"].Findings) != 1 || v.Reviewers["security"].Findings[0] != "sdplab-10" {
		t.Fatalf("expected security finding sdplab-10, got %v", v.Reviewers["security"].Findings)
	}
	if v.Reviewers["docs"].Verdict != "FAIL" {
		t.Fatalf("expected docs FAIL, got %q", v.Reviewers["docs"].Verdict)
	}
	if len(v.Reviewers["docs"].Findings) != 1 || v.Reviewers["docs"].Findings[0] != "sdplab-11" {
		t.Fatalf("expected docs finding sdplab-11, got %v", v.Reviewers["docs"].Findings)
	}
	if v.Reviewers["qa"].Verdict != "PASS" {
		t.Fatalf("expected qa PASS, got %q", v.Reviewers["qa"].Verdict)
	}
	if len(v.Reviewers["qa"].Findings) != 0 {
		t.Fatalf("expected qa no findings, got %v", v.Reviewers["qa"].Findings)
	}
	if len(v.FindingIDs) != 2 {
		t.Fatalf("expected 2 total finding ids, got %d", len(v.FindingIDs))
	}
}

func TestBuildEscalatedReviewVerdict(t *testing.T) {
	cp := &Checkpoint{FeatureID: "F098", Review: &ReviewStatus{Iteration: 4}}
	v := buildEscalatedReviewVerdict(cp, "escalated to human", "sdplab-99")
	if v.Verdict != "ESCALATED" {
		t.Fatalf("expected ESCALATED, got %q", v.Verdict)
	}
	if v.EscalationIssue != "sdplab-99" {
		t.Fatalf("expected escalation issue sdplab-99, got %q", v.EscalationIssue)
	}
	for _, role := range []string{"qa", "security", "devops", "sre", "techlead", "docs", "promptops"} {
		if v.Reviewers[role].Verdict != "FAIL" {
			t.Fatalf("expected reviewer %s FAIL, got %q", role, v.Reviewers[role].Verdict)
		}
	}
}

func TestValidateOverrideReason(t *testing.T) {
	if err := ValidateOverrideReason(""); err == nil {
		t.Fatal("expected error for empty override reason")
	}
	if err := ValidateOverrideReason("   "); err == nil {
		t.Fatal("expected error for whitespace-only override reason")
	}
	if err := ValidateOverrideReason("valid reason"); err != nil {
		t.Fatalf("unexpected error for valid reason: %v", err)
	}
}
