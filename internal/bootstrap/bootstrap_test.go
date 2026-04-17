package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupRepoWithScout creates a temp dir with a minimal .sdp/scout.json.
func setupRepoWithScout(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(sdpPath, 0o755))
	scout := ScoutData{
		PrimaryLanguage: "Go",
		BuildSystem:     "go",
		HasTests:        true,
		TestRatio:       0.4,
		TotalFiles:      50,
	}
	writeJSON(t, filepath.Join(sdpPath, "scout.json"), scout)
	return dir
}

func TestPlanner_Plan_Minimal(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{RepoPath: dir})
	plan, err := planner.Plan()
	require.NoError(t, err)

	// Should plan to create CLAUDE.md, AGENTS.md, policies, hooks
	assert.NotEmpty(t, plan.WillCreate)
	assert.Empty(t, plan.WillSkip)
	assert.Empty(t, plan.WillMerge)
	assert.NotNil(t, plan.DataSources.Scout)
}

func TestPlanner_Plan_WithExistingCLAUDEMd(t *testing.T) {
	dir := setupRepoWithScout(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# existing"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir})
	plan, err := planner.Plan()
	require.NoError(t, err)

	// CLAUDE.md should be in WillSkip since it already exists.
	var skipTypes []string
	for _, a := range plan.WillSkip {
		skipTypes = append(skipTypes, a.Type)
	}
	assert.Contains(t, skipTypes, "claude_md")

	// Other artifacts should still be created.
	var createTypes []string
	for _, a := range plan.WillCreate {
		createTypes = append(createTypes, a.Type)
	}
	assert.Contains(t, createTypes, "agents_md")
}

func TestPlanner_Plan_ForceMerge(t *testing.T) {
	dir := setupRepoWithScout(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# existing"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, Force: true})
	plan, err := planner.Plan()
	require.NoError(t, err)

	// CLAUDE.md should be in WillMerge because of --force.
	var mergeTypes []string
	for _, a := range plan.WillMerge {
		mergeTypes = append(mergeTypes, a.Type)
	}
	assert.Contains(t, mergeTypes, "claude_md")
}

func TestPlanner_Plan_OnlyFilter(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{
		RepoPath: dir,
		Only:     []string{"hooks"},
	})
	plan, err := planner.Plan()
	require.NoError(t, err)

	// Should only plan hooks.
	var allTypes []string
	for _, a := range plan.WillCreate {
		allTypes = append(allTypes, a.Type)
	}
	for _, a := range plan.WillSkip {
		allTypes = append(allTypes, a.Type)
	}
	for _, a := range plan.WillMerge {
		allTypes = append(allTypes, a.Type)
	}
	// Only hook type should appear.
	for _, typ := range allTypes {
		assert.Equal(t, "hook", typ)
	}
}

func TestPlanner_Plan_NoVerify(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{
		RepoPath: dir,
		NoVerify: true,
	})
	plan, err := planner.Plan()
	require.NoError(t, err)

	// NoVerify skips verification execution, not command detection.
	// Commands are still detected for plan reporting.
	assert.NotEmpty(t, plan.Commands.Test, "command detection should still run")
}

func TestPlanner_Plan_WithVerify(t *testing.T) {
	dir := setupRepoWithScout(t)
	// Add Makefile
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"),
		[]byte("build:\n\tgo build ./...\ntest:\n\tgo test ./...\nlint:\n\tgolangci-lint run\n"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir})
	plan, err := planner.Plan()
	require.NoError(t, err)

	assert.Equal(t, "make build", plan.Commands.Build)
	assert.Equal(t, "make test", plan.Commands.Test)
	assert.Equal(t, "make lint", plan.Commands.Lint)
}

