package evidence

import (
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
