package architect_test

import (
	"context"
	"testing"

	"sdp_dev/internal/architect"
	"sdp_dev/internal/architect/extract"

	"github.com/stretchr/testify/assert"
)

// TestAdaptersImplementInterface verifies that all adapters implement the Extractor interface.
func TestAdaptersImplementInterface(t *testing.T) {
	// Test each adapter individually
	adapters := []struct {
		name     string
		extractor architect.Extractor
	}{
		{"GoAdapter", extract.GoAdapter{}},
		{"PythonAdapter", extract.PythonAdapter{}},
		{"JavaAdapter", extract.JavaAdapter{}},
		{"TypeScriptAdapter", extract.TypeScriptAdapter{}},
	}

	for _, tc := range adapters {
		t.Run(tc.name, func(t *testing.T) {
			// Verify Name() returns non-empty
			assert.NotEmpty(t, tc.extractor.Name())

			// Verify Extract() can be called (may return error for non-matching repos)
			ctx := context.Background()
			frag, err := tc.extractor.Extract(ctx, "/tmp/test")
			// Java extractor returns error for non-Java repos, others return empty fragment
			if tc.name == "JavaAdapter" {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, frag)
			}
		})
	}
}

// TestDefaultExtractorsIncludesAdapters verifies that DefaultExtractors includes all language adapters.
func TestDefaultExtractorsIncludesAdapters(t *testing.T) {
	extractors := extract.DefaultExtractors()

	names := make(map[string]bool)
	for _, e := range extractors {
		names[e.Name()] = true
	}

	// Verify all adapter names are present
	expectedAdapters := []string{"go", "python", "java", "typescript"}
	for _, name := range expectedAdapters {
		assert.True(t, names[name], "Expected adapter '%s' not found in DefaultExtractors()", name)
	}
}
