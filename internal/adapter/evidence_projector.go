package adapter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// EvidenceProjector projects operator outputs into strict SDP evidence schema.
type EvidenceProjector struct {
	workDir string
}

// NewEvidenceProjector returns a projector for the given working directory.
func NewEvidenceProjector(workDir string) *EvidenceProjector {
	return &EvidenceProjector{workDir: workDir}
}

// ProjectFromIntent builds an evidence envelope from TaskIntent and role outputs.
// Writes to .sdp/evidence/<issue_id>.json.
func (p *EvidenceProjector) ProjectFromIntent(intent *TaskIntent, roleOutputs map[string]string, runID string) (string, error) {
	if intent == nil {
		return "", fmt.Errorf("intent required")
	}
	path := filepath.Join(p.workDir, ".sdp", "evidence", intent.IssueID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}

	payload := map[string]any{
		"intent": map[string]any{
			"issue_id":    intent.IssueID,
			"trigger":     "adapter",
			"spec_hash":   intent.SpecHash,
			"prompt":      intent.Prompt,
			"objective":   intent.Objective,
		},
		"plan": map[string]any{
			"role_binding": intent.AgentRef,
			"run_id":       runID,
		},
		"execution": map[string]any{
			"claimed_issue_ids": []string{intent.IssueID},
			"role_outputs":      roleOutputs,
		},
		"verification": map[string]any{},
		"review":       map[string]any{},
		"risk_notes":   []string{},
		"boundary": map[string]any{
			"declared": map[string]any{
				"allowed_path_prefixes":  []string{"internal/", "cmd/", "docs/"},
				"control_path_prefixes":  []string{".beads/", ".sdp/"},
				"forbidden_path_prefixes": []string{".git/"},
			},
			"observed": map[string]any{
				"touched_paths":         []string{},
				"out_of_boundary_paths": []string{},
			},
			"compliance": map[string]any{
				"ok":     true,
				"reason": "adapter-projected",
			},
		},
		"provenance": map[string]any{
			"run_id":           runID,
			"orchestrator":     "adapter-controller",
			"runtime":          "kubeopencode",
			"source_issue_id":  intent.IssueID,
			"artifact_id":      intent.IssueID + ":strict-evidence",
			"contract_version": "artifact-provenance/v1",
			"hash_algorithm":   "sha256",
			"sequence":         0,
			"captured_at":      time.Now().UTC().Format(time.RFC3339),
		},
		"trace": map[string]any{
			"beads_ids":            []string{intent.IssueID},
			"evidence_context_link": path,
		},
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return "", err
	}
	return path, nil
}
