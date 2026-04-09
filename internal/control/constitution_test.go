package control

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConstitutionReturnsDefaultWhenMissing(t *testing.T) {
	root := t.TempDir()
	constitution, err := LoadConstitution(root)
	if err != nil {
		t.Fatal(err)
	}
	if constitution == nil {
		t.Fatal("constitution is nil")
		return
	}
	if constitution.Vision == "" {
		t.Fatal("default vision should be set")
	}
	if got := constitution.MaxRiskDefault; got != "medium" {
		t.Fatalf("max_risk_default = %q, want medium", got)
	}
	if len(constitution.AllowedTaskTypes) == 0 {
		t.Fatal("default allowed_task_types should be set")
	}
}

func TestValidateCardWithValidTaskTypeHasNoWarnings(t *testing.T) {
	constitution := defaultConstitution()
	warnings := constitution.ValidateCard(&FeatureCard{TaskType: "feature", RiskLevel: "medium"})
	if len(warnings) != 0 {
		t.Fatalf("warnings = %+v, want none", warnings)
	}
}

func TestValidateCardWithInvalidTaskTypeWarns(t *testing.T) {
	constitution := defaultConstitution()
	warnings := constitution.ValidateCard(&FeatureCard{TaskType: "migration", RiskLevel: "medium"})
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1 (%+v)", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "task_type") {
		t.Fatalf("warning = %q, want task_type message", warnings[0])
	}
}

func TestValidateCardWithHigherRiskWarns(t *testing.T) {
	constitution := defaultConstitution()
	warnings := constitution.ValidateCard(&FeatureCard{TaskType: "feature", RiskLevel: "high"})
	if len(warnings) != 1 {
		t.Fatalf("warnings len = %d, want 1 (%+v)", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "risk_level") {
		t.Fatalf("warning = %q, want risk_level message", warnings[0])
	}
}

func TestCreateCardStoresConstitutionWarnings(t *testing.T) {
	store := setupStore(t)
	if err := os.MkdirAll(filepath.Join(store.ProjectRoot, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	constitutionDoc := []byte("allowed_task_types:\n  - feature\nmax_risk_default: medium\n")
	if err := os.WriteFile(filepath.Join(store.ProjectRoot, "docs", "constitution.yaml"), constitutionDoc, 0o644); err != nil {
		t.Fatal(err)
	}

	card, err := store.CreateCard("openclaw", "Risky migration", "do a weird thing")
	if err != nil {
		t.Fatal(err)
	}
	card.TaskType = "migration"
	card.RiskLevel = "high"
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	constitution, err := LoadConstitution(filepath.Join(store.ProjectRoot, "docs"))
	if err != nil {
		t.Fatal(err)
	}
	card.ConstitutionWarnings = constitution.ValidateCard(card)
	if err := store.SaveCard(card); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.LoadCard("openclaw", card.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := len(loaded.ConstitutionWarnings); got != 2 {
		t.Fatalf("constitution_warnings len = %d, want 2 (%+v)", got, loaded.ConstitutionWarnings)
	}
}
