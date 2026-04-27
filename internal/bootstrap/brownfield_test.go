package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/scout"
)

// --- CompareSections tests ---

func TestCompareSections_NoChanges(t *testing.T) {
	existing := map[string]string{
		"language": "go(10 files, 90%)",
		"ci":      "system=github-actions",
	}
	fresh := map[string]string{
		"language": "go(10 files, 90%)",
		"ci":      "system=github-actions",
	}
	deltas := CompareSections(existing, fresh)
	if len(deltas) != 0 {
		t.Fatalf("expected 0 deltas for identical maps, got %d: %+v", len(deltas), deltas)
	}
}

func TestCompareSections_AllAdded(t *testing.T) {
	existing := map[string]string{}
	fresh := map[string]string{
		"ci":     "system=github-actions",
		"testing": "test_files=5, test_ratio=0.50",
	}
	deltas := CompareSections(existing, fresh)
	if len(deltas) != 2 {
		t.Fatalf("expected 2 ADDED deltas, got %d", len(deltas))
	}
	for _, d := range deltas {
		if d.ChangeType != DeltaAdded {
			t.Errorf("expected ADDED, got %s for section %q", d.ChangeType, d.Section)
		}
		if d.Old != "" {
			t.Errorf("ADDED delta should have empty Old, got %q for section %q", d.Old, d.Section)
		}
		if d.New == "" {
			t.Errorf("ADDED delta should have non-empty New for section %q", d.Section)
		}
	}
}

func TestCompareSections_AllRemoved(t *testing.T) {
	existing := map[string]string{
		"ci":     "system=github-actions",
		"testing": "test_files=5",
	}
	fresh := map[string]string{}
	deltas := CompareSections(existing, fresh)
	if len(deltas) != 2 {
		t.Fatalf("expected 2 REMOVED deltas, got %d", len(deltas))
	}
	for _, d := range deltas {
		if d.ChangeType != DeltaRemoved {
			t.Errorf("expected REMOVED, got %s for section %q", d.ChangeType, d.Section)
		}
		if d.New != "" {
			t.Errorf("REMOVED delta should have empty New, got %q for section %q", d.New, d.Section)
		}
	}
}

func TestCompareSections_Modified(t *testing.T) {
	existing := map[string]string{
		"linting": "tool=golangci-lint, config=.golangci.yml",
	}
	fresh := map[string]string{
		"linting": "tool=golangci-lint, config=.golangci.yaml",
	}
	deltas := CompareSections(existing, fresh)
	if len(deltas) != 1 {
		t.Fatalf("expected 1 delta, got %d", len(deltas))
	}
	d := deltas[0]
	if d.ChangeType != DeltaModified {
		t.Errorf("expected MODIFIED, got %s", d.ChangeType)
	}
	if d.Section != "linting" {
		t.Errorf("expected section 'linting', got %q", d.Section)
	}
	if d.Old != existing["linting"] {
		t.Errorf("Old mismatch: got %q, want %q", d.Old, existing["linting"])
	}
	if d.New != fresh["linting"] {
		t.Errorf("New mismatch: got %q, want %q", d.New, fresh["linting"])
	}
}

func TestCompareSections_DeterministicOrder(t *testing.T) {
	existing := map[string]string{"a": "1", "b": "2", "c": "3"}
	fresh := map[string]string{"a": "changed", "d": "4"}
	deltas := CompareSections(existing, fresh)
	// Expect: a=MODIFIED, b=REMOVED, c=REMOVED, d=ADDED
	if len(deltas) != 4 {
		t.Fatalf("expected 4 deltas, got %d", len(deltas))
	}
	// Verify sorted by section name
	for i := 1; i < len(deltas); i++ {
		if deltas[i].Section < deltas[i-1].Section {
			t.Errorf("deltas not sorted: %q before %q", deltas[i-1].Section, deltas[i].Section)
		}
	}
}

// --- Delta JSON serialization ---

