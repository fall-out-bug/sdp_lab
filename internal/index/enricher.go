package index

import (
	"fmt"
	"strconv"
)

// Enrich updates the store with data from optional toolkit artifacts.
// Every enrichment source is optional — missing data degrades gracefully
// without blocking index usage.
func Enrich(s ManifestStore, input *EnrichmentInput) error {
	if input == nil {
		return nil
	}

	modules, err := s.LoadModules()
	if err != nil {
		return fmt.Errorf("enrich: load modules: %w", err)
	}

	// Load metadata values we need to update
	meta, err := s.LoadMeta(
		"repo_name", "languages", "arch_style",
		"commit_style", "build_system",
		"active_branches",
	)
	if err != nil {
		return fmt.Errorf("enrich: load meta: %w", err)
	}

	// Architect enrichment
	if input.ArchitectReport != nil {
		ar := input.ArchitectReport
		if ar.ArchStyle != "" {
			meta["arch_style"] = ar.ArchStyle
		}
		for i := range modules {
			path := modules[i].Path
			if purpose, ok := ar.ModulePurposes[path]; ok && modules[i].Purpose == "" {
				modules[i].Purpose = purpose
			}
		}
	}

	// Metrics enrichment
	if input.MetricsReport != nil {
		mr := input.MetricsReport
		if mr.CommitStyle != "" {
			meta["commit_style"] = mr.CommitStyle
		}
		if mr.ActiveBranches > 0 {
			meta["active_branches"] = strconv.Itoa(mr.ActiveBranches)
		}
		// Build module risk lookup
		riskMap := make(map[string]ModuleRiskEntry)
		for _, r := range mr.ModuleRisks {
			riskMap[r.Module] = r
		}
		for i := range modules {
			path := modules[i].Path
			if risk, ok := riskMap[path]; ok {
				if risk.BusFactor > 0 {
					modules[i].BusFactor = risk.BusFactor
				}
				if risk.PrimaryAuthor != "" {
					modules[i].Owner = risk.PrimaryAuthor
				}
			}
		}
	}

	// Scout enrichment
	if input.ScoutCard != nil {
		sc := input.ScoutCard
		if sc.BuildSystem != "" {
			meta["build_system"] = sc.BuildSystem
		}
		if sc.PrimaryLanguage != "" {
			meta["languages"] = sc.PrimaryLanguage
		}
	}

	// Git blame enrichment (fallback for owner)
	if len(input.GitBlame) > 0 {
		enrichFromBlame(modules, input.GitBlame)
	}

	// Update modules in store
	if err := s.UpdateModules(modules); err != nil {
		return fmt.Errorf("enrich: update modules: %w", err)
	}

	// Update meta if enrichment provided new values
	if input.ArchitectReport != nil || input.MetricsReport != nil || input.ScoutCard != nil {
		if err := saveEnrichedMeta(s, meta); err != nil {
			return fmt.Errorf("enrich: update meta: %w", err)
		}
	}

	return nil
}

// enrichFromBlame sets module owners based on git blame data.
// The owner is the most frequent author across files in the module.
// Only sets owner if not already set.
func enrichFromBlame(modules []ModuleMeta, blame map[string]string) {
	for i := range modules {
		if modules[i].Owner != "" {
			continue // don't override existing owner
		}
		authorCounts := make(map[string]int)
		modPath := modules[i].Path
		for file, author := range blame {
			if isFileInModule(file, modPath) {
				authorCounts[author]++
			}
		}
		if len(authorCounts) > 0 {
			modules[i].Owner = topAuthor(authorCounts)
		}
	}
}

// isFileInModule checks whether a file path belongs to a module directory.
// It ensures the file is under the module path with a path separator boundary,
// preventing "internal/core" from matching "internal/core_utils/file.go".
func isFileInModule(filePath, modulePath string) bool {
	if len(filePath) < len(modulePath) {
		return false
	}
	if filePath[:len(modulePath)] != modulePath {
		return false
	}
	if len(filePath) == len(modulePath) {
		return true // exact match
	}
	return filePath[len(modulePath)] == '/'
}

// topAuthor returns the author with the highest count.
func topAuthor(counts map[string]int) string {
	top := ""
	topN := 0
	for a, n := range counts {
		if n > topN {
			top = a
			topN = n
		}
	}
	return top
}

// saveEnrichedMeta persists enriched metadata keys back to the store.
func saveEnrichedMeta(s ManifestStore, meta map[string]string) error {
	for k, v := range meta {
		if err := s.SaveMeta(k, v); err != nil {
			return fmt.Errorf("save meta %s: %w", k, err)
		}
	}
	return nil
}
