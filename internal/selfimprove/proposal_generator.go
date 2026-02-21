package selfimprove

import (
	"sdp_dev/internal/beads"
)

// ProposalGenerator creates Beads tasks from weakness patterns.
type ProposalGenerator struct {
	beads *beads.Adapter
}

// NewProposalGenerator returns a generator for the given work dir.
func NewProposalGenerator(workDir string) *ProposalGenerator {
	return &ProposalGenerator{beads: beads.NewAdapter(workDir)}
}

// Generate creates Beads tasks for the given patterns.
// Labels: autonomy, strict-evidence, workstream:self-improvement, risk:medium
func (g *ProposalGenerator) Generate(patterns []WeaknessPattern, maxItems int) ([]string, error) {
	if maxItems <= 0 {
		maxItems = 3
	}
	var created []string
	for i, p := range patterns {
		if i >= maxItems {
			break
		}
		title := SuggestImprovement(p)
		id, err := g.beads.Create(beads.CreateOpts{
			Title:       title,
			Type:        "task",
			Priority:    2,
			Description: p.Description,
			Labels:      []string{"autonomy", "strict-evidence", "workstream:self-improvement", "risk:medium"},
		})
		if err != nil {
			continue
		}
		created = append(created, id)
	}
	return created, nil
}
