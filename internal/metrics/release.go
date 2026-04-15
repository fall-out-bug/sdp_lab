package metrics

import (
	"strings"
	"time"
)

// AnalyzeReleaseQuality computes release quality metrics from tags and commits.
func AnalyzeReleaseQuality(data *GitData) *ReleaseQuality {
	if data == nil {
		return nil
	}
	rq := &ReleaseQuality{}

	// Collect semver tags
	var semverTags []TagInfo
	for _, tag := range data.Tags {
		if tag.IsSemver {
			semverTags = append(semverTags, tag)
		}
	}
	rq.ReleasesAnalyzed = len(semverTags)

	if len(semverTags) == 0 || len(data.Commits) == 0 {
		return rq
	}

	// For each release tag, count fixes in windows after the tag
	var ttfhSum float64
	var ttfhCount int

	for i, tag := range semverTags {
		ri := ReleaseInfo{
			Tag:  tag.Tag,
			Date: tag.Date,
		}

		// Find the next tag's date for window end, or use tag+60d
		var windowEnd time.Time
		if i+1 < len(semverTags) {
			windowEnd = semverTags[i+1].Date
		} else {
			windowEnd = tag.Date.AddDate(0, 0, 60)
		}

		var firstFixH float64 = -1
		for _, c := range data.Commits {
			if c.Date.Before(tag.Date) || c.Date.After(windowEnd) {
				continue
			}
			lower := toLower(c.Subject)
			isFix := strings.Contains(lower, "fix") || strings.Contains(lower, "bugfix") || strings.Contains(lower, "patch")
			if !isFix {
				continue
			}
			hours := c.Date.Sub(tag.Date).Hours()
			days := hours / 24

			if days <= 7 {
				ri.Fixes7d++
			}
			if days <= 14 {
				ri.Fixes14d++
			}
			if days <= 30 {
				ri.Fixes30d++
			}
			if firstFixH < 0 || hours < firstFixH {
				firstFixH = hours
			}
		}

		if firstFixH >= 0 {
			ri.TimeToFirstFixH = firstFixH
			ttfhSum += firstFixH
			ttfhCount++
		}

		rq.Releases = append(rq.Releases, ri)
	}

	if ttfhCount > 0 {
		rq.AvgTimeToFirstHotfixH = ttfhSum / float64(ttfhCount)
	}

	return rq
}

// AnalyzeStabilization computes release stabilization from semver tags.
func AnalyzeStabilization(data *GitData) *Stabilization {
	if data == nil {
		return nil
	}
	s := &Stabilization{}

	// Collect semver tags
	var semverTags []TagInfo
	for _, tag := range data.Tags {
		if tag.IsSemver {
			semverTags = append(semverTags, tag)
		}
	}
	if len(semverTags) == 0 {
		return s
	}

	// Group by base version (major.minor)
	type baseInfo struct {
		base      string
		patches   int
		stableAt  int
		lastPatch string
	}
	baseMap := make(map[string]*baseInfo)
	var baseOrder []string

	for _, tag := range semverTags {
		base := semverBase(tag.Tag)
		if base == "" {
			continue
		}
		info, ok := baseMap[base]
		if !ok {
			info = &baseInfo{base: base}
			baseMap[base] = info
			baseOrder = append(baseOrder, base)
		}
		info.patches++
		info.lastPatch = tag.Tag

		// Check if this patch is stable: <=1 fix in 14 days after tag
		if info.stableAt == 0 {
			fixCount := countFixesAfter(data, tag.Date, tag.Date.AddDate(0, 0, 14))
			if fixCount <= 1 {
				info.stableAt = info.patches
			}
		}
	}

	var totalStable float64
	var stableCount int
	for _, base := range baseOrder {
		info := baseMap[base]
		sa := info.stableAt
		if sa == 0 {
			sa = info.patches // never stabilized
		}
		s.Releases = append(s.Releases, StabilizedRelease{
			Base:              info.base,
			StabilizedAtPatch: sa,
			PatchesTotal:      info.patches,
		})
		totalStable += float64(sa)
		stableCount++
	}

	if stableCount > 0 {
		s.AvgPatchesToStable = totalStable / float64(stableCount)
	}

	// Trend: compare last half vs first half
	if len(baseOrder) >= 2 {
		s.Trend = computeTrend(s.Releases)
	} else {
		s.Trend = "unknown"
	}

	return s
}

func countFixesAfter(data *GitData, start, end time.Time) int {
	var count int
	for _, c := range data.Commits {
		if c.Date.Before(start) || c.Date.After(end) {
			continue
		}
		lower := toLower(c.Subject)
		if strings.Contains(lower, "fix") || strings.Contains(lower, "bugfix") || strings.Contains(lower, "patch") {
			count++
		}
	}
	return count
}

func semverBase(tag string) string {
	// Extract major.minor from semver tag like v1.18.3
	s := tag
	if len(s) > 0 && s[0] == 'v' {
		s = s[1:]
	}
	// Find first two dots
	dots := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			dots++
			if dots == 2 {
				return s[:i]
			}
		}
	}
	return ""
}

func computeTrend(releases []StabilizedRelease) string {
	n := len(releases)
	if n < 2 {
		return "unknown"
	}
	first := float64(releases[0].StabilizedAtPatch)
	last := float64(releases[n-1].StabilizedAtPatch)
	diff := last - first
	if diff < -0.5 {
		return "improving"
	}
	if diff > 0.5 {
		return "degrading"
	}
	return "stable"
}
