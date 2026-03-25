package control

import (
	"fmt"
)

// DeployPhase represents a deployment phase.
type DeployPhase string

const (
	DeployPhaseStaging DeployPhase = "staging"
	DeployPhaseProd    DeployPhase = "prod"
)

// DeployGate creates gates for a deployment phase.
// Creates: gate:ci + gate:human:{phase} under the parent card.
// Returns gate IDs for later resolution.
func (s *Store) DeployGate(cardID string, phase DeployPhase) (ciGateID, humanGateID string, err error) {
	if s.beadsRepo == nil {
		return "", "", fmt.Errorf("deploy gates require beads/dual mode (current: %s)", s.RepoMode)
	}

	// Create CI gate
	ciGateID, err = s.beadsRepo.CreateGate(cardID, fmt.Sprintf("ci-%s", phase))
	if err != nil {
		return "", "", fmt.Errorf("create ci gate: %w", err)
	}

	// Create human gate
	humanGateID, err = s.beadsRepo.CreateGate(cardID, fmt.Sprintf("human:%s-approve", phase))
	if err != nil {
		return "", "", fmt.Errorf("create human gate: %w", err)
	}

	return ciGateID, humanGateID, nil
}

// DeployPhaseTransition moves a card through a deploy phase:
// 1. Check CI gate (if auto-check supported)
// 2. Create human approval gate
// Returns the human gate ID that needs approval.
func (s *Store) DeployPhaseTransition(cardID string, phase DeployPhase) (humanGateID string, err error) {
	_, humanGateID, err = s.DeployGate(cardID, phase)
	if err != nil {
		return "", err
	}

	// Set deploy phase state in Beads
	if s.beadsRepo != nil {
		err = s.beadsRepo.SetState(cardID, "deploy_phase", string(phase),
			fmt.Sprintf("entering %s deploy phase", phase))
		if err != nil {
			return "", fmt.Errorf("set deploy phase: %w", err)
		}
	}

	return humanGateID, nil
}

// RecordDeployEvidence stores deploy evidence in Beads metadata.
func (s *Store) RecordDeployEvidence(cardID string, phase DeployPhase, evidence map[string]any) error {
	if s.beadsRepo == nil {
		return fmt.Errorf("record deploy evidence requires beads mode")
	}

	meta := map[string]any{
		fmt.Sprintf("deploy_%s", phase): evidence,
	}
	return s.beadsRepo.UpdateMetadata(cardID, meta)
}

// CreateDeploySubtask creates a deploy subtask under a parent feature.
func (s *Store) CreateDeploySubtask(parentID string, phase DeployPhase, title string) (string, error) {
	if s.beadsRepo == nil {
		return "", fmt.Errorf("create deploy subtask requires beads mode")
	}

	args := []string{
		"create", title,
		"--type", "chore",
		"--parent", parentID,
		"--labels", fmt.Sprintf("sdp:deploy-%s", phase),
		"--priority", "0",
		"--silent",
	}

	data, err := s.beadsRepo.runBDWrite(args...)
	if err != nil {
		return "", fmt.Errorf("create deploy subtask: %w", err)
	}

	return trimOutput(data), nil
}

func trimOutput(data []byte) string {
	s := string(data)
	// bd --silent may return JSON or plain ID
	if len(s) > 50 {
		return s
	}
	return s
}
