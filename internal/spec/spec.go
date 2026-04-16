package spec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// RunOptions controls optional behaviour in the spec extraction pipeline.
type RunOptions struct {
	Enrich bool            // opt-in: attempt LLM enrichment (stub, not implemented)
	Ctx    context.Context // optional: cancellation support (nil means background)
}

func (o RunOptions) ctx() context.Context {
	if o.Ctx != nil {
		return o.Ctx
	}
	return context.Background()
}

// Run executes the full deterministic spec extraction pipeline on a directory.
func Run(repoPath string) (*SpecReport, error) {
	return RunWithOptions(repoPath, RunOptions{})
}

// RunWithOptions executes the spec pipeline with optional enrichment.
// Without opts.Enrich the output is identical to Run().
func RunWithOptions(repoPath string, opts RunOptions) (*SpecReport, error) {
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
	ctx := opts.ctx()
	var warnings []string
	start := time.Now()
	api, apiErr := ExtractAPIContracts(abs)
	if apiErr != nil {
		warnings = append(warnings, fmt.Sprintf("api: %v", apiErr))
	}
	if api == nil {
		api = &APIContracts{}
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	rules, scanned, withSpecs, rulesErr := extractAllRules(abs)
	if rulesErr != nil {
		warnings = append(warnings, fmt.Sprintf("rules: %v", rulesErr))
	}
	sql, sqlErr := extractAllSQL(abs)
	if sqlErr != nil {
		warnings = append(warnings, fmt.Sprintf("sql: %v", sqlErr))
	}
	for _, sc := range sql {
		rules.Validations = append(rules.Validations, ValidationRule{
			Category: "sql_constraint",
			Description: fmt.Sprintf("%s on %s.%s", sc.Type, sc.Table, sc.Column),
			Enforcement: "database", Location: sc.SourceFile,
			Field: sc.Column, Constraints: sqlToRules(sc),
		})
	}
	rules.Total = len(rules.Validations)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	inv, invErr := ExtractInvariants(abs)
	if invErr != nil {
		warnings = append(warnings, fmt.Sprintf("invariants: %v", invErr))
	}
	sla, slaErr := ExtractSLAParameters(abs)
	if slaErr != nil {
		warnings = append(warnings, fmt.Sprintf("sla: %v", slaErr))
	}
	cfgParams, cfgErr := ExtractConfigParameters(abs)
	if cfgErr != nil {
		warnings = append(warnings, fmt.Sprintf("config: %v", cfgErr))
	}
	mergeConfigSLA(&sla, cfgParams)
	var density float64
	if scanned > 0 {
		density = float64(withSpecs) / float64(scanned)
	}
	report := &SpecReport{
		Version: "1.0.0", Repo: filepath.Base(abs), GeneratedAt: start.UTC(),
		DurationMs:    time.Since(start).Milliseconds(),
		APIContracts:  *api,
		BusinessRules: *rules,
		Invariants:    inv,
		SLAParameters: sla,
		Coverage:      Coverage{FilesScanned: scanned, FilesWithSpecs: withSpecs, SpecDensity: density},
		Warnings:      warnings,
	}
	if opts.Enrich {
		report.Enrichment = &EnrichmentInfo{
			Attempted: true,
			Status:    "not_configured",
			Note:      "enrichment is not yet implemented; output is deterministic-only",
		}
	}
	return report, nil
}

// mergeConfigSLA folds config-extracted parameters into the SLA struct.
func mergeConfigSLA(sla *SLAParameters, params []SLAParam) {
	for _, p := range params {
		if p.Category == "secret" {
			continue
		}
		switch p.Category {
		case "timeout":
			sla.Timeouts = append(sla.Timeouts, p)
		case "retry":
			sla.Retries = append(sla.Retries, p)
		case "rate_limit":
			sla.RateLimits = append(sla.RateLimits, p)
		case "resource_pool":
			sla.ResourcePools = append(sla.ResourcePools, p)
		}
	}
	sla.Total = len(sla.Timeouts) + len(sla.Retries) + len(sla.RateLimits) +
		len(sla.CircuitBreakers) + len(sla.ResourcePools) + len(sla.HealthChecks)
}

func extractAllRules(root string) (*BusinessRules, int, int, error) {
	var all []ValidationRule
	scanned, withSpecs := 0, 0
	filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if fi, e := d.Info(); e == nil && fi.Size() > 10*1024*1024 {
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
		if fi, e := d.Info(); e == nil && fi.Size() > 10*1024*1024 {
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
