// Package python provides a Python ecosystem extractor for architecture analysis.
// It detects imports, dependencies, and framework patterns (Flask, FastAPI, Django).
package python

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"sdp_dev/internal/architect"
)

// PythonExtractor implements architect.Extractor for Python projects.
// It uses pragmatic regex/text parsing for import extraction to avoid heavy
// tree-sitter CGo dependencies.
type PythonExtractor struct{}

// Language returns "python".
func (p *PythonExtractor) Language() string { return "python" }

// Extract walks rootDir, parses .py files for imports, reads dependency manifests,
// and detects frameworks.
func (p *PythonExtractor) Extract(ctx context.Context, rootDir string) (*architect.ExtractionResult, error) {
	result := &architect.ExtractionResult{
		Language:         "python",
		ExtractionMethod: "regex",
		AccuracyEstimate: 0.60,
	}

	seen := make(map[string]bool)
	frameworks := make(map[string]architect.Framework)

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rel, _ := filepath.Rel(rootDir, path)
			_ = rel // relative path used for import resolution below

		if info.IsDir() {
			name := info.Name()
			if skipDirs[name] {
				return filepath.SkipDir
			}
			if strings.HasSuffix(name, ".egg-info") {
				return filepath.SkipDir
			}
			return nil
		}

		switch {
		case strings.HasSuffix(info.Name(), ".py"):
			imports, fws, err := ParseImports(path)
			if err != nil {
				return nil
			}
			result.FileCount++

			for _, imp := range imports {
				key := imp.Source + ":" + imp.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   imp.Name,
						Source: imp.Source,
						Kind:   imp.Kind,
					})
				}
			}

			for _, fw := range fws {
				if existing, ok := frameworks[fw.Name]; !ok || fw.Confidence > existing.Confidence {
					frameworks[fw.Name] = architect.Framework{
						Name:       fw.Name,
						Confidence: fw.Confidence,
						Evidence:   fw.Evidence,
					}
				}
			}

		case info.Name() == "requirements.txt":
			deps := ParseRequirementsTxt(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   d.Name,
						Source: d.Source,
						Kind:   d.Kind,
					})
				}
			}

		case info.Name() == "pyproject.toml":
			deps := ParsePyprojectToml(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   d.Name,
						Source: d.Source,
						Kind:   d.Kind,
					})
				}
			}

		case info.Name() == "setup.py":
			deps := ParseSetupPy(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   d.Name,
						Source: d.Source,
						Kind:   d.Kind,
					})
				}
			}

		case info.Name() == "setup.cfg":
			deps := ParseSetupCfg(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   d.Name,
						Source: d.Source,
						Kind:   d.Kind,
					})
				}
			}

		case info.Name() == "Pipfile":
			deps := ParsePipfile(path)
			for _, d := range deps {
				key := d.Source + ":" + d.Name
				if !seen[key] {
					seen[key] = true
					result.Dependencies = append(result.Dependencies, architect.Dependency{
						Name:   d.Name,
						Source: d.Source,
						Kind:   d.Kind,
					})
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, fw := range frameworks {
		result.Frameworks = append(result.Frameworks, fw)
	}

	return result, nil
}
