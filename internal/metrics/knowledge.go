package metrics

import (
	"time"
)

type authorCommits struct {
	name  string
	count int
}

// AnalyzeKnowledge computes knowledge-risk metrics from raw git data.
func AnalyzeKnowledge(data *GitData) *KnowledgeRisk {
	if data == nil || len(data.Commits) == 0 {
		return nil
	}
	kr := &KnowledgeRisk{}

	// ── Per-module bus factor ──
	moduleCommits := make(map[string]map[string]int) // module -> author -> count
	moduleFiles := make(map[string]int)               // module -> unique file count

	for _, c := range data.Commits {
		mod := topModule(c.Files)
		if mod == "" {
			mod = "<root>"
		}
		if moduleCommits[mod] == nil {
			moduleCommits[mod] = make(map[string]int)
		}
		moduleCommits[mod][c.Author]++
		fileSet := make(map[string]bool)
		for _, f := range c.Files {
			fileSet[f.Path] = true
		}
		for p := range fileSet {
			_ = p
			moduleFiles[mod]++
		}
	}

	// Compute bus factor per module
	var allAuthors []authorCommits
	for mod, authors := range moduleCommits {
		var sorted []authorCommits
		total := 0
		for name, cnt := range authors {
			sorted = append(sorted, authorCommits{name, cnt})
			total += cnt
			allAuthors = append(allAuthors, authorCommits{name, cnt})
		}
		// Sort descending by count
		for i := 1; i < len(sorted); i++ {
			for j := i; j > 0 && sorted[j].count > sorted[j-1].count; j-- {
				sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
			}
		}
		bf := busFactor(sorted, total)
		var primary string
		var primaryRatio float64
		if len(sorted) > 0 {
			primary = sorted[0].name
			primaryRatio = float64(sorted[0].count) / float64(total)
		}
		kr.BusFactorByModule = append(kr.BusFactorByModule, ModuleRisk{
			Module:             mod,
			BusFactor:          bf,
			PrimaryAuthor:      primary,
			PrimaryAuthorRatio: primaryRatio,
			FilesCount:         moduleFiles[mod],
		})
	}

	// Overall bus factor
	authorTotals := make(map[string]int)
	for _, ac := range allAuthors {
		authorTotals[ac.name] += ac.count
	}
	var overallSorted []authorCommits
	grandTotal := 0
	for name, cnt := range authorTotals {
		overallSorted = append(overallSorted, authorCommits{name, cnt})
		grandTotal += cnt
	}
	for i := 1; i < len(overallSorted); i++ {
		for j := i; j > 0 && overallSorted[j].count > overallSorted[j-1].count; j-- {
			overallSorted[j], overallSorted[j-1] = overallSorted[j-1], overallSorted[j]
		}
	}
	kr.OverallBusFactor = busFactor(overallSorted, grandTotal)

	// Gini coefficient
	var vals []float64
	for _, cnt := range authorTotals {
		vals = append(vals, float64(cnt))
	}
	kr.GiniCoefficient = gini(vals)

	// ── Former contributors ──
	cutoff := time.Now().AddDate(0, -6, 0)
	active := make(map[string]bool)
	ever := make(map[string]bool)
	for _, c := range data.Commits {
		ever[c.Author] = true
		if !c.Date.Before(cutoff) {
			active[c.Author] = true
		}
	}
	var former []string
	for author := range ever {
		if !active[author] {
			former = append(former, author)
		}
	}
	kr.FormerContributors = former
	if len(ever) > 0 {
		kr.FormerContributorRatio = float64(len(former)) / float64(len(ever))
	}

	return kr
}

func busFactor(sorted []authorCommits, total int) int {
	if total == 0 {
		return 0
	}
	cumulative := 0
	for i, ac := range sorted {
		cumulative += ac.count
		if float64(cumulative) > float64(total)*0.5 {
			return i + 1
		}
	}
	return len(sorted)
}

func gini(vals []float64) float64 {
	n := len(vals)
	if n == 0 {
		return 0
	}
	var totalDiff, sum float64
	for _, v := range vals {
		sum += v
	}
	if sum == 0 {
		return 0
	}
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			if vals[i] > vals[j] {
				totalDiff += vals[i] - vals[j]
			} else {
				totalDiff += vals[j] - vals[i]
			}
		}
	}
	return totalDiff / (2 * float64(n) * sum)
}

func topModule(files []FileChange) string {
	for _, f := range files {
		for i := 0; i < len(f.Path); i++ {
			if f.Path[i] == '/' {
				return f.Path[:i]
			}
		}
		return f.Path
	}
	return ""
}