func TestDeltaJSONSerialization(t *testing.T) {
	d := Delta{
		Section:    "ci",
		ChangeType: DeltaAdded,
		New:        "system=github-actions",
		Evidence:   "CI detected by fresh scout",
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal delta: %v", err)
	}
	var decoded Delta
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal delta: %v", err)
	}
	if decoded.Section != d.Section {
		t.Errorf("Section: got %q, want %q", decoded.Section, d.Section)
	}
	if decoded.ChangeType != d.ChangeType {
		t.Errorf("ChangeType: got %q, want %q", decoded.ChangeType, d.ChangeType)
	}
	if decoded.Old != "" {
		t.Errorf("Old should be empty for ADDED, got %q", decoded.Old)
	}
	if decoded.New != d.New {
		t.Errorf("New: got %q, want %q", decoded.New, d.New)
	}
	if decoded.Evidence != d.Evidence {
		t.Errorf("Evidence: got %q, want %q", decoded.Evidence, d.Evidence)
	}
}

func TestDeltaJSONSerialization_Removed(t *testing.T) {
	d := Delta{
		Section:    "testing",
		ChangeType: DeltaRemoved,
		Old:        "test_files=5",
		Evidence:   "tests section absent from fresh scout",
	}
	data, err := json.Marshal(d)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	// Verify "new" is omitted via omitempty when empty
	raw := string(data)
	if strings.Contains(raw, `"new"`) {
		t.Errorf("REMOVED delta should omit 'new' field, got: %s", raw)
	}
	var decoded Delta
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.ChangeType != DeltaRemoved {
		t.Errorf("ChangeType: got %q, want %q", decoded.ChangeType, DeltaRemoved)
	}
}

// --- RunBrownfield integration tests ---

func TestRunBrownfield_NoChanges(t *testing.T) {
	card := sampleProjectCard()
	existing := extractFreshSections(card)

	result := RunBrownfield(card, existing)
	if len(result.Deltas) != 0 {
		t.Fatalf("identical rules should produce 0 deltas, got %d", len(result.Deltas))
	}
	if len(result.ExistingRules) != len(result.FreshScout) {
		t.Errorf("ExistingRules and FreshScout should have equal length")
	}
}

