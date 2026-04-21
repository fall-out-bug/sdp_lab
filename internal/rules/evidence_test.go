package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadEvidenceDir_Empty(t *testing.T) {
	dir := t.TempDir()

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestReadEvidenceDir_WithEntries(t *testing.T) {
	dir := t.TempDir()
	writeJSON(t, dir, "run-1.json", EvidenceEntry{
		RunID: "run-1", Phase: "test", Verdict: "fail",
		Summary: "test A failed", Timestamp: "2026-04-19T10:00:00Z",
	})

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "run-1", entries[0].RunID)
	assert.Equal(t, "fail", entries[0].Verdict)
	assert.Equal(t, filepath.Join(dir, "run-1.json"), entries[0].FilePath)
}

func TestReadEvidenceDir_ArrayFile(t *testing.T) {
	dir := t.TempDir()
	arr := []EvidenceEntry{
		{RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "fail A"},
		{RunID: "run-1", Phase: "build", Verdict: "error", Summary: "err B"},
	}
	data, err := json.Marshal(arr)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "batch.json"), data, 0o644))

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 2)
}

func TestReadEvidenceDir_MalformedJSON(t *testing.T) {
	dir := t.TempDir()

	// Write a valid file.
	writeJSON(t, dir, "good.json", EvidenceEntry{
		RunID: "run-1", Phase: "test", Verdict: "fail", Summary: "ok",
	})

	// Write a malformed file.
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, "bad.json"), []byte(`{not valid json}`), 0o644))

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1, "malformed file should be skipped")
	assert.Equal(t, "run-1", entries[0].RunID)
}

func TestReadEvidenceDir_NonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "data.csv"), []byte("a,b"), 0o644))

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "non-.json files should be ignored")
}

func TestReadEvidenceDir_MissingDir(t *testing.T) {
	_, err := ReadEvidenceDir("/nonexistent/path/that/does/not/exist")
	assert.Error(t, err)
}

func TestReadEvidenceDir_Subdirectories(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "subdir"), 0o755))

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	assert.Empty(t, entries, "subdirectories should be skipped")
}

func TestReadEvidenceDir_PreservesFilePath(t *testing.T) {
	dir := t.TempDir()
	entry := EvidenceEntry{
		RunID: "run-1", Phase: "test", Verdict: "fail",
		Summary: "test", FilePath: "/custom/path.json",
	}
	writeJSON(t, dir, "custom.json", entry)

	entries, err := ReadEvidenceDir(dir)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	// FilePath from JSON should be preserved when set.
	assert.Equal(t, "/custom/path.json", entries[0].FilePath)
}

// --- helpers ---

func writeJSON(t *testing.T, dir, name string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), data, 0o644))
}
