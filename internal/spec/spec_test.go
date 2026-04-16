package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// td is a shorthand for the testdata directory used across tests.
func td() string { return filepath.Join("testdata") }

func TestRun_GeneratesReport(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, "1.0.0", report.Version)
	assert.False(t, report.GeneratedAt.IsZero())
	assert.Greater(t, report.DurationMs, int64(0))
}

func TestRun_APIContracts(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Greater(t, report.APIContracts.Total, 0)
}

func TestRun_BusinessRules(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Greater(t, report.BusinessRules.Total, 0)
}

func TestRun_Coverage(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Greater(t, report.Coverage.FilesScanned, 0)
}

func TestRun_NonexistentDir(t *testing.T) {
	report, err := Run("testdata/nonexistent_dir_xyz")
	assert.Error(t, err)
	assert.Nil(t, report)
}

func TestRun_SQLConstraints(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestRun_Invariants(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Greater(t, report.Invariants.Total, 0)
}

func TestRun_SLAParameters(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Greater(t, report.SLAParameters.Total, 0)
}

func TestRun_InvariantCategories(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.NotEmpty(t, report.Invariants.TypeSystem)
	assert.NotEmpty(t, report.Invariants.Concurrency)
	assert.NotEmpty(t, report.Invariants.Architectural)
}

func TestRun_SLATimeouts(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.NotEmpty(t, report.SLAParameters.Timeouts)
}

func TestRun_SecretRedaction(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	for _, params := range [][]SLAParam{
		report.SLAParameters.Timeouts, report.SLAParameters.Retries,
		report.SLAParameters.RateLimits, report.SLAParameters.ResourcePools,
		report.SLAParameters.HealthChecks,
	} {
		for _, p := range params {
			assert.NotContains(t, p.Value, "s3cret")
			assert.NotContains(t, p.Value, "password")
		}
	}
}

func TestRun_Determinism(t *testing.T) {
	r1, err := Run(td())
	require.NoError(t, err)
	r2, err := Run(td())
	require.NoError(t, err)
	assert.Equal(t, r1.APIContracts.Total, r2.APIContracts.Total)
	assert.Equal(t, r1.BusinessRules.Total, r2.BusinessRules.Total)
	assert.Equal(t, r1.Invariants.Total, r2.Invariants.Total)
	assert.Equal(t, r1.SLAParameters.Total, r2.SLAParameters.Total)
	assert.Equal(t, r1.Coverage.FilesScanned, r2.Coverage.FilesScanned)
}

func TestRun_DeterministicJSON(t *testing.T) {
	r1, err := Run(td())
	require.NoError(t, err)
	r2, err := Run(td())
	require.NoError(t, err)
	j1, _ := json.Marshal(r1)
	j2, _ := json.Marshal(r2)
	var m1, m2 map[string]interface{}
	require.NoError(t, json.Unmarshal(j1, &m1))
	require.NoError(t, json.Unmarshal(j2, &m2))
	assert.Equal(t, m1["version"], m2["version"])
	assert.Equal(t, m1["repo"], m2["repo"])
}

// --- Enrichment tests ---

func TestRun_NoEnrichmentByDefault(t *testing.T) {
	report, err := Run(td())
	require.NoError(t, err)
	assert.Nil(t, report.Enrichment)
}

func TestRunWithOptions_EnrichmentOff(t *testing.T) {
	report, err := RunWithOptions(td(), RunOptions{Enrich: false})
	require.NoError(t, err)
	assert.Nil(t, report.Enrichment)
}

func TestRunWithOptions_EnrichmentOn(t *testing.T) {
	report, err := RunWithOptions(td(), RunOptions{Enrich: true})
	require.NoError(t, err)
	require.NotNil(t, report.Enrichment)
	assert.True(t, report.Enrichment.Attempted)
	assert.Equal(t, "not_configured", report.Enrichment.Status)
	assert.Contains(t, report.Enrichment.Note, "not yet implemented")
}

func TestRunWithOptions_EnrichmentDoesNotChangeDeterministicOutput(t *testing.T) {
	plain, err := Run(td())
	require.NoError(t, err)
	enriched, err := RunWithOptions(td(), RunOptions{Enrich: true})
	require.NoError(t, err)
	assert.Equal(t, plain.APIContracts.Total, enriched.APIContracts.Total)
	assert.Equal(t, plain.BusinessRules.Total, enriched.BusinessRules.Total)
	assert.Equal(t, plain.Invariants.Total, enriched.Invariants.Total)
	assert.Equal(t, plain.SLAParameters.Total, enriched.SLAParameters.Total)
	assert.Equal(t, plain.Coverage.FilesScanned, enriched.Coverage.FilesScanned)
}

// --- Diff integration tests ---

func TestDiffSpecs_Integration(t *testing.T) {
	v1 := filepath.Join("testdata", "spec_v1.json")
	v2 := filepath.Join("testdata", "spec_v2.json")
	diff, err := DiffSpecs(v1, v2)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", diff.Version)
	assert.Equal(t, v1, diff.OldSnapshot)
	assert.Equal(t, v2, diff.NewSnapshot)
}

func TestDiffSpecs_FalsePositiveGuard(t *testing.T) {
	r := baseReport()
	p1 := writeSnapshot(t, "old", r)
	p2 := writeSnapshot(t, "new", r)
	diff, err := DiffSpecs(p1, p2)
	require.NoError(t, err)
	assert.Empty(t, diff.APIChanges)
	assert.Empty(t, diff.RuleChanges)
	assert.Empty(t, diff.InvChanges)
	assert.Empty(t, diff.SLAChanges)
	assert.Equal(t, DiffSummary{}, diff.Summary)
}

func TestWriteArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	report := &SpecReport{
		Version:       "1.0.0",
		APIContracts:  APIContracts{Total: 5, HTTPEndpoints: []Endpoint{{Method: "GET", Path: "/test"}}},
		BusinessRules: BusinessRules{Total: 3},
		Invariants:    Invariants{Total: 2},
		SLAParameters: SLAParameters{Total: 4},
	}
	path, err := WriteArtifact(tmpDir, report)
	require.NoError(t, err)
	assert.FileExists(t, path)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"version": "1.0.0"`)
}