func TestPlanner_Plan_BeadsOptIn(t *testing.T) {
	dir := setupRepoWithScout(t)

	// Default (Beads: false) should NOT plan beads.
	planner := NewPlanner(BootstrapConfig{
		RepoPath: dir,
	})
	plan, err := planner.Plan()
	require.NoError(t, err)

	var types []string
	for _, a := range plan.WillCreate {
		types = append(types, a.Type)
	}
	assert.NotContains(t, types, "beads")

	// With Beads: true, beads should be planned.
	planner2 := NewPlanner(BootstrapConfig{
		RepoPath: dir,
		Beads:    true,
	})
	plan2, err := planner2.Plan()
	require.NoError(t, err)

	var types2 []string
	for _, a := range plan2.WillCreate {
		types2 = append(types2, a.Type)
	}
	assert.Contains(t, types2, "beads")
}

func TestPlanner_DryRun(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{
		RepoPath: dir,
		DryRun:   true,
	})
	report, err := planner.DryRun()
	require.NoError(t, err)

	assert.Equal(t, version, report.Version)
	assert.NotZero(t, report.GeneratedAt)
	assert.NotEmpty(t, report.Artifacts)
	assert.NotEmpty(t, report.DataSources)
	assert.NotEmpty(t, report.Confidence)

	// All artifacts should have dry_run status
	for _, a := range report.Artifacts {
		if a.Action == "skip" {
			assert.Equal(t, "skipped", a.Status)
		} else {
			assert.Equal(t, "dry_run", a.Status)
		}
	}

	// No files should have been created.
	_, err = os.Stat(filepath.Join(dir, "CLAUDE.md"))
	assert.True(t, os.IsNotExist(err))
}

func TestPlanner_Execute(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	assert.Equal(t, version, report.Version)
	assert.NotEmpty(t, report.Artifacts)

	// Files should have been created.
	assert.FileExists(t, filepath.Join(dir, "CLAUDE.md"))
	assert.FileExists(t, filepath.Join(dir, "AGENTS.md"))
	// Beads is opt-in; default bootstrap should NOT create .beads.
	assert.NoDirExists(t, filepath.Join(dir, ".beads"))

	// Check artifact statuses.
	var okCount, skipCount int
	for _, a := range report.Artifacts {
		switch a.Status {
		case "ok":
			okCount++
		case "skipped":
			skipCount++
		}
	}
	assert.Positive(t, okCount)
}

func TestPlanner_Execute_NoOverwriteWithoutForce(t *testing.T) {
	dir := setupRepoWithScout(t)
	existingContent := "# original content"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(existingContent), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	_, err := planner.Execute()
	require.NoError(t, err)

	// CLAUDE.md should not have been overwritten.
	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Equal(t, existingContent, string(data))
}

func TestPlanner_Execute_ForceOverwrites(t *testing.T) {
	dir := setupRepoWithScout(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# original"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, Force: true, NoVerify: true})
	_, err := planner.Execute()
	require.NoError(t, err)

	// CLAUDE.md should have been overwritten.
	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "generated by sdp bootstrap")
}

