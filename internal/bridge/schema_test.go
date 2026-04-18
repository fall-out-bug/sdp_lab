package bridge

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// schemaBase returns the path to schema/findings relative to repo root.
func schemaBase() string {
	// Test binary runs from the module root; schemas live at
	// <repo>/schema/findings/*.schema.json.
	base, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	// Walk up until we find go.mod to determine repo root.
	dir := base
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "schema", "findings")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			// Fallback: assume cwd is <repo>/internal/bridge → go up two levels.
			return filepath.Join(base, "..", "..", "schema", "findings")
		}
		dir = parent
	}
}

// loadSchema compiles a JSON Schema file using the standard http loader.
func loadSchema(t *testing.T, name string) *jsonschema.Schema {
	t.Helper()
	path := filepath.Join(schemaBase(), name)
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile("file://" + path)
	require.NoError(t, err, "failed to compile schema %s", name)
	return schema
}

// loadExample reads an example JSON file from schema/findings/examples/.
func loadExample(t *testing.T, name string) interface{} {
	t.Helper()
	path := filepath.Join(schemaBase(), "examples", name)
	data, err := os.ReadFile(path)
	require.NoError(t, err, "failed to read example %s", name)
	var v interface{}
	require.NoError(t, json.Unmarshal(data, &v), "failed to parse example %s", name)
	return v
}

// --- Tests ------------------------------------------------------------------

func TestSchemaProtocolFindingsValidatesExample(t *testing.T) {
	schema := loadSchema(t, "protocol-findings.schema.json")
	example := loadExample(t, "protocol-findings-example.json")
	assert.NoError(t, schema.Validate(example),
		"protocol-findings-example.json must validate against its schema")
}

func TestSchemaDocsFindingsValidatesExample(t *testing.T) {
	schema := loadSchema(t, "docs-findings.schema.json")
	example := loadExample(t, "docs-findings-example.json")
	assert.NoError(t, schema.Validate(example),
		"docs-findings-example.json must validate against its schema")
}

func TestSchemaRequiredFieldsProtocol(t *testing.T) {
	_ = loadSchema(t, "protocol-findings.schema.json")
	example := loadExample(t, "protocol-findings-example.json")

	// We already know the example validates; now verify the parsed Go types
	// carry all required fields.
	data, err := json.Marshal(example)
	require.NoError(t, err)

	var pf ProtocolFindings
	require.NoError(t, json.Unmarshal(data, &pf))

	// Top-level required fields.
	assert.NotEmpty(t, pf.SpecVersion, "spec_version is required")
	assert.NotEmpty(t, pf.FindingsID, "findings_id is required")
	assert.NotEmpty(t, pf.Timestamp, "timestamp is required")
	assert.NotEmpty(t, pf.Source.CheckName, "source.check_name is required")
	assert.NotEmpty(t, pf.Source.Workflow, "source.workflow is required")
	assert.NotZero(t, pf.Source.RunID, "source.run_id is required")
	assert.NotEmpty(t, pf.Findings, "findings array is required and must not be empty")

	// Per-finding required fields.
	for i, f := range pf.Findings {
		assert.NotEmpty(t, f.FindingKey,
			"findings[%d].finding_key is required", i)
		assert.NotEmpty(t, f.Severity,
			"findings[%d].severity is required", i)
		assert.NotEmpty(t, f.Category,
			"findings[%d].category is required", i)
		assert.NotEmpty(t, f.File,
			"findings[%d].file is required (file path)", i)
		assert.NotEmpty(t, f.Message,
			"findings[%d].message is required", i)
	}
}

func TestSchemaRequiredFieldsDocs(t *testing.T) {
	// Verify the example validates first.
	_ = loadSchema(t, "docs-findings.schema.json")
	example := loadExample(t, "docs-findings-example.json")

	data, err := json.Marshal(example)
	require.NoError(t, err)

	var df DocsFindings
	require.NoError(t, json.Unmarshal(data, &df))

	assert.NotEmpty(t, df.SpecVersion, "spec_version is required")
	assert.NotEmpty(t, df.FindingsID, "findings_id is required")
	assert.NotEmpty(t, df.Timestamp, "timestamp is required")
	assert.NotEmpty(t, df.Source.CheckName, "source.check_name is required")
	assert.NotEmpty(t, df.Source.Workflow, "source.workflow is required")
	assert.NotZero(t, df.Source.RunID, "source.run_id is required")
	assert.NotEmpty(t, df.Findings, "findings array is required and must not be empty")

	for i, f := range df.Findings {
		assert.NotEmpty(t, f.FindingKey,
			"findings[%d].finding_key is required", i)
		assert.NotEmpty(t, f.Severity,
			"findings[%d].severity is required", i)
		assert.NotEmpty(t, f.Category,
			"findings[%d].category is required", i)
		assert.NotEmpty(t, f.File,
			"findings[%d].file is required (file path)", i)
		assert.NotEmpty(t, f.Message,
			"findings[%d].message is required", i)
	}

}

