package evidence

import "testing"

func TestAllTestsPass(t *testing.T) {
	stmt := CodingWorkflowStatement{}
	stmt.Predicate.Verification.Tests = []GateResult{
		{Name: "go-test", Status: "pass (10 passed, 0 failed)"},
	}

	if !AllTestsPass(stmt) {
		t.Fatal("expected pass=true when all statuses are pass")
	}
}

func TestAllTestsPass_FailsOnFailPrefix(t *testing.T) {
	stmt := CodingWorkflowStatement{}
	stmt.Predicate.Verification.Tests = []GateResult{
		{Name: "go-test", Status: "fail (9 passed, 1 failed)"},
	}

	if AllTestsPass(stmt) {
		t.Fatal("expected pass=false when any status has fail prefix")
	}
}
