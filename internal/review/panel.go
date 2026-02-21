package review

import (
	"context"
	"sync"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/federation"
	"sdp_dev/internal/roles"
)

// PanelConfig holds options for the review panel.
type PanelConfig struct {
	Personas []string // e.g. ["correctness", "security", "dx"]
}

// Panel dispatches to N persona reviewers and aggregates verdicts.
type Panel struct {
	Personas []string
}

// NewPanel creates a Panel.
func NewPanel(cfg PanelConfig) *Panel {
	if len(cfg.Personas) == 0 {
		cfg.Personas = []string{"correctness", "security", "dx"}
	}
	return &Panel{Personas: cfg.Personas}
}

// RunReviews dispatches to each persona and returns consensus.
func (p *Panel) RunReviews(ctx context.Context, task federation.FederatedTask, agentCtx *agent.Context) (ConsensusResult, error) {
	var verdicts []ReviewVerdict
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, personaID := range p.Personas {
		personaID := personaID
		wg.Add(1)
		go func() {
			defer wg.Done()
			strategy := roles.Get("reviewer-" + personaID)
			if strategy == nil {
				strategy = roles.Get("reviewer")
			}
			if strategy == nil {
				return
			}
			input := roles.TaskInput{
				FederatedTask: task,
				Ctx:          agentCtx,
			}
			res, err := strategy.Execute(ctx, input)
			if err != nil {
				mu.Lock()
				verdicts = append(verdicts, ReviewVerdict{
					PersonaID: personaID,
					Verdict:   "reject",
					Summary:   err.Error(),
				})
				mu.Unlock()
				return
			}
			mu.Lock()
			verdicts = append(verdicts, ReviewVerdict{
				PersonaID: personaID,
				Verdict:   res.Verdict,
				Summary:   res.Summary,
				Comments:  res.Comments,
			})
			mu.Unlock()
		}()
	}

	wg.Wait()
	return Consensus(verdicts), nil
}
