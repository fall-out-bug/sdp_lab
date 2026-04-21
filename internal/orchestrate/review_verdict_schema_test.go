package orchestrate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func loadReviewVerdictSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	schemaPath := filepath.Join(repoRootForTest(t), "schema", "review-verdict.schema.json")
	schema, err := compiler.Compile("file://" + schemaPath)
	if err != nil {
		t.Fatalf("compile review verdict schema: %v", err)
	}
	return schema
}

func repoRootForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("repo root not found from %s", dir)
		}
		dir = parent
	}
}

func validReviewVerdictBase(verdict string) map[string]interface{} {
	return map[string]interface{}{
		"feature":   "F104",
		"verdict":   verdict,
		"round":     3,
		"timestamp": "2026-04-21T12:00:00Z",
		"reviewers": map[string]interface{}{
			"qa":        map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"security":  map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"devops":    map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"sre":       map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"techlead":  map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"docs":      map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
			"promptops": map[string]interface{}{"verdict": "PASS", "findings": []interface{}{}},
		},
	}
}

func TestReviewVerdictSchemaEscapeHatchContracts(t *testing.T) {
	schema := loadReviewVerdictSchema(t)

	t.Run("override reason must not be empty when present", func(t *testing.T) {
		payload := validReviewVerdictBase("APPROVED")
		payload["override_reason"] = ""
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected empty override_reason to fail schema validation")
		}
	})

	t.Run("partial verdict requires failing roles", func(t *testing.T) {
		payload := validReviewVerdictBase("PARTIALLY_APPROVED")
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected PARTIALLY_APPROVED without partial_failing_roles to fail schema validation")
		}
		payload["partial_failing_roles"] = []interface{}{"docs"}
		if err := schema.Validate(payload); err != nil {
			t.Fatalf("expected PARTIALLY_APPROVED with failing roles to validate: %v", err)
		}
	})

	t.Run("escalated verdict requires escalation issue", func(t *testing.T) {
		payload := validReviewVerdictBase("ESCALATED")
		if err := schema.Validate(payload); err == nil {
			t.Fatal("expected ESCALATED without escalation_issue to fail schema validation")
		}
		payload["escalation_issue"] = "sdplab-123"
		if err := schema.Validate(payload); err != nil {
			t.Fatalf("expected ESCALATED with escalation issue to validate: %v", err)
		}
	})
}
