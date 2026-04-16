package bootstrap

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSensitivePaths_WithEnvFiles(t *testing.T) {
	dir := t.TempDir()

	// Create sensitive files.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env"), []byte("KEY=val"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.production"), []byte("KEY=val"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets.yaml"), []byte("secret: value"), 0o644))

	// Create a non-sensitive file.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644))

	paths := DetectSensitivePaths(dir)

	assert.Contains(t, paths, ".env")
	assert.Contains(t, paths, ".env.production")
	assert.Contains(t, paths, "secrets.yaml")
	assert.NotContains(t, paths, "main.go")
}

func TestDetectSensitivePaths_WithKeyFiles(t *testing.T) {
	dir := t.TempDir()

	require.NoError(t, os.WriteFile(filepath.Join(dir, "server.key"), []byte("key"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cert.pem"), []byte("cert"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("{}"), 0o644))

	paths := DetectSensitivePaths(dir)

	assert.Contains(t, paths, "server.key")
	assert.Contains(t, paths, "cert.pem")
	assert.Contains(t, paths, "credentials.json")
}

func TestDetectSensitivePaths_WithSensitiveDirs(t *testing.T) {
	dir := t.TempDir()

	// Create files in sensitive directories.
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "config", "production"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "production", "app.yaml"), []byte("app"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "deploy"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "deploy", "manifest.yaml"), []byte("deploy"), 0o644))

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "auth"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "auth", "handler.go"), []byte("auth"), 0o644))

	paths := DetectSensitivePaths(dir)

	assert.Contains(t, paths, filepath.Join("config", "production", "app.yaml"))
	assert.Contains(t, paths, filepath.Join("deploy", "manifest.yaml"))
	assert.Contains(t, paths, filepath.Join("auth", "handler.go"))
}

func TestDetectSensitivePaths_SkipsVendorAndHidden(t *testing.T) {
	dir := t.TempDir()

	// Create sensitive files in vendor (should be skipped).
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "vendor", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "vendor", "pkg", ".env"), []byte("KEY=val"), 0o644))

	// Create sensitive files in node_modules (should be skipped).
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "node_modules", "pkg"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "node_modules", "pkg", "secrets.json"), []byte("{}"), 0o644))

	// Create a real sensitive file at root.
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".env.local"), []byte("KEY=val"), 0o644))

	paths := DetectSensitivePaths(dir)

	assert.NotContains(t, paths, filepath.Join("vendor", "pkg", ".env"))
	assert.NotContains(t, paths, filepath.Join("node_modules", "pkg", "secrets.json"))
	assert.Contains(t, paths, ".env.local")
}

