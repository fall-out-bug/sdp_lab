package handoff

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

var (
	schemaDir     string
	schemaDirOnce sync.Once
)

func getSchemaDir() string {
	schemaDirOnce.Do(func() {
		_, file, _, _ := runtime.Caller(0)
		dir := filepath.Dir(file)
		for d := dir; d != filepath.Dir(d); d = filepath.Dir(d) {
			p := filepath.Join(d, "schema", "handoff-analyst.schema.json")
			if _, err := os.Stat(p); err == nil {
				schemaDir = filepath.Join(d, "schema")
				return
			}
		}
	})
	return schemaDir
}

func validateAgainstSchema(schemaName string, data []byte) error {
	dir := getSchemaDir()
	if dir == "" {
		return os.ErrNotExist
	}
	b, err := os.ReadFile(filepath.Join(dir, schemaName))
	if err != nil {
		return err
	}
	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource(schemaName, bytes.NewReader(b)); err != nil {
		return err
	}
	schema, err := compiler.Compile(schemaName)
	if err != nil {
		return err
	}
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	return schema.Validate(doc)
}

// ValidateAnalyst validates analyst handoff JSON against the schema.
func ValidateAnalyst(data []byte) error {
	return validateAgainstSchema("handoff-analyst.schema.json", data)
}

// ValidateCoder validates coder handoff JSON against the schema.
func ValidateCoder(data []byte) error {
	return validateAgainstSchema("handoff-coder.schema.json", data)
}

// ValidateReviewer validates reviewer handoff JSON against the schema.
func ValidateReviewer(data []byte) error {
	return validateAgainstSchema("handoff-reviewer.schema.json", data)
}
