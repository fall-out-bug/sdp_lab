package scout

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

// goldenPath points to the locked JSON fixture for the ProjectCard contract.
const goldenPath = "testdata/scout_golden.json"

func TestProjectCardJSONShape(t *testing.T) {
	desc := "A fast SDP toolkit"
	repo := "github.com/example/repo"
	bs := "go-modules"
	pm := "go-modules"
	df := "go.mod"
	lr := "v1.0.0"
	ci := "github-actions"
	first := "2024-06-15"
	last := "2026-04-13"

	card := ProjectCard{
		Version:   "1.0.0",
		ScannedAt: mustParseTime(t, "2026-04-13T15:00:00Z"),
		DurationMs: 2700,
		Identity: Identity{
			Name:            "example",
			Description:     &desc,
			RepoURL:         &repo,
			PrimaryLanguage: "go",
			Languages: map[string]LangStats{
				"go": {Files: 312, Ratio: 0.72},
			},
			BuildSystem: &bs,
			BuildFiles:  []string{"go.mod"},
			Monorepo:    false,
		},
		Scale: Scale{
			TotalFiles: 433, TotalLoc: 48200, SourceFiles: 312,
			TestFiles: 67, TestRatio: 0.18, GeneratedFiles: 12,
			VendorFiles: 0, MaxFileLoc: 1240, MedianFileLoc: 89,
			Directories: 47, DepthMax: 6,
		},
		Activity: Activity{
			FirstCommit: &first, LastCommit: &last,
			AgeMonths: 22, TotalCommits: 1247, Contributors: 8,
			ActiveContributors90d: 4, Commits30d: 87, Commits90d: 234,
			ActiveBranches: 5,
		},
		Maturity: Maturity{
			HasReadme: true, HasLicense: true, HasCI: true,
			CISystem: &ci, HasTests: true, HasLinter: true,
			HasDocker: true, HasReleases: true, LatestRelease: &lr,
			ReleaseCount: 14, HasCodeowners: false,
			HasContributing: true, HasChangelog: false,
		},
		Build: Build{
			EntryPoints:     []string{"cmd/app/main.go"},
			ConfigFiles:     []string{"go.mod"},
			PackageManager:  &pm,
			DependencyCount: 42,
			DependencyFile:  &df,
		},
		Health: HealthSignals{
			BusFactorEstimate: 3, CommitFrequency: CommitFreqHigh,
			Staleness: StalenessActive, TestCoverageHint: CovPartial,
			ComplexityHint: ComplexityMedium,
		},
	}

	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		t.Fatalf("marshal ProjectCard: %v", err)
	}

	// Lock the golden fixture — update only when contract changes intentionally.
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("mkdir testdata: %v", err)
		}
		if err := os.WriteFile(goldenPath, data, 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}
		t.Log("golden updated")
		return
	}

	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden fixture (run with UPDATE_GOLDEN=1 to create): %v", err)
	}

	if string(data) != string(golden) {
		t.Errorf("JSON shape diverges from golden fixture.\nGot:\n%s\nWant:\n%s", data, golden)
	}
}

func TestProjectCardNullFieldsOmitted(t *testing.T) {
	card := ProjectCard{
		Version:   "1.0.0",
		ScannedAt: mustParseTime(t, "2026-04-13T00:00:00Z"),
		// Description, RepoURL, BuildSystem, CISystem, LatestRelease,
		// PackageManager, DependencyFile are all nil → must serialize as null
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Nil pointer fields must be "null" not absent
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	identity := m["identity"].(map[string]any)
	for _, field := range []string{"description", "repo_url", "build_system"} {
		v, ok := identity[field]
		if !ok {
			t.Errorf("identity.%s: field missing (should be null)", field)
		} else if v != nil {
			t.Errorf("identity.%s: expected null, got %v", field, v)
		}
	}

	maturity := m["maturity"].(map[string]any)
	for _, field := range []string{"ci_system", "latest_release"} {
		v, ok := maturity[field]
		if !ok {
			t.Errorf("maturity.%s: field missing (should be null)", field)
		} else if v != nil {
			t.Errorf("maturity.%s: expected null, got %v", field, v)
		}
	}

	build := m["build"].(map[string]any)
	for _, field := range []string{"package_manager", "dependency_file"} {
		v, ok := build[field]
		if !ok {
			t.Errorf("build.%s: field missing (should be null)", field)
		} else if v != nil {
			t.Errorf("build.%s: expected null, got %v", field, v)
		}
	}
}

func TestHealthSignalsUseExplicitUnknown(t *testing.T) {
	card := ProjectCard{
		Health: HealthSignals{
			CommitFrequency:  Unknown,
			Staleness:        Unknown,
			TestCoverageHint: Unknown,
			ComplexityHint:   Unknown,
		},
	}

	data, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	hs := m["health_signals"].(map[string]any)
	for _, field := range []string{"commit_frequency", "staleness", "test_coverage_hint", "complexity_hint"} {
		v := hs[field].(string)
		if v != "unknown" {
			t.Errorf("health_signals.%s: expected 'unknown', got %q", field, v)
		}
	}
}

func mustParseTime(t *testing.T, s string) time.Time {
	t.Helper()
	tm, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse time %q: %v", s, err)
	}
	return tm
}
