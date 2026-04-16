package metrics

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// AnalyzeGitFlow detects the branch model from git history.
func AnalyzeGitFlow(data *GitData) *GitFlow {
	if data == nil {
		return nil
	}
	gf := &GitFlow{}

	// ── Score each model ──
	trunkScore := 0
	ghFlowScore := 0
	gitFlowScore := 0

	// Signal: branch naming
	hasRelease := false
	hasDevelop := false
	hasHotfix := false
	featureFromMain := 0
	totalBranches := 0

	for _, b := range data.Branches {
		totalBranches++
		name := b.Name
		if strings.Contains(name, "release/") {
			hasRelease = true
		}
		if strings.Contains(name, "develop") || strings.Contains(name, "dev") {
			hasDevelop = true
		}
		if strings.Contains(name, "hotfix/") {
			hasHotfix = true
		}
		if strings.Contains(name, "main") || strings.Contains(name, "master") {
			continue
		}
		if !strings.Contains(name, "release/") && !strings.Contains(name, "hotfix/") && !strings.Contains(name, "develop") {
			featureFromMain++
		}
	}

	// Signal: merge frequency per week
	var mergeFreq float64
	if len(data.Commits) > 0 {
		first := data.Commits[len(data.Commits)-1].Date
		last := data.Commits[0].Date
		weeks := last.Sub(first).Hours() / (24 * 7)
		if weeks > 0 {
			mergeFreq = float64(data.MergeCount) / weeks
		}
	}
	gf.MergeFrequencyPerWeek = mergeFreq

	// Signal: branch lifetimes
	var lifetimes []float64
	now := time.Now()
	for _, b := range data.Branches {
		if b.LastCommit != nil {
			hours := now.Sub(*b.LastCommit).Hours()
			if hours > 0 && hours < 365*24 {
				lifetimes = append(lifetimes, hours)
			}
		}
	}
	if len(lifetimes) > 0 {
		slices.Sort(lifetimes)
		gf.BranchLifetimeMedianH = median(lifetimes)
		gf.BranchLifetimeP95H = percentile(lifetimes, 95)
	}

	// Long-lived branches (>7 days)
	longLived := 0
	for _, lt := range lifetimes {
		if lt > 7*24 {
			longLived++
		}
	}
	gf.LongLivedBranches = longLived

	// ── Score trunk-based ──
	if gf.BranchLifetimeMedianH > 0 && gf.BranchLifetimeMedianH < 3*24 {
		trunkScore += 3
	}
	if mergeFreq > 20 {
		trunkScore += 2
	}
	if totalBranches > 0 && featureFromMain*100/totalBranches > 80 {
		trunkScore += 2
	}

	// ── Score GitFlow ──
	if hasRelease {
		gitFlowScore += 3
	}
	if hasDevelop {
		gitFlowScore += 3
	}
	if hasHotfix {
		gitFlowScore += 2
	}
	if longLived > 0 {
		gitFlowScore += 2
	}

	// ── Score GitHub Flow ──
	if gf.BranchLifetimeMedianH >= 3*24 && gf.BranchLifetimeMedianH <= 7*24 {
		ghFlowScore += 1
	}
	if !hasRelease && !hasDevelop {
		ghFlowScore += 2
	}
	if mergeFreq > 10 && mergeFreq <= 20 {
		ghFlowScore += 1
	}

	// Determine winner
	maxScore := trunkScore
	model := "trunk-based"
	if ghFlowScore > maxScore {
		maxScore = ghFlowScore
		model = "github-flow"
	}
	if gitFlowScore > maxScore {
		maxScore = gitFlowScore
		model = "gitflow"
	}
	gf.DetectedModel = model

	// Confidence
	totalPossible := 7
	if maxScore > 0 && totalPossible > 0 {
		gf.Confidence = float64(maxScore) / float64(totalPossible)
	}

	// Evidence
	if gf.BranchLifetimeMedianH < 3*24 {
		gf.Evidence = append(gf.Evidence, "median branch lifetime <3 days")
	}
	if hasRelease {
		gf.Evidence = append(gf.Evidence, "release/* branches found")
	}
	if hasDevelop {
		gf.Evidence = append(gf.Evidence, "develop branch found")
	}
	if mergeFreq > 0 {
		gf.Evidence = append(gf.Evidence, fmt.Sprintf("%.1f", mergeFreq)+" merges/week")
	}

	return gf
}

func median(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// percentile returns the p-th percentile (0-100) from a sorted slice.
// Uses nearest-rank method with float index to avoid integer truncation.
func percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n == 1 {
		return sorted[0]
	}
	idx := int(float64(n) * p / 100)
	if idx >= n {
		idx = n - 1
	}
	return sorted[idx]
}