func TestPlanner_Status(t *testing.T) {
	dir := setupRepoWithScout(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# claude"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("# agents"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir})
	status, err := planner.Status()
	require.NoError(t, err)

	assert.Equal(t, dir, status.RepoPath)
	assert.True(t, status.Bootstrapped)
	assert.Contains(t, status.ExistingFiles, "CLAUDE.md")
	assert.Contains(t, status.ExistingFiles, "AGENTS.md")
	assert.True(t, status.DataSources["scout"])
	assert.False(t, status.DataSources["architect"])
}

func TestPlanner_Status_NotBootstrapped(t *testing.T) {
	dir := setupRepoWithScout(t)

	planner := NewPlanner(BootstrapConfig{RepoPath: dir})
	status, err := planner.Status()
	require.NoError(t, err)

	assert.False(t, status.Bootstrapped)
	assert.NotEmpty(t, status.MissingFiles)
	assert.NotEmpty(t, status.Suggestions)
}

func TestComputeConfidence(t *testing.T) {
	plan := &BootstrapPlan{
		DataSources: DataSourceInfo{
			Scout:     &ScoutData{PrimaryLanguage: "Go"},
			Architect: &ArchitectData{Components: []string{"a"}},
			Metrics:   &MetricsData{BusFactor: 3},
			Spec:      &SpecData{Files: []SpecFile{{Name: "spec.md"}}},
			Index:     &IndexData{Symbols: 100},
		},
	}
	conf := computeConfidence(plan)

	assert.Equal(t, 1.0, conf["overall"])
	assert.Equal(t, 1.0, conf["scout"])
	assert.InDelta(t, 0.9, conf["architect"], 0.01)
}

func TestComputeConfidence_ScoutOnly(t *testing.T) {
	plan := &BootstrapPlan{
		DataSources: DataSourceInfo{
			Scout: &ScoutData{PrimaryLanguage: "Go"},
		},
	}
	conf := computeConfidence(plan)

	assert.InDelta(t, 0.6, conf["overall"], 0.01)
	assert.Equal(t, 1.0, conf["scout"])
}

func TestGenerateNotes(t *testing.T) {
	plan := &BootstrapPlan{
		WillCreate: []PlannedArtifact{{Type: "claude_md", Path: "CLAUDE.md", Action: "create", Description: "test"}},
		DataSources: DataSourceInfo{
			Scout: &ScoutData{PrimaryLanguage: "Go", BuildSystem: "go"},
		},
	}
	notes := generateNotes(plan)
	assert.NotEmpty(t, notes)

	var foundLang, foundCreate bool
	for _, n := range notes {
		if n == "Primary language: Go" {
			foundLang = true
		}
		if n == "Will create 1 new artifact(s)" {
			foundCreate = true
		}
	}
	assert.True(t, foundLang)
	assert.True(t, foundCreate)
}

func TestFormatPlanText(t *testing.T) {
	plan := &BootstrapPlan{
		DataSources: DataSourceInfo{
			Scout: &ScoutData{PrimaryLanguage: "Go", BuildSystem: "go"},
		},
		Commands: BuildCommands{Build: "go build ./...", Test: "go test ./...", Lint: "golangci-lint run"},
		WillCreate: []PlannedArtifact{
			{Type: "claude_md", Path: "CLAUDE.md", Action: "create", Description: "config"},
		},
	}
	text := FormatPlanText(plan)
	assert.Contains(t, text, "Bootstrap Plan")
	assert.Contains(t, text, "Will Create:")
	assert.Contains(t, text, "CLAUDE.md")
	assert.Contains(t, text, "go build")
}

func TestFormatStatusText(t *testing.T) {
	status := &BootstrapStatus{
		RepoPath:      "/tmp/repo",
		Bootstrapped:  false,
		ExistingFiles: []string{"CLAUDE.md"},
		MissingFiles:  []string{"AGENTS.md"},
		DataSources:   map[string]bool{"scout": true, "architect": false},
		Suggestions:   []string{"Run sdp bootstrap"},
	}
	text := FormatStatusText(status)
	assert.Contains(t, text, "Bootstrap Status")
	assert.Contains(t, text, "CLAUDE.md")
	assert.Contains(t, text, "AGENTS.md")
	assert.Contains(t, text, "Suggestions")
}

func TestFormatReportJSON(t *testing.T) {
	report := &BootstrapReport{
		Version:     version,
		GeneratedAt: time.Now().UTC(),
		Repo:        "/tmp/repo",
		Artifacts:   []ArtifactResult{{Type: "claude_md", Path: "CLAUDE.md", Status: "ok"}},
		DataSources: map[string]bool{"scout": true},
		Confidence:  map[string]float64{"overall": 0.6},
	}
	out, err := FormatReportJSON(report)
	require.NoError(t, err)
	assert.Contains(t, out, `"version"`)
	assert.Contains(t, out, `"claude_md"`)

	// Verify it's valid JSON.
	var parsed BootstrapReport
	require.NoError(t, json.Unmarshal([]byte(out), &parsed))
	assert.Equal(t, report.Version, parsed.Version)
}
