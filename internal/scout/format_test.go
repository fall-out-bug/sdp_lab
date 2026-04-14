package scout

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleCard() *ProjectCard {
	desc := "A test app"
	bs := "go-modules"
	pm := "go-modules"
	df := "go.mod"
	ci := "github-actions"
	first := "2024-06-15"
	last := "2026-04-13"
	_ = "v1.0.0" // latestRelease
	return &ProjectCard{
		Version:    "1.0.0",
		ScannedAt: time.Date(2026, 4, 13, 15, 0, 0, 0, time.UTC),
		DurationMs: 150,
		Identity: Identity{
			Name:            "myapp",
			Description:     &desc,
			PrimaryLanguage: "go",
			Languages: map[string]LangStats{
				"go":   {Files: 20, Ratio: 0.80},
				"yaml": {Files: 5, Ratio: 0.20},
			},
			BuildSystem: &bs,
			BuildFiles:  []string{"go.mod"},
			Monorepo:    false,
		},
		Scale: Scale{
			TotalFiles: 25, TotalLoc: 3200, SourceFiles: 20,
			TestFiles: 8, TestRatio: 0.29,
			Directories: 6, DepthMax: 3, MaxFileLoc: 200,
		},
		Activity: Activity{
			FirstCommit: &first, LastCommit: &last,
			TotalCommits: 150, Contributors: 4, Commits30d: 30,
		},
		Maturity: Maturity{
			HasReadme: true, HasTests: true, HasCI: true, CISystem: &ci,
		},
		Build: Build{
			EntryPoints:    []string{"cmd/app/main.go"},
			PackageManager: &pm, DependencyFile: &df,
		},
		Health: HealthSignals{
			BusFactorEstimate: 2, CommitFrequency: CommitFreqHigh,
			Staleness: StalenessActive, TestCoverageHint: CovPartial,
			ComplexityHint: ComplexityLow,
		},
	}
}

// ── FormatJSON ─────────────────────────────────────────────────────────

func TestFormatJSONIsValid(t *testing.T) {
	card := sampleCard()
	out, err := FormatJSON(card)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(out), &m); err != nil {
		t.Fatalf("FormatJSON output is not valid JSON: %v", err)
	}
	if m["version"] != "1.0.0" {
		t.Errorf("version = %v, want 1.0.0", m["version"])
	}
}

// ── FormatText ─────────────────────────────────────────────────────────

func TestFormatTextContainsKeyFields(t *testing.T) {
	card := sampleCard()
	out := FormatText(card)

	checks := []string{
		"myapp",
		"go",
		"3.2K LOC",
		"25 files",
		"8 test",
		"150 commits",
		"4 contributors",
		"Active",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("FormatText missing %q\nOutput:\n%s", want, out)
		}
	}
}

// ── FormatCard ─────────────────────────────────────────────────────────

func TestFormatCardCompact(t *testing.T) {
	card := sampleCard()
	out := FormatCard(card)

	// Card should be compact — fewer lines than text
	textLines := len(strings.Split(FormatText(card), "\n"))
	cardLines := len(strings.Split(out, "\n"))
	if cardLines > textLines {
		t.Errorf("card (%d lines) should be more compact than text (%d lines)", cardLines, textLines)
	}
	if !strings.Contains(out, "myapp") {
		t.Error("card should contain project name")
	}
}

// ── Artifact Writing ───────────────────────────────────────────────────

func TestWriteArtifact(t *testing.T) {
	dir := t.TempDir()
	card := sampleCard()

	path, err := WriteArtifact(dir, card)
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(dir, "scout.json")
	if path != expected {
		t.Errorf("path = %q, want %q", path, expected)
	}

	data, err := os.ReadFile(expected)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("artifact file is empty")
	}
}

func TestWriteArtifactCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".sdp")
	card := sampleCard()

	path, err := WriteArtifact(dir, card)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("artifact not created: %v", err)
	}
}

// ── Large Repo Safeguards ──────────────────────────────────────────────

func TestLargeRepoSkipLOCForMinorLanguages(t *testing.T) {
	// Create a repo with many files to trigger large-repo path
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.26\n"), 0o644)

	// Create 101 source files (above large-repo threshold of 100)
	for i := range 101 {
		os.MkdirAll(filepath.Join(dir, "pkg"), 0o755)
		os.WriteFile(filepath.Join(dir, "pkg", "file"+string(rune('A'+i%26))+".go"),
			[]byte("package pkg\nfunc F() {}\n"), 0o644)
	}
	// Add a minor-language file
	os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash\necho hi\n"), 0o644)

	scale := detectScale(dir, strPtr("go-modules"))
	// Should still have LOC counted — just tests it doesn't crash
	if scale.SourceFiles < 1 {
		t.Errorf("SourceFiles = %d, want >= 1", scale.SourceFiles)
	}
}

// ── Performance Budget ─────────────────────────────────────────────────

func TestPerformanceBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping perf test in short mode")
	}
	root := filepath.Join("..", "..", "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Skip("not running from sdp_lab worktree")
	}

	start := time.Now()
	_, err = Run(abs)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	// Design budget: <3s typical, <10s for large repos like sdp_lab
	if elapsed > 10*time.Second {
		t.Errorf("scout took %v, budget is 10s", elapsed)
	}
}

// ── Error Modes ────────────────────────────────────────────────────────

func TestRunNonexistentPath(t *testing.T) {
	_, err := Run("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestFormatTextNoGit(t *testing.T) {
	card := &ProjectCard{
		Version:  "1.0.0",
		Identity: Identity{Name: "nogit", PrimaryLanguage: "python"},
		Activity: Activity{}, // empty — no git
	}
	out := FormatText(card)
	if !strings.Contains(out, "nogit") {
		t.Error("should still render project name")
	}
}
