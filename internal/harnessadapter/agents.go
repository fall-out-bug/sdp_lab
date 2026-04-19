package harnessadapter

import (
	"fmt"
	"sort"
	"strings"

	"sdp_dev/internal/rules"
	"sdp_dev/internal/scout"
)

const agentsAdapterName = "agents"

type agentsAdapter struct{}

func newAgentsAdapter() Adapter { return &agentsAdapter{} }

func (a *agentsAdapter) Name() string { return agentsAdapterName }

// Render produces an AGENTS.md rules section snippet.
func (a *agentsAdapter) Render(card *scout.ProjectCard, rlist []rules.Rule) ([]byte, error) {
	var b strings.Builder

	if card != nil {
		writeAgentsConventions(&b, &card.Conventions)
	}
	writeAgentsRules(&b, rlist)

	return []byte(b.String()), nil
}

func writeAgentsConventions(b *strings.Builder, conv *scout.Conventions) {
	if conv == nil {
		return
	}

	patterns := conv.ModulePatterns
	if len(patterns) == 0 {
		return
	}

	sorted := make([]scout.ModulePattern, len(patterns))
	copy(sorted, patterns)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	b.WriteString("## Code Conventions\n\n")
	for _, mp := range sorted {
		fmt.Fprintf(b, "- **%s** (`%s`)\n", mp.Name, mp.Pattern)
	}
	b.WriteByte('\n')
}

func writeAgentsRules(b *strings.Builder, rlist []rules.Rule) {
	if len(rlist) == 0 {
		return
	}

	sorted := make([]rules.Rule, len(rlist))
	copy(sorted, rlist)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	b.WriteString("## Observed Rules\n\n")
	for _, r := range sorted {
		fmt.Fprintf(b, "- **%s** [%s]: %s\n", r.ID, r.Severity, r.Title)
		if r.Description != "" {
			fmt.Fprintf(b, "  %s\n", r.Description)
		}
	}
	b.WriteByte('\n')
}
