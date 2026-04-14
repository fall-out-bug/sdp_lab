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
