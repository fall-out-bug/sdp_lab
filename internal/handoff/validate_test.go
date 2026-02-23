package handoff

import (
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	root := moduleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "testdata", "handoff", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestValidateAnalyst_Valid(t *testing.T) {
	data := readFixture(t, "analyst.json")
	if err := ValidateAnalyst(data); err != nil {
		t.Errorf("ValidateAnalyst(valid): %v", err)
	}
}

func TestValidateAnalyst_Invalid(t *testing.T) {
	invalid := []byte(`{"risk_class": "invalid_enum", "decomposed_steps": [], "recommended_approach": "", "estimated_complexity": "S", "scope_files": []}`)
	if err := ValidateAnalyst(invalid); err == nil {
		t.Error("ValidateAnalyst(invalid): expected error")
	}
}

func TestValidateCoder_Valid(t *testing.T) {
	data := readFixture(t, "coder.json")
	if err := ValidateCoder(data); err != nil {
		t.Errorf("ValidateCoder(valid): %v", err)
	}
}

func TestValidateCoder_Invalid(t *testing.T) {
	invalid := []byte(`{"changed_files": [], "test_results": {"passed": -1, "failed": 0, "coverage": 0}, "implementation_notes": "", "branch": "", "commits": []}`)
	if err := ValidateCoder(invalid); err == nil {
		t.Error("ValidateCoder(invalid): expected error")
	}
}

func TestValidateReviewer_Valid(t *testing.T) {
	data := readFixture(t, "reviewer.json")
	if err := ValidateReviewer(data); err != nil {
		t.Errorf("ValidateReviewer(valid): %v", err)
	}
}

func TestValidateReviewer_Invalid(t *testing.T) {
	invalid := []byte(`{"verdict": "invalid", "findings": [], "suggestions": [], "risk_assessment": ""}`)
	if err := ValidateReviewer(invalid); err == nil {
		t.Error("ValidateReviewer(invalid): expected error")
	}
}
