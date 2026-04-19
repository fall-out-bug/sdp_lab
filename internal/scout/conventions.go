package scout

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// extractConventions runs Phase 5: detects code patterns and conventions.
func extractConventions(root string, identity Identity, scale Scale) Conventions {
	var conv Conventions
	conv.ModulePatterns = detectModulePatterns(root)
	conv.TestStructure = detectTestLayout(root)
	conv.LintConfig = detectLintConfig(root)
	conv.CIWorkflow = detectCIWorkflow(root)
	return conv
}

// goModuleDirs maps Go module directory names to human-readable pattern names.
var goModuleDirs = map[string]string{
	"cmd":      "CLI entry points",
	"internal": "private packages",
	"pkg":      "public packages",
	"api":      "API definitions",
	"web":      "web assets",
}

// detectModulePatterns scans for conventional Go directory layouts.
func detectModulePatterns(root string) []ModulePattern {
	var patterns []ModulePattern

	entries, err := os.ReadDir(root)
	if err != nil {
		return patterns
	}

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		_, ok := goModuleDirs[name]
		if !ok {
			continue
		}
		pattern := name + "/"
		examples := collectSubDirNames(root, name)
		patterns = append(patterns, ModulePattern{
			Name:     name,
			Pattern:  pattern,
			Examples: examples,
		})
	}

	return patterns
}

// collectSubDirNames returns the immediate subdirectory or file names under
// root/name, up to 5 entries, using slash-separated relative paths.
func collectSubDirNames(root, name string) []string {
	dir := filepath.Join(root, name)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		rel := filepath.Join(name, e.Name())
		names = append(names, filepath.ToSlash(rel))
		if len(names) >= 5 {
			break
		}
	}
	return names
}

// skipDirNames are directories that detectTestLayout skips entirely.
var skipDirNames = map[string]bool{
	"vendor": true, "node_modules": true, ".git": true, ".claude": true,
	"archive": true, "dist": true, "build": true, "out": true,
	".worktrees": true, "testdata": true, "__pycache__": true,
	".tox": true, ".venv": true, "venv": true, ".mypy_cache": true,
	".next": true, ".nuxt": true, "coverage": true, ".cache": true,
}

// detectTestLayout determines whether tests are colocated, in a dedicated
// test directory, mixed, or unknown. Walks up to depth 5 and samples up to
// 200 files for performance. Exits early once both colocated and testdir
// patterns are detected (no need to keep walking).
func detectTestLayout(root string) TestLayout {
	var hasColocated bool
	var hasTestDir bool
	var testDirPattern string
	var visited int

	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}

		// Limit walk depth to 5 levels for performance.
		if depth := strings.Count(rel, string(filepath.Separator)) + 1; depth > 5 {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()
		if d.IsDir() {
			if skipDirNames[name] {
				return filepath.SkipDir
			}
			if isTestDirectoryName(name) {
				hasTestDir = true
				testDirPattern = name + "/"
			}
			// Early exit: both patterns found, no need to walk further.
			if hasColocated && hasTestDir {
				return filepath.SkipDir
			}
			return nil
		}

		visited++
		if visited > 200 {
			return filepath.SkipDir
		}

		// Only check for colocated test files if we haven't found one yet.
		if !hasColocated && strings.HasSuffix(name, "_test.go") {
			dir := filepath.Dir(rel)
			// Skip if the directory itself is in the skip list.
			dirBase := filepath.Base(dir)
			if skipDirNames[dirBase] {
				return nil
			}
			baseName := strings.TrimSuffix(name, "_test.go") + ".go"
			if _, statErr := os.Stat(filepath.Join(root, dir, baseName)); statErr == nil {
				hasColocated = true
			}
		}

		// Early exit: both patterns found.
		if hasColocated && hasTestDir {
			return filepath.SkipDir
		}
		return nil
	})

	switch {
	case hasColocated && hasTestDir:
		return TestLayout{Style: "mixed", DirPattern: testDirPattern}
	case hasColocated:
		return TestLayout{Style: "colocated", DirPattern: "*_test.go"}
	case hasTestDir:
		return TestLayout{Style: "testdir", DirPattern: testDirPattern}
	default:
		return TestLayout{Style: "unknown"}
	}
}

// isTestDirectoryName reports whether a directory name indicates a test directory.
func isTestDirectoryName(name string) bool {
	lower := strings.ToLower(name)
	return lower == "test" || lower == "tests" || lower == "testdata" ||
		lower == "__tests__" || lower == "spec"
}

