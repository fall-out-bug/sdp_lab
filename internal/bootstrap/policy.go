package bootstrap

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/main_rego.tmpl
var regoTemplateFS embed.FS

// PolicyInput holds the data for rendering the main.rego policy template.
type PolicyInput struct {
	// AnalysisSource describes where the policy data came from (e.g., "scout.json + architect/report.json").
	AnalysisSource string
	// SensitivePaths lists paths that should never be auto-edited without approval.
	SensitivePaths []string
	// GeneratedPaths lists paths whose contents are auto-generated and should not be hand-edited.
	GeneratedPaths []string
	// TestRequiredDirs is a Rego set literal of directories that require tests.
	TestRequiredDirs string
	// MaxFileLOC is the maximum allowed lines of code for new files.
	MaxFileLOC int
	// CommitPattern is the regex pattern for commit messages.
	CommitPattern string
}

// DetectSensitivePaths scans the repository for files that match known sensitive
// patterns. It does not read file contents -- only checks file names and paths.
//
// Detection algorithm:
//  1. Scan for known sensitive file-name patterns (.env*, credentials.*, etc.)
//  2. Check for CODEOWNERS-protected paths
//  3. Check for files with high coordination cost (>3 contributors is a heuristic;
//     this implementation skips that since we do not have git history in a
//     filesystem scan, and relies on CODEOWNERS and pattern matching instead)
func DetectSensitivePaths(repoPath string) []string {
	var paths []string
	seen := make(map[string]bool)

	// Patterns that indicate sensitive files.
	sensitivePatterns := []string{
		".env*",
		"credentials.*",
		"secrets.*",
		"*.pem",
		"*.key",
	}

	// Directory prefixes that indicate sensitive areas.
	sensitivePrefixes := []string{
		"config/production",
		"deploy/",
		"infrastructure/",
		"auth/",
		"security/",
		"crypto/",
	}

	// Walk the repo looking for sensitive file names.
	filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Skip hidden and vendor directories.
			name := d.Name()
			if strings.HasPrefix(name, ".") && name != "." && name != ".." {
				return filepath.SkipDir
			}
			if name == "vendor" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(repoPath, path)
		if err != nil {
			return nil
		}

		// Check file-name patterns.
		for _, pattern := range sensitivePatterns {
			matched, _ := filepath.Match(pattern, d.Name())
			if matched && !seen[rel] {
				paths = append(paths, rel)
				seen[rel] = true
				break
			}
		}

		// Check directory prefix patterns.
		for _, prefix := range sensitivePrefixes {
			if strings.HasPrefix(rel, prefix) && !seen[rel] {
				paths = append(paths, rel)
				seen[rel] = true
				break
			}
		}

		return nil
	})

	// Add CODEOWNERS-protected paths if a CODEOWNERS file exists.
	codeownersPaths := parseCODEOWNERS(repoPath)
	for _, p := range codeownersPaths {
		if !seen[p] {
			paths = append(paths, p)
			seen[p] = true
		}
	}

	return paths
}

// DetectGeneratedPaths scans for directories and files that are typically
// auto-generated and should not be hand-edited.
func DetectGeneratedPaths(repoPath string) []string {
	var paths []string
	seen := make(map[string]bool)

	generatedIndicators := []struct {
		path    string
		pattern string
	}{
		{".sdp/policies", "*.rego"},
		{".sdp/metrics", "*.json"},
		{".sdp/architect", "*.json"},
		{".sdp/index.db", ""},
	}

	for _, gi := range generatedIndicators {
		fullPath := filepath.Join(repoPath, gi.path)
		if pathExists(fullPath) {
			rel := gi.path
			if !seen[rel] {
				paths = append(paths, rel)
				seen[rel] = true
			}
		}
	}

	return paths
}

