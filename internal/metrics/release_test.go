package metrics

import (
	"testing"
	"time"
)

func TestAnalyzeReleaseQualityNil(t *testing.T) {
	if AnalyzeReleaseQuality(nil) != nil {
		t.Fatal("expected nil for nil input")
	}
}

func TestAnalyzeReleaseQualityEmpty(t *testing.T) {
	rq := AnalyzeReleaseQuality(&GitData{})
	if rq == nil {
		t.Fatal("expected non-nil for empty data")
	}
	if rq.ReleasesAnalyzed != 0 {
		t.Fatalf("expected 0 releases got %d", rq.ReleasesAnalyzed)
	}
}

func TestAnalyzeReleaseQualitySemverOnly(t *testing.T) {
	now := time.Now()
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: now.Add(-48 * time.Hour), IsSemver: true},
			{Tag: "not-semver", Date: now.Add(-24 * time.Hour), IsSemver: false},
		},
		Commits: []RawCommit{},
	}
	rq := AnalyzeReleaseQuality(data)
	if rq.ReleasesAnalyzed != 1 {
		t.Fatalf("expected 1 release analyzed got %d", rq.ReleasesAnalyzed)
	}
}

func TestAnalyzeReleaseQualityFixesInWindow(t *testing.T) {
	tagDate := time.Now().Add(-10 * 24 * time.Hour)
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: tagDate, IsSemver: true},
		},
		Commits: []RawCommit{
			{Subject: "fix: urgent bug", Date: tagDate.Add(3 * 24 * time.Hour)},   // 3d → 7d window
			{Subject: "fix: another bug", Date: tagDate.Add(10 * 24 * time.Hour)},  // 10d → 14d window
			{Subject: "fix: later bug", Date: tagDate.Add(20 * 24 * time.Hour)},    // 20d → 30d window
			{Subject: "feat: new thing", Date: tagDate.Add(5 * 24 * time.Hour)},    // not a fix
		},
	}
	rq := AnalyzeReleaseQuality(data)
	if len(rq.Releases) != 1 {
		t.Fatalf("expected 1 release got %d", len(rq.Releases))
	}
	ri := rq.Releases[0]
	if ri.Fixes7d != 1 {
		t.Fatalf("expected 1 fix in 7d got %d", ri.Fixes7d)
	}
	if ri.Fixes14d != 2 {
		t.Fatalf("expected 2 fixes in 14d got %d", ri.Fixes14d)
	}
	if ri.Fixes30d != 3 {
		t.Fatalf("expected 3 fixes in 30d got %d", ri.Fixes30d)
	}
}

func TestAnalyzeReleaseQualityTimeToFirstFix(t *testing.T) {
	tagDate := time.Now().Add(-10 * 24 * time.Hour)
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: tagDate, IsSemver: true},
		},
		Commits: []RawCommit{
			{Subject: "fix: quick patch", Date: tagDate.Add(6 * time.Hour)},
			{Subject: "fix: slow patch", Date: tagDate.Add(48 * time.Hour)},
		},
	}
	rq := AnalyzeReleaseQuality(data)
	if rq.AvgTimeToFirstHotfixH != 6.0 {
		t.Fatalf("expected avg_ttfh=6.0 got %.1f", rq.AvgTimeToFirstHotfixH)
	}
	if len(rq.Releases) != 1 {
		t.Fatalf("expected 1 release got %d", len(rq.Releases))
	}
	if rq.Releases[0].TimeToFirstFixH != 6.0 {
		t.Fatalf("expected ttfh=6.0 got %.1f", rq.Releases[0].TimeToFirstFixH)
	}
}

func TestAnalyzeReleaseQualityMultipleReleases(t *testing.T) {
	t1 := time.Now().Add(-60 * 24 * time.Hour)
	t2 := time.Now().Add(-30 * 24 * time.Hour)
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: t1, IsSemver: true},
			{Tag: "v2.0.0", Date: t2, IsSemver: true},
		},
		Commits: []RawCommit{
			{Subject: "fix: bug after v1", Date: t1.Add(5 * 24 * time.Hour)},
			{Subject: "fix: bug after v2", Date: t2.Add(10 * 24 * time.Hour)},
		},
	}
	rq := AnalyzeReleaseQuality(data)
	if rq.ReleasesAnalyzed != 2 {
		t.Fatalf("expected 2 releases got %d", rq.ReleasesAnalyzed)
	}
	if len(rq.Releases) != 2 {
		t.Fatalf("expected 2 release infos got %d", len(rq.Releases))
	}
	avg := (5*24 + 10*24) / 2.0
	if rq.AvgTimeToFirstHotfixH != avg {
		t.Fatalf("expected avg_ttfh=%.1f got %.1f", avg, rq.AvgTimeToFirstHotfixH)
	}
}

func TestAnalyzeReleaseQualityNoFixes(t *testing.T) {
	tagDate := time.Now().Add(-10 * 24 * time.Hour)
	data := &GitData{
		Tags: []TagInfo{
			{Tag: "v1.0.0", Date: tagDate, IsSemver: true},
		},
		Commits: []RawCommit{
			{Subject: "feat: new feature", Date: tagDate.Add(5 * 24 * time.Hour)},
		},
	}
	rq := AnalyzeReleaseQuality(data)
	if rq.AvgTimeToFirstHotfixH != 0 {
		t.Fatalf("expected avg_ttfh=0 for no fixes got %.1f", rq.AvgTimeToFirstHotfixH)
	}
}
