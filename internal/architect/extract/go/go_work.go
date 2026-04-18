// Package golang implements go.work file parsing.
package golang

import (
	"os"
	"path/filepath"
	"strings"
)

// goWorkModule represents a single module in a go.work file.
type goWorkModule struct {
	Dir        string
	ModulePath string
}

// detectGoWorkModules checks for a go.work file and returns listed modules.
func detectGoWorkModules(rootDir string) []goWorkModule {
	data, err := os.ReadFile(filepath.Join(rootDir, "go.work"))
	if err != nil {
		return nil
	}

	var modules []goWorkModule
	var inUse bool

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)

		if line == "use (" || line == "use(" {
			inUse = true
			continue
		}
		if inUse && line == ")" {
			inUse = false
			continue
		}
		if inUse && line != "" && !strings.HasPrefix(line, "//") {
			dir := strings.Trim(line, "\"")
			absDir := dir
			if !filepath.IsAbs(dir) {
				absDir = filepath.Join(rootDir, dir)
			}
			modules = append(modules, goWorkModule{Dir: absDir})
			continue
		}

		if strings.HasPrefix(line, "use ") && !strings.HasPrefix(line, "use (") {
			dir := strings.TrimSpace(strings.TrimPrefix(line, "use"))
			dir = strings.Trim(dir, "\"()")
			absDir := dir
			if !filepath.IsAbs(dir) {
				absDir = filepath.Join(rootDir, dir)
			}
			modules = append(modules, goWorkModule{Dir: absDir})
		}
	}

	return modules
}