// BuildPolicyInput constructs a PolicyInput from the collected data sources
// and filesystem scan results.
func BuildPolicyInput(ds *DataSourceInfo, repoPath string, cmds BuildCommands) *PolicyInput {
	// Determine analysis source.
	sources := []string{}
	if ds.Scout != nil {
		sources = append(sources, "scout.json")
	}
	if ds.Architect != nil {
		sources = append(sources, "architect/report.json")
	}
	if ds.Metrics != nil {
		sources = append(sources, "metrics/report.json")
	}
	analysisSource := "unknown"
	if len(sources) > 0 {
		analysisSource = strings.Join(sources, " + ")
	}

	// Detect sensitive and generated paths.
	sensitivePaths := DetectSensitivePaths(repoPath)
	generatedPaths := DetectGeneratedPaths(repoPath)

	// Build test-required dirs set literal.
	testRequiredDirs := buildTestRequiredDirsRego(ds)

	// Determine commit pattern.
	commitPattern := `.+` // default: any non-empty message
	if ds.Scout != nil && hasConventionalCommits(repoPath) {
		commitPattern = `^(feat|fix|chore|refactor|docs|test|ci|build|perf|style)(\(.+\))?: .+`
	}

	// Max file LOC.
	maxLOC := 500
	if ds.Metrics != nil && ds.Metrics.ComplexityHint == "high" {
		maxLOC = 300
	}

	return &PolicyInput{
		AnalysisSource:   analysisSource,
		SensitivePaths:   sensitivePaths,
		GeneratedPaths:   generatedPaths,
		TestRequiredDirs: testRequiredDirs,
		MaxFileLOC:       maxLOC,
		CommitPattern:    commitPattern,
	}
}

// GeneratePolicy renders the main.rego policy from the embedded template.
// Returns the rendered content as a string.
func GeneratePolicy(input *PolicyInput) (string, error) {
	tmplData, err := regoTemplateFS.ReadFile("templates/main_rego.tmpl")
	if err != nil {
		return "", fmt.Errorf("policy: reading embedded template: %w", err)
	}

	tmpl, err := template.New("main.rego").Parse(string(tmplData))
	if err != nil {
		return "", fmt.Errorf("policy: parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, input); err != nil {
		return "", fmt.Errorf("policy: executing template: %w", err)
	}

	return buf.String(), nil
}

// GeneratePolicyToDir generates the main.rego policy and writes it to the
// specified directory. Creates the directory if it does not exist.
func GeneratePolicyToDir(input *PolicyInput, dirPath string) error {
	content, err := GeneratePolicy(input)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("policy: creating directory %s: %w", dirPath, err)
	}

	policyPath := filepath.Join(dirPath, "main.rego")
	if err := os.WriteFile(policyPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("policy: writing %s: %w", policyPath, err)
	}

	return nil
}

// hasConventionalCommits detects whether the repo uses Conventional Commits
// by checking for convention indicators in commit history or config files.
func hasConventionalCommits(repoPath string) bool {
	// Check for commitlint config files.
	conventionFiles := []string{
		".commitlintrc",
		".commitlintrc.json",
		".commitlintrc.yaml",
		".commitlintrc.yml",
		"commitlint.config.js",
		"commitlint.config.cjs",
		".versionrc",
	}
	for _, f := range conventionFiles {
		if pathExists(filepath.Join(repoPath, f)) {
			return true
		}
	}

	// Check for conventional-changelog or standard-version config.
	if pathExists(filepath.Join(repoPath, ".versionrc.json")) {
		return true
	}

	return false
}

// buildTestRequiredDirsRego produces a Rego set literal string for directories
// that require tests, based on architect component data.
func buildTestRequiredDirsRego(ds *DataSourceInfo) string {
	if ds.Architect == nil || len(ds.Architect.Components) == 0 {
		return `set()`
	}

	var parts []string
	for _, comp := range ds.Architect.Components {
		// Convert component paths to directory format.
		parts = append(parts, fmt.Sprintf("%q", comp))
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

// parseCODEOWNERS reads a CODEOWNERS file (if present) and returns the
// paths that are owner-protected. Returns nil if no CODEOWNERS file exists.
func parseCODEOWNERS(repoPath string) []string {
	candidates := []string{
		"CODEOWNERS",
		".github/CODEOWNERS",
		"docs/CODEOWNERS",
	}

	for _, candidate := range candidates {
		path := filepath.Join(repoPath, candidate)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		return extractCODEOWNERSPaths(string(data))
	}
	return nil
}

// extractCODEOWNERSPaths parses the paths from a CODEOWNERS file.
// Each non-comment, non-empty line has a pattern followed by owner(s).
func extractCODEOWNERSPaths(content string) []string {
	var paths []string
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// First field is the path pattern.
		fields := strings.Fields(line)
		if len(fields) > 0 {
			paths = append(paths, fields[0])
		}
	}
	return paths
}
