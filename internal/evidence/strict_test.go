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
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},
		"boundary":{"declared":{"allowed_path_prefixes":[],"control_path_prefixes":[],"forbidden_path_prefixes":[]},"observed":{"touched_paths":[],"out_of_boundary_paths":[]},"compliance":{"ok":true,"reason":"ok"}},
		"provenance":{"run_id":"run-1","orchestrator":"autonomy-worker","runtime":"opencode","model":"glm-5","gate_results":[]},
		"trace":{}
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
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},
		"boundary":{"declared":{"allowed_path_prefixes":[],"control_path_prefixes":[],"forbidden_path_prefixes":[]},"observed":{"touched_paths":[],"out_of_boundary_paths":[]},"compliance":{"ok":true,"reason":"ok"}},
		"provenance":{"run_id":"run-1","orchestrator":"autonomy-worker","runtime":"opencode","model":"glm-5","gate_results":[]},
		"trace":{"pr_url":"https://example/pr/1"}
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
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},
		"boundary":{"declared":{"allowed_path_prefixes":[],"control_path_prefixes":[],"forbidden_path_prefixes":[]},"observed":{"touched_paths":[],"out_of_boundary_paths":[]},"compliance":{"ok":true,"reason":"ok"}},
		"provenance":{"run_id":"run-1","orchestrator":"autonomy-worker","runtime":"opencode","model":"glm-5","gate_results":[]},
		"trace":{}
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

func TestValidateStrictFileInvalidBoundaryContract(t *testing.T) {
	dir := t.TempDir()
	body := `{
		"intent":{},"plan":{},"execution":{},"verification":{},"review":{},"risk_notes":{},
		"boundary":{"declared":{},"observed":{},"compliance":{}},
		"provenance":{"run_id":"run-1","orchestrator":"autonomy-worker","runtime":"opencode","model":"glm-5","gate_results":[]},
		"trace":{}
	}`
	path := writeFile(t, dir, "evidence.json", body)

	res, err := ValidateStrictFile(path, false)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if res.OK {
		t.Fatalf("expected failure for invalid boundary contract")
	}
}
