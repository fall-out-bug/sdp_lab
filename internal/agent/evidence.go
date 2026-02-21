package agent

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/llm"
)

// EvidenceCollector initializes and updates .sdp/evidence/{issue_id}.json.
type EvidenceCollector struct {
	workDir string
}

// NewEvidenceCollector creates an EvidenceCollector.
func NewEvidenceCollector(workDir string) *EvidenceCollector {
	return &EvidenceCollector{workDir: workDir}
}

// CollectResult holds execution result for evidence.
type CollectResult struct {
	ChangedFiles      []string
	ModelUsed         string
	Duration          time.Duration
	BoundaryViolation error
	TestsPassed       bool
}

// Initialize creates the evidence file from template and populates intent/boundary/provenance.
func (e *EvidenceCollector) Initialize(issueID, branch, riskClass, model, role string, boundary llm.BoundarySpec) (string, error) {
	tmpl, err := e.loadTemplate()
	if err != nil {
		return "", err
	}
	populateIntentSection(tmpl, issueID, riskClass)
	populateExecutionSection(tmpl, branch, issueID)
	populateBoundarySection(tmpl, boundary, role)
	populateProvenanceSection(tmpl, issueID, model, role)
	populateTraceSection(tmpl, branch, issueID)

	path := filepath.Join(e.workDir, ".sdp", "evidence", issueID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(tmpl, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

// UpdateExecution updates the evidence file with execution results.
func (e *EvidenceCollector) UpdateExecution(issueID string, result CollectResult) error {
	path := filepath.Join(e.workDir, ".sdp", "evidence", issueID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	updateExecutionSection(doc, result)
	updateBoundarySection(doc, result)
	updateVerificationSection(doc, result)

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func getOrCreateMap(parent map[string]any, key string) map[string]any {
	m, _ := parent[key].(map[string]any)
	if m == nil {
		m = map[string]any{}
		parent[key] = m
	}
	return m
}

func populateIntentSection(tmpl map[string]any, issueID, riskClass string) {
	intent := getOrCreateMap(tmpl, "intent")
	intent["issue_id"] = issueID
	intent["trigger"] = "agent"
	intent["risk_class"] = riskClass
}

func populateExecutionSection(tmpl map[string]any, branch, issueID string) {
	exec := getOrCreateMap(tmpl, "execution")
	exec["branch"] = branch
	exec["claimed_issue_ids"] = []string{issueID}
	exec["changed_files"] = []string{}
}

func populateBoundarySection(tmpl map[string]any, boundary llm.BoundarySpec, role string) {
	boundSec := getOrCreateMap(tmpl, "boundary")
	declared := getOrCreateMap(boundSec, "declared")
	declared["allowed_path_prefixes"] = boundary.AllowedPathPrefixes
	declared["control_path_prefixes"] = boundary.ControlPathPrefixes
	declared["forbidden_path_prefixes"] = boundary.ForbiddenPathPrefixes
	declared["role"] = role
	declared["lane"] = "commit"
	observed := getOrCreateMap(boundSec, "observed")
	observed["touched_paths"] = []string{}
	observed["out_of_boundary_paths"] = []string{}
	compliance := getOrCreateMap(boundSec, "compliance")
	compliance["ok"] = true
	compliance["reason"] = "declared boundary initialized"
}

func populateProvenanceSection(tmpl map[string]any, issueID, model, role string) {
	prov := getOrCreateMap(tmpl, "provenance")
	prov["run_id"] = issueID
	prov["orchestrator"] = "agent"
	prov["runtime"] = os.Getenv("SDP_RUNTIME")
	prov["model"] = model
	prov["phase"] = "intake"
	prov["role"] = role
	prov["captured_at"] = time.Now().UTC().Format(time.RFC3339)
	prov["source_issue_id"] = issueID
	prov["artifact_id"] = issueID + ":strict-evidence"
	prov["contract_version"] = "artifact-provenance/v1"
	prov["hash_algorithm"] = "sha256"
}

func populateTraceSection(tmpl map[string]any, branch, issueID string) {
	trace := getOrCreateMap(tmpl, "trace")
	trace["branch"] = branch
	trace["beads_ids"] = []string{issueID}
}

func updateExecutionSection(doc map[string]any, result CollectResult) {
	exec := getOrCreateMap(doc, "execution")
	exec["changed_files"] = result.ChangedFiles
}

func updateBoundarySection(doc map[string]any, result CollectResult) {
	boundSec, _ := doc["boundary"].(map[string]any)
	if boundSec == nil {
		return
	}
	observed, _ := boundSec["observed"].(map[string]any)
	if observed != nil {
		observed["touched_paths"] = result.ChangedFiles
		if result.BoundaryViolation != nil {
			observed["out_of_boundary_paths"] = result.ChangedFiles
		}
	}
	compliance, _ := boundSec["compliance"].(map[string]any)
	if compliance != nil {
		compliance["ok"] = result.BoundaryViolation == nil
		if result.BoundaryViolation != nil {
			compliance["reason"] = result.BoundaryViolation.Error()
		}
	}
}

func updateVerificationSection(doc map[string]any, result CollectResult) {
	verif := getOrCreateMap(doc, "verification")
	verif["go_test_passed"] = result.TestsPassed
}

func (e *EvidenceCollector) loadTemplate() (map[string]any, error) {
	tmplPath := filepath.Join(e.workDir, "specs", "strict-evidence-template.json")
	data, err := os.ReadFile(tmplPath)
	if err != nil {
		return nil, err
	}
	var tmpl map[string]any
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return nil, err
	}
	if tmpl == nil {
		return nil, errors.New("empty evidence template")
	}
	return tmpl, nil
}
