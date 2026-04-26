package index

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// GenerateManifest builds the manifest.md content from the store.
// The output targets <=2K tokens for always-on agent context.
func GenerateManifest(s ManifestStore) (string, error) {
	data, err := collectManifestData(s)
	if err != nil {
		return "", fmt.Errorf("manifest: collect data: %w", err)
	}
	return renderManifest(data), nil
}

// WriteManifest generates the manifest and writes it to dir/manifest.md.
// Creates the directory if it does not exist. Returns the written file path.
func WriteManifest(dir string, s ManifestStore) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("manifest: create dir %s: %w", dir, err)
	}
	content, err := GenerateManifest(s)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "manifest.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("manifest: write: %w", err)
	}
	return path, nil
}

// collectManifestData reads all necessary data from the store.
func collectManifestData(s ManifestStore) (*ManifestData, error) {
	meta, err := s.LoadMeta(
		"repo_name", "languages", "arch_style",
		"commit_style", "build_system",
		"last_commit_date", "last_author", "active_branches",
		"summary",
	)
	if err != nil {
		return nil, err
	}

	modules, err := s.LoadModules()
	if err != nil {
		return nil, err
	}

	entryPoints, err := s.LoadEntryPoints()
	if err != nil {
		return nil, err
	}

	// Sort modules by LOC descending
	sort.Slice(modules, func(i, j int) bool {
		return modules[i].Loc > modules[j].Loc
	})

	data := &ManifestData{
		RepoName:        meta["repo_name"],
		PrimaryLanguage: primaryLang(meta["languages"]),
		ArchStyle:       meta["arch_style"],
		Summary:         meta["summary"],
		Modules:         modules,
		EntryPoints:     entryPoints,
		Conventions: ConventionSet{
			CommitStyle: meta["commit_style"],
			BuildSystem: meta["build_system"],
		},
		ActiveWork: ActiveWork{
			LastCommitDate: meta["last_commit_date"],
			LastAuthor:     meta["last_author"],
		},
	}

	if v := meta["active_branches"]; v != "" {
		_, _ = fmt.Sscanf(v, "%d", &data.ActiveWork.ActiveBranches)
	}

	return data, nil
}

// renderManifest produces the markdown string from ManifestData.
func renderManifest(d *ManifestData) string {
	var b strings.Builder

	// Header
	repoName := d.RepoName
	if repoName == "" {
		repoName = "unknown"
	}
	header := repoName
	if d.PrimaryLanguage != "" {
		header += " — " + d.PrimaryLanguage
	}
	if d.ArchStyle != "" {
		header += " " + d.ArchStyle
	}
	fmt.Fprintf(&b, "# %s\n\n", header)

	// Summary
	if d.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", d.Summary)
	}

	// Modules
	fmt.Fprintf(&b, "## Modules (%d)\n", len(d.Modules))
	if len(d.Modules) > 0 {
		b.WriteString("| Module | Purpose | LOC | Owner | Bus Factor |\n")
		b.WriteString("|--------|---------|-----|-------|------------|\n")
		for _, m := range d.Modules {
			purpose := m.Purpose
			if purpose == "" {
				purpose = "-"
			}
			owner := m.Owner
			if owner == "" {
				owner = "-"
			}
			busFactor := "-"
			if m.BusFactor > 0 {
				busFactor = fmt.Sprintf("%d", m.BusFactor)
			}
			fmt.Fprintf(&b, "| %s | %s | %d | %s | %s |\n",
				m.Name, purpose, m.Loc, owner, busFactor)
		}
	}
	b.WriteByte('\n')

	// Entry Points
	if len(d.EntryPoints) > 0 {
		b.WriteString("## Entry Points\n")
		for _, ep := range d.EntryPoints {
			fmt.Fprintf(&b, "- `%s`\n", ep)
		}
		b.WriteByte('\n')
	}

	// Conventions
	convs := d.Conventions
	if convs.CommitStyle != "" || convs.BuildSystem != "" || len(convs.KeyPatterns) > 0 {
		b.WriteString("## Conventions\n")
		if convs.CommitStyle != "" {
			fmt.Fprintf(&b, "- Commit style: %s\n", convs.CommitStyle)
		}
		if convs.TestFramework != "" {
			fmt.Fprintf(&b, "- Test framework: %s\n", convs.TestFramework)
		}
		if convs.BuildSystem != "" {
			fmt.Fprintf(&b, "- Build system: %s\n", convs.BuildSystem)
		}
		for _, p := range convs.KeyPatterns {
			fmt.Fprintf(&b, "- %s\n", p)
		}
		b.WriteByte('\n')
	}

	// Active Work
	aw := d.ActiveWork
	if aw.LastCommitDate != "" || aw.LastAuthor != "" || aw.ActiveBranches > 0 || aw.OpenIssues > 0 {
		b.WriteString("## Active Work\n")
		if aw.LastCommitDate != "" {
			author := aw.LastAuthor
			if author == "" {
				author = "unknown"
			}
			fmt.Fprintf(&b, "- Last commit: %s by %s\n", aw.LastCommitDate, author)
		}
		if aw.ActiveBranches > 0 {
			fmt.Fprintf(&b, "- Active branches: %d\n", aw.ActiveBranches)
		}
		if aw.OpenIssues > 0 {
			fmt.Fprintf(&b, "- Open issues: %d\n", aw.OpenIssues)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

// primaryLang extracts the first language from a comma-separated list.
func primaryLang(langs string) string {
	if langs == "" {
		return ""
	}
	parts := strings.Split(langs, ",")
	return strings.TrimSpace(parts[0])
}
