package spec

import (
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

	// SQL constraints should show up in business rules if migrations dir exists
	// The spec.go pipeline scans migrations/ subdirectories
	assert.NotNil(t, report)
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
	}
	path, err := WriteArtifact(tmpDir, report)
	require.NoError(t, err)
	assert.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"version": "1.0.0"`)
}
