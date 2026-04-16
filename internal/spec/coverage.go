package spec

// coverage.go — helpers for computing cross-source coverage metrics.

// filesWithSpecs counts unique files that contributed specs across all extraction sources.
func filesWithSpecs(api *APIContracts, inv Invariants, sla SLAParameters, rulesWithSpecs int) int {
	total := rulesWithSpecs
	if len(api.HTTPEndpoints) > 0 {
		total += countUniqueFiles(apiFiles(api.HTTPEndpoints))
	}
	if inv.Total > 0 {
		total += countUniqueFiles(invFiles(inv))
	}
	if sla.Total > 0 {
		total += countUniqueFiles(slaLocationFiles(sla))
	}
	return total
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
