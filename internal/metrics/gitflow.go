package metrics

import "time"

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
		if contains(name, "release/") {
			hasRelease = true
		}
		if contains(name, "develop") || contains(name, "dev") {
			hasDevelop = true
		}
		if contains(name, "hotfix/") {
			hasHotfix = true
		}
		if contains(name, "main") || contains(name, "master") {
			continue
		}
		if !contains(name, "release/") && !contains(name, "hotfix/") && !contains(name, "develop") {
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
		sortFloats(lifetimes)
		gf.BranchLifetimeMedianH = median(lifetimes)
		gf.BranchLifetimeP95H = lifetimes[len(lifetimes)*95/100]
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
	if !hasRelease {
		trunkScore += 0
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
		gf.Evidence = append(gf.Evidence, ftoa(mergeFreq)+" merges/week")
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

func sortFloats(a []float64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

func ftoa(f float64) string {
	if f == float64(int(f)) {
		return itoa(int(f))
	}
	// Simple 1-decimal conversion
	whole := int(f)
	frac := int((f - float64(whole)) * 10)
	if frac < 0 {
		frac = -frac
	}
	return itoa(whole) + "." + string(rune('0'+frac))
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
