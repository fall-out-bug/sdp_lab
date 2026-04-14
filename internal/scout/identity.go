package scout

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/common"
)

// programmingLanguages are eligible for "primary language" selection.
// Markup, config, and data formats are excluded.
var programmingLanguages = map[string]bool{
	"go": true, "typescript": true, "javascript": true, "python": true,
	"java": true, "kotlin": true, "scala": true, "rust": true, "ruby": true,
	"c": true, "cpp": true, "csharp": true, "swift": true, "dart": true,
	"lua": true, "r": true, "php": true, "shell": true, "powershell": true,
	"elixir": true, "erlang": true, "clojure": true, "haskell": true,
	"ocaml": true, "zig": true, "nim": true, "vlang": true,
}

func isProgrammingLanguage(lang string) bool { return programmingLanguages[lang] }

// detectIdentity runs Phase 1: language distribution, build system, README, monorepo.
func detectIdentity(root string) (Identity, Maturity, Build) {
	var id Identity
	var mat Maturity
	var bld Build

	id.Name = filepath.Base(root)
	id.Languages = make(map[string]LangStats)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			if common.DefaultMatcher.Match(info.Name(), true) {
				return filepath.SkipDir
			}
			return nil
		}
		if common.DefaultMatcher.Match(info.Name(), false) {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if lang, ok := ExtToLanguage(ext); ok {
			st := id.Languages[lang]
			st.Files++
			id.Languages[lang] = st
		}
		return nil
	})

	// Compute language ratios
	var totalFiles int
	for _, st := range id.Languages {
		totalFiles += st.Files
	}
	for lang, st := range id.Languages {
		if totalFiles > 0 {
			st.Ratio = float64(st.Files) / float64(totalFiles)
		}
		id.Languages[lang] = st
	}

	// Primary language = highest file count among programming languages only
	maxFiles := 0
	for lang, st := range id.Languages {
		if !isProgrammingLanguage(lang) {
			continue
		}
		if st.Files > maxFiles {
			maxFiles = st.Files
			id.PrimaryLanguage = lang
		}
	}

	detectBuildSystemFiles(root, &id, &bld)
	mat.HasReadme = detectReadme(root, &id)
	id.Monorepo = detectMonorepo(root)
	detectMaturitySignals(root, &mat)

	return id, mat, bld
}

func detectBuildSystemFiles(root string, id *Identity, bld *Build) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if sys, ok := DetectBuildSystem(e.Name()); ok {
			s := sys
			id.BuildSystem = &s
			id.BuildFiles = append(id.BuildFiles, e.Name())
			bld.PackageManager = &s
			df := e.Name()
			bld.DependencyFile = &df
		}
	}
}

func detectReadme(root string, id *Identity) bool {
	for _, name := range []string{"README.md", "README.rst", "README.txt", "README"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		desc := extractFirstParagraph(string(data))
		if desc != "" {
			id.Description = &desc
		}
		return true
	}
	return false
}

func extractFirstParagraph(content string) string {
	scanner := bufio.NewScanner(strings.NewReader(content))
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" && len(lines) > 0 {
			break
		}
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	text := strings.Join(lines, " ")
	if len(text) > 200 {
		text = text[:197] + "..."
	}
	return text
}

func detectMonorepo(root string) bool {
	for _, f := range []string{"lerna.json", "pnpm-workspace.yaml", "pnpm-workspace.yml"} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			return true
		}
	}
	if pkgs, err := filepath.Glob(filepath.Join(root, "packages", "*", "package.json")); err == nil && len(pkgs) >= 2 {
		return true
	}
	if mods, err := filepath.Glob(filepath.Join(root, "*", "go.mod")); err == nil && len(mods) >= 2 {
		return true
	}
	return false
}

func detectMaturitySignals(root string, mat *Maturity) {
	for name, field := range map[string]*bool{
		"LICENSE": &mat.HasLicense, "LICENSE.md": &mat.HasLicense, "LICENSE.txt": &mat.HasLicense,
		"Dockerfile": &mat.HasDocker, "CODEOWNERS": &mat.HasCodeowners,
		"CONTRIBUTING.md": &mat.HasContributing, "CHANGELOG.md": &mat.HasChangelog,
	} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			*field = true
		}
	}
	for _, f := range []string{".golangci.yml", ".golangci.yaml", ".eslintrc.js", ".eslintrc.json", ".flake8", ".pylintrc"} {
		if _, err := os.Stat(filepath.Join(root, f)); err == nil {
			mat.HasLinter = true
			break
		}
	}
	for _, ci := range []struct {
		path, name string
	}{
		{".github/workflows", "github-actions"},
		{".gitlab-ci.yml", "gitlab-ci"},
		{"Jenkinsfile", "jenkins"},
		{".circleci", "circleci"},
		{".travis.yml", "travis"},
	} {
		if _, err := os.Stat(filepath.Join(root, ci.path)); err == nil {
			mat.HasCI = true
			s := ci.name
			mat.CISystem = &s
			break
		}
	}
}
