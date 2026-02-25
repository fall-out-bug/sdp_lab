package evidence

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// moduleRoot returns the path to the module root (directory containing go.mod).
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	dir := filepath.Dir(file)
	for d := dir; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
	}
	t.Fatal("could not find module root")
	return ""
}

// validEvidenceFixture is a minimal valid evidence envelope that passes ValidateStrictFile.
var validEvidenceFixture = []byte(`{
  "intent": {"issue_id": "sdp_dev-abc", "trigger": "user", "acceptance": [], "risk_class": "low"},
  "plan": {"workstreams": [], "ordering_rationale": ""},
  "execution": {"claimed_issue_ids": [], "branch": "main", "changed_files": []},
  "verification": {"tests": [], "lint": [], "contracts": [], "coverage": {"value": 80, "threshold": 80}},
  "review": {"self_review": [], "adversarial_review": []},
  "risk_notes": {"residual_risks": [], "out_of_scope": []},
  "boundary": {
    "declared": {"allowed_path_prefixes": [], "control_path_prefixes": [], "forbidden_path_prefixes": [], "role": "", "lane": ""},
    "observed": {"touched_paths": [], "out_of_boundary_paths": []},
    "compliance": {"ok": true, "reason": ""}
  },
  "provenance": {
    "run_id": "run-1",
    "orchestrator": "test",
    "runtime": "local",
    "model": "test",
    "gate_results": [],
    "phase": "execute",
    "role": "coder",
    "captured_at": "2026-01-01T00:00:00Z",
    "source_issue_id": "sdp_dev-abc",
    "artifact_id": "art-1",
    "contract_version": "artifact-provenance/v1",
    "hash_algorithm": "sha256",
    "sequence": 0,
    "payload_digest": "",
    "hash": "",
    "hash_prev": ""
  },
  "trace": {"beads_ids": [], "branch": "main", "commits": [], "pr_url": "https://github.com/org/repo/pull/1"}
}`)

func TestSchemaValidationMatchesEvidenceValidate(t *testing.T) {
	root := moduleRoot(t)
	schemaPath := filepath.Join(root, "schema", "evidence-envelope.schema.json")
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("evidence-envelope.schema.json", bytes.NewReader(mustReadFile(t, schemaPath))); err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	schema, err := compiler.Compile("evidence-envelope.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	tests := []struct {
		name       string
		payload    []byte
		requirePR  bool
		wantStrict bool // ValidateStrictFile OK
	}{
		{
			name:       "valid_full",
			payload:    validEvidenceFixture,
			requirePR:  true,
			wantStrict: true,
		},
		{
			name:       "valid_prepublish",
			payload:    validEvidenceFixture,
			requirePR:  false,
			wantStrict: true,
		},
		{
			name:       "missing_sections",
			payload:    []byte(`{"intent":{}}`),
			requirePR:  false,
			wantStrict: false,
		},
		{
			name: "invalid_boundary_missing_declared",
			payload: mustMerge(t, validEvidenceFixture, map[string]any{
				"boundary": map[string]any{
					"declared":  map[string]any{},
					"observed": map[string]any{"touched_paths": []any{}, "out_of_boundary_paths": []any{}},
					"compliance": map[string]any{"ok": true, "reason": ""},
				},
			}),
			requirePR:  false,
			wantStrict: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write payload to temp file for ValidateStrictFile
			f := filepath.Join(t.TempDir(), "evidence.json")
			if err := os.WriteFile(f, tt.payload, 0o644); err != nil {
				t.Fatal(err)
			}

			res, err := ValidateStrictFile(f, tt.requirePR)
			if err != nil {
				t.Fatalf("ValidateStrictFile: %v", err)
			}
			strictOK := res.OK

			var doc any
			if err := json.Unmarshal(tt.payload, &doc); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			schemaErr := schema.Validate(doc)
			schemaOK := schemaErr == nil

			if strictOK != tt.wantStrict {
				t.Errorf("ValidateStrictFile: got OK=%v, want %v (reason=%q)", strictOK, tt.wantStrict, res.Reason)
			}
			if schemaOK != strictOK {
				t.Errorf("schema validation disagrees with evidence.Validate: schemaOK=%v, strictOK=%v, schemaErr=%v",
					schemaOK, strictOK, schemaErr)
			}
		})
	}
}

func TestSchemaValidatesTemplate(t *testing.T) {
	root := moduleRoot(t)
	templatePath := filepath.Join(root, "specs", "strict-evidence-template.json")
	b := mustReadFile(t, templatePath)
	var doc any
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("unmarshal template: %v", err)
	}

	schemaPath := filepath.Join(root, "schema", "evidence-envelope.schema.json")
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("evidence-envelope.schema.json", bytes.NewReader(mustReadFile(t, schemaPath))); err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	schema, err := compiler.Compile("evidence-envelope.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}

	if err := schema.Validate(doc); err != nil {
		t.Errorf("template should validate against schema: %v", err)
	}
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return b
}

// TestCodingWorkflowStatementSchemaRefResolves verifies that coding-workflow-statement
// $ref to predicate $id resolves when predicate is pre-loaded (AC 00-053-13: $ref works from any cwd).
func TestCodingWorkflowStatementSchemaRefResolves(t *testing.T) {
	root := moduleRoot(t)
	schemaDir := filepath.Join(root, "sdp", "schema")
	stmtPath := filepath.Join(schemaDir, "coding-workflow-statement.schema.json")
	if _, err := os.Stat(stmtPath); err != nil {
		t.Skip("sdp/schema/coding-workflow-statement.schema.json not found (submodule?)")
	}
	compiler := jsonschema.NewCompiler()
	// Pre-load predicate by its $id so $ref "https://sdp.dev/attestation/coding-workflow/v1" resolves
	predPath := filepath.Join(schemaDir, "coding-workflow-predicate.schema.json")
	predID := "https://sdp.dev/attestation/coding-workflow/v1"
	if err := compiler.AddResource(predID, bytes.NewReader(mustReadFile(t, predPath))); err != nil {
		t.Fatalf("add predicate schema: %v", err)
	}
	stmtURL := "file://" + stmtPath
	if err := compiler.AddResource(stmtURL, bytes.NewReader(mustReadFile(t, stmtPath))); err != nil {
		t.Fatalf("add statement schema: %v", err)
	}
	_, err := compiler.Compile(stmtURL)
	if err != nil {
		t.Fatalf("compile coding-workflow-statement (with $ref): %v", err)
	}
}

func mustMerge(t *testing.T, base []byte, overrides map[string]any) []byte {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(base, &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range overrides {
		m[k] = v
	}
	out, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
