package spec

import (
	"os"
	"path/filepath"
	"strings"
)

// coverage.go — helpers for computing cross-source coverage metrics.

// filesWithSpecs counts globally-unique files that contributed specs across all
// extraction sources. Deduplicates across sources so a file with both routes
// and validation tags counts once, not twice.
func filesWithSpecs(api *APIContracts, inv Invariants, sla SLAParameters, rulesFiles []string) int {
	seen := map[string]bool{}
	mark := func(files []string) {
		for _, f := range files {
			if idx := strings.Index(f, ":"); idx >= 0 {
				f = f[:idx]
			}
			seen[f] = true
		}
	}
	mark(rulesFiles)
	if len(api.HTTPEndpoints) > 0 {
		mark(apiFiles(api.HTTPEndpoints))
	}
	if inv.Total > 0 {
		mark(invFiles(inv))
	}
	if sla.Total > 0 {
		mark(slaLocationFiles(sla))
	}
	return len(seen)
}

func apiFiles(eps []Endpoint) []string {
	seen := map[string]bool{}
	var out []string
	for _, e := range eps {
		if !seen[e.SourceFile] {
			seen[e.SourceFile] = true
			out = append(out, e.SourceFile)
		}
	}
	return out
}

func invFiles(inv Invariants) []string {
	var out []string
	for _, v := range inv.Database {
		out = append(out, v.Location)
	}
	for _, v := range inv.TypeSystem {
		out = append(out, v.Location)
	}
	for _, v := range inv.Concurrency {
		out = append(out, v.Location)
	}
	for _, v := range inv.Architectural {
		out = append(out, v.Location)
	}
	return out
}

func slaLocationFiles(sla SLAParameters) []string {
	var out []string
	for _, p := range sla.Timeouts {
		out = append(out, p.Location)
	}
	for _, p := range sla.Retries {
		out = append(out, p.Location)
	}
	for _, p := range sla.RateLimits {
		out = append(out, p.Location)
	}
	for _, p := range sla.CircuitBreakers {
		out = append(out, p.Location)
	}
	for _, p := range sla.ResourcePools {
		out = append(out, p.Location)
	}
	for _, p := range sla.HealthChecks {
		out = append(out, p.Location)
	}
	return out
}

func countUniqueFiles(files []string) int {
	seen := map[string]bool{}
	for _, f := range files {
		seen[f] = true
	}
	return len(seen)
}

// specExts lists file extensions that can contribute specs.
var specExts = map[string]bool{
	".go": true, ".sql": true, ".yaml": true, ".yml": true, ".json": true,
}

// countScannedFiles walks root and counts files with spec-capable extensions,
// skipping _test.go and files >10MB.
func countScannedFiles(root string) int {
	n := 0
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !specExts[ext] {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if fi, e := d.Info(); e == nil && fi.Size() > 10*1024*1024 {
			return nil
		}
		n++
		return nil
	})
	return n
}
