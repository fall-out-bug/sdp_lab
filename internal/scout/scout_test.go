package scout

import (
	"os"
	"path/filepath"
	"testing"
)

// ── Pipeline Tests ──────────────────────────────────────────────────────

func TestPipelineOnTempRepo(t *testing.T) {
	dir := createTempRepo(t, map[string]string{
		"go.mod":          "module example.com/app\ngo 1.26\nrequire (\n\tfmt v0.0.0\n)\n",
		"main.go":         "package main\nfunc main() { println(\"hello\") }\n",
		"main_test.go":    "package main\nimport \"testing\"\nfunc TestMain(t *testing.T) {}\n",
		"README.md":       "# App\nA test app.\n",
		".goreleaser.yml": "builds:\n  - main: .\n",
	}, true)

	card, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if card.Version != "1.0.0" {
		t.Errorf("Version = %q", card.Version)
	}
	if card.DurationMs <= 0 {
		t.Error("DurationMs should be > 0")
	}
	if card.Identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q", card.Identity.PrimaryLanguage)
	}
	if card.Scale.SourceFiles < 1 {
		t.Errorf("SourceFiles = %d, want >= 1", card.Scale.SourceFiles)
	}
	if card.Activity.TotalCommits < 1 {
		t.Errorf("TotalCommits = %d, want >= 1", card.Activity.TotalCommits)
	}
	if card.Health.CommitFrequency == "" {
		t.Error("Health signals should be derived")
	}
}

// ── Health Signals ──────────────────────────────────────────────────────

func TestHealthDerivesCorrectSignals(t *testing.T) {
	card := &ProjectCard{
		Scale: Scale{
			SourceFiles: 100, TestFiles: 30, TestRatio: 0.3,
			TotalFiles: 200, MaxFileLoc: 150, DepthMax: 4,
		},
		Activity: Activity{
			Contributors: 5, Commits30d: 100, LastCommit: strPtr("2026-04-10"),
		},
	}

	deriveHealthSignals(card)

	if card.Health.CommitFrequency != CommitFreqHigh {
		t.Errorf("CommitFrequency = %q, want %q", card.Health.CommitFrequency, CommitFreqHigh)
	}
	if card.Health.TestCoverageHint != CovGood {
		t.Errorf("TestCoverageHint = %q, want %q", card.Health.TestCoverageHint, CovGood)
	}
	if card.Health.Staleness != StalenessActive {
		t.Errorf("Staleness = %q, want %q", card.Health.Staleness, StalenessActive)
	}
}

// ── Integration: Run on sdp_lab ────────────────────────────────────────

func TestIntegrationOnSelf(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(abs, "go.mod")); err != nil {
		t.Skip("not running from sdp_lab worktree")
	}

	card, err := Run(abs)
	if err != nil {
		t.Fatalf("Run(%s): %v", abs, err)
	}

	if card.Identity.PrimaryLanguage != "go" {
		t.Errorf("PrimaryLanguage = %q, want go", card.Identity.PrimaryLanguage)
	}
	if card.Identity.BuildSystem == nil || *card.Identity.BuildSystem != "go-modules" {
		t.Error("BuildSystem should be go-modules")
	}
	if card.Scale.TotalFiles < 10 {
		t.Errorf("TotalFiles = %d, want >= 10", card.Scale.TotalFiles)
	}
	if card.Scale.TotalLoc < 1000 {
		t.Errorf("TotalLoc = %d, want >= 1000", card.Scale.TotalLoc)
	}
	if card.Activity.TotalCommits < 10 {
		t.Errorf("TotalCommits = %d, want >= 10", card.Activity.TotalCommits)
	}
	if !card.Maturity.HasReadme {
		t.Error("sdp_lab should have README")
	}
}
