package spec

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRun_GeneratesReport(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	require.NotNil(t, report)

	assert.Equal(t, "1.0.0", report.Version)
	assert.False(t, report.GeneratedAt.IsZero())
	assert.Greater(t, report.DurationMs, int64(0))
}

func TestRun_APIContracts(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)

	assert.Greater(t, report.APIContracts.Total, 0, "should find API endpoints")
}

func TestRun_BusinessRules(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)

	assert.Greater(t, report.BusinessRules.Total, 0, "should find business rules")
}

func TestRun_Coverage(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)

	assert.Greater(t, report.Coverage.FilesScanned, 0, "should scan files")
}

func TestRun_NonexistentDir(t *testing.T) {
	report, err := Run("testdata/nonexistent_dir_xyz")
	assert.Error(t, err)
	assert.Nil(t, report)
}

func TestRun_SQLConstraints(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	assert.NotNil(t, report)
}

func TestRun_Invariants(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	assert.Greater(t, report.Invariants.Total, 0, "should extract invariants")
}

func TestRun_SLAParameters(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	assert.Greater(t, report.SLAParameters.Total, 0, "should extract SLA parameters")
}

func TestRun_InvariantCategories(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	// Should have at least one type assertion
	assert.NotEmpty(t, report.Invariants.TypeSystem, "should find type system invariants")
	// Should have at least one mutex guard
	assert.NotEmpty(t, report.Invariants.Concurrency, "should find concurrency invariants")
	// Should have at least one architectural invariant (interface compliance)
	assert.NotEmpty(t, report.Invariants.Architectural, "should find architectural invariants")
}

func TestRun_SLATimeouts(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, report.SLAParameters.Timeouts, "should find timeout params")
}

func TestRun_SecretRedaction(t *testing.T) {
	dir := filepath.Join("testdata")
	report, err := Run(dir)
	require.NoError(t, err)
	// Check all SLA params for leaked secrets
	checkNoSecrets(t, report.SLAParameters.Timeouts)
	checkNoSecrets(t, report.SLAParameters.Retries)
	checkNoSecrets(t, report.SLAParameters.RateLimits)
	checkNoSecrets(t, report.SLAParameters.ResourcePools)
	checkNoSecrets(t, report.SLAParameters.HealthChecks)
}

func checkNoSecrets(t *testing.T, params []SLAParam) {
	t.Helper()
	for _, p := range params {
		assert.NotContains(t, p.Value, "s3cret", "no raw secret in output")
		assert.NotContains(t, p.Value, "password", "no raw password in output")
		assert.NotContains(t, p.Value, "hunter2", "no raw password in output")
	}
}

func TestRun_Determinism(t *testing.T) {
	dir := filepath.Join("testdata")
	r1, err := Run(dir)
	require.NoError(t, err)
	r2, err := Run(dir)
	require.NoError(t, err)
	// Same counts on same input
	assert.Equal(t, r1.APIContracts.Total, r2.APIContracts.Total)
	assert.Equal(t, r1.BusinessRules.Total, r2.BusinessRules.Total)
	assert.Equal(t, r1.Invariants.Total, r2.Invariants.Total)
	assert.Equal(t, r1.SLAParameters.Total, r2.SLAParameters.Total)
	assert.Equal(t, r1.Coverage.FilesScanned, r2.Coverage.FilesScanned)
}

func TestRun_DeterministicJSON(t *testing.T) {
	dir := filepath.Join("testdata")
	r1, err := Run(dir)
	require.NoError(t, err)
	r2, err := Run(dir)
	require.NoError(t, err)
	// Structural determinism: same counts, same top-level keys
	j1, _ := json.Marshal(r1)
	j2, _ := json.Marshal(r2)
	var m1, m2 map[string]interface{}
	require.NoError(t, json.Unmarshal(j1, &m1))
	require.NoError(t, json.Unmarshal(j2, &m2))
	// Compare structural keys exist
	assert.Equal(t, m1["version"], m2["version"])
	assert.Equal(t, m1["repo"], m2["repo"])
	// Compare nested totals
	inv1, _ := m1["invariants"].(map[string]interface{})
	inv2, _ := m2["invariants"].(map[string]interface{})
	assert.Equal(t, inv1["total"], inv2["total"])
	sla1, _ := m1["sla_parameters"].(map[string]interface{})
	sla2, _ := m2["sla_parameters"].(map[string]interface{})
	assert.Equal(t, sla1["total"], sla2["total"])
}

func TestWriteArtifact(t *testing.T) {
	tmpDir := t.TempDir()
	report := &SpecReport{
		Version: "1.0.0",
		APIContracts: APIContracts{
			Total: 5,
			HTTPEndpoints: []Endpoint{
				{Method: "GET", Path: "/test"},
			},
		},
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
	assert.Contains(t, string(data), `"invariants"`)
	assert.Contains(t, string(data), `"sla_parameters"`)
}
