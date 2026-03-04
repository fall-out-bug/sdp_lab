package main

import (
	"testing"

	"sdp_dev/internal/evidence"
)

func TestShouldFailClosed_FalseWhenTestsPass(t *testing.T) {
	stmt := evidence.CodingWorkflowStatement{}
	stmt.Predicate.Verification.Tests = []evidence.GateResult{
		{Name: "go-test", Status: "pass (3 passed, 0 failed)"},
	}

	if shouldFailClosed(stmt) {
		t.Fatal("expected shouldFailClosed=false for passing tests")
	}
}

func TestShouldFailClosed_TrueWhenTestsFail(t *testing.T) {
	stmt := evidence.CodingWorkflowStatement{}
	stmt.Predicate.Verification.Tests = []evidence.GateResult{
		{Name: "go-test", Status: "fail (2 passed, 1 failed)"},
	}

	if !shouldFailClosed(stmt) {
		t.Fatal("expected shouldFailClosed=true for failing tests")
	}
}
