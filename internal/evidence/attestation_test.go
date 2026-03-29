package evidence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	intoto "github.com/in-toto/in-toto-golang/in_toto"
	"github.com/in-toto/in-toto-golang/in_toto/slsa_provenance/common"
)

func TestNewStatement(t *testing.T) {
	subjects := []intoto.Subject{{ //nolint:staticcheck // intoto v0 types
		Name:   "PR #42",
		Digest: common.DigestSet{"sha256": "abc123"},
	}}
	predicate := CodingWorkflowPredicate{
		Intent: Intent{IssueID: "sdp_dev-abc", Trigger: "ci"},
		Plan:   Plan{Workstreams: []string{"00-053-01"}},
		Trace:  Trace{Branch: "main", PRURL: "https://github.com/org/repo/pull/42"},
	}
	stmt := NewStatement(subjects, predicate)
	if stmt.Type != StatementType {
		t.Errorf("Type = %q, want %q", stmt.Type, StatementType)
	}
	if stmt.PredicateType != PredicateTypeCodingWorkflow {
		t.Errorf("PredicateType = %q", stmt.PredicateType)
	}
	if len(stmt.Subject) != 1 || stmt.Subject[0].Name != "PR #42" {
		t.Errorf("Subject = %+v", stmt.Subject)
	}
	if stmt.Predicate.Intent.IssueID != "sdp_dev-abc" {
		t.Errorf("Intent.IssueID = %q", stmt.Predicate.Intent.IssueID)
	}
}

func TestWriteAttestation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "attestation.json")
	stmt := NewStatement(
		[]intoto.Subject{{Name: "test", Digest: common.DigestSet{"sha256": "abc"}}}, //nolint:staticcheck
		CodingWorkflowPredicate{Intent: Intent{IssueID: "x"}, Trace: Trace{Branch: "main"}},
	)
	if err := WriteAttestation(path, stmt); err != nil {
		t.Fatalf("WriteAttestation: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("file should exist: %v", err)
	}
	read, err := ReadAttestation(path)
	if err != nil {
		t.Fatalf("ReadAttestation: %v", err)
	}
	if read.Predicate.Intent.IssueID != "x" {
		t.Errorf("read Intent.IssueID = %q", read.Predicate.Intent.IssueID)
	}
}

func TestDispatchEvidence_JSON(t *testing.T) {
	orig := &DispatchEvidence{
		Harness:   "claude-code",
		Provider:  "anthropic",
		Model:     "claude-sonnet-4-20250514",
		Score:     0.92,
		Reason:    "best match for Go refactoring",
		ColdStart: true,
		Alternatives: []struct {
			Harness string  `json:"harness"`
			Score   float64 `json:"score"`
		}{
			{Harness: "aider", Score: 0.78},
			{Harness: "copilot-workspace", Score: 0.65},
		},
	}

	b, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got DispatchEvidence
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Harness != orig.Harness {
		t.Errorf("Harness = %q, want %q", got.Harness, orig.Harness)
	}
	if got.Provider != orig.Provider {
		t.Errorf("Provider = %q, want %q", got.Provider, orig.Provider)
	}
	if got.Model != orig.Model {
		t.Errorf("Model = %q, want %q", got.Model, orig.Model)
	}
	if got.Score != orig.Score {
		t.Errorf("Score = %v, want %v", got.Score, orig.Score)
	}
	if got.Reason != orig.Reason {
		t.Errorf("Reason = %q, want %q", got.Reason, orig.Reason)
	}
	if !got.ColdStart {
		t.Error("ColdStart should be true")
	}
	if len(got.Alternatives) != 2 {
		t.Fatalf("Alternatives len = %d, want 2", len(got.Alternatives))
	}
	if got.Alternatives[0].Harness != "aider" || got.Alternatives[0].Score != 0.78 {
		t.Errorf("Alternatives[0] = %+v", got.Alternatives[0])
	}
}

func TestDispatchEvidence_JSON_OmitEmpty(t *testing.T) {
	de := &DispatchEvidence{
		Harness:  "claude-code",
		Provider: "anthropic",
		Model:    "claude-sonnet-4-20250514",
		Score:    0.85,
	}
	b, err := json.Marshal(de)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(b)
	if contains(s, "reason") {
		t.Error("empty reason should be omitted")
	}
	if contains(s, "cold_start") {
		t.Error("false cold_start should be omitted")
	}
	if contains(s, "alternatives") {
		t.Error("nil alternatives should be omitted")
	}
}

func TestCodingWorkflowPredicate_WithDispatch(t *testing.T) {
	predicate := CodingWorkflowPredicate{
		Intent: Intent{IssueID: "sdplab-0lxh", Trigger: "sdp-orchestrate"},
		Provenance: Provenance{
			RunID:      "orch-F028-abc12345",
			CapturedAt: "2026-03-28T00:00:00Z",
		},
		Boundary: Boundary{
			Compliance: BoundaryCompliance{OK: true, Reason: "all files in scope"},
		},
		Dispatch: &DispatchEvidence{
			Harness:  "claude-code",
			Provider: "anthropic",
			Model:    "claude-sonnet-4-20250514",
			Score:    0.92,
		},
	}

	stmt := NewStatement(
		[]intoto.Subject{{Name: "branch:feature/test", Digest: common.DigestSet{"sha256": "deadbeef"}}}, //nolint:staticcheck
		predicate,
	)

	// Roundtrip through JSON
	b, err := json.MarshalIndent(stmt, "", "  ")
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var got CodingWorkflowStatement
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if got.Predicate.Dispatch == nil {
		t.Fatal("Dispatch should not be nil after roundtrip")
	}
	if got.Predicate.Dispatch.Harness != "claude-code" {
		t.Errorf("Dispatch.Harness = %q", got.Predicate.Dispatch.Harness)
	}
	if got.Predicate.Dispatch.Score != 0.92 {
		t.Errorf("Dispatch.Score = %v", got.Predicate.Dispatch.Score)
	}

	// Validate the statement
	result := ValidateAttestation(got, false)
	if !result.OK {
		t.Errorf("ValidateAttestation failed: %s", result.Reason)
	}
}

func TestCodingWorkflowPredicate_WithoutDispatch(t *testing.T) {
	predicate := CodingWorkflowPredicate{
		Intent: Intent{IssueID: "sdplab-0lxh", Trigger: "sdp-orchestrate"},
		Provenance: Provenance{
			RunID:      "orch-F028-abc12345",
			CapturedAt: "2026-03-28T00:00:00Z",
		},
	}
	b, err := json.Marshal(predicate)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if contains(string(b), "dispatch") {
		t.Error("nil dispatch should be omitted from JSON")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsSubstring(s, substr)
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
