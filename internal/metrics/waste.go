package metrics

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// AnalyzeWaste computes wasted-work metrics from raw git data.
func AnalyzeWaste(data *GitData) *Waste {
	if data == nil || len(data.Commits) == 0 {
		return nil
	}
	w := &Waste{
		ChurnWindowDays: 14,
	}

	// ── Churn analysis ──
	type fileChurn struct {
		added, deleted int
		commits        int
	}
	churnMap := make(map[string]*fileChurn)
	var totalAdded, churnAdded int

	for _, c := range data.Commits {
		for _, f := range c.Files {
			totalAdded += f.Added
			entry, ok := churnMap[f.Path]
			if !ok {
				entry = &fileChurn{}
				churnMap[f.Path] = entry
			}
			entry.added += f.Added
			entry.deleted += f.Deleted
			entry.commits++
		}
	}

	// Churn = min(added, deleted) per file
	var totalChurn int
	for _, entry := range churnMap {
		churn := min(entry.added, entry.deleted)
		totalChurn += churn
	}
	if totalAdded > 0 {
		w.ChurnRatio = float64(totalChurn) / float64(totalAdded)
	}

	// Top churn files
	churnAdded = totalChurn
	_ = churnAdded
	type churnEntry struct {
		path  string
		churn int
		entry *fileChurn
	}
	var sorted []churnEntry
	for path, e := range churnMap {
		ch := min(e.added, e.deleted)
		if ch > 0 {
			sorted = append(sorted, churnEntry{path, ch, e})
		}
	}
	// Sort by churn descending
	slices.SortFunc(sorted, func(a, b churnEntry) int {
		return cmp.Compare(b.churn, a.churn)
	})
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	for i := 0; i < limit; i++ {
		w.ChurnFilesTop = append(w.ChurnFilesTop, ChurnFile{
			Path:    sorted[i].path,
			Added:   sorted[i].entry.added,
			Deleted: sorted[i].entry.deleted,
			Commits: sorted[i].entry.commits,
		})
	}

	// ── Revert detection ──
	var revertCount int
	for _, c := range data.Commits {
		lower := strings.ToLower(c.Subject)
		if strings.Contains(lower, "revert") || strings.Contains(lower, "rollback") {
			revertCount++
		}
	}
	w.RevertCount = revertCount
	w.RevertRate = safeRatio(revertCount, len(data.Commits))

	// ── Abandoned branches ──
	now := time.Now()
	cutoff := now.AddDate(0, 0, -30)
	var abandoned int
	for _, b := range data.Branches {
		if b.LastCommit != nil && b.LastCommit.Before(cutoff) {
			abandoned++
		}
	}
	w.AbandonedBranches = abandoned

	return w
}
