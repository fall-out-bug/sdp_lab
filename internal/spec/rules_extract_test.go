package spec

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractRules_ValidationTags(t *testing.T) {
	rules := extractRulesFromTestdata(t, "validation_tags.go")
	require.NotEmpty(t, rules, "should find validation tag rules")

	// Should find rules with category "validation_tag"
	var tagRules []ValidationRule
	for _, r := range rules {
		if r.Category == "validation_tag" {
			tagRules = append(tagRules, r)
		}
	}
	assert.NotEmpty(t, tagRules, "should find validation_tag rules")

	// Check fields have constraints
	for _, r := range tagRules {
		assert.NotEmpty(t, r.Field, "rule should have a field name")
		assert.NotEmpty(t, r.Location, "rule should have a location")
	}
}

func TestExtractRules_GuardClauses(t *testing.T) {
	rules := extractRulesFromTestdata(t, "guard_clauses.go")
	require.NotEmpty(t, rules, "should find guard clause rules")

	var guards []ValidationRule
	for _, r := range rules {
		if r.Category == "guard_clause" {
			guards = append(guards, r)
		}
	}
	assert.NotEmpty(t, guards, "should find guard_clause rules")
}

func TestExtractRules_ErrorConstants(t *testing.T) {
	rules := extractRulesFromTestdata(t, "guard_clauses.go")
	require.NotEmpty(t, rules, "should find rules")

	var errConsts []ValidationRule
	for _, r := range rules {
		if r.Category == "error_constant" {
			errConsts = append(errConsts, r)
		}
	}
	assert.NotEmpty(t, errConsts, "should find error_constant rules")
	assert.GreaterOrEqual(t, len(errConsts), 3, "should find multiple error constants")
}

func TestExtractRules_NonexistentFile(t *testing.T) {
	rules, err := ExtractBusinessRules(filepath.Join("testdata", "nonexistent.go"))
	assert.NoError(t, err)
	assert.Empty(t, rules)
}

func extractRulesFromTestdata(t *testing.T, filename string) []ValidationRule {
	t.Helper()
	path := filepath.Join("testdata", filename)
	rules, err := ExtractBusinessRules(path)
	require.NoError(t, err)
	return rules
}
