package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func updateEvidence(issueID, branch, workstream string, changedFiles []string, testsPassed bool) (string, error) {
	path := filepath.Join(".sdp", "evidence", issueID+".json")
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return "", err
	}
	execSection, _ := payload["execution"].(map[string]any)
	if execSection == nil {
		execSection = map[string]any{}
		payload["execution"] = execSection
	}
	execSection["branch"] = branch
	execSection["changed_files"] = changedFiles
	execSection["claimed_issue_ids"] = []string{issueID}

	trace, _ := payload["trace"].(map[string]any)
	if trace == nil {
		trace = map[string]any{}
		payload["trace"] = trace
	}
	trace["branch"] = branch
	trace["beads_ids"] = []string{issueID}

	verification, _ := payload["verification"].(map[string]any)
	if verification == nil {
		verification = map[string]any{}
		payload["verification"] = verification
	}
	verification["tests"] = []string{"go test ./..."}
	verification["go_test_passed"] = testsPassed

	runPath := filepath.Join(".sdp", "runs", issueID+".json")
	var runPacket map[string]any
	if runBytes, runErr := os.ReadFile(runPath); runErr == nil {
		if err := json.Unmarshal(runBytes, &runPacket); err != nil {
			return "", err
		}
	}

	note := ""
	if workstream == "oneshot-swarm-orchestrator" {
		note, err = applyOneShotVerification(payload, runPacket, changedFiles, testsPassed)
		if err != nil {
			return "", err
		}
	}

	boundary, _ := payload["boundary"].(map[string]any)
	if boundary == nil {
		boundary = map[string]any{}
		payload["boundary"] = boundary
	}
	declared, _ := boundary["declared"].(map[string]any)
	if declared == nil {
		declared = map[string]any{}
		boundary["declared"] = declared
	}
	allowed := toStringSlice(declared["allowed_path_prefixes"])
	control := toStringSlice(declared["control_path_prefixes"])
	forbidden := toStringSlice(declared["forbidden_path_prefixes"])

	outOfBoundary := make([]string, 0)
	for _, f := range changedFiles {
		if hasPrefixAny(f, control) {
			continue
		}
		if hasPrefixAny(f, forbidden) {
			outOfBoundary = append(outOfBoundary, f)
			continue
		}
		if len(allowed) > 0 && !hasPrefixAny(f, allowed) {
			outOfBoundary = append(outOfBoundary, f)
		}
	}
	sort.Strings(outOfBoundary)

	observed, _ := boundary["observed"].(map[string]any)
	if observed == nil {
		observed = map[string]any{}
		boundary["observed"] = observed
	}
	observed["touched_paths"] = changedFiles
	observed["out_of_boundary_paths"] = outOfBoundary

	compliance, _ := boundary["compliance"].(map[string]any)
	if compliance == nil {
		compliance = map[string]any{}
		boundary["compliance"] = compliance
	}
	compliance["ok"] = len(outOfBoundary) == 0
	if len(outOfBoundary) == 0 {
		compliance["reason"] = "changed paths within declared boundary"
	} else {
		compliance["reason"] = "changed paths exceed declared boundary"
	}

	provenance, _ := payload["provenance"].(map[string]any)
	if provenance == nil {
		provenance = map[string]any{}
		payload["provenance"] = provenance
	}
	provenance["orchestrator"] = "swarm-worker"
	provenance["runtime"] = os.Getenv("SDP_RUNTIME")
	if model := os.Getenv("SDP_MODEL"); model != "" {
		provenance["model"] = model
	}
	provenance["phase"] = "verify"
	provenance["role"] = workstream
	provenance["captured_at"] = time.Now().UTC().Format(time.RFC3339)
	provenance["source_issue_id"] = issueID
	if _, ok := provenance["artifact_id"].(string); !ok {
		provenance["artifact_id"] = issueID + ":strict-evidence"
	}
	if _, ok := provenance["contract_version"].(string); !ok {
		provenance["contract_version"] = "artifact-provenance/v1"
	}
	if _, ok := provenance["hash_algorithm"].(string); !ok {
		provenance["hash_algorithm"] = "sha256"
	}
	if _, ok := provenance["sequence"].(float64); !ok {
		if _, intOK := provenance["sequence"].(int); !intOK {
			provenance["sequence"] = 0
		}
	}
	if _, ok := provenance["payload_digest"].(string); !ok {
		provenance["payload_digest"] = ""
	}
	if _, ok := provenance["hash"].(string); !ok {
		provenance["hash"] = ""
	}
	if _, ok := provenance["hash_prev"].(string); !ok {
		provenance["hash_prev"] = ""
	}
	gates := toStringSlice(provenance["gate_results"])
	gates = append(gates, "verification:go test ./...", fmt.Sprintf("boundary:ok=%t", len(outOfBoundary) == 0))
	provenance["gate_results"] = uniqueStrings(gates)

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", err
	}

	if runPacket != nil {
		runOut, err := json.MarshalIndent(runPacket, "", "  ")
		if err != nil {
			return "", err
		}
		runOut = append(runOut, '\n')
		if err := os.WriteFile(runPath, runOut, 0o644); err != nil {
			return "", err
		}
	}

	return note, nil
}