func TestSchemaDeduplicationKeyFormat(t *testing.T) {
	keyPattern := regexp.MustCompile(`^[a-f0-9]{16}$`)

	t.Run("protocol", func(t *testing.T) {
		example := loadExample(t, "protocol-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var pf ProtocolFindings
		require.NoError(t, json.Unmarshal(data, &pf))

		for i, f := range pf.Findings {
			assert.Regexp(t, keyPattern, f.FindingKey,
				"findings[%d].finding_key must be 16 hex chars", i)
		}
	})

	t.Run("docs", func(t *testing.T) {
		example := loadExample(t, "docs-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var df DocsFindings
		require.NoError(t, json.Unmarshal(data, &df))

		for i, f := range df.Findings {
			assert.Regexp(t, keyPattern, f.FindingKey,
				"findings[%d].finding_key must be 16 hex chars", i)
		}
	})
}

func TestSchemaVersioningPattern(t *testing.T) {
	versionPattern := regexp.MustCompile(`^v\d+\.\d+$`)

	cases := []struct {
		name    string
		example string
	}{
		{"protocol", "protocol-findings-example.json"},
		{"docs", "docs-findings-example.json"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			example := loadExample(t, tc.example)
			data, err := json.Marshal(example)
			require.NoError(t, err)

			var raw struct {
				SpecVersion string `json:"spec_version"`
			}
			require.NoError(t, json.Unmarshal(data, &raw))
			assert.Regexp(t, versionPattern, raw.SpecVersion,
				"spec_version must match vMAJOR.MINOR pattern")
		})
	}
}

func TestSchemaSeverityValues(t *testing.T) {
	validSeverities := map[string]bool{
		"error":   true,
		"warning": true,
		"info":    true,
		"hint":    true,
	}

	t.Run("protocol", func(t *testing.T) {
		example := loadExample(t, "protocol-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var pf ProtocolFindings
		require.NoError(t, json.Unmarshal(data, &pf))

		for i, f := range pf.Findings {
			assert.True(t, validSeverities[f.Severity],
				"findings[%d].severity=%q must be one of [error,warning,info,hint]", i, f.Severity)
		}
	})

	t.Run("docs", func(t *testing.T) {
		example := loadExample(t, "docs-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var df DocsFindings
		require.NoError(t, json.Unmarshal(data, &df))

		for i, f := range df.Findings {
			assert.True(t, validSeverities[f.Severity],
				"findings[%d].severity=%q must be one of [error,warning,info,hint]", i, f.Severity)
		}
	})
}

func TestSchemaRejectsInvalidPayload(t *testing.T) {
	schema := loadSchema(t, "protocol-findings.schema.json")

	// Minimal payload missing required fields.
	invalid := map[string]interface{}{
		"spec_version": "v1.0",
	}

	err := schema.Validate(invalid)
	assert.Error(t, err,
		"payload missing required fields must fail validation")
	assert.Contains(t, fmt.Sprintf("%v", err), "findings_id",
		"error should mention missing findings_id")
}

func TestSchemaRejectsInvalidFindingKey(t *testing.T) {
	schema := loadSchema(t, "protocol-findings.schema.json")
	example := loadExample(t, "protocol-findings-example.json")

	// Deep copy and corrupt a finding_key.
	data, err := json.Marshal(example)
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &payload))

	findings := payload["findings"].([]interface{})
	first := findings[0].(map[string]interface{})
	first["finding_key"] = "ZZZZZZZZZZZZZZZZ" // invalid: uppercase, not hex

	err = schema.Validate(payload)
	assert.Error(t, err,
		"finding_key with non-hex characters must fail validation")
}

func TestSchemaRemediationHintPresentInExamples(t *testing.T) {
	// Verify that at least one finding in each example has a remediation hint,
	// confirming the field is exercised.
	t.Run("protocol", func(t *testing.T) {
		example := loadExample(t, "protocol-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var pf ProtocolFindings
		require.NoError(t, json.Unmarshal(data, &pf))

		hasRemediation := false
		for _, f := range pf.Findings {
			if f.Remediation != nil && f.Remediation.Hint != "" {
				hasRemediation = true
				break
			}
		}
		assert.True(t, hasRemediation,
			"at least one protocol finding must include remediation.hint")
	})

	t.Run("docs", func(t *testing.T) {
		example := loadExample(t, "docs-findings-example.json")
		data, err := json.Marshal(example)
		require.NoError(t, err)
		var df DocsFindings
		require.NoError(t, json.Unmarshal(data, &df))

		hasRemediation := false
		for _, f := range df.Findings {
			if f.Remediation != nil && f.Remediation.Hint != "" {
				hasRemediation = true
				break
			}
		}
		assert.True(t, hasRemediation,
			"at least one docs finding must include remediation.hint")
	})
}
