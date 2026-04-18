// Package golang implements go.mod parsing and module graph analysis.
package golang

import (
	"bufio"
	"os"
	"strings"
)

// parseModuleInfo reads a go.mod file and extracts module metadata.
func parseModuleInfo(path string) *ModuleInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	info := &ModuleInfo{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))

	var inRequire, inExclude bool

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		if strings.HasPrefix(line, "module ") {
			info.ModulePath = strings.TrimSpace(strings.TrimPrefix(line, "module"))
			continue
		}

		if strings.HasPrefix(line, "go ") {
			info.GoVersion = strings.TrimSpace(strings.TrimPrefix(line, "go"))
			continue
		}

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}
		if strings.HasPrefix(line, "require ") && !strings.HasPrefix(line, "require (") {
			info.Requires = append(info.Requires, parseModuleDep(strings.TrimPrefix(line, "require ")))
			continue
		}
		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			info.Requires = append(info.Requires, parseModuleDep(line))
			continue
		}

		if line == "exclude (" {
			inExclude = true
			continue
		}
		if inExclude && line == ")" {
			inExclude = false
			continue
		}
		if strings.HasPrefix(line, "exclude ") {
			fields := strings.Fields(strings.TrimPrefix(line, "exclude"))
			if len(fields) > 0 {
				info.Excludes = append(info.Excludes, fields[0])
			}
			continue
		}
		if inExclude && line != "" {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				info.Excludes = append(info.Excludes, fields[0])
			}
			continue
		}

		if strings.HasPrefix(line, "replace ") {
			content := strings.TrimSpace(strings.TrimPrefix(line, "replace"))
			if dep := parseModuleReplace(content); dep != nil {
				info.Replaces = append(info.Replaces, *dep)
			}
			continue
		}
	}

	return info
}

// parseModuleDep parses a dependency line from go.mod.
func parseModuleDep(line string) ModuleDep {
	line = strings.TrimSpace(line)
	indirect := false

	if idx := strings.Index(line, "//"); idx >= 0 {
		comment := strings.TrimSpace(line[idx+2:])
		line = strings.TrimSpace(line[:idx])
		indirect = strings.Contains(comment, "indirect")
	}

	fields := strings.Fields(line)
	if len(fields) >= 2 {
		return ModuleDep{
			Path:     fields[0],
			Version:  fields[1],
			Indirect: indirect,
		}
	}
	return ModuleDep{}
}

// parseModuleReplace parses a replace directive.
func parseModuleReplace(line string) *ModuleDep {
	line = strings.TrimSpace(line)

	if idx := strings.Index(line, "=>"); idx >= 0 {
		oldPart := strings.TrimSpace(line[:idx])
		newPart := strings.TrimSpace(line[idx+2:])
		return &ModuleDep{
			Path:      oldPart,
			Version:   newPart,
			IsReplace: true,
		}
	}
	return nil
}
