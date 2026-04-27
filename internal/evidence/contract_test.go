package evidence

import (
	"testing"
)

func TestContractVersion(t *testing.T) {
	if ContractVersion != "v1.0.0" {
		t.Errorf("ContractVersion = %s, want v1.0.0", ContractVersion)
	}
}

// mockAttester implements Attester for compile-time interface checks
type mockAttester struct{}

func (m *mockAttester) Attest(opts AutoAttestOptions) (CodingWorkflowStatement, error) {
	return CodingWorkflowStatement{}, nil
}

func (m *mockAttester) Validate(stmt CodingWorkflowStatement, requirePRURL bool) Result {
	return Result{OK: true}
}

func (m *mockAttester) Sign(stmt CodingWorkflowStatement) ([]byte, error) {
	return []byte("{}"), nil
}

func (m *mockAttester) Verify(signed []byte) (CodingWorkflowStatement, error) {
	return CodingWorkflowStatement{}, nil
}

func TestAttesterInterface(t *testing.T) {
	// Compile-time check that mockAttester implements Attester
	var _ Attester = (*mockAttester)(nil)

	ma := &mockAttester{}

	// Test Attest method exists and has correct signature
	opts := AutoAttestOptions{BaseBranch: "main"}
	stmt, err := ma.Attest(opts)
	if err != nil {
		t.Fatalf("Attest() failed: %v", err)
	}
	// We don't check stmt content, just that it returns the right types
	_ = stmt

	// Test Validate method exists
	stmt = CodingWorkflowStatement{}
	result := ma.Validate(stmt, false)
	if !result.OK {
		t.Fatalf("Validate() returned OK=false, want true")
	}

	// Test Sign method exists
	signed, err := ma.Sign(stmt)
	if err != nil {
		t.Fatalf("Sign() failed: %v", err)
	}
	if len(signed) == 0 {
		t.Fatal("Sign() returned empty bytes")
	}

	// Test Verify method exists
	verifiedStmt, err := ma.Verify(signed)
	if err != nil {
		t.Fatalf("Verify() failed: %v", err)
	}
	_ = verifiedStmt
}

// mockDiscrepancyDetector implements DiscrepancyDetector
type mockDiscrepancyDetector struct{}

func (m *mockDiscrepancyDetector) Compare(runID string, agentDir, ciDir string) (DiscrepancyReport, error) {
	return DiscrepancyReport{OK: true, RunID: runID}, nil
}

func TestDiscrepancyDetectorInterface(t *testing.T) {
	// Compile-time check
	var _ DiscrepancyDetector = (*mockDiscrepancyDetector)(nil)

	mdd := &mockDiscrepancyDetector{}
	report, err := mdd.Compare("test-run", "/tmp/agent", "/tmp/ci")
	if err != nil {
		t.Fatalf("Compare() failed: %v", err)
	}
	if report.RunID != "test-run" {
		t.Errorf("Compare() RunID = %s, want test-run", report.RunID)
	}
}

// mockInspector implements Inspector
type mockInspector struct{}

func (m *mockInspector) Inspect(path string, requirePRURL bool) (string, Result, error) {
	return "summary", Result{OK: true}, nil
}

func TestInspectorInterface(t *testing.T) {
	// Compile-time check
	var _ Inspector = (*mockInspector)(nil)

	mi := &mockInspector{}
	summary, result, err := mi.Inspect("/path/to/file.json", false)
	if err != nil {
		t.Fatalf("Inspect() failed: %v", err)
	}
	if summary != "summary" {
		t.Errorf("Inspect() summary = %s, want summary", summary)
	}
	if !result.OK {
		t.Error("Inspect() returned OK=false, want true")
	}
}

// mockTraceValidator implements TraceValidator
type mockTraceValidator struct{}

func (m *mockTraceValidator) ValidateChain(events []TraceEvent) TraceValidationResult {
	return TraceValidationResult{OK: true}
}

func TestTraceValidatorInterface(t *testing.T) {
	// Compile-time check
	var _ TraceValidator = (*mockTraceValidator)(nil)

	mtv := &mockTraceValidator{}
	result := mtv.ValidateChain([]TraceEvent{})
	if !result.OK {
		t.Error("ValidateChain() returned OK=false, want true")
	}
}

func TestIngestContractDocumentation(t *testing.T) {
	// Verify IngestContract has the expected documentation fields
	ic := IngestContract{
		RunID:         "test-run",
		SubjectDigest: "abc123",
		Timestamp:     "2024-01-01T00:00:00Z",
	}

	// Just verify the struct compiles with expected fields
	if ic.RunID == "" {
		t.Error("IngestContract.RunID is empty")
	}
	if ic.SubjectDigest == "" {
		t.Error("IngestContract.SubjectDigest is empty")
	}
	if ic.Timestamp == "" {
		t.Error("IngestContract.Timestamp is empty")
	}
}

func TestRenderContractDocumentation(t *testing.T) {
	// Verify RenderContract has the expected documentation fields
	rc := RenderContract{
		Format:                  "json",
		CanonicalJSON:           true,
		StatementHeaderRequired: true,
	}

	// Just verify the struct compiles with expected fields
	if rc.Format != "json" {
		t.Errorf("RenderContract.Format = %s, want json", rc.Format)
	}
	if !rc.CanonicalJSON {
		t.Error("RenderContract.CanonicalJSON = false, want true")
	}
	if !rc.StatementHeaderRequired {
		t.Error("RenderContract.StatementHeaderRequired = false, want true")
	}
}
