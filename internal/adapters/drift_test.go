package adapters_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/adapters"
)

// setupGeneratedDir writes a generated map into dir/outDir and returns the
// absolute path to outDir.
func setupGeneratedDir(t *testing.T, dir string, files map[string][]byte) string {
	t.Helper()
	for rel, data := range files {
		dest := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(dest), err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", dest, err)
		}
	}
	return dir
}

// TestDoctorAdapters_Clean verifies that when on-disk files exactly match the
// generated map, CheckDrift reports no drifts and no orphans.
func TestDoctorAdapters_Clean(t *testing.T) {
	generated := map[string][]byte{
		".claude/commands/build.md": []byte("# build\nsome content\n"),
		".codex/skills/build.md":    []byte("# build skill\n"),
	}

	outDir := t.TempDir()
	repoRoot := t.TempDir()

	// Write identical files to outDir.
	setupGeneratedDir(t, outDir, generated)

	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		t.Fatalf("CheckDrift returned error: %v", err)
	}
	if len(result.Drifts) != 0 {
		t.Errorf("expected 0 drifts, got %d: %v", len(result.Drifts), result.Drifts)
	}
	if len(result.Orphans) != 0 {
		t.Errorf("expected 0 orphans, got %d: %v", len(result.Orphans), result.Orphans)
	}
	if !result.IsClean() {
		t.Error("IsClean() should be true")
	}
}

// TestDoctorAdapters_DriftDetected verifies that CheckDrift returns a non-empty
// Drifts slice when a generated file has been modified on disk.
func TestDoctorAdapters_DriftDetected(t *testing.T) {
	generated := map[string][]byte{
		".claude/commands/build.md": []byte("# build\noriginal content\n"),
	}

	outDir := t.TempDir()
	repoRoot := t.TempDir()

	// Write a DIFFERENT version to simulate manual edit.
	modifiedFiles := map[string][]byte{
		".claude/commands/build.md": []byte("# build\nMANUALLY MODIFIED\n"),
	}
	setupGeneratedDir(t, outDir, modifiedFiles)

	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		t.Fatalf("CheckDrift returned error: %v", err)
	}
	if len(result.Drifts) == 0 {
		t.Error("expected at least 1 drift but got none")
	}
	if result.IsClean() {
		t.Error("IsClean() should be false when drift is present")
	}
}

// TestDoctorAdapters_MissingFile verifies that a file present in generated but
// absent on disk is reported as a drift (MISSING).
func TestDoctorAdapters_MissingFile(t *testing.T) {
	generated := map[string][]byte{
		".claude/commands/build.md":   []byte("# build\n"),
		".claude/commands/feature.md": []byte("# feature\n"),
	}

	outDir := t.TempDir()
	repoRoot := t.TempDir()

	// Only write one of the two files.
	setupGeneratedDir(t, outDir, map[string][]byte{
		".claude/commands/build.md": []byte("# build\n"),
	})

	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		t.Fatalf("CheckDrift returned error: %v", err)
	}
	found := false
	for _, d := range result.Drifts {
		if len(d) > 8 && d[:8] == "MISSING " {
			found = true
		}
	}
	if !found {
		t.Errorf("expected a MISSING drift entry; got: %v", result.Drifts)
	}
}

// TestDoctorAdapters_OrphanFile verifies that a file present in the live harness
// tree but absent from the generated map is reported as an orphan (not a drift).
func TestDoctorAdapters_OrphanFile(t *testing.T) {
	generated := map[string][]byte{
		".claude/commands/build.md": []byte("# build\n"),
	}

	outDir := t.TempDir()
	repoRoot := t.TempDir()

	// Set up on-disk generated files (clean).
	setupGeneratedDir(t, outDir, generated)

	// Create an extra file in the live .claude/commands/ tree of repoRoot.
	liveDir := filepath.Join(repoRoot, ".claude", "commands")
	if err := os.MkdirAll(liveDir, 0o755); err != nil {
		t.Fatalf("mkdir live dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(liveDir, "mystery.md"), []byte("# mystery\n"), 0o644); err != nil {
		t.Fatalf("write mystery.md: %v", err)
	}

	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		t.Fatalf("CheckDrift returned error: %v", err)
	}

	// No drifts expected (generated files are in sync).
	if len(result.Drifts) != 0 {
		t.Errorf("expected 0 drifts, got %d: %v", len(result.Drifts), result.Drifts)
	}

	// Orphan should be detected.
	found := false
	for _, o := range result.Orphans {
		if o == ".claude/commands/mystery.md" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected orphan .claude/commands/mystery.md; got orphans: %v", result.Orphans)
	}
}

// TestDoctorAdapters_WhitelistedOrphanIgnored verifies that known non-manifest
// files (e.g. sweep.md) are NOT reported as orphans.
func TestDoctorAdapters_WhitelistedOrphanIgnored(t *testing.T) {
	generated := map[string][]byte{}

	outDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create the whitelisted file in the live tree.
	liveDir := filepath.Join(repoRoot, ".claude", "commands")
	if err := os.MkdirAll(liveDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(liveDir, "sweep.md"), []byte("# sweep\n"), 0o644); err != nil {
		t.Fatalf("write sweep.md: %v", err)
	}

	result, err := adapters.CheckDrift(generated, outDir, repoRoot)
	if err != nil {
		t.Fatalf("CheckDrift: %v", err)
	}
	if len(result.Orphans) != 0 {
		t.Errorf("sweep.md should be whitelisted; got orphans: %v", result.Orphans)
	}
}

// TestFormatDriftReport_Clean verifies that a clean result renders an "ok" message.
func TestFormatDriftReport_Clean(t *testing.T) {
	result := &adapters.DriftResult{}
	report := adapters.FormatDriftReport(result, ".sdp/generated", false)
	if len(report) == 0 {
		t.Error("FormatDriftReport returned empty string")
	}
	// Should contain an "ok" line.
	if !containsSubstring(report, "ok:") {
		t.Errorf("expected 'ok:' in report; got:\n%s", report)
	}
}

// TestFormatDriftReport_StrictOrphansAreErrors verifies that --strict mode labels orphans as ERROR.
func TestFormatDriftReport_StrictOrphansAreErrors(t *testing.T) {
	result := &adapters.DriftResult{
		Orphans: []string{".claude/commands/extra.md"},
	}
	report := adapters.FormatDriftReport(result, ".sdp/generated", true)
	if !containsSubstring(report, "ERROR") {
		t.Errorf("expected 'ERROR' in strict-mode report; got:\n%s", report)
	}
}

// TestFormatDriftReport_NonStrictOrphansAreWarnings verifies that without --strict, orphans are warnings.
func TestFormatDriftReport_NonStrictOrphansAreWarnings(t *testing.T) {
	result := &adapters.DriftResult{
		Orphans: []string{".claude/commands/extra.md"},
	}
	report := adapters.FormatDriftReport(result, ".sdp/generated", false)
	if !containsSubstring(report, "WARNING") {
		t.Errorf("expected 'WARNING' in non-strict report; got:\n%s", report)
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && stringContains(s, sub))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
