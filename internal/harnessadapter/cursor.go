package harnessadapter

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fall-out-bug/sdp_lab/internal/rules"
	"github.com/fall-out-bug/sdp_lab/internal/scout"
)

const cursorAdapterName = "cursor"

type cursorAdapter struct{}

func newCursorAdapter() Adapter { return &cursorAdapter{} }

func (c *cursorAdapter) Name() string { return cursorAdapterName }

// Render produces a .cursorrules snippet with rules as comment-style entries.
func (c *cursorAdapter) Render(card *scout.ProjectCard, rlist []rules.Rule) ([]byte, error) {
	var b strings.Builder

	if card != nil && card.Conventions.ModulePatterns != nil {
		writeCursorConventions(&b, &card.Conventions)
	}

	writeCursorRules(&b, rlist)

	return []byte(b.String()), nil
}

func writeCursorConventions(b *strings.Builder, conv *scout.Conventions) {
	patterns := conv.ModulePatterns
	if len(patterns) == 0 {
		return
	}

	sorted := make([]scout.ModulePattern, len(patterns))
	copy(sorted, patterns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	b.WriteString("# Conventions\n")
	for _, mp := range sorted {
		fmt.Fprintf(b, "# %s: %s\n", mp.Name, mp.Pattern)
	}
	b.WriteByte('\n')
}

func writeCursorRules(b *strings.Builder, rlist []rules.Rule) {
	if len(rlist) == 0 {
		return
	}

	sorted := make([]rules.Rule, len(rlist))
	copy(sorted, rlist)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	for _, r := range sorted {
		fmt.Fprintf(b, "# %s: %s\n", r.ID, r.Title)
		fmt.Fprintf(b, "# Severity: %s\n", r.Severity)
		if r.Description != "" {
			fmt.Fprintf(b, "# %s\n", r.Description)
		}
		b.WriteByte('\n')
	}
}
