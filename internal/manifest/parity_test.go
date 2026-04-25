package manifest_test

import (
	"strings"
	"testing"
	"time"

	"sdp_dev/internal/manifest"
)

func fixedTime() time.Time {
	return time.Date(2026, 4, 25, 12, 0, 0, 0, time.UTC)
}

func TestParityMatrix_AllHarnessesByDefault(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Commands: []manifest.Command{
			{Name: "build", Path: "x.md"}, // no harnesses → all
		},
	}
	got := m.ParityMatrix(fixedTime())
	if !strings.Contains(got, "| `build` | ✓ | ✓ | ✓ | ✓ |") {
		t.Errorf("expected all-harness coverage, got:\n%s", got)
	}
}

func TestParityMatrix_PartialCoverage(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Commands: []manifest.Command{
			{
				Name:      "claude-only",
				Path:      "x.md",
				Harnesses: []manifest.Harness{manifest.HarnessClaudeCode},
			},
		},
	}
	got := m.ParityMatrix(fixedTime())
	if !strings.Contains(got, "| `claude-only` | ✓ | — | — | — |") {
		t.Errorf("expected single-harness row, got:\n%s", got)
	}
}

func TestParityMatrix_IntentionalGapMarker(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Commands: []manifest.Command{
			{
				Name:        "deliver",
				Path:        "x.md",
				Harnesses:   []manifest.Harness{manifest.HarnessClaudeCode},
				ParityNotes: "OpenCode lacks long-running session — see F127-05 deadlock.",
			},
		},
	}
	got := m.ParityMatrix(fixedTime())
	if !strings.Contains(got, "| `deliver` | ✓ | ⚠ | ⚠ | ⚠ |") {
		t.Errorf("expected gap marker for non-claude harnesses, got:\n%s", got)
	}
	if !strings.Contains(got, "**command/deliver**") {
		t.Errorf("expected note line, got:\n%s", got)
	}
}

func TestParityMatrix_DeterministicOutput(t *testing.T) {
	m := &manifest.Manifest{
		Version:    "1.0.0",
		SDPVersion: "1.0.0",
		Commands: []manifest.Command{
			{Name: "z", Path: "a.md"},
			{Name: "a", Path: "b.md"},
			{Name: "m", Path: "c.md"},
		},
	}
	first := m.ParityMatrix(fixedTime())
	second := m.ParityMatrix(fixedTime())
	if first != second {
		t.Fatal("output is not deterministic")
	}
	// Sorting check: 'a' must appear before 'm' before 'z'.
	ai := strings.Index(first, "| `a` |")
	mi := strings.Index(first, "| `m` |")
	zi := strings.Index(first, "| `z` |")
	if !(ai >= 0 && ai < mi && mi < zi) {
		t.Errorf("rows not sorted alphabetically: a=%d m=%d z=%d\n%s", ai, mi, zi, first)
	}
}

func TestParityMatrix_OmitsEmptyMCPSection(t *testing.T) {
	m := &manifest.Manifest{Version: "1.0.0", SDPVersion: "1.0.0"}
	got := m.ParityMatrix(fixedTime())
	if strings.Contains(got, "MCP Servers") {
		t.Errorf("MCP section should be omitted when empty, got:\n%s", got)
	}
}
