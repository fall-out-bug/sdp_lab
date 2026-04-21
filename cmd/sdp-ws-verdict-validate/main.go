// sdp-ws-verdict-validate validates docs/ws-verdicts/*.json against schema/ws-verdict.schema.json.
// Used by post-build hook to catch invalid verdicts before merge.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

func main() {
	projectRoot := "."
	if len(os.Args) > 1 {
		projectRoot = os.Args[1]
	}

	schemaPath := filepath.Join(projectRoot, "sdp", "schema", "ws-verdict.schema.json")
	schemaData, err := os.ReadFile(schemaPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws-verdict-validate: cannot read schema: %v\n", err)
		os.Exit(1)
	}

	compiler := jsonschema.NewCompiler()
	if err := compiler.AddResource("ws-verdict.schema.json", bytes.NewReader(schemaData)); err != nil {
		fmt.Fprintf(os.Stderr, "ws-verdict-validate: compile schema: %v\n", err)
		os.Exit(1)
	}
	schema, err := compiler.Compile("ws-verdict.schema.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws-verdict-validate: compile schema: %v\n", err)
		os.Exit(1)
	}

	verdictsDir := filepath.Join(projectRoot, "docs", "ws-verdicts")
	entries, err := filepath.Glob(filepath.Join(verdictsDir, "*.json"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "ws-verdict-validate: glob: %v\n", err)
		os.Exit(1)
	}

	var failed int
	for _, p := range entries {
		data, err := os.ReadFile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ws-verdict-validate: read %s: %v\n", p, err)
			failed++
			continue
		}
		var doc any
		if err := json.Unmarshal(data, &doc); err != nil {
			fmt.Fprintf(os.Stderr, "ws-verdict-validate: %s: invalid JSON: %v\n", filepath.Base(p), err)
			failed++
			continue
		}
		if err := schema.Validate(doc); err != nil {
			fmt.Fprintf(os.Stderr, "ws-verdict-validate: %s: schema validation failed: %v\n", filepath.Base(p), err)
			failed++
		}
	}

	if failed > 0 {
		fmt.Fprintf(os.Stderr, "ws-verdict-validate: %d verdict(s) failed validation\n", failed)
		os.Exit(1)
	}
}
