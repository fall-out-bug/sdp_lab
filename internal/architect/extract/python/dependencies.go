package python

import (
	"bufio"
	"os"
	"regexp"
	"strings"
)

type Dependency struct {
	Name   string
	Source string
	Kind   string
}

func ParseRequirementsTxt(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if m := reRequirement.FindStringSubmatch(line); m != nil {
			name := m[1]
			if name != "" {
				deps = append(deps, Dependency{
					Name:   name,
					Source: "requirements.txt",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

func ParsePyprojectToml(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	inDeps := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			lower := strings.ToLower(trimmed)
			inDeps = strings.Contains(lower, "[project") && strings.Contains(lower, "dependencies") ||
				strings.Contains(lower, "[tool.poetry") && strings.Contains(lower, "dependencies")
			continue
		}

		if !inDeps {
			continue
		}

		if m := rePyprojectDep.FindStringSubmatch(trimmed); m != nil {
			name := strings.Trim(m[1], `"`)
			if name != "" && name != "python" {
				deps = append(deps, Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
			continue
		}

		if m := rePoetryDep.FindStringSubmatch(trimmed); m != nil {
			name := m[1]
			if name != "" && name != "python" {
				deps = append(deps, Dependency{
					Name:   name,
					Source: "pyproject.toml",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

func ParseSetupPy(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	inInstallRequires := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.Contains(trimmed, "install_requires") {
			inInstallRequires = true
			extractSetupDepsFromLine(trimmed, &deps)
			continue
		}

		if inInstallRequires {
			if strings.Contains(trimmed, "]") {
				inInstallRequires = false
				extractSetupDepsFromLine(trimmed, &deps)
				continue
			}
			extractSetupDepsFromLine(trimmed, &deps)
		}
	}
	return deps
}

func extractSetupDepsFromLine(line string, deps *[]Dependency) {
	matches := reSetupDep.FindAllStringSubmatch(line, -1)
	for _, m := range matches {
		name := m[1]
		if name == "" {
			continue
		}
		*deps = append(*deps, Dependency{
			Name:   name,
			Source: "setup.py",
			Kind:   "third-party",
		})
	}
}

func ParseSetupCfg(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	inInstallRequires := false
	inOptionsSection := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			inOptionsSection = strings.Contains(strings.ToLower(trimmed), "options")
			inInstallRequires = false
			continue
		}

		if !inOptionsSection {
			continue
		}

		if strings.HasPrefix(trimmed, "install_requires") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				if val != "" && !strings.HasPrefix(val, "\n") {
					name := strings.Trim(val, `'"`)
					if name != "" {
						deps = append(deps, Dependency{
							Name:   name,
							Source: "setup.cfg",
							Kind:   "third-party",
						})
					}
				}
			}
			inInstallRequires = true
			continue
		}

		if inInstallRequires {
			if trimmed != "" && !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inInstallRequires = false
				continue
			}

			if trimmed != "" {
				name := trimmed
				if idx := strings.IndexAny(name, "><=!~;"); idx >= 0 {
					name = strings.TrimSpace(name[:idx])
				}
				name = strings.Trim(name, `'"`)
				if name != "" {
					deps = append(deps, Dependency{
						Name:   name,
						Source: "setup.cfg",
						Kind:   "third-party",
					})
				}
			}
		}
	}
	return deps
}

func ParsePipfile(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	inPackages := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "[") {
			inPackages = trimmed == "[packages]" || trimmed == "[dev-packages]"
			continue
		}

		if !inPackages || trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		if m := rePipfileDep.FindStringSubmatch(trimmed); m != nil {
			name := strings.TrimSpace(m[1])
			if name != "" {
				deps = append(deps, Dependency{
					Name:   name,
					Source: "Pipfile",
					Kind:   "third-party",
				})
			}
		}
	}
	return deps
}

var (
	reRequirement   = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)`)
	rePyprojectDep  = regexp.MustCompile(`^\s*"?([A-Za-z0-9][A-Za-z0-9._-]*)"??\s*[>=<~!]`)
	rePoetryDep     = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)
	reSetupDep      = regexp.MustCompile(`['"]([A-Za-z0-9][A-Za-z0-9._-]*)['"]`)
	rePipfileDep    = regexp.MustCompile(`^\s*([A-Za-z0-9][A-Za-z0-9._-]*)\s*=`)
)
