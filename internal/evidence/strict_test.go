package evidence

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func TestValidateStrictFileMissingSections(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "evidence.json", `{"intent":{},"trace":{"pr_url":"https://example/pr/1"}}`)

	res, err := ValidateStrictFile(path, false)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK {
		t.Fatalf("expected failure for missing sections")
	}
}

func TestValidateStrictFileMissingPRURL(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},"trace":{}
	}`
	path := writeFile(t, dir, "evidence.json", body)

	res, err := ValidateStrictFile(path, true)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK {
		t.Fatalf("expected failure for missing trace.pr_url")
	}
}

func TestValidateStrictFileOK(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},"trace":{"pr_url":"https://example/pr/1"}
	}`
	path := writeFile(t, dir, "evidence.json", body)

	res, err := ValidateStrictFile(path, true)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected success, got %+v", res)
	}
}

func TestValidateStrictFileVerifiedModeAllowsMissingPRURL(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},"trace":{}
	}`
	path := writeFile(t, dir, "evidence.json", body)

	res, err := ValidateStrictFile(path, false)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if !res.OK {
		t.Fatalf("expected success in verified mode, got %+v", res)
	}
}
