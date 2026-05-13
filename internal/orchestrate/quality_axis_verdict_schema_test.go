package orchestrate

import (
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func loadQualityAxisVerdictSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	schemaPath := filepath.Join(repoRootForTest(t), "schema", "quality-axis-verdict.schema.json")
	schema, err := compiler.Compile("file://" + schemaPath)
	if err != nil {
		t.Fatalf("compile quality axis verdict schema: %v", err)
	}
	return schema
}

func validQualityAxisVerdictBase() map[string]interface{} {
	return map[string]interface{}{
		"feature_id":     "F168",
		"workstream_id":  "00-168-05",
		"generated_at":   "2026-05-13T12:00:00Z",
		"schema_version": "v1",
		"axes": []interface{}{
			map[string]interface{}{
				"axis":   "modern_go_patterns",
				"status": "evidence_only",
				"assessed_scope": map[string]interface{}{
					"kind":  "changed_files",
					"paths": []interface{}{"internal/example/foo.go"},
				},
				"source": map[string]interface{}{
					"type":      "deterministic_command",
					"name":      "golangci-lint",
					"command":   "golangci-lint run ./...",
					"exit_code": float64(0),
				},
				"omitted_files": []interface{}{},
				"evidence_refs": []interface{}{
					map[string]interface{}{
						"kind": "command_output",
						"ref":  ".sdp/evidence/f168-modern-go.txt",
					},
				},
				"confidence": "high",
			},
			map[string]interface{}{
				"axis":   "clean_architecture",
				"status": "warn",
				"assessed_scope": map[string]interface{}{
					"kind":  "branch_diff",
					"paths": []interface{}{"internal/example/foo.go"},
				},
				"source": map[string]interface{}{
					"type":     "model_review",
					"name":     "pi-review architecture plane",
					"reviewer": "clean_architecture",
					"provider": "openrouter",
					"model":    "review-model",
				},
				"omitted_files": []interface{}{
					map[string]interface{}{
						"path":   ".sdp/runs/pi-review/raw.json",
						"reason": "secret_risk",
					},
				},
				"evidence_refs": []interface{}{
					map[string]interface{}{
						"kind": "review_artifact",
						"ref":  ".sdp/review_verdict.json",
					},
				},
				"confidence": "medium",
			},
		},
	}
}

func TestQualityAxisVerdictSchemaContracts(t *testing.T) {
	schema := loadQualityAxisVerdictSchema(t)

	t.Run("accepts deterministic and model-review evidence in one artifact", func(t *testing.T) {
		payload := validQualityAxisVerdictBase()
		if err := schema.Validate(payload); err != nil {
			t.Fatalf("expected valid quality axis verdict: %v", err)
		}
	})

	t.Run("requires per-axis status", func(t *testing.T) {
		payload := validQualityAxisVerdictBase()
		firstAxis := payload["axes"].([]interface{})[0].(map[string]interface{})
		delete(firstAxis, "status")
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected missing status to fail schema validation")
		}
	})

	t.Run("requires assessed scope", func(t *testing.T) {
		payload := validQualityAxisVerdictBase()
		firstAxis := payload["axes"].([]interface{})[0].(map[string]interface{})
		delete(firstAxis, "assessed_scope")
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected missing assessed_scope to fail schema validation")
		}
	})

	t.Run("rejects unsupported status values", func(t *testing.T) {
		payload := validQualityAxisVerdictBase()
		firstAxis := payload["axes"].([]interface{})[0].(map[string]interface{})
		firstAxis["status"] = "green"
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected unsupported status to fail schema validation")
		}
	})
}
