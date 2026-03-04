package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func TestValidatorCompilesSchema(t *testing.T) {
	root := findModuleRoot(t)
	schemaPath := filepath.Join(root, "sdp", "schema", "ws-verdict.schema.json")
	if _, err := os.Stat(schemaPath); err != nil {
		t.Skip("sdp submodule or schema not present")
	}
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("ws-verdict.schema.json", bytes.NewReader(schemaData)); err != nil {
		t.Fatalf("AddResource: %v", err)
	}
	_, err = compiler.Compile("ws-verdict.schema.json")
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
}

func TestValidatorRejectsInvalid(t *testing.T) {
	root := findModuleRoot(t)
	schemaPath := filepath.Join(root, "sdp", "schema", "ws-verdict.schema.json")
	if _, err := os.Stat(schemaPath); err != nil {
		t.Skip("sdp submodule or schema not present")
	}
	schemaData, _ := os.ReadFile(schemaPath)
	compiler := jsonschema.NewCompiler()
	_ = compiler.AddResource("ws-verdict.schema.json", bytes.NewReader(schemaData))
	schema, _ := compiler.Compile("ws-verdict.schema.json")
	invalid := []byte(`{"ws_id": "bad"}`) // missing required fields
	var doc any
	_ = json.Unmarshal(invalid, &doc)
	err := schema.Validate(doc)
	if err == nil {
		t.Error("expected validation error for invalid verdict")
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	wd, _ := os.Getwd()
	for d := wd; d != filepath.Dir(d); d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d
		}
	}
	t.Fatal("could not find module root")
	return ""
}
