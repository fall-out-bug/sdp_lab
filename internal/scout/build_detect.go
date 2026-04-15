package scout

import (
	"os"
	"path/filepath"
	"strings"
)

// configFiles lists filenames considered config for Build.ConfigFiles.
var configFiles = map[string]bool{
	".goreleaser.yml": true, ".goreleaser.yaml": true,
	"Makefile": true, "docker-compose.yml": true, "Dockerfile": true,
}

func isConfigFile(name string) bool {
	return configFiles[name]
}

func populateDependencyCount(root string, bld *Build) {
	if bld.DependencyFile == nil {
		return
	}
	path := filepath.Join(root, *bld.DependencyFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	if strings.HasSuffix(*bld.DependencyFile, "go.mod") {
		bld.DependencyCount = countGoModRequires(string(data))
	} else if *bld.DependencyFile == "package.json" {
		bld.DependencyCount = countPackageJSONDeps(string(data))
	}
}

func countGoModRequires(content string) int {
	count := 0
	inRequire := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "require (") {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire && trimmed != "" && !strings.HasPrefix(trimmed, "//") {
			count++
		}
		if !inRequire && strings.HasPrefix(trimmed, "require ") && !strings.Contains(trimmed, "(") {
			count++
		}
	}
	return count
}

func countPackageJSONDeps(content string) int {
	count := 0
	for _, section := range []string{"\"dependencies\"", "\"devDependencies\""} {
		idx := strings.Index(content, section)
		if idx < 0 {
			continue
		}
		sub := content[idx:]
		endIdx := strings.Index(sub, "}")
		if endIdx < 0 {
			continue
		}
		block := sub[len(section):endIdx]
		for _, line := range strings.Split(block, "\n") {
			if strings.Contains(line, ":") && strings.Contains(line, "\"") {
				count++
			}
		}
	}
	return count
}

// detectMaturitySignals populates Maturity flags from filesystem signals.
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
