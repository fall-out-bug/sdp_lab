package control

import (
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/harness"
)

func TestGenerateContractFromCard(t *testing.T) {
	card := &FeatureCard{
		ID:               "feature-openclaw-2026-03-24-001",
		Title:            "Fallback title",
		NormalizedIntent: "Ship the ready-gate derived contract",
		AcceptanceShape:  []string{"first criterion\nsecond criterion", "third criterion"},
		ScopeOut:         []string{"contract json", "evidence index"},
	}

	contract, err := GenerateContractFromCard(card)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Objective != card.NormalizedIntent {
		t.Fatalf("objective = %q, want %q", contract.Objective, card.NormalizedIntent)
	}
	if got := len(contract.AcceptanceCriteria); got != 3 {
		t.Fatalf("acceptance criteria len = %d, want 3", got)
	}
	if contract.AcceptanceCriteria[0].ID != "ac-1" || contract.AcceptanceCriteria[1].ID != "ac-2" || contract.AcceptanceCriteria[2].ID != "ac-3" {
		t.Fatalf("unexpected AC IDs: %+v", contract.AcceptanceCriteria)
	}
	for _, ac := range contract.AcceptanceCriteria {
		if ac.Priority != "required" {
			t.Fatalf("AC priority = %q, want required", ac.Priority)
		}
	}
	if len(contract.RequiredEvidence) != 2 {
		t.Fatalf("required evidence len = %d, want 2", len(contract.RequiredEvidence))
	}
	assertAllQualityGatesEnabled(t, contract)
	if contract.Constraints.AllowScopeReduction {
		t.Fatal("allow_scope_reduction = true, want false")
	}
	if contract.Constraints.AllowMetricReduction {
		t.Fatal("allow_metric_reduction = true, want false")
	}
	if contract.Version != "v1" {
		t.Fatalf("version = %q, want v1", contract.Version)
	}
	if contract.RunID != card.ID {
		t.Fatalf("run_id = %q, want %q", contract.RunID, card.ID)
	}
	if contract.CreatedAt == "" {
		t.Fatal("created_at should be set")
	}
}

func TestGenerateContractFromCardFallsBackToTitle(t *testing.T) {
	contract, err := GenerateContractFromCard(&FeatureCard{Title: "Use title", AcceptanceShape: nil})
	if err != nil {
		t.Fatal(err)
	}
	if contract.Objective != "Use title" {
		t.Fatalf("objective = %q, want title fallback", contract.Objective)
	}
}

func TestGenerateContractFromCardWithoutAcceptanceShape(t *testing.T) {
	contract, err := GenerateContractFromCard(&FeatureCard{Title: "No ACs"})
	if err != nil {
		t.Fatal(err)
	}
	if len(contract.AcceptanceCriteria) != 0 {
		t.Fatalf("acceptance criteria len = %d, want 0", len(contract.AcceptanceCriteria))
	}
	assertAllQualityGatesEnabled(t, contract)
}

func TestMarkReadyGeneratesAndPersistsContract(t *testing.T) {
	store := setupStore(t)
	card, err := store.CreateCard("openclaw", "Unify reminders", "make reminders smarter")
	if err != nil {
		t.Fatal(err)
	}
	card, err = store.ClarifyCard("openclaw", card.ID, "unify reminder policy", "feature", "openclaw", "medium", "ask one product question", []string{"escalation levels"}, []string{"contract json", "evidence note"})
	if err != nil {
		t.Fatal(err)
	}
	card.AcceptanceShape = []string{"AC one", "AC two"}
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	card, err = store.MarkReady("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}

	contractPath := filepath.Join(store.ControlRoot, "contracts", card.ID+".json")
	contract, err := harness.LoadTaskContract(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if contract.Objective != "unify reminder policy" {
		t.Fatalf("objective = %q, want normalized intent", contract.Objective)
	}
	if len(contract.AcceptanceCriteria) != 2 {
		t.Fatalf("acceptance criteria len = %d, want 2", len(contract.AcceptanceCriteria))
	}
	found := false
	for _, artifact := range card.RequiredArtifacts {
		if artifact == filepath.ToSlash(contractPath) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("required_artifacts missing contract path %q: %+v", filepath.ToSlash(contractPath), card.RequiredArtifacts)
	}
}

func assertAllQualityGatesEnabled(t *testing.T, contract *harness.TaskContract) {
	t.Helper()
	if !contract.QualityGates.Build || !contract.QualityGates.Test || !contract.QualityGates.Lint || !contract.QualityGates.Typecheck {
		t.Fatalf("quality gates not all enabled: %+v", contract.QualityGates)
	}
}
