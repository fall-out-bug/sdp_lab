package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Run executes the full deterministic spec extraction pipeline on a directory.
func Run(repoPath string) (*SpecReport, error) {
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, fmt.Errorf("spec: resolve path: %w", err)
	}
	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("spec: cannot access %q: %w", repoPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("spec: %q is not a directory", repoPath)
	}
	start := time.Now()
	api, _ := ExtractAPIContracts(abs)
	if api == nil {
		api = &APIContracts{}
	}
	rules, scanned, withSpecs, _ := extractAllRules(abs)
	sql, _ := extractAllSQL(abs)
	for _, sc := range sql {
		rules.Validations = append(rules.Validations, ValidationRule{
			Category: "sql_constraint",
			Description: fmt.Sprintf("%s on %s.%s", sc.Type, sc.Table, sc.Column),
			Enforcement: "database", Location: sc.SourceFile,
			Field: sc.Column, Constraints: sqlToRules(sc),
		})
	}
	rules.Total = len(rules.Validations)
	var density float64
	if scanned > 0 {
		density = float64(withSpecs) / float64(scanned)
	}
	return &SpecReport{
		Version: "1.0.0", Repo: filepath.Base(abs), GeneratedAt: start.UTC(),
		DurationMs:    time.Since(start).Milliseconds(),
		APIContracts:  *api,
		BusinessRules: *rules,
		Coverage:      Coverage{FilesScanned: scanned, FilesWithSpecs: withSpecs, SpecDensity: density},
	}, nil
}

func extractAllRules(root string) (*BusinessRules, int, int, error) {
	var all []ValidationRule
	scanned, withSpecs := 0, 0
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		scanned++
		rules, _ := ExtractBusinessRules(path)
		if len(rules) > 0 {
			withSpecs++
			all = append(all, rules...)
		}
		return nil
	})
	return &BusinessRules{Validations: all, Total: len(all)}, scanned, withSpecs, nil
}

func extractAllSQL(root string) ([]SQLConstraint, error) {
	var all []SQLConstraint
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".sql") {
			return nil
		}
		cs, _ := ParseSQLFile(path)
		all = append(all, cs...)
		return nil
	})
	return all, nil
}

func sqlToRules(sc SQLConstraint) []Constraint {
	cs := []Constraint{{Type: sc.Type, Value: sc.Value}}
	if sc.References != "" {
		cs = append(cs, Constraint{Type: "references", Value: sc.References})
	}
	return cs
}
