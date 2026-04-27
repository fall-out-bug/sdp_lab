package replayutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sdp_dev/internal/inference/replayutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadCorpus_RealCorpus(t *testing.T) {
	corpusPath := "../../confidence/testdata/ws-verdict"
	if _, err := os.Stat(corpusPath); os.IsNotExist(err) {
		t.Skip("corpus not found, skipping")
	}

	fixtures, err := replayutil.LoadCorpus(corpusPath)
	require.NoError(t, err)
	assert.NotEmpty(t, fixtures, "expected at least one fixture")

	for _, f := range fixtures {
		assert.NotEmpty(t, f.ID, "fixture ID should not be empty")
		assert.NotEmpty(t, f.Category, "fixture category should not be empty")
		assert.NotNil(t, f.Raw, "fixture raw bytes should not be nil")
		assert.NotNil(t, f.Data, "fixture data should not be nil")
	}
}

func TestLoadCorpus_SyntheticFixtures(t *testing.T) {
	dir := t.TempDir()
	catDir := filepath.Join(dir, "correct")
	require.NoError(t, os.MkdirAll(catDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(catDir, "a.json"),
		[]byte(`{"verdict":"PASS","ws_id":"t1"}`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(catDir, "b.json"),
		[]byte(`{"verdict":"FAIL","ws_id":"t2"}`), 0o644))

	fixtures, err := replayutil.LoadCorpus(dir)
	require.NoError(t, err)
	require.Len(t, fixtures, 2)

	for _, f := range fixtures {
		assert.Equal(t, "correct", f.Category)
	}
	assert.Equal(t, "passed", findByWS("t1", fixtures).GoldenStatus)
	assert.Equal(t, "failed", findByWS("t2", fixtures).GoldenStatus)
}

func TestLoadCorpus_InvalidPath(t *testing.T) {
	_, err := replayutil.LoadCorpus("/no/such/path")
	require.Error(t, err)
}

func TestLoadCorpus_FileNotDir(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.json")
	require.NoError(t, err)
	_ = f.Close()
	_, err = replayutil.LoadCorpus(f.Name())
	require.Error(t, err)
}

func TestLoadCorpus_MalformedJSON(t *testing.T) {
	// Malformed JSON fixtures are silently skipped (adversarial corpus entries).
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{bad`), 0o644))
	fixtures, err := replayutil.LoadCorpus(dir)
	require.NoError(t, err)
	assert.Empty(t, fixtures, "malformed JSON fixture should be skipped")
}

func TestWriteEvidence_MultipleRecords(t *testing.T) {
	dir := t.TempDir()
	evidence := replayutil.AggregateEvidence{
		RunID:     "multi",
		Timestamp: "2026-04-26T00:00:00Z",
		Records: []replayutil.RunRecord{
			{ID: "a", Category: "correct", MonolithicStatus: "passed", DecomposedStatus: "passed",
				LatencyMonoMs: 50, LatencyDecompMs: 150, TokensMono: 200, TokensDecomp: 100, CostMonoUSD: 0.001, CostDecompUSD: 0.0005},
			{ID: "b", Category: "adversarial", MonolithicStatus: "failed", DecomposedStatus: "failed",
				LatencyMonoMs: 80, LatencyDecompMs: 220, TokensMono: 300, TokensDecomp: 180, CostMonoUSD: 0.002, CostDecompUSD: 0.0008},
		},
	}
	err := replayutil.WriteEvidence(dir, "multi", evidence)
	require.NoError(t, err)

	csvData, _ := os.ReadFile(filepath.Join(dir, "multi", "results.csv"))
	lines := strings.Split(strings.TrimSpace(string(csvData)), "\n")
	assert.Len(t, lines, 3) // header + 2 data rows
}

func TestWriteEvidence(t *testing.T) {
	dir := t.TempDir()
	evidence := replayutil.AggregateEvidence{
		RunID:     "test-run",
		Timestamp: "2026-04-26T00:00:00Z",
		Records: []replayutil.RunRecord{
			{ID: "correct/a", Category: "correct", MonolithicStatus: "passed", DecomposedStatus: "passed",
				LatencyMonoMs: 100, LatencyDecompMs: 200, TokensMono: 500, TokensDecomp: 300},
		},
	}
	err := replayutil.WriteEvidence(dir, "test-run", evidence)
	require.NoError(t, err)

	// Verify JSON written.
	jsonData, err := os.ReadFile(filepath.Join(dir, "test-run", "evidence.json"))
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "correct/a")

	// Verify CSV written.
	csvData, err := os.ReadFile(filepath.Join(dir, "test-run", "results.csv"))
	require.NoError(t, err)
	assert.Contains(t, string(csvData), "correct/a")
	assert.Contains(t, string(csvData), "id,category") // header
}

func TestWriteEvidence_ReadOnlyDir(t *testing.T) {
	dir := t.TempDir()
	// Make dir read-only so MkdirAll fails for the sub-run directory.
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := replayutil.WriteEvidence(dir, "run-id", replayutil.AggregateEvidence{})
	require.Error(t, err)
}

func TestNewRunID(t *testing.T) {
	id := replayutil.NewRunID()
	assert.NotEmpty(t, id)
	assert.Len(t, id, len("20060102-150405"))
}

func TestLoadCorpus_NormalisedVerdicts(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"pass.json":    `{"verdict":"PASS"}`,
		"fail.json":    `{"verdict":"FAIL"}`,
		"warn.json":    `{"verdict":"WARN"}`,
		"partial.json": `{"verdict":"partial"}`,
		"other.json":   `{"verdict":"other"}`,
	}
	for name, content := range cases {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
	}
	fixtures, err := replayutil.LoadCorpus(dir)
	require.NoError(t, err)

	m := map[string]string{}
	for _, f := range fixtures {
		m[f.ID] = f.GoldenStatus
	}
	assert.Equal(t, "passed", m["pass"])
	assert.Equal(t, "failed", m["fail"])
	assert.Equal(t, "partial", m["warn"])
	assert.Equal(t, "partial", m["partial"])
	assert.Equal(t, "other", m["other"])
}

func findByWS(wsID string, fixtures []replayutil.Fixture) replayutil.Fixture {
	for _, f := range fixtures {
		if ws, ok := f.Data["ws_id"].(string); ok && ws == wsID {
			return f
		}
	}
	return replayutil.Fixture{}
}
