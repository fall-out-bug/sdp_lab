package metrics

import (
	"cmp"
	"slices"
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
		slices.SortFunc(sorted, func(a, b authorCommits) int {
			return cmp.Compare(b.count, a.count)
		})
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
	slices.SortFunc(overallSorted, func(a, b authorCommits) int {
		return cmp.Compare(b.count, a.count)
	})
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
	// Sort for O(n log n) formula
	sorted := make([]float64, n)
	copy(sorted, vals)
	slices.Sort(sorted)

	var sum float64
	for _, v := range sorted {
		sum += v
	}
	if sum == 0 {
		return 0
	}
	// Gini = (2 * sum((i+1)*x_i)) / (n * sum(x_i)) - (n+1)/n
	var weightedSum float64
	for i, v := range sorted {
		weightedSum += float64(i+1) * v
	}
	return (2*weightedSum)/(float64(n)*sum) - (float64(n)+1)/float64(n)
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
