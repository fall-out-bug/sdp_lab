package metrics

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestIntegrationPipeline exercises the full Collect → Filter → Analyzers → MetricsReport
// pipeline against a synthetic git repo with known data.
func TestIntegrationPipeline(t *testing.T) {
	// ── Setup synthetic repo ──
	dir := t.TempDir()
	runIG(t, dir, "init")
	runIG(t, dir, "config", "user.email", "alice@example.com")
	runIG(t, dir, "config", "user.name", "Alice")

	// Create commits with known authors and patterns
	commitDated(t, dir, "internal/app.go", "package internal\n", "2025-06-01T10:00:00Z", "feat: add app")
	commitDated(t, dir, "internal/handler.go", "package internal\n", "2025-07-01T10:00:00Z", "feat: add handler")
	commitDated(t, dir, "internal/util.go", "package internal\n", "2025-08-01T10:00:00Z", "fix: fix util")

	// Second author
	runIG(t, dir, "config", "user.email", "bob@example.com")
	runIG(t, dir, "config", "user.name", "Bob")
	commitDated(t, dir, "pkg/client.go", "package pkg\n", "2025-09-01T10:00:00Z", "feat: add client")
	commitDated(t, dir, "pkg/server.go", "package pkg\n", "2025-10-01T10:00:00Z", "refactor: simplify server")

	// Tags
	runIG(t, dir, "tag", "v1.0.0")
	commitDated(t, dir, "README.md", "# project\n", "2025-11-01T10:00:00Z", "docs: update readme")
	runIG(t, dir, "tag", "v1.1.0")

	// ── Collect ──
	data, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect failed: %v", err)
	}
	if len(data.Commits) < 5 {
		t.Errorf("Commits = %d, want >= 5", len(data.Commits))
	}
	if len(data.Tags) < 2 {
		t.Errorf("Tags = %d, want >= 2", len(data.Tags))
	}

	// ── Filter ──
	filtered := Filter(data)
	if len(filtered.Commits) == 0 {
		t.Fatal("Filter removed all commits")
	}
	// All filtered commits should have real authors
	for _, c := range filtered.Commits {
		if IsBot(c.Author) {
			t.Errorf("bot author passed filter: %q", c.Author)
		}
	}

	// ── All Analyzers ──
	report := MetricsReport{
		Version:         "1.0.0",
		GeneratedAt:     time.Now(),
		RepoPath:        dir,
		CommitsAnalyzed: len(filtered.Commits),
		Period: TimePeriod{
			From: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
			To:   time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC),
		},
		Hygiene:        AnalyzeHygiene(filtered),
		Waste:          AnalyzeWaste(filtered),
		GitFlow:        AnalyzeGitFlow(filtered),
		ReleaseQuality: AnalyzeReleaseQuality(filtered),
		Stabilization:  AnalyzeStabilization(filtered),
		KnowledgeRisk:  AnalyzeKnowledge(filtered),
		Decay:          AnalyzeDecay(filtered),
	}

	// ── Verify analyzers produced non-nil results ──
	if report.Hygiene == nil {
		t.Error("AnalyzeHygiene returned nil")
	}
	if report.Waste == nil {
		t.Error("AnalyzeWaste returned nil")
	}
	if report.GitFlow == nil {
		t.Error("AnalyzeGitFlow returned nil")
	}
	if report.KnowledgeRisk == nil {
		t.Error("AnalyzeKnowledge returned nil")
	}
	if report.Decay == nil {
		t.Error("AnalyzeDecay returned nil")
	}

	// ── Verify specific metric values from known data ──
	// Knowledge: 2 authors (Alice, Bob), bus factor should be reasonable
	if report.KnowledgeRisk != nil {
		if report.KnowledgeRisk.OverallBusFactor < 1 {
			t.Errorf("OverallBusFactor = %d, want >= 1", report.KnowledgeRisk.OverallBusFactor)
		}
		gini := report.KnowledgeRisk.GiniCoefficient
		if gini < 0 || gini > 1 {
			t.Errorf("GiniCoefficient = %f, want [0,1]", gini)
		}
	}

	// Hygiene: all subjects follow conventional commits → ratio should be 1.0
	if report.Hygiene != nil {
		if report.Hygiene.ConventionalCommitsRatio < 0.8 {
			t.Errorf("ConventionalCommitsRatio = %f, want >= 0.8 (all subjects are conventional)", report.Hygiene.ConventionalCommitsRatio)
		}
	}

	// ReleaseQuality: 2 tags → at least 1 release analyzed
	if report.ReleaseQuality != nil && len(data.Tags) >= 2 {
		if report.ReleaseQuality.ReleasesAnalyzed < 1 {
			t.Errorf("ReleasesAnalyzed = %d, want >= 1 (have %d tags)", report.ReleaseQuality.ReleasesAnalyzed, len(data.Tags))
		}
	}

	// ── JSON round-trip ──
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	var roundTripped MetricsReport
	if err := json.Unmarshal(b, &roundTripped); err != nil {
		t.Fatalf("JSON unmarshal failed: %v", err)
	}

	if roundTripped.Version != "1.0.0" {
		t.Errorf("round-trip version = %q, want 1.0.0", roundTripped.Version)
	}
	if roundTripped.CommitsAnalyzed != len(filtered.Commits) {
		t.Errorf("round-trip commits_analyzed = %d, want %d", roundTripped.CommitsAnalyzed, len(filtered.Commits))
	}
	if roundTripped.Hygiene == nil {
		t.Error("round-trip hygiene lost")
	}
	if roundTripped.KnowledgeRisk == nil {
		t.Error("round-trip knowledge_risk lost")
	}
}

func runIG(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func commitDated(t *testing.T, dir, path, content, date, msg string) {
	t.Helper()
	fullPath := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	runIG(t, dir, "add", path)
	env := append(os.Environ(),
		"GIT_AUTHOR_DATE="+date,
		"GIT_COMMITTER_DATE="+date,
	)
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	cmd.Env = env
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("commit: %v\n%s", err, out)
	}
}