func TestDetectSensitivePaths_WithCODEOWNERS(t *testing.T) {
	dir := t.TempDir()

	// Create a CODEOWNERS file.
	codeowners := `# Important files
/internal/core/ @senior-devs
/config/ @devops
*.pem @security-team
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "CODEOWNERS"), []byte(codeowners), 0o644))

	paths := DetectSensitivePaths(dir)

	assert.Contains(t, paths, "/internal/core/")
	assert.Contains(t, paths, "/config/")
	assert.Contains(t, paths, "*.pem")
}

func TestDetectSensitivePaths_EmptyRepo(t *testing.T) {
	dir := t.TempDir()
	paths := DetectSensitivePaths(dir)
	assert.Empty(t, paths)
}

func TestDetectGeneratedPaths_WithSDPData(t *testing.T) {
	dir := t.TempDir()

	sdpPath := filepath.Join(dir, ".sdp")
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "policies"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "metrics"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(sdpPath, "architect"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sdpPath, "index.db"), []byte("data"), 0o644))

	paths := DetectGeneratedPaths(dir)

	assert.Contains(t, paths, ".sdp/policies")
	assert.Contains(t, paths, ".sdp/metrics")
	assert.Contains(t, paths, ".sdp/architect")
	assert.Contains(t, paths, ".sdp/index.db")
}

func TestDetectGeneratedPaths_NoSDPData(t *testing.T) {
	dir := t.TempDir()
	paths := DetectGeneratedPaths(dir)
	assert.Empty(t, paths)
}

func TestBuildPolicyInput_WithAllSources(t *testing.T) {
	dir := setupRepoWithScout(t)

	ds := &DataSourceInfo{
		Scout:     &ScoutData{PrimaryLanguage: "Go", BuildSystem: "go"},
		Architect: &ArchitectData{Components: []string{"cmd/sdp", "internal/scout"}},
		Metrics:   &MetricsData{BusFactor: 3, ComplexityHint: "high"},
	}
	cmds := BuildCommands{Build: "go build ./...", Test: "go test ./...", Lint: "golangci-lint run"}

	input := BuildPolicyInput(ds, dir, cmds)

	assert.Equal(t, "scout.json + architect/report.json + metrics/report.json", input.AnalysisSource)
	assert.Equal(t, 300, input.MaxFileLOC) // high complexity -> 300
	assert.Contains(t, input.TestRequiredDirs, "cmd/sdp")
	assert.Contains(t, input.TestRequiredDirs, "internal/scout")
}

func TestBuildPolicyInput_ScoutOnly(t *testing.T) {
	dir := t.TempDir()

	ds := &DataSourceInfo{
		Scout: &ScoutData{PrimaryLanguage: "Go"},
	}
	cmds := BuildCommands{Build: "go build ./...", Test: "go test ./...", Lint: "golangci-lint run"}

	input := BuildPolicyInput(ds, dir, cmds)

	assert.Equal(t, "scout.json", input.AnalysisSource)
	assert.Equal(t, 500, input.MaxFileLOC) // default
	assert.Equal(t, `set()`, input.TestRequiredDirs) // no architect data
}

func TestBuildPolicyInput_NoSources(t *testing.T) {
	dir := t.TempDir()

	ds := &DataSourceInfo{}
	cmds := BuildCommands{}

	input := BuildPolicyInput(ds, dir, cmds)

	assert.Equal(t, "unknown", input.AnalysisSource)
	assert.Equal(t, 500, input.MaxFileLOC)
	assert.Equal(t, `.+`, input.CommitPattern) // default: any message
}

func TestGeneratePolicy_Basic(t *testing.T) {
	input := &PolicyInput{
		AnalysisSource:   "scout.json",
		SensitivePaths:   []string{".env", "secrets.yaml"},
		GeneratedPaths:   []string{".sdp/policies"},
		TestRequiredDirs: `{"cmd/sdp", "internal/scout"}`,
		MaxFileLOC:       500,
		CommitPattern:    `.+`,
	}

	content, err := GeneratePolicy(input)
	require.NoError(t, err)

	assert.Contains(t, content, "package sdp.policy")
	assert.Contains(t, content, "scout.json")
	assert.Contains(t, content, ".env")
	assert.Contains(t, content, "secrets.yaml")
	assert.Contains(t, content, ".sdp/policies")
	assert.Contains(t, content, "500")
}

func TestGeneratePolicy_NoSensitivePaths(t *testing.T) {
	input := &PolicyInput{
		AnalysisSource:   "scout.json",
		SensitivePaths:   nil,
		GeneratedPaths:   nil,
		TestRequiredDirs: `set()`,
		MaxFileLOC:       500,
		CommitPattern:    `.+`,
	}

	content, err := GeneratePolicy(input)
	require.NoError(t, err)

	assert.Contains(t, content, "package sdp.policy")
	assert.Contains(t, content, "sensitive_paths :=")
	assert.Contains(t, content, "generated_paths :=")
}

func TestGeneratePolicy_ConventionalCommits(t *testing.T) {
	input := &PolicyInput{
		AnalysisSource:   "scout.json + architect/report.json",
		SensitivePaths:   []string{"credentials.json"},
		GeneratedPaths:   []string{".sdp/index.db"},
		TestRequiredDirs: `{"cmd/sdp"}`,
		MaxFileLOC:       300,
		CommitPattern:    `^(feat|fix|chore|refactor|docs|test|ci|build|perf|style)(\(.+\))?: .+`,
	}

	content, err := GeneratePolicy(input)
	require.NoError(t, err)

	assert.Contains(t, content, `^(feat|fix|chore|refactor|docs|test|ci|build|perf|style)(\(.+\))?: .+`)
	assert.Contains(t, content, "300")
	assert.Contains(t, content, "credentials.json")
}

func TestGeneratePolicyToDir(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, ".sdp", "policies")

	input := &PolicyInput{
		AnalysisSource:   "scout.json",
		SensitivePaths:   []string{".env"},
		GeneratedPaths:   []string{},
		TestRequiredDirs: `set()`,
		MaxFileLOC:       500,
		CommitPattern:    `.+`,
	}

	err := GeneratePolicyToDir(input, policyDir)
	require.NoError(t, err)

	// Should have created main.rego
	repoPath := filepath.Join(policyDir, "main.rego")
	assert.FileExists(t, repoPath)

	data, err := os.ReadFile(repoPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "package sdp.policy")
}

func TestGeneratePolicyToDir_NestedPath(t *testing.T) {
	dir := t.TempDir()
	policyDir := filepath.Join(dir, "deep", "nested", "policies")

	input := &PolicyInput{
		AnalysisSource:   "scout.json",
		SensitivePaths:   nil,
		GeneratedPaths:   nil,
		TestRequiredDirs: `set()`,
		MaxFileLOC:       500,
		CommitPattern:    `.+`,
	}

	err := GeneratePolicyToDir(input, policyDir)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(policyDir, "main.rego"))
}

func TestHasConventionalCommits_WithCommitlint(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".commitlintrc.json"), []byte(`{"extends": ["@commitlint/config-conventional"]}`), 0o644))
	assert.True(t, hasConventionalCommits(dir))
}

func TestHasConventionalCommits_WithVersionrc(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, ".versionrc"), []byte(`{}`), 0o644))
	assert.True(t, hasConventionalCommits(dir))
}

func TestHasConventionalCommits_NotConfigured(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, hasConventionalCommits(dir))
}

func TestBuildTestRequiredDirsRego_WithComponents(t *testing.T) {
	ds := &DataSourceInfo{
		Architect: &ArchitectData{
			Components: []string{"cmd/sdp", "internal/scout", "internal/bootstrap"},
		},
	}
	result := buildTestRequiredDirsRego(ds)
	assert.Contains(t, result, `"cmd/sdp"`)
	assert.Contains(t, result, `"internal/scout"`)
	assert.Contains(t, result, `"internal/bootstrap"`)
}

func TestBuildTestRequiredDirsRego_NoArchitect(t *testing.T) {
	ds := &DataSourceInfo{}
	result := buildTestRequiredDirsRego(ds)
	assert.Equal(t, `set()`, result)
}

func TestExtractCODEOWNERSPaths(t *testing.T) {
	content := `# Owners file
/src/core/ @team-a
/docs/ @team-b
*.go @senior-devs

# More rules
/internal/ @team-c
`
	paths := extractCODEOWNERSPaths(content)
	assert.Equal(t, []string{"/src/core/", "/docs/", "*.go", "/internal/"}, paths)
}

func TestExtractCODEOWNERSPaths_Empty(t *testing.T) {
	paths := extractCODEOWNERSPaths("")
	assert.Empty(t, paths)
}

func TestExtractCODEOWNERSPaths_CommentsOnly(t *testing.T) {
	content := `# Just a comment
# Another comment
`
	paths := extractCODEOWNERSPaths(content)
	assert.Empty(t, paths)
}

func TestParseCODEOWNERS_NoFile(t *testing.T) {
	dir := t.TempDir()
	paths := parseCODEOWNERS(dir)
	assert.Nil(t, paths)
}

func TestParseCODEOWNERS_GitHubLocation(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".github"), 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(dir, ".github", "CODEOWNERS"),
		[]byte("/src/ @team\n"),
		0o644))

	paths := parseCODEOWNERS(dir)
	assert.Contains(t, paths, "/src/")
}

func TestPolicyGeneration_IntegratedWithPlanner(t *testing.T) {
	dir := setupRepoWithScout(t)
	// Add a Makefile for command detection.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Makefile"),
		[]byte("build:\n\tgo build ./...\ntest:\n\tgo test ./...\nlint:\n\tgolangci-lint run\n"), 0o644))

	planner := NewPlanner(BootstrapConfig{RepoPath: dir, NoVerify: true})
	report, err := planner.Execute()
	require.NoError(t, err)

	// Find the policy artifact result.
	var policyResult *ArtifactResult
	for i := range report.Artifacts {
		if report.Artifacts[i].Type == "policy" {
			policyResult = &report.Artifacts[i]
			break
		}
	}
	require.NotNil(t, policyResult, "policy artifact should be in report")
	assert.Equal(t, "ok", policyResult.Status)
	assert.Contains(t, policyResult.Message, "main.rego")

	// Verify the file was actually created.
	assert.FileExists(t, filepath.Join(dir, ".sdp", "policies", "main.rego"))

	data, err := os.ReadFile(filepath.Join(dir, ".sdp", "policies", "main.rego"))
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "package sdp.policy")
}

func TestPolicyGeneration_ScoutOnlyMode(t *testing.T) {
	dir := setupRepoWithScout(t)

	ds := &DataSourceInfo{
		Scout: &ScoutData{PrimaryLanguage: "Go"},
	}
	cmds := BuildCommands{Build: "go build ./...", Test: "go test ./...", Lint: "golangci-lint run"}

	input := BuildPolicyInput(ds, dir, cmds)
	content, err := GeneratePolicy(input)
	require.NoError(t, err)

	// Should work fine without architect data.
	assert.Contains(t, content, "package sdp.policy")
	assert.Contains(t, content, "scout.json")
	// TestRequiredDirs should be empty set.
	assert.Contains(t, content, "set()")
}
