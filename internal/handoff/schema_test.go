package handoff

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

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

func TestHandoffFixturesValidateAgainstSchemas(t *testing.T) {
	root := moduleRoot(t)
	schemas := []struct {
		name   string
		schema string
		fixture string
	}{
		{"analyst", "handoff-analyst.schema.json", "analyst.json"},
		{"coder", "handoff-coder.schema.json", "coder.json"},
		{"reviewer", "handoff-reviewer.schema.json", "reviewer.json"},
	}

	for _, s := range schemas {
		t.Run(s.name, func(t *testing.T) {
			schemaPath := filepath.Join(root, "schema", s.schema)
			schemaData, err := os.ReadFile(schemaPath)
			if err != nil {
				t.Fatalf("read schema: %v", err)
			}

			compiler := jsonschema.NewCompiler()
			if err := compiler.AddResource(s.schema, bytes.NewReader(schemaData)); err != nil {
				t.Fatalf("add schema: %v", err)
			}
			schema, err := compiler.Compile(s.schema)
			if err != nil {
				t.Fatalf("compile schema: %v", err)
			}

			fixturePath := filepath.Join(root, "testdata", "handoff", s.fixture)
			fixtureData, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatalf("read fixture: %v", err)
			}

			var doc any
			if err := json.Unmarshal(fixtureData, &doc); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}

			if err := schema.Validate(doc); err != nil {
				t.Errorf("fixture should validate: %v", err)
			}
		})
	}
}
