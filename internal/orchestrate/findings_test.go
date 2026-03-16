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
