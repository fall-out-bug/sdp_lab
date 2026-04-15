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
