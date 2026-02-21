package evidence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateStrictFile_missing(t *testing.T) {
	_, err := ValidateStrictFile("/nonexistent", false)
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestValidateStrictFile_invalidJSON(t *testing.T) {
	f := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(f, []byte(`{invalid`), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ValidateStrictFile(f, false)
	if err == nil {
		t.Error("invalid JSON should return error")
	}
}

func TestValidateStrictFile_missingSections(t *testing.T) {
	f := filepath.Join(t.TempDir(), "partial.json")
	if err := os.WriteFile(f, []byte(`{"intent":{}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := ValidateStrictFile(f, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.OK {
		t.Error("missing sections should not be OK")
	}
	if len(r.Missing) == 0 {
		t.Error("expected missing sections")
	}
}

func TestFormatMissing(t *testing.T) {
	got := FormatMissing([]string{"a", "b"})
	if got != "missing: a, b" {
		t.Errorf("FormatMissing = %q", got)
	}
	got = FormatMissing(nil)
	if got != "" {
		t.Errorf("FormatMissing(nil) = %q", got)
	}
}
