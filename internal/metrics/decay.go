package metrics

import (
	"strings"
	"time"
)

// AnalyzeDecay computes code-decay metrics from raw git data.
func AnalyzeDecay(data *GitData) *Decay {
	if data == nil || len(data.Commits) == 0 {
		return nil
	}
	d := &Decay{}

	// ── Shotgun surgery ──
	// Commits where: files >= 5 AND directories >= 3 AND avg lines/file <= 20
	var shotgun int
	for _, c := range data.Commits {
		if len(c.Files) < 5 {
			continue
		}
		dirs := make(map[string]bool)
		var totalLines int
		for _, f := range c.Files {
			dirs[dirOf(f.Path)] = true
			totalLines += f.Added + f.Deleted
		}
		if len(dirs) < 3 {
			continue
		}
		avgLines := float64(totalLines) / float64(len(c.Files))
		if avgLines <= 20 {
			shotgun++
		}
	}
	d.ShotgunCommits = shotgun
	d.ShotgunSurgeryRatio = safeRatio(shotgun, len(data.Commits))

	// ── Fix recurrence ──
	// Files with >= 5 fix-classified commits
	type fileStats struct {
		fixCount     int
		totalCommits int
	}
	fileMap := make(map[string]*fileStats)
	for _, c := range data.Commits {
		lower := toLower(c.Subject)
		isFix := strings.Contains(lower, "fix") || strings.Contains(lower, "bugfix")
		for _, f := range c.Files {
			entry, ok := fileMap[f.Path]
			if !ok {
				entry = &fileStats{}
				fileMap[f.Path] = entry
			}
			entry.totalCommits++
			if isFix {
				entry.fixCount++
			}
		}
	}
	for path, st := range fileMap {
		if st.fixCount >= 5 {
			d.FixRecurrence = append(d.FixRecurrence, FixRecurrenceEntry{
				Path:         path,
				FixCount:     st.fixCount,
				TotalCommits: st.totalCommits,
				FixDensity:   safeRatio(st.fixCount, st.totalCommits),
			})
		}
	}

	// ── Monotonic growth ──
	// Files that keep growing without significant deletions
	type fileGrowth struct {
		firstAdded int
		currentNet int
		months     int
		firstDate  time.Time
		lastDate   time.Time
		hasRefactor bool
	}
	growMap := make(map[string]*fileGrowth)
	for _, c := range data.Commits {
		for _, f := range c.Files {
			entry, ok := growMap[f.Path]
			if !ok {
				entry = &fileGrowth{firstDate: c.Date, firstAdded: f.Added}
				growMap[f.Path] = entry
			}
			entry.lastDate = c.Date
			entry.currentNet += f.Added - f.Deleted
			// Refactoring event: significant deletion (>30% of added in this commit)
			if f.Added > 0 && f.Deleted > f.Added*3/10 {
				entry.hasRefactor = true
			}
		}
	}
	for path, g := range growMap {
		months := int(g.lastDate.Sub(g.firstDate).Hours() / (24 * 30))
		if months >= 6 && g.currentNet > 0 && !g.hasRefactor {
			d.MonotonicGrowthFiles = append(d.MonotonicGrowthFiles, MonotonicFile{
				Path:          path,
				MonthsGrowing: months,
				StartLOC:      g.firstAdded,
				CurrentLOC:    g.firstAdded + g.currentNet,
				ZeroRefactor:  true,
			})
		}
	}

	return d
}

func dirOf(path string) string {
	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
