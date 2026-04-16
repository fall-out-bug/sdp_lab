package metrics

import "strings"

// Filter removes noise commits from raw data according to the design's
// global filtering rules: bots, generated files, formatting-only, CI-only.
// This is called once after collection, before any analysis.
func Filter(data *GitData) *GitData {
	if data == nil {
		return nil
	}
	filtered := &GitData{
		Tags:       data.Tags,
		Branches:   data.Branches,
		MergeCount: data.MergeCount,
	}

	for _, c := range data.Commits {
		if IsBot(c.Author) {
			continue
		}
		// Filter file-level noise
		var cleanFiles []FileChange
		for _, f := range c.Files {
			if IsGeneratedFile(f.Path) {
				continue
			}
			if isVendorPath(f.Path) {
				continue
			}
			cleanFiles = append(cleanFiles, f)
		}
		c.Files = cleanFiles

		if IsCIOnly(c.Files) {
			continue
		}
		if IsFormattingOnly(c.Files) {
			continue
		}

		filtered.Commits = append(filtered.Commits, c)
	}
	return filtered
}

func isVendorPath(path string) bool {
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == "vendor" || p == "node_modules" {
			return true
		}
	}
	return false
}
