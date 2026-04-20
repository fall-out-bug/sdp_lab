package harnessadapter

import (
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/rules"
	"sdp_dev/internal/scout"
)

const claudeAdapterName = "claude-code"

type claudeAdapter struct{}

func newClaudeAdapter() Adapter { return &claudeAdapter{} }

func (c *claudeAdapter) Name() string { return claudeAdapterName }

// Render produces a CLAUDE.md snippet with Language Patterns and Rules sections.
func (c *claudeAdapter) Render(card *scout.ProjectCard, rlist []rules.Rule) ([]byte, error) {
	var b strings.Builder

	if card != nil {
		writeConventions(&b, &card.Conventions)
	}
	writeRulesSection(&b, rlist)

	return []byte(b.String()), nil
}

func writeConventions(b *strings.Builder, conv *scout.Conventions) {
	if conv == nil {
		return
	}

	hasContent := false

	patterns := conv.ModulePatterns
	if len(patterns) > 0 {
		hasContent = true
	}

	testStyle := conv.TestStructure.Style
	if testStyle != "" && testStyle != "unknown" {
		hasContent = true
	}

	if !hasContent {
		return
	}

	b.WriteString("## Language Patterns\n\n")

	if len(patterns) > 0 {
		sorted := make([]scout.ModulePattern, len(patterns))
		copy(sorted, patterns)
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].Name < sorted[j].Name
		})
		for _, mp := range sorted {
			fmt.Fprintf(b, "### %s\n", mp.Name)
			fmt.Fprintf(b, "- Pattern: `%s`\n", mp.Pattern)
			if len(mp.Examples) > 0 {
				b.WriteString("- Examples:\n")
				for _, ex := range mp.Examples {
					fmt.Fprintf(b, "  - `%s`\n", ex)
				}
			}
			b.WriteByte('\n')
		}
	}

	if testStyle != "" && testStyle != "unknown" {
		b.WriteString("### Test Layout\n")
		fmt.Fprintf(b, "- Style: %s\n", testStyle)
		if conv.TestStructure.DirPattern != "" {
			fmt.Fprintf(b, "- Directory pattern: `%s`\n", conv.TestStructure.DirPattern)
		}
		b.WriteByte('\n')
	}
}

func writeRulesSection(b *strings.Builder, rlist []rules.Rule) {
	if len(rlist) == 0 {
		return
	}

	sorted := make([]rules.Rule, len(rlist))
	copy(sorted, rlist)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	b.WriteString("## Rules\n\n")
	for _, r := range sorted {
		fmt.Fprintf(b, "### %s: %s\n", r.ID, r.Title)
		if r.Description != "" {
			fmt.Fprintf(b, "- Description: %s\n", r.Description)
		}
		fmt.Fprintf(b, "- Severity: %s\n", r.Severity)
		if r.EvidenceRef != "" {
			fmt.Fprintf(b, "- Evidence: %s\n", r.EvidenceRef)
		}
		b.WriteByte('\n')
	}
}