func TestRunBrownfield_NewCI(t *testing.T) {
	card := sampleProjectCard()
	// Remove CI from existing rules to simulate CI being newly detected
	existing := extractFreshSections(card)
	delete(existing, "ci")

	result := RunBrownfield(card, existing)
	found := false
	for _, d := range result.Deltas {
		if d.Section == "ci" && d.ChangeType == DeltaAdded {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected ADDED delta for 'ci' section")
	}
}

func TestRunBrownfield_RemovedTests(t *testing.T) {
	card := sampleProjectCard()
	existing := extractFreshSections(card)
	// Add a testing section that won't be in fresh (simulate removal)
	// by modifying the card to have no test files.
	cardNoTests := sampleProjectCard()
	cardNoTests.Scale.TestFiles = 0
	cardNoTests.Scale.TestRatio = 0

	result := RunBrownfield(cardNoTests, existing)
	// The testing section should differ because test_files changed
	found := false
	for _, d := range result.Deltas {
		if d.Section == "testing" {
			found = true
			if d.ChangeType != DeltaModified {
				t.Errorf("expected MODIFIED for testing, got %s", d.ChangeType)
			}
			break
		}
	}
	if !found {
		t.Error("expected a delta for 'testing' section")
	}
}

func TestRunBrownfield_ModifiedLint(t *testing.T) {
	card := sampleProjectCard()
	existing := extractFreshSections(card)
	// Modify linting in existing to simulate a change
	if _, hasLint := existing["linting"]; hasLint {
		existing["linting"] = "tool=old-linter, config=.oldconfig"
	}

	result := RunBrownfield(card, existing)
	found := false
	for _, d := range result.Deltas {
		if d.Section == "linting" && d.ChangeType == DeltaModified {
			found = true
			if d.Old != "tool=old-linter, config=.oldconfig" {
				t.Errorf("Old mismatch: got %q", d.Old)
			}
			break
		}
	}
	if !found {
		t.Error("expected MODIFIED delta for 'linting' section")
	}
}

func TestRunBrownfield_NilCard(t *testing.T) {
	existing := map[string]string{"ci": "system=github-actions"}
	result := RunBrownfield(nil, existing)
	if len(result.Deltas) != 1 {
		t.Fatalf("expected 1 REMOVED delta for nil card, got %d", len(result.Deltas))
	}
	if result.Deltas[0].ChangeType != DeltaRemoved {
		t.Errorf("expected REMOVED, got %s", result.Deltas[0].ChangeType)
	}
}

func TestRunBrownfield_NilExistingRules(t *testing.T) {
	card := sampleProjectCard()
	result := RunBrownfield(card, nil)
	// All fresh sections should appear as ADDED
	for _, d := range result.Deltas {
		if d.ChangeType != DeltaAdded {
			t.Errorf("expected all ADDED for nil existing, got %s for %q", d.ChangeType, d.Section)
		}
	}
	if len(result.Deltas) == 0 {
		t.Error("expected at least one ADDED delta for nil existing rules")
	}
}

func TestRunBrownfield_Deterministic(t *testing.T) {
	card := sampleProjectCard()
	existing := map[string]string{"ci": "old-value", "language": "python(5 files, 50%)"}

	r1 := RunBrownfield(card, existing)
	r2 := RunBrownfield(card, existing)

	d1, _ := json.Marshal(r1.Deltas)
	d2, _ := json.Marshal(r2.Deltas)
	if string(d1) != string(d2) {
		t.Errorf("RunBrownfield not deterministic:\n%s\nvs\n%s", d1, d2)
	}
}

// --- BrownfieldResult JSON round-trip ---

func TestBrownfieldResult_WriteReadJSON(t *testing.T) {
	card := sampleProjectCard()
	existing := map[string]string{"language": "old-value"}
	result := RunBrownfield(card, existing)

	data, err := MarshalBrownfieldResult(result)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	decoded, err := UnmarshalBrownfieldResult(data)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(decoded.Deltas) != len(result.Deltas) {
		t.Fatalf("delta count mismatch: got %d, want %d", len(decoded.Deltas), len(result.Deltas))
	}
	for i, d := range decoded.Deltas {
		if d.Section != result.Deltas[i].Section {
			t.Errorf("delta[%d] Section: got %q, want %q", i, d.Section, result.Deltas[i].Section)
		}
		if d.ChangeType != result.Deltas[i].ChangeType {
			t.Errorf("delta[%d] ChangeType: got %q, want %q", i, d.ChangeType, result.Deltas[i].ChangeType)
		}
	}
}

// --- Delta type constant tests ---

func TestDeltaTypeConstants(t *testing.T) {
	types := map[string]string{
		DeltaAdded:    "ADDED",
		DeltaModified: "MODIFIED",
		DeltaRemoved:  "REMOVED",
	}
	for constVal, expected := range types {
		if constVal != expected {
			t.Errorf("delta type constant mismatch: got %q, want %q", constVal, expected)
		}
	}
}

// --- helpers ---

// sampleProjectCard returns a fully populated scout.ProjectCard for testing.
func sampleProjectCard() *scout.ProjectCard {
	buildSystem := "go_modules"
	pkgMgr := "go_modules"
	lang := "go"
	desc := "test project"
	ciSystem := "github-actions"
	return &scout.ProjectCard{
		Version: "1.0.0",
		Identity: scout.Identity{
			Name:            "test-project",
			Description:     &desc,
			PrimaryLanguage: lang,
			Languages: map[string]scout.LangStats{
				"go":   {Files: 10, Ratio: 0.85},
				"yaml": {Files: 2, Ratio: 0.15},
			},
			BuildSystem: &buildSystem,
			BuildFiles:  []string{"go.mod"},
			Monorepo:    false,
		},
		Scale: scout.Scale{
			TotalFiles:  12,
			SourceFiles: 10,
			TestFiles:   5,
			TestRatio:   0.42,
			Directories: 4,
		},
		Maturity: scout.Maturity{
			HasCI:    true,
			CISystem: &ciSystem,
		},
		Build: scout.Build{
			EntryPoints:     []string{"cmd/main.go"},
			PackageManager:  &pkgMgr,
			DependencyCount: 3,
		},
		Conventions: scout.Conventions{
			TestStructure: scout.TestLayout{
				Style:      "colocated",
				DirPattern: "*_test.go",
			},
			LintConfig: &scout.LintInfo{
				Tool:       "golangci-lint",
				ConfigFile: ".golangci.yml",
				Rules:      []string{"errcheck", "govet"},
			},
			CIWorkflow: &scout.CIInfo{
				System:     "github-actions",
				ConfigFile: ".github/workflows/ci.yml",
				Steps:      []string{"checkout", "test"},
			},
		},
	}
}
