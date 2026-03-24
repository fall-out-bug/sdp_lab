package control

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/harness"
)

func GenerateContractFromCard(card *FeatureCard) (*harness.TaskContract, error) {
	if card == nil {
		return nil, fmt.Errorf("card is required")
	}

	objective := strings.TrimSpace(card.NormalizedIntent)
	if objective == "" {
		objective = strings.TrimSpace(card.Title)
	}

	acceptanceCriteria := make([]harness.AcceptanceCriterion, 0, len(card.AcceptanceShape))
	for _, line := range card.AcceptanceShape {
		for _, part := range strings.Split(line, "\n") {
			statement := strings.TrimSpace(part)
			if statement == "" {
				continue
			}
			acceptanceCriteria = append(acceptanceCriteria, harness.AcceptanceCriterion{
				ID:        fmt.Sprintf("ac-%d", len(acceptanceCriteria)+1),
				Statement: statement,
				Priority:  "required",
			})
		}
	}

	requiredEvidence := append([]string(nil), cleanList(card.ScopeOut)...)

	return &harness.TaskContract{
		Version:            "v1",
		RunID:              card.ID,
		CreatedAt:          time.Now().UTC().Format(time.RFC3339),
		Objective:          objective,
		AcceptanceCriteria: acceptanceCriteria,
		RequiredEvidence:   requiredEvidence,
		QualityGates: harness.QualityGates{
			Build:     true,
			Test:      true,
			Lint:      true,
			Typecheck: true,
		},
		Constraints: harness.Constraints{
			AllowScopeReduction:  false,
			AllowMetricReduction: false,
			SecurityPolicy:       "",
		},
	}, nil
}

func (s *Store) writeGeneratedContract(card *FeatureCard) (string, error) {
	if s == nil {
		return "", fmt.Errorf("store is required")
	}
	contract, err := GenerateContractFromCard(card)
	if err != nil {
		return "", err
	}
	path := filepath.Join(s.ControlRoot, "contracts", card.ID+".json")
	if err := harness.SaveTaskContract(path, contract); err != nil {
		return "", err
	}
	return filepath.ToSlash(path), nil
}
