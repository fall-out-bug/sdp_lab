package evidence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInspectValid(t *testing.T) {
	// Use template with requirePRURL=false (specs at repo root)
	wd, _ := os.Getwd()
	repoRoot := filepath.Dir(filepath.Dir(wd)) // internal/evidence -> repo
	template := filepath.Join(repoRoot, "specs", "strict-evidence-template.json")
	summary, res, err := Inspect(template, false)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected OK, got %v", res)
	}
	if !strings.Contains(summary, "intent") {
		t.Error("summary should include intent")
	}
	if !strings.Contains(summary, "plan") {
		t.Error("summary should include plan")
	}
	if !strings.Contains(summary, "boundary_compliance") {
		t.Error("summary should include boundary_compliance")
	}
	if !strings.Contains(summary, "provenance") {
		t.Error("summary should include provenance")
	}
}

func TestInspectInvalidFile(t *testing.T) {
	_, _, err := Inspect("/nonexistent/path.json", false)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestInspectInvalidEvidence(t *testing.T) {
	tmp := t.TempDir()
	bad := filepath.Join(tmp, "bad.json")
	os.WriteFile(bad, []byte(`{"intent":{}}`), 0644)
	summary, res, err := Inspect(bad, false)
	if err != nil {
		t.Fatalf("Inspect should not return error for invalid evidence: %v", err)
	}
	if res.OK {
		t.Fatal("expected !res.OK for invalid evidence")
	}
	if summary != "" {
		t.Error("summary should be empty for invalid evidence")
	}
}
