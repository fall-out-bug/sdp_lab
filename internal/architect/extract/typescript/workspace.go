package typescript

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// detectMonorepo checks for monorepo markers and returns workspace packages.
func detectMonorepo(rootDir string) ([]TSWorkspaceInfo, bool, string) {
	var workspaces []TSWorkspaceInfo
	isMonorepo := false
	tool := ""

	// Check package.json workspaces.
	pkgWorkspaces := parsePackageJSONWorkspaces(filepath.Join(rootDir, "package.json"))
	if len(pkgWorkspaces) > 0 {
		isMonorepo = true
		tool = "npm"
		workspaces = expandWorkspacePatterns(rootDir, pkgWorkspaces)
	}

	// Check yarn.lock — indicates yarn workspaces.
	if fileExists(filepath.Join(rootDir, "yarn.lock")) && len(pkgWorkspaces) > 0 {
		tool = "yarn"
	}

	// Check pnpm-lock.yaml + pnpm-workspace.yaml.
	if fileExists(filepath.Join(rootDir, "pnpm-lock.yaml")) {
		pnpmWorkspaces := parsePnpmWorkspace(filepath.Join(rootDir, "pnpm-workspace.yaml"))
		if len(pnpmWorkspaces) > 0 {
			isMonorepo = true
			tool = "pnpm"
			ws := expandWorkspacePatterns(rootDir, pnpmWorkspaces)
			workspaces = mergeWorkspaces(workspaces, ws)
		}
	}

	// Check lerna.json.
	if fileExists(filepath.Join(rootDir, "lerna.json")) {
		isMonorepo = true
		if tool == "" {
			tool = "lerna"
		}
	}

	// Check turborepo (turbo.json).
	if fileExists(filepath.Join(rootDir, "turbo.json")) {
		isMonorepo = true
		if tool == "" || tool == "npm" {
			tool = "turborepo"
		}
	}

	return workspaces, isMonorepo, tool
}

// parsePackageJSONWorkspaces reads package.json and returns workspace glob patterns.
func parsePackageJSONWorkspaces(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Workspaces interface{} `json:"workspaces"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}
	return parseWorkspaces(pkg.Workspaces)
}

// parsePnpmWorkspace reads a pnpm-workspace.yaml file for workspace packages.
func parsePnpmWorkspace(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var patterns []string
	inPackages := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "packages:" {
			inPackages = true
			continue
		}
		if inPackages {
			if line == "" || (!strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "\"")) {
				if len(patterns) > 0 {
					break
				}
				continue
			}
			pattern := strings.TrimPrefix(line, "- ")
			pattern = strings.Trim(pattern, "\"' ")
			if pattern != "" {
				patterns = append(patterns, pattern)
			}
		}
	}
	return patterns
}

// expandWorkspacePatterns expands workspace glob patterns into actual directory paths.
func expandWorkspacePatterns(rootDir string, patterns []string) []TSWorkspaceInfo {
	var ws []TSWorkspaceInfo
	seen := make(map[string]bool)
	for _, pattern := range patterns {
		// Only handle simple star patterns like "packages/*".
		if !strings.Contains(pattern, "*") {
			// Direct path reference.
			absPath := filepath.Join(rootDir, pattern)
			pkgJSON := filepath.Join(absPath, "package.json")
			if fileExists(pkgJSON) {
				name := readPkgName(pkgJSON)
				rel := pattern
				if !seen[rel] {
					seen[rel] = true
					ws = append(ws, TSWorkspaceInfo{Name: name, Path: rel})
				}
			}
			continue
		}

		// Glob pattern: split on "/*" to find parent directory.
		parts := strings.SplitN(pattern, "*", 2)
		parentDir := filepath.Join(rootDir, parts[0])
		entries, err := os.ReadDir(parentDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			relPath := filepath.Join(parts[0], e.Name())
			pkgJSON := filepath.Join(rootDir, relPath, "package.json")
			if fileExists(pkgJSON) {
				name := readPkgName(pkgJSON)
				if !seen[relPath] {
					seen[relPath] = true
					ws = append(ws, TSWorkspaceInfo{Name: name, Path: relPath})
				}
			}
		}
	}
	sort.Slice(ws, func(i, j int) bool { return ws[i].Path < ws[j].Path })
	return ws
}

// readPkgName reads the "name" field from a package.json.
func readPkgName(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return pkg.Name
}

// mergeWorkspaces merges two workspace slices, deduplicating by path.
func mergeWorkspaces(a, b []TSWorkspaceInfo) []TSWorkspaceInfo {
	seen := make(map[string]bool)
	for _, w := range a {
		seen[w.Path] = true
	}
	for _, w := range b {
		if !seen[w.Path] {
			seen[w.Path] = true
			a = append(a, w)
		}
	}
	return a
}

// parseWorkspaces handles both array and object forms of the "workspaces" field.
func parseWorkspaces(raw interface{}) []string {
	if raw == nil {
		return nil
	}
	// Array form: ["packages/*", "apps/*"]
	if arr, ok := raw.([]interface{}); ok {
		var out []string
		for _, v := range arr {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	// Object form: {"packages": ["packages/*"]}
	if obj, ok := raw.(map[string]interface{}); ok {
		if pkgs, ok := obj["packages"]; ok {
			return parseWorkspaces(pkgs)
		}
	}
	return nil
}
