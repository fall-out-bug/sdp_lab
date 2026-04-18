package typescript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// parseTSConfigAliases reads tsconfig.json and extracts path aliases.
func parseTSConfigAliases(rootDir string) []TSPathAlias {
	path := filepath.Join(rootDir, "tsconfig.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var cfg struct {
		CompilerOptions struct {
			BaseURL string              `json:"baseUrl"`
			Paths   map[string][]string `json:"paths"`
		} `json:"compilerOptions"`
		References []struct {
			Path string `json:"path"`
		} `json:"references"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}

	var aliases []TSPathAlias
	for alias, targets := range cfg.CompilerOptions.Paths {
		if len(targets) == 0 {
			continue
		}
		cleanAlias := strings.TrimSuffix(alias, "*")
		cleanTarget := strings.TrimSuffix(targets[0], "*")
		aliases = append(aliases, TSPathAlias{
			Alias:  cleanAlias,
			Target: cleanTarget,
		})
	}
	sort.Slice(aliases, func(i, j int) bool { return aliases[i].Alias < aliases[j].Alias })
	return aliases
}

// parsePackageJSONDeps reads package.json and returns all dependencies.
func parsePackageJSONDeps(rootDir string) []TSDependencyEntry {
	path := filepath.Join(rootDir, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var deps []TSDependencyEntry
	for name, version := range pkg.Dependencies {
		deps = append(deps, TSDependencyEntry{Name: name, Version: version, Dev: false})
	}
	for name, version := range pkg.DevDependencies {
		deps = append(deps, TSDependencyEntry{Name: name, Version: version, Dev: true})
	}
	sort.Slice(deps, func(i, j int) bool { return deps[i].Name < deps[j].Name })
	return deps
}
