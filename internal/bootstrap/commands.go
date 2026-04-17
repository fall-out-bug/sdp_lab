package bootstrap

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectBuildCommands determines the build, test, and lint commands for the
// repository based on its language and configuration files.
//
// Detection order:
//  1. Makefile targets (build, test, lint)
//  2. package.json scripts (build, test, lint)
//  3. CI config (GitHub Actions, GitLab CI)
//  4. Language-default fallback
func DetectBuildCommands(repoPath string, lang string) BuildCommands {
	// Check Makefile first — highest priority.
	if cmds, ok := detectFromMakefile(repoPath); ok {
		return cmds
	}

	// Check package.json for Node projects.
	if cmds, ok := detectFromPackageJSON(repoPath); ok {
		return cmds
	}

	// Check CI config.
	if cmds, ok := detectFromCI(repoPath); ok {
		return cmds
	}

	// Language-default fallback.
	return defaultCommands(lang)
}

// detectFromMakefile checks for a Makefile with build/test/lint targets.
func detectFromMakefile(repoPath string) (BuildCommands, bool) {
	makefileNames := []string{"Makefile", "makefile", "GNUmakefile"}
	for _, name := range makefileNames {
		path := filepath.Join(repoPath, name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return parseMakefileCommands(path)
		}
	}
	return BuildCommands{}, false
}

// parseMakefileCommands extracts build/test/lint from a Makefile.
func parseMakefileCommands(path string) (BuildCommands, bool) {
	f, err := os.Open(path)
	if err != nil {
		return BuildCommands{}, false
	}
	defer f.Close()

	var cmds BuildCommands
	found := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Detect target lines (e.g., "build:" or "test:")
		if !strings.HasPrefix(line, "#") && strings.Contains(line, ":") {
			target := strings.SplitN(line, ":", 2)[0]
			target = strings.TrimSpace(target)
			switch target {
			case "build":
				cmds.Build = "make build"
				found = true
			case "test":
				cmds.Test = "make test"
				found = true
			case "lint":
				cmds.Lint = "make lint"
				found = true
			case "run":
				cmds.Run = "make run"
				found = true
			}
		}
	}
	return cmds, found
}

// detectFromPackageJSON checks for npm/pnpm/yarn scripts by parsing
// the package.json scripts object properly.
func detectFromPackageJSON(repoPath string) (BuildCommands, bool) {
	path := filepath.Join(repoPath, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return BuildCommands{}, false
	}

	// Parse the JSON to extract the scripts object.
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return BuildCommands{}, false
	}
	if len(pkg.Scripts) == 0 {
		return BuildCommands{}, false
	}

	var cmds BuildCommands
	found := false
	if _, ok := pkg.Scripts["build"]; ok {
		cmds.Build = "npm run build"
		found = true
	}
	if _, ok := pkg.Scripts["test"]; ok {
		cmds.Test = "npm test"
		found = true
	}
	if _, ok := pkg.Scripts["lint"]; ok {
		cmds.Lint = "npm run lint"
		found = true
	}
	if _, ok := pkg.Scripts["start"]; ok {
		cmds.Run = "npm start"
		found = true
	}
	return cmds, found
}

// detectFromCI checks for GitHub Actions or GitLab CI configuration.
func detectFromCI(repoPath string) (BuildCommands, bool) {
	var cmds BuildCommands
	found := false

	// GitHub Actions
	ghDir := filepath.Join(repoPath, ".github", "workflows")
	if entries, err := os.ReadDir(ghDir); err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(ghDir, e.Name()))
			if err != nil {
				continue
			}
			content := string(data)
			if strings.Contains(content, "go build") && cmds.Build == "" {
				cmds.Build = "go build ./..."
				found = true
			}
			if strings.Contains(content, "go test") && cmds.Test == "" {
				cmds.Test = "go test ./..."
				found = true
			}
			if strings.Contains(content, "golangci-lint") && cmds.Lint == "" {
				cmds.Lint = "golangci-lint run"
				found = true
			}
			if strings.Contains(content, "npm run build") && cmds.Build == "" {
				cmds.Build = "npm run build"
				found = true
			}
			if strings.Contains(content, "npm test") && cmds.Test == "" {
				cmds.Test = "npm test"
				found = true
			}
			if strings.Contains(content, "cargo build") && cmds.Build == "" {
				cmds.Build = "cargo build"
				found = true
			}
			if strings.Contains(content, "cargo test") && cmds.Test == "" {
				cmds.Test = "cargo test"
				found = true
			}
		}
	}

	// GitLab CI
	gitlabPath := filepath.Join(repoPath, ".gitlab-ci.yml")
	if data, err := os.ReadFile(gitlabPath); err == nil {
		content := string(data)
		if strings.Contains(content, "go build") && cmds.Build == "" {
			cmds.Build = "go build ./..."
			found = true
		}
		if strings.Contains(content, "go test") && cmds.Test == "" {
			cmds.Test = "go test ./..."
			found = true
		}
	}

	return cmds, found
}

// defaultCommands returns language-default build/test/lint commands.
func defaultCommands(lang string) BuildCommands {
	switch strings.ToLower(lang) {
	case "go":
		return BuildCommands{
			Build: "go build ./...",
			Test:  "go test ./...",
			Lint:  "golangci-lint run",
			Run:   "go run ./...",
		}
	case "javascript", "typescript", "node":
		return BuildCommands{
			Build: "npm run build",
			Test:  "npm test",
			Lint:  "npm run lint",
			Run:   "npm start",
		}
	case "rust":
		return BuildCommands{
			Build: "cargo build",
			Test:  "cargo test",
			Lint:  "cargo clippy",
			Run:   "cargo run",
		}
	case "python":
		return BuildCommands{
			Build: "python -m build",
			Test:  "pytest",
			Lint:  "ruff check",
		}
	case "java":
		return BuildCommands{
			Build: "mvn compile",
			Test:  "mvn test",
			Lint:  "mvn checkstyle:check",
			Run:   "mvn exec:java",
		}
	default:
		return BuildCommands{
			Build: "make build",
			Test:  "make test",
			Lint:  "make lint",
		}
	}
}
