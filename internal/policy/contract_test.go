package policy

import (
	"context"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/evidence"
)

// mockEvaluator implements Evaluator for testing
type mockEvaluator struct {
	allowed bool
	err     error
}

func (m *mockEvaluator) Evaluate(ctx context.Context, input EvidenceInput) (Decision, error) {
	if m.err != nil {
		return Decision{}, m.err
	}
	return NewDecision(m.allowed, ExplainResult{
		Decision: "allow",
		Reason:   "mock evaluation",
	}), nil
}

// mockExplainer implements Explainer for testing
type mockExplainer struct{}

func (m *mockExplainer) Explain(ctx context.Context, allowed bool, reasons []string, severityCount map[string]int, mode ExplainMode) (ExplainResult, error) {
	return ExplainResult{
		Decision:  decisionString(allowed),
		Reason:    "mock explanation",
		Sanitized: mode != ExplainModeInternal,
	}, nil
}

// mockRuleStore implements RuleStore for testing
type mockRuleStore struct {
	rules []Rule
	err   error
}

func (m *mockRuleStore) LoadRules(ctx context.Context) ([]Rule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.rules, nil
}

func (m *mockRuleStore) SaveRule(ctx context.Context, rule Rule) error {
	return m.err
}

func TestContractVersion(t *testing.T) {
	if ContractVersion != "v1.0.0" {
		t.Errorf("Expected ContractVersion to be v1.0.0, got %s", ContractVersion)
	}
}

func TestEvaluatorInterface(t *testing.T) {
	var _ Evaluator = (*mockEvaluator)(nil)

	ctx := context.Background()
	eval := &mockEvaluator{allowed: true}

	input := EvidenceInput{
		Attestation: []byte("{}"),
		DiscrepancyReport: evidence.DiscrepancyReport{
			RunID: "test-run",
			OK:    true,
		},
		GateConfig: GateConfig{
			RequireSignedAttestation: false,
			Thresholds: DiscrepancyThresholds{
				Critical: 0,
				High:     0,
				Medium:   5,
				Low:      20,
			},
		},
	}

	decision, err := eval.Evaluate(ctx, input)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}

	if !decision.Allowed {
		t.Error("Expected decision to be allowed")
	}
}

func TestExplainerInterface(t *testing.T) {
	var _ Explainer = (*mockExplainer)(nil)

	ctx := context.Background()
	explainer := &mockExplainer{}

	result, err := explainer.Explain(ctx, true, []string{"test reason"}, map[string]int{"critical": 0}, ExplainModeDetailed)
	if err != nil {
		t.Fatalf("Explain failed: %v", err)
	}

	if result.Decision != "allow" {
		t.Errorf("Expected decision 'allow', got %s", result.Decision)
	}

	if !result.Sanitized {
		t.Error("Expected Detailed mode to be sanitized")
	}
}

func TestRuleStoreInterface(t *testing.T) {
	var _ RuleStore = (*mockRuleStore)(nil)

	ctx := context.Background()
	store := &mockRuleStore{
		rules: []Rule{
			{ID: "rule-1", Type: "boundary", Severity: "critical", Condition: "true", Enabled: true},
			{ID: "rule-2", Type: "verification", Severity: "high", Condition: "false", Enabled: false},
		},
	}

	rules, err := store.LoadRules(ctx)
	if err != nil {
		t.Fatalf("LoadRules failed: %v", err)
	}

	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	rule := Rule{ID: "rule-3", Type: "discrepancy", Severity: "medium", Condition: "count > 5", Enabled: true}
	err = store.SaveRule(ctx, rule)
	if err != nil {
		t.Fatalf("SaveRule failed: %v", err)
	}
}

func TestEvidenceInputStructure(t *testing.T) {
	input := EvidenceInput{
		Attestation: []byte(`{"signatures": []}`),
		DiscrepancyReport: evidence.DiscrepancyReport{
			RunID: "test-123",
			OK:    false,
			Discrepancies: []evidence.Discrepancy{
				{Type: "test", Severity: "high", Description: "test discrepancy"},
			},
		},
		GateConfig: GateConfig{
			RequireSignedAttestation: true,
			Thresholds: DiscrepancyThresholds{
				Critical: 0,
				High:     1,
				Medium:   10,
				Low:      50,
			},
		},
	}

	if len(input.Attestation) == 0 {
		t.Error("Expected attestation to be set")
	}

	if input.GateConfig.Thresholds.High != 1 {
		t.Error("Expected high threshold to be 1")
	}

	if len(input.DiscrepancyReport.Discrepancies) != 1 {
		t.Error("Expected 1 discrepancy")
	}
}

func TestRuleStructure(t *testing.T) {
	rule := Rule{
		ID:        "test-rule",
		Type:      "boundary",
		Severity:  "critical",
		Condition: "boundary.compliance.ok == true",
		Enabled:   true,
	}

	if rule.ID != "test-rule" {
		t.Errorf("Expected ID 'test-rule', got %s", rule.ID)
	}

	if rule.Type != "boundary" {
		t.Errorf("Expected Type 'boundary', got %s", rule.Type)
	}

	if rule.Severity != "critical" {
		t.Errorf("Expected Severity 'critical', got %s", rule.Severity)
	}

	if !rule.Enabled {
		t.Error("Expected rule to be enabled")
	}
}

func TestEvaluationContract(t *testing.T) {
	contract := DefaultEvaluationContract()

	if contract.Version != ContractVersion {
		t.Errorf("Expected version %s, got %s", ContractVersion, contract.Version)
	}

	if !contract.DeterministicOrdering {
		t.Error("Expected deterministic ordering to be enabled")
	}

	if !contract.ShortCircuitDeny {
		t.Error("Expected short-circuit deny to be enabled")
	}

	if !contract.OverrideHooksEnabled {
		t.Error("Expected override hooks to be enabled")
	}

	if !contract.AuditTrailEnabled {
		t.Error("Expected audit trail to be enabled")
	}
}

func TestOverrideHookSignature(t *testing.T) {
	var hook OverrideHook = func(ctx context.Context, decision Decision) (Decision, error) {
		decision.Allowed = true
		return decision, nil
	}

	ctx := context.Background()
	decision := Decision{Allowed: false}

	result, err := hook(ctx, decision)
	if err != nil {
		t.Fatalf("OverrideHook failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected hook to flip decision to allowed")
	}
}

func TestEvaluateEvidenceGate(t *testing.T) {
	ctx := context.Background()

	// Create a minimal valid attestation payload
	attestation := []byte(`{"predicate": {"provenance": {"run_id": "test-run"}}}`)

	input := EvidenceInput{
		Attestation: attestation,
		DiscrepancyReport: evidence.DiscrepancyReport{
			RunID:         "test-run",
			OK:            true,
			Discrepancies: []evidence.Discrepancy{},
		},
		GateConfig: GateConfig{
			RequireSignedAttestation: false,
			Thresholds: DiscrepancyThresholds{
				Critical: 0,
				High:     0,
				Medium:   5,
				Low:      20,
			},
		},
	}

	decision, err := EvaluateEvidenceGate(ctx, input)
	if err != nil {
		t.Fatalf("EvaluateEvidenceGate failed: %v", err)
	}

	// Note: Verification may fail if attestation is not properly formatted
	// The test verifies the contract API is callable, not that attestation is valid
	if decision.Explanation.Decision == "" {
		t.Error("Expected explanation decision to be set")
	}
}
