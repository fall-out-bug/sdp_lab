package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/policy"
)

func populateEvidence(root string, picked *issue, branch string, decision policy.DecisionResponse) (string, error) {
	tmpl, err := loadEvidenceTemplate(root)
	if err != nil {
		return "", err
	}
	intent, ok := tmpl["intent"].(map[string]any)
	if !ok {
		return "", errors.New("invalid evidence template: intent")
	}
	intent["issue_id"] = picked.ID
	intent["trigger"] = "agent"
	intent["risk_class"] = decision.RiskClass

	execSection, ok := tmpl["execution"].(map[string]any)
	if !ok {
		return "", errors.New("invalid evidence template: execution")
	}
	execSection["branch"] = branch
	execSection["claimed_issue_ids"] = []string{picked.ID}

	boundary, ok := tmpl["boundary"].(map[string]any)
	if !ok {
		return "", errors.New("invalid evidence template: boundary")
	}
	declared, _ := boundary["declared"].(map[string]any)
	if declared == nil {
		declared = map[string]any{}
		boundary["declared"] = declared
	}
	declared["allowed_path_prefixes"] = allowedPrefixesFromLabels(picked.Labels)
	declared["control_path_prefixes"] = []string{".beads/", ".sdp/"}
	declared["forbidden_path_prefixes"] = []string{".git/"}
	declared["role"] = "builder"
	declared["lane"] = decision.Lane

	observed, _ := boundary["observed"].(map[string]any)
	if observed == nil {
		observed = map[string]any{}
		boundary["observed"] = observed
	}
	observed["touched_paths"] = []string{}
	observed["out_of_boundary_paths"] = []string{}

	compliance, _ := boundary["compliance"].(map[string]any)
	if compliance == nil {
		compliance = map[string]any{}
		boundary["compliance"] = compliance
	}
	compliance["ok"] = true
	compliance["reason"] = "declared boundary initialized"

	provenance, ok := tmpl["provenance"].(map[string]any)
	if !ok {
		return "", errors.New("invalid evidence template: provenance")
	}
	provenance["run_id"] = picked.ID
	provenance["orchestrator"] = "autonomy-worker"
	provenance["runtime"] = os.Getenv("SDP_RUNTIME")
	provenance["model"] = decision.SelectedModel
	provenance["gate_results"] = []string{"policy:allow"}
	provenance["phase"] = "intake"
	provenance["role"] = "builder"
	provenance["captured_at"] = time.Now().UTC().Format(time.RFC3339)
	provenance["source_issue_id"] = picked.ID
	provenance["artifact_id"] = picked.ID + ":strict-evidence"
	provenance["contract_version"] = "artifact-provenance/v1"
	provenance["hash_algorithm"] = "sha256"
	provenance["sequence"] = 0
	provenance["payload_digest"] = ""
	provenance["hash"] = ""
	provenance["hash_prev"] = ""

	trace, ok := tmpl["trace"].(map[string]any)
	if !ok {
		return "", errors.New("invalid evidence template: trace")
	}
	trace["branch"] = branch
	trace["beads_ids"] = []string{picked.ID}

	evidencePath := filepath.Join(root, ".sdp", "evidence", picked.ID+".json")
	if err := writeJSON(evidencePath, tmpl); err != nil {
		return "", err
	}
	return evidencePath, nil
}