// lintConfigFiles maps lint config filenames to tool identifiers.
var lintConfigFiles = []struct {
	file string
	tool string
}{
	{".golangci.yml", "golangci-lint"},
	{".golangci.yaml", "golangci-lint"},
	{".eslintrc.js", "eslint"},
	{".eslintrc.json", "eslint"},
	{".eslintrc.yml", "eslint"},
	{".flake8", "flake8"},
	{".pylintrc", "pylint"},
	{"ruff.toml", "ruff"},
	{".rubocop.yml", "rubocop"},
}

// detectLintConfig discovers linting tool configuration.
func detectLintConfig(root string) *LintInfo {
	for _, lc := range lintConfigFiles {
		path := filepath.Join(root, lc.file)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		info := &LintInfo{
			Tool:       lc.tool,
			ConfigFile: lc.file,
		}
		info.Rules = parseLintRules(path, lc.tool)
		return info
	}
	return nil
}

// parseLintRules reads a lint config file and extracts enabled rule names.
func parseLintRules(path, tool string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var rules []string
	switch {
	case strings.HasPrefix(tool, "golangci"):
		rules = parseGolangciLinters(data)
	case tool == "eslint":
		rules = parseEslintRules(data)
	}
	return rules
}

// parseGolangciLinters extracts enabled linter names from golangci config.
func parseGolangciLinters(data []byte) []string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inEnable := false
	var rules []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.Contains(line, "enable:") {
			inEnable = true
			continue
		}
		if inEnable {
			if strings.HasPrefix(line, "- ") {
				rules = append(rules, strings.TrimPrefix(line, "- "))
			} else if line != "" && !strings.HasPrefix(line, "#") {
				break
			}
		}
	}
	return rules
}

// parseEslintRules extracts rule names from eslint config.
func parseEslintRules(data []byte) []string {
	content := string(data)
	var rules []string
	// Simple extraction: find "rules": { block and collect keys
	idx := strings.Index(content, "\"rules\"")
	if idx < 0 {
		return rules
	}
	block := content[idx:]
	depth := 0
	inBlock := false
	for i, ch := range block {
		if ch == '{' {
			depth++
			inBlock = true
		} else if ch == '}' {
			depth--
			if inBlock && depth == 0 {
				break
			}
		} else if inBlock && depth == 1 && ch == '"' {
			// Find the closing quote
			end := strings.IndexByte(block[i+1:], '"')
			if end > 0 {
				rule := block[i+1 : i+1+end]
				rules = append(rules, rule)
			}
		}
	}
	return rules
}

// ciConfigPaths maps CI config paths to system identifiers.
var ciConfigPaths = []struct {
	path   string
	system string
}{
	{".github/workflows", "github-actions"},
	{".gitlab-ci.yml", "gitlab-ci"},
	{"Jenkinsfile", "jenkins"},
	{".circleci", "circleci"},
	{".travis.yml", "travis"},
}

// detectCIWorkflow discovers CI/CD configuration.
func detectCIWorkflow(root string) *CIInfo {
	for _, ci := range ciConfigPaths {
		fullPath := filepath.Join(root, ci.path)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			// For directory-based configs (GitHub Actions), read YAML files
			yamlFiles := findYAMLFiles(fullPath)
			if len(yamlFiles) == 0 {
				continue
			}
			first := yamlFiles[0]
			relPath := filepath.Join(ci.path, first)
			ciInfo := &CIInfo{
				System:     ci.system,
				ConfigFile: filepath.ToSlash(relPath),
			}
			ciInfo.Steps = parseCISteps(filepath.Join(fullPath, first))
			return ciInfo
		}

		// File-based configs
		ciInfo := &CIInfo{
			System:     ci.system,
			ConfigFile: ci.path,
		}
		ciInfo.Steps = parseCISteps(fullPath)
		return ciInfo
	}
	return nil
}

// findYAMLFiles returns .yml/.yaml files in a directory, up to 10.
func findYAMLFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".yml" || ext == ".yaml" {
			files = append(files, e.Name())
			if len(files) >= 10 {
				break
			}
		}
	}
	return files
}

// parseCISteps extracts step descriptions from a CI config file.
func parseCISteps(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var steps []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "- run:") || strings.HasPrefix(line, "- uses:") {
			step := strings.TrimSpace(strings.TrimPrefix(line, "- run:"))
			if strings.HasPrefix(line, "- uses:") {
				step = strings.TrimSpace(strings.TrimPrefix(line, "- uses:"))
			}
			if step != "" {
				steps = append(steps, step)
			}
		}
	}
	return steps
}
