package architect_test

import (
	"context"
	"os"
	"path/filepath"
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

// helper to create a temp dir with files.
func setupAdapterTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestPythonAdapter_Bug6_NotableDepsFiltering verifies that NotableDeps only includes
// dependencies from manifest files (requirements.txt/pyproject.toml), NOT from source imports.
// This prevents internal imports like "pyspark.pandas.utils" or stdlib like "contextvars"
// from appearing as notable third-party dependencies.
func TestPythonAdapter_Bug6_NotableDepsFiltering(t *testing.T) {
	// Simulate an Apache Spark-like project with:
	// - Internal imports (pyspark.*)
	// - Stdlib imports (os, json, contextvars)
	// - Actual third-party deps from requirements.txt
	root := setupAdapterTree(t, map[string]string{
		"pyspark/__init__.py": `# Internal imports that should NOT appear in NotableDeps
from pyspark.pandas import utils
from pyspark.ml import classification
import contextvars
import cProfile
import _collections_abc
`,
		"pyspark/pandas/utils.py": `import os
import json
`,
		"requirements.txt": `# These ARE actual third-party deps and SHOULD appear in NotableDeps
py4j==0.10.9.7
pyspark>=3.0.0
`,
	})

	adapter := extract.PythonAdapter{}
	ctx := context.Background()
	frag, err := adapter.Extract(ctx, root)
	assert.NoError(t, err)
	assert.NotNil(t, frag)

	// Verify Dependencies were extracted
	assert.NotEmpty(t, frag.Dependencies, "Should have Dependencies")
	depInfo := frag.Dependencies[0]

	// NotableDeps should only contain deps from requirements.txt/pyproject.toml
	// NOT internal imports or stdlib
	notableNames := make(map[string]bool)
	for _, dep := range depInfo.NotableDeps {
		notableNames[dep.Name] = true
	}

	// Should contain actual third-party deps from requirements.txt
	assert.True(t, notableNames["py4j"], "NotableDeps should include py4j from requirements.txt")
	assert.True(t, notableNames["pyspark"], "NotableDeps should include pyspark from requirements.txt")

	// Should NOT contain internal imports
	assert.False(t, notableNames["pyspark.pandas.utils"], "NotableDeps should NOT include internal import pyspark.pandas.utils")
	assert.False(t, notableNames["pyspark.ml.classification"], "NotableDeps should NOT include internal import pyspark.ml.classification")

	// Should NOT contain stdlib
	assert.False(t, notableNames["os"], "NotableDeps should NOT include stdlib os")
	assert.False(t, notableNames["json"], "NotableDeps should NOT include stdlib json")
	assert.False(t, notableNames["contextvars"], "NotableDeps should NOT include stdlib contextvars")
	assert.False(t, notableNames["cProfile"], "NotableDeps should NOT include stdlib cProfile")
	assert.False(t, notableNames["_collections_abc"], "NotableDeps should NOT include stdlib _collections_abc")

	// Verify architectural signal detection
	for _, dep := range depInfo.NotableDeps {
		if dep.Name == "py4j" {
			assert.Equal(t, "dependency", dep.Signal, "py4j should have generic signal")
		}
	}
}

// TestPythonAdapter_ArchitecturalSignals verifies that dependencies get appropriate
// architectural signals based on their names (e.g., kafka -> event_driven).
func TestPythonAdapter_ArchitecturalSignals(t *testing.T) {
	root := setupAdapterTree(t, map[string]string{
		"requirements.txt": `kafka==2.0.0
confluent-kafka==1.8.0
fastapi==0.68.0
flask==2.0.0
grpcio==1.40.0
celery==5.2.0
redis==4.0.0
sqlalchemy==1.4.0
pandas==1.3.0
pydantic==1.8.0
pytest==7.0.0
`,
	})

	adapter := extract.PythonAdapter{}
	ctx := context.Background()
	frag, err := adapter.Extract(ctx, root)
	assert.NoError(t, err)
	assert.NotNil(t, frag)

	assert.NotEmpty(t, frag.Dependencies, "Should have Dependencies")
	depInfo := frag.Dependencies[0]

	signals := make(map[string]string)
	for _, dep := range depInfo.NotableDeps {
		signals[dep.Name] = dep.Signal
	}

	// Verify architectural signals
	assert.Equal(t, "event_driven", signals["kafka"], "kafka should have event_driven signal")
	assert.Equal(t, "event_driven", signals["confluent-kafka"], "confluent-kafka should have event_driven signal")
	assert.Equal(t, "web_framework", signals["fastapi"], "fastapi should have web_framework signal")
	assert.Equal(t, "web_framework", signals["flask"], "flask should have web_framework signal")
	assert.Equal(t, "rpc", signals["grpcio"], "grpcio should have rpc signal")
	assert.Equal(t, "task_queue", signals["celery"], "celery should have task_queue signal")
	assert.Equal(t, "cache", signals["redis"], "redis should have cache signal")
	assert.Equal(t, "orm", signals["sqlalchemy"], "sqlalchemy should have orm signal")
	assert.Equal(t, "data_processing", signals["pandas"], "pandas should have data_processing signal")
	assert.Equal(t, "validation", signals["pydantic"], "pydantic should have validation signal")
	assert.Equal(t, "testing", signals["pytest"], "pytest should have testing signal")
}
