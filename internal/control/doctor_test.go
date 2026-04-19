package control

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDraftFiles_WithDraftFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create DRAFT- prefixed files
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "DRAFT-design.md"), []byte("draft"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "DRAFT-artifact.yaml"), []byte("draft"), 0644))

	checks := checkDraftFiles(tmpDir)
	require.Len(t, checks, 1, "expected exactly one check when DRAFT files are present")

	check := checks[0]
	assert.Equal(t, "draft-files", check.CheckID)
	assert.Equal(t, "info", check.Severity, "DRAFT check must be INFO severity, not error")
	assert.Contains(t, check.Message, "bootstrap incomplete")
	assert.Contains(t, check.Message, "2 DRAFT file(s)")
}

func TestCheckDraftFiles_CleanDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	checks := checkDraftFiles(tmpDir)
	assert.Empty(t, checks, "no checks should be returned when no DRAFT files are present")
}

func TestCheckDraftFiles_SeverityIsInfo(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "DRAFT-test.md"), []byte("draft"), 0644))

	checks := checkDraftFiles(tmpDir)
	require.NotEmpty(t, checks)
	assert.Equal(t, "info", checks[0].Severity,
		"DRAFT check severity must be 'info', not 'error' or 'warning'")
}

func TestCheckDraftFiles_ListsEachPath(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "DRAFT-alpha.md"), []byte("a"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "DRAFT-beta.md"), []byte("b"), 0644))

	checks := checkDraftFiles(tmpDir)
	require.Len(t, checks, 1)

	// The message should contain both file names
	assert.Contains(t, checks[0].Message, "DRAFT-alpha.md")
	assert.Contains(t, checks[0].Message, "DRAFT-beta.md")
}

func TestCheckDraftFiles_IgnoresNonDraftFiles(t *testing.T) {
	tmpDir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "README.md"), []byte("readme"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "draft-lowercase.md"), []byte("lower"), 0644))

	checks := checkDraftFiles(tmpDir)
	assert.Empty(t, checks, "non-DRAFT- prefixed files should be ignored")
}
