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
	payload, err := loadEvidencePayload(path)
	if err != nil {
		return "", err
	}

	mergeEvidenceIntent(payload)
	mergeEvidenceExecution(payload, issueID, branch, changedFiles)
	mergeEvidenceTrace(payload, issueID, branch)
	mergeEvidenceVerification(payload, testsPassed)

	runPath := filepath.Join(".sdp", "runs", issueID+".json")
	runPacket, runErr := loadRunPacket(runPath)
	if runErr != nil && !os.IsNotExist(runErr) {
		return "", runErr
	}

	note := ""
	if workstream == "oneshot-swarm-orchestrator" {
		note, err = applyOneShotVerification(payload, runPacket, changedFiles, testsPassed)
		if err != nil {
			return "", err
		}
	}

	outOfBoundary := computeOutOfBoundaryPaths(payload, changedFiles)
	mergeEvidenceBoundary(payload, changedFiles, outOfBoundary)
	mergeEvidenceProvenance(payload, issueID, workstream, len(outOfBoundary) == 0)

	if err := writeEvidencePayload(path, payload); err != nil {
		return "", err
	}
	if runPacket != nil {
		_ = writeRunPacket(runPath, runPacket)
	}
	return note, nil
}

func loadEvidencePayload(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func loadRunPacket(runPath string) (map[string]any, error) {
	runBytes, err := os.ReadFile(runPath)
	if err != nil {
		return nil, err
	}
	var runPacket map[string]any
	if err := json.Unmarshal(runBytes, &runPacket); err != nil {
		return nil, err
	}
	return runPacket, nil
}

func writeEvidencePayload(path string, payload map[string]any) error {
	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(path, out, 0o644)
}

func writeRunPacket(runPath string, runPacket map[string]any) error {
	runOut, err := json.MarshalIndent(runPacket, "", "  ")
	if err != nil {
		return err
	}
	runOut = append(runOut, '\n')
	return os.WriteFile(runPath, runOut, 0o644)
}

func getOrCreateMap(parent map[string]any, key string) map[string]any {
	m, _ := parent[key].(map[string]any)
	if m == nil {
		m = map[string]any{}
		parent[key] = m
	}
	return m
}

func mergeEvidenceIntent(payload map[string]any) {
	if lastBuilderResult == nil || lastBuilderResult.Prompt == "" {
		return
	}
	intent := getOrCreateMap(payload, "intent")
	intent["llm_prompt"] = lastBuilderResult.Prompt
}

func mergeEvidenceExecution(payload map[string]any, issueID, branch string, changedFiles []string) {
	exec := getOrCreateMap(payload, "execution")
	exec["branch"] = branch
	exec["changed_files"] = changedFiles
	exec["claimed_issue_ids"] = []string{issueID}
	if lastBuilderResult != nil {
		exec["model"] = lastBuilderResult.ModelUsed
		exec["duration_ms"] = lastBuilderResult.Duration.Milliseconds()
		if lastBuilderResult.SessionID != "" {
			exec["opencode_session_id"] = lastBuilderResult.SessionID
		}
	}
}

func mergeEvidenceTrace(payload map[string]any, issueID, branch string) {
	trace := getOrCreateMap(payload, "trace")
	trace["branch"] = branch
	trace["beads_ids"] = []string{issueID}
}

func mergeEvidenceVerification(payload map[string]any, testsPassed bool) {
	verif := getOrCreateMap(payload, "verification")
	verif["tests"] = []string{"go test ./..."}
	verif["go_test_passed"] = testsPassed
}

func computeOutOfBoundaryPaths(payload map[string]any, changedFiles []string) []string {
	boundary, _ := payload["boundary"].(map[string]any)
	if boundary == nil {
		return nil
	}
	declared, _ := boundary["declared"].(map[string]any)
	if declared == nil {
		return nil
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
	return outOfBoundary
}

func mergeEvidenceBoundary(payload map[string]any, changedFiles, outOfBoundary []string) {
	boundary := getOrCreateMap(payload, "boundary")
	observed := getOrCreateMap(boundary, "observed")
	observed["touched_paths"] = changedFiles
	observed["out_of_boundary_paths"] = outOfBoundary
	compliance := getOrCreateMap(boundary, "compliance")
	compliance["ok"] = len(outOfBoundary) == 0
	if len(outOfBoundary) == 0 {
		compliance["reason"] = "changed paths within declared boundary"
	} else {
		compliance["reason"] = "changed paths exceed declared boundary"
	}
}

func mergeEvidenceProvenance(payload map[string]any, issueID, workstream string, boundaryOK bool) {
	prov := getOrCreateMap(payload, "provenance")
	prov["orchestrator"] = "swarm-worker"
	prov["runtime"] = os.Getenv("SDP_RUNTIME")
	if lastBuilderResult != nil && lastBuilderResult.ModelUsed != "" {
		prov["model"] = lastBuilderResult.ModelUsed
	} else if model := os.Getenv("SDP_MODEL"); model != "" {
		prov["model"] = model
	}
	prov["phase"] = "verify"
	prov["role"] = workstream
	prov["captured_at"] = time.Now().UTC().Format(time.RFC3339)
	prov["source_issue_id"] = issueID
	setProvenanceDefaultString(prov, "artifact_id", issueID+":strict-evidence")
	setProvenanceDefaultString(prov, "contract_version", "artifact-provenance/v1")
	setProvenanceDefaultString(prov, "hash_algorithm", "sha256")
	setProvenanceDefaultString(prov, "payload_digest", "")
	setProvenanceDefaultString(prov, "hash", "")
	setProvenanceDefaultString(prov, "hash_prev", "")
	if _, ok := prov["sequence"].(float64); !ok {
		if _, intOK := prov["sequence"].(int); !intOK {
			prov["sequence"] = 0
		}
	}
	gates := toStringSlice(prov["gate_results"])
	gates = append(gates, "verification:go test ./...", fmt.Sprintf("boundary:ok=%t", boundaryOK))
	prov["gate_results"] = uniqueStrings(gates)
}

func setProvenanceDefaultString(prov map[string]any, key, defaultVal string) {
	if _, ok := prov[key].(string); !ok {
		prov[key] = defaultVal
	}
}
