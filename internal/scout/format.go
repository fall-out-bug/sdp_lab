package scout

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FormatJSON returns the ProjectCard as pretty-printed JSON.
func FormatJSON(card *ProjectCard) (string, error) {
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal project card: %w", err)
	}
	return string(data), nil
}

// FormatText returns a human-readable summary of the project card.
func FormatText(card *ProjectCard) string {
	var b strings.Builder

	name := card.Identity.Name
	lang := card.Identity.PrimaryLanguage
	loc := formatLOC(card.Scale.TotalLoc)
	files := card.Scale.TotalFiles

	fmt.Fprintf(&b, " %s — %s project (%s LOC, %d files)\n", name, lang, loc, files)
	b.WriteString(" ─────────────────────────────────────────\n")

	// Languages
	var langs []string
	for l, s := range card.Identity.Languages {
		langs = append(langs, fmt.Sprintf("%s %.0f%%", l, s.Ratio*100))
	}
	fmt.Fprintf(&b, " Languages:  %s\n", strings.Join(langs, " | "))

	// Build
	if card.Identity.BuildSystem != nil {
		fmt.Fprintf(&b, " Build:      %s (%s)\n", *card.Identity.BuildSystem,
			strings.Join(card.Identity.BuildFiles, ", "))
	}

	// Tests
	fmt.Fprintf(&b, " Tests:      %d test files (%.0f%% ratio)\n",
		card.Scale.TestFiles, card.Scale.TestRatio*100)

	// Activity
	fmt.Fprintf(&b, " Activity:   %d commits, %d contributors",
		card.Activity.TotalCommits, card.Activity.Contributors)
	if card.Activity.Commits30d > 0 {
		fmt.Fprintf(&b, ", %d in last 30d", card.Activity.Commits30d)
	}
	b.WriteByte('\n')

	// Age
	if card.Activity.FirstCommit != nil && card.Activity.LastCommit != nil {
		fmt.Fprintf(&b, " Age:        %d months (%s – %s)\n",
			card.Activity.AgeMonths, *card.Activity.FirstCommit, *card.Activity.LastCommit)
	}

	// Maturity
	var matFlags []string
	if card.Maturity.HasReadme { matFlags = append(matFlags, "README") }
	if card.Maturity.HasCI { matFlags = append(matFlags, "CI") }
	if card.Maturity.HasTests { matFlags = append(matFlags, "Tests") }
	if card.Maturity.HasDocker { matFlags = append(matFlags, "Docker") }
	if card.Maturity.HasReleases && card.Maturity.LatestRelease != nil {
		matFlags = append(matFlags, "Releases("+*card.Maturity.LatestRelease+")")
	}
	if len(matFlags) > 0 {
		fmt.Fprintf(&b, " Maturity:   %s\n", strings.Join(matFlags, " "))
	}

	// Health
	fmt.Fprintf(&b, " Health:     Bus factor ~%d | %s | %s complexity\n",
		card.Health.BusFactorEstimate,
		titleCase(card.Health.Staleness),
		titleCase(card.Health.ComplexityHint))

	// Entry points
	if len(card.Build.EntryPoints) > 0 {
		fmt.Fprintf(&b, " Entry:      %s\n", strings.Join(card.Build.EntryPoints, ", "))
	}

	return b.String()
}

// FormatCard returns a compact one-screen card summary.
func FormatCard(card *ProjectCard) string {
	var b strings.Builder
	loc := formatLOC(card.Scale.TotalLoc)
	fmt.Fprintf(&b, "%s | %s | %s LOC | %d files", card.Identity.Name, card.Identity.PrimaryLanguage, loc, card.Scale.TotalFiles)
	if card.Activity.TotalCommits > 0 {
		fmt.Fprintf(&b, " | %d commits", card.Activity.TotalCommits)
	}
	b.WriteByte('\n')

	if card.Identity.BuildSystem != nil {
		fmt.Fprintf(&b, "build:%s", *card.Identity.BuildSystem)
	}
	fmt.Fprintf(&b, " tests:%d(%.0f%%)", card.Scale.TestFiles, card.Scale.TestRatio*100)
	fmt.Fprintf(&b, " health:%s/%s\n", card.Health.Staleness, card.Health.ComplexityHint)

	return b.String()
}

// WriteArtifact writes the ProjectCard as scout.json into the given directory.
// Creates the directory if it does not exist. Returns the written file path.
func WriteArtifact(dir string, card *ProjectCard) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create artifact dir %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(card, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal artifact: %w", err)
	}
	path := filepath.Join(dir, "scout.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write artifact: %w", err)
	}
	return path, nil
}

func formatLOC(loc int64) string {
	if loc >= 1000 {
		return fmt.Sprintf("%.1fK", float64(loc)/1000)
	}
	return fmt.Sprintf("%d", loc)
}

func titleCase(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
