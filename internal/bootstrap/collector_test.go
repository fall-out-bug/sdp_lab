package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollector_Collect_AllSourcesPresent(t *testing.T) {
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpPath, 0o755))

	// Write scout.json (required)
	scout := ScoutData{
		PrimaryLanguage: "Go",
		BuildSystem:     "go",
		Languages:       map[string]float64{"Go": 0.9, "Shell": 0.1},
		HasTests:        true,
		HasCI:           true,
		CISystem:        "github",
		TestRatio:       0.35,
		TotalFiles:      120,
	}
	writeJSON(t, filepath.Join(sdpPath, "scout.json"), scout)

	// Write architect/report.json
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "architect"), 0o755))
	arch := ArchitectData{
		Components: []string{"cmd/sdp", "internal/scout"},
		Decisions: []ArchDecision{{ID: "ADR-001", Title: "Use Go", Status: "accepted"}},
		Patterns:  []string{"hexagonal"},
	}
	writeJSON(t, filepath.Join(sdpPath, "architect", "report.json"), arch)

	// Write metrics/report.json
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "metrics"), 0o755))
	met := MetricsData{BusFactor: 3, CommitFreq: "high", Staleness: "active"}
	writeJSON(t, filepath.Join(sdpPath, "metrics", "report.json"), met)

	// Write specs
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "specs"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sdpPath, "specs", "feat-1.md"), []byte("# spec"), 0o644))

	// Write index.db (just a non-empty file)
	require.NoError(t, os.WriteFile(filepath.Join(sdpPath, "index.db"), []byte("SQLITE"), 0o644))

	c := NewCollector(dir)
	ds, err := c.Collect()
	require.NoError(t, err)

	assert.NotNil(t, ds.Scout)
	assert.Equal(t, "Go", ds.Scout.PrimaryLanguage)
	assert.NotNil(t, ds.Architect)
	assert.Len(t, ds.Architect.Components, 2)
	assert.NotNil(t, ds.Metrics)
	assert.Equal(t, 3, ds.Metrics.BusFactor)
	assert.NotNil(t, ds.Spec)
	assert.Len(t, ds.Spec.Files, 1)
	assert.NotNil(t, ds.Index)
}

func TestCollector_Collect_PartialSources(t *testing.T) {
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpPath, 0o755))

	// Only scout.json + metrics
	scout := ScoutData{PrimaryLanguage: "Go", BuildSystem: "go"}
	writeJSON(t, filepath.Join(sdpPath, "scout.json"), scout)

	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "metrics"), 0o755))
	met := MetricsData{BusFactor: 2}
	writeJSON(t, filepath.Join(sdpPath, "metrics", "report.json"), met)

	c := NewCollector(dir)
	ds, err := c.Collect()
	require.NoError(t, err)

	assert.NotNil(t, ds.Scout)
	assert.NotNil(t, ds.Metrics)
	assert.Nil(t, ds.Architect)
	assert.Nil(t, ds.Spec)
	assert.Nil(t, ds.Index)
}

func TestCollector_Collect_NoScoutFails(t *testing.T) {
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpPath, 0o755))

	// No scout.json — should fail
	c := NewCollector(dir)
	_, err := c.Collect()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scout.json is required")
}

func TestCollector_CollectOptional_NoSourcesNoError(t *testing.T) {
	dir := t.TempDir()

	c := NewCollector(dir)
	ds := c.CollectOptional()

	assert.Nil(t, ds.Scout)
	assert.Nil(t, ds.Architect)
	assert.Nil(t, ds.Metrics)
	assert.Nil(t, ds.Spec)
	assert.Nil(t, ds.Index)
}

func TestCollector_DataSourcesAvailable(t *testing.T) {
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpPath, 0o755))

	// Only scout.json
	scout := ScoutData{PrimaryLanguage: "Go"}
	writeJSON(t, filepath.Join(sdpPath, "scout.json"), scout)

	c := NewCollector(dir)
	avail := c.DataSourcesAvailable()

	assert.True(t, avail["scout"])
	assert.False(t, avail["architect"])
	assert.False(t, avail["metrics"])
	assert.False(t, avail["spec"])
	assert.False(t, avail["index"])
}

func TestCollector_DataSourcesAvailable_None(t *testing.T) {
	dir := t.TempDir()
	c := NewCollector(dir)
	avail := c.DataSourcesAvailable()

	assert.False(t, avail["scout"])
	assert.False(t, avail["architect"])
	assert.False(t, avail["metrics"])
	assert.False(t, avail["spec"])
	assert.False(t, avail["index"])
}

func TestCollector_ExistingConfig(t *testing.T) {
	dir := t.TempDir()

	// Create some config files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# claude"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".sdp"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".beads"), 0o755))

	c := NewCollector(dir)
	existing := c.ExistingConfig()

	assert.Contains(t, existing, "claude_md")
	assert.Contains(t, existing, "sdp_dir")
	assert.Contains(t, existing, "beads_dir")
	assert.NotContains(t, existing, "agents_md")
	assert.NotContains(t, existing, "hooks_dir")
}

func TestCollector_ExistingConfig_None(t *testing.T) {
	dir := t.TempDir()
	c := NewCollector(dir)
	existing := c.ExistingConfig()
	assert.Empty(t, existing)
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := json.MarshalIndent(v, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}
