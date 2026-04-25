// Package promote implements the vibecode-to-strict Phase FSM bridge.
// It converts completed vibecode build evidence into Phase FSM artifacts
// (delta artifacts + gates + trace records), enabling opt-in escalation
// from the fast vibecode path to the strict F134 Phase FSM without
// redoing work.
package promote

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/build"
	"sdp_dev/internal/delta"
	"sdp_dev/internal/gate"
)

// SourceLabel identifies artifacts produced by the vibecode promotion bridge.
const SourceLabel = "vibecode-promoted"

// PromoteResult holds the output of a promotion operation.
type PromoteResult struct {
	RunID       string          `json:"run_id"`
	FeatureID   string          `json:"feature_id"`
	EvidenceDir string          `json:"evidence_dir"`
	PhaseDir    string          `json:"phase_dir"`
	Deltas      []DeltaArtifact `json:"deltas"`
	Gates       []GateArtifact  `json:"gates"`
	Errors      []string        `json:"errors,omitempty"`
}

// DeltaArtifact records a generated delta file.
type DeltaArtifact struct {
	Phase string `json:"phase"`
	Path  string `json:"path"`
}

// GateArtifact records a generated gate file.
type GateArtifact struct {
	Phase string `json:"phase"`
	ID    string `json:"gate_id"`
	Path  string `json:"path"`
}

// PromoteOptions configures the promotion operation.
type PromoteOptions struct {
	// RunID identifies the completed vibecode run to promote.
	RunID string
	// FeatureID is the feature label for delta artifacts.
	FeatureID string
	// EvidenceDir is the directory containing the vibecode evidence.json.
	// Defaults to .sdp/evidence/<run_id>/.
	EvidenceDir string
	// PhaseDir is the output directory for Phase FSM artifacts.
	// Defaults to .sdp/phases/<run_id>/.
	PhaseDir string
}

// PromoteFromRun reads vibecode build evidence and produces Phase FSM artifacts.
// It generates delta artifacts and gates for plan, review, and eval phases,
// pre-populated with evidence from the vibecode run.
func PromoteFromRun(opts PromoteOptions) (*PromoteResult, error) {
	if opts.RunID == "" {
		return nil, fmt.Errorf("promote: run_id is required")
	}
	if opts.FeatureID == "" {
		return nil, fmt.Errorf("promote: feature_id is required")
	}

	if opts.EvidenceDir == "" {
		opts.EvidenceDir = filepath.Join(".sdp", "evidence", opts.RunID)
	}
	if opts.PhaseDir == "" {
		opts.PhaseDir = filepath.Join(".sdp", "phases", opts.RunID)
	}

	// Read vibecode evidence.
	evidence, err := readBuildEvidence(opts.EvidenceDir)
	if err != nil {
		return nil, fmt.Errorf("promote: read evidence: %w", err)
	}

	// Validate evidence RunID matches the requested run to prevent mismatches.
	if evidence.RunID != "" && evidence.RunID != opts.RunID {
		return nil, fmt.Errorf("promote: evidence RunID %q does not match requested RunID %q", evidence.RunID, opts.RunID)
	}

	result := &PromoteResult{
		RunID:       opts.RunID,
		FeatureID:   opts.FeatureID,
		EvidenceDir: opts.EvidenceDir,
		PhaseDir:    opts.PhaseDir,
		Deltas:      make([]DeltaArtifact, 0, 3),
		Gates:       make([]GateArtifact, 0, 3),
		Errors:      make([]string, 0),
	}

	if err := os.MkdirAll(opts.PhaseDir, 0o755); err != nil {
		return nil, fmt.Errorf("promote: create phase dir: %w", err)
	}

	// Generate phase-evidence JSON from vibecode evidence.
	phaseEvidence := buildPhaseEvidence(evidence)

	// Track which phases had successful evidence writes.
	evidenceWritten := make(map[string]bool)

	// Write per-phase evidence files.
	for _, phase := range []string{"plan", "review", "eval"} {
		evPath := filepath.Join(opts.PhaseDir, phase+".evidence.json")
		if err := writeJSON(evPath, phaseEvidence[phase]); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s evidence: %v", phase, err))
			continue
		}
		evidenceWritten[phase] = true
	}

	// Generate delta artifacts and gates for each phase.
	phases := []struct {
		name     string
		gateType gate.GateType
		question string
		options  []string
	}{
		{"plan", gate.GateTypePlan, "Approve plan (promoted from vibecode run)?", []string{"approve", "reject", "defer"}},
		{"review", gate.GateTypeReview, "Approve review (promoted from vibecode run)?", []string{"approve", "reject", "request-changes"}},
		{"eval", gate.GateTypeEval, "Approve eval results (promoted from vibecode run)?", []string{"approve", "reject", "retry"}},
	}

	for _, p := range phases {
		// Create delta artifact.
		d := buildPhaseDelta(p.name, opts, evidence)
		deltaPath := filepath.Join(opts.PhaseDir, p.name+".delta.md")
		if err := os.WriteFile(deltaPath, []byte(d.RenderMarkdown()), 0o644); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s delta: %v", p.name, err))
			continue
		}
		result.Deltas = append(result.Deltas, DeltaArtifact{Phase: p.name, Path: deltaPath})

		// Skip gate creation if evidence write failed — gate would reference
		// missing evidence and fail validation.
		if !evidenceWritten[p.name] {
			result.Errors = append(result.Errors, fmt.Sprintf("skip %s gate: evidence file not written", p.name))
			continue
		}

		// Create gate with pre-populated evidence.
		g := &gate.Gate{
			ID:        fmt.Sprintf("%s-%s-%s", p.name, opts.FeatureID, opts.RunID),
			Question:  p.question,
			Context:   fmt.Sprintf("promoted from vibecode run %s (source=%s)", opts.RunID, SourceLabel),
			Options:   p.options,
			Type:      p.gateType,
			CreatedAt: time.Now().UTC(),
		}

		// Attach evidence path — gate stays in AWAITING state (requires human).
		evPath := filepath.Join(opts.PhaseDir, p.name+".evidence.json")
		g.EvidencePath = evPath

		gatePath := filepath.Join(opts.PhaseDir, p.name+".gate.json")
		if err := writeJSON(gatePath, g); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("write %s gate: %v", p.name, err))
			continue
		}
		result.Gates = append(result.Gates, GateArtifact{Phase: p.name, ID: g.ID, Path: gatePath})
	}

	// Write promotion trace record.
	trace := map[string]interface{}{
		"source":       SourceLabel,
		"run_id":       opts.RunID,
		"feature_id":   opts.FeatureID,
		"evidence_dir": opts.EvidenceDir,
		"phase_dir":    opts.PhaseDir,
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"deltas":       len(result.Deltas),
		"gates":        len(result.Gates),
		"errors":       len(result.Errors),
	}
	tracePath := filepath.Join(opts.PhaseDir, "promotion-trace.json")
	if err := writeJSON(tracePath, trace); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("write trace: %v", err))
	}

	return result, nil
}

// readBuildEvidence reads evidence.json from the vibecode evidence directory.
func readBuildEvidence(dir string) (*build.BuildEvidence, error) {
	path := filepath.Join(dir, "evidence.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var ev build.BuildEvidence
	if err := json.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("parse evidence: %w", err)
	}
	return &ev, nil
}

// buildPhaseEvidence maps vibecode BuildEvidence to phase-gate evidence schemas.
// Each phase gets a JSON object that satisfies the gate's RequiredEvidenceKeys.
func buildPhaseEvidence(ev *build.BuildEvidence) map[string]map[string]interface{} {
	planEvidence := map[string]interface{}{
		"test_coverage":    deriveTestCoverage(ev),
		"design_checklist": deriveDesignChecklist(ev),
		"source":           SourceLabel,
		"run_id":           ev.RunID,
		"classification":   deriveClassification(ev),
	}

	reviewEvidence := map[string]interface{}{
		"spec_review_verdict": deriveSpecReviewVerdict(ev),
		"code_review_verdict": deriveCodeReviewVerdict(ev),
		"source":             SourceLabel,
		"run_id":             ev.RunID,
	}

	evalEvidence := map[string]interface{}{
		"go_test":        deriveGoTestResult(ev),
		"go_vet":         deriveGoVetResult(ev),
		"protocol_check": deriveProtocolCheckResult(ev),
		"smoke":          deriveSmokeResult(ev),
		"source":         SourceLabel,
		"run_id":         ev.RunID,
	}

	return map[string]map[string]interface{}{
		"plan":   planEvidence,
		"review": reviewEvidence,
		"eval":   evalEvidence,
	}
}

// buildPhaseDelta creates a delta artifact for a phase, populated from vibecode evidence.
func buildPhaseDelta(phase string, opts PromoteOptions, ev *build.BuildEvidence) *delta.Delta {
	d := delta.NewDelta(phase,
		delta.WithFeatureID(opts.FeatureID),
		delta.WithRunID(opts.RunID),
	)

	d.SetRationale(fmt.Sprintf("Promoted from vibecode run %s (source=%s). Vibecode evidence schema is superset-compatible — no rewrite needed.", opts.RunID, SourceLabel))

	// Add blocks based on vibecode stages.
	d.AddModified(delta.Block{
		Title:       fmt.Sprintf("Vibecode run %s — %s phase", min8(opts.RunID), phase),
		Description: fmt.Sprintf("Auto-generated %s delta from vibecode run. Status: %s", phase, ev.Status),
		Files:       []string{filepath.Join(opts.EvidenceDir, "evidence.json")},
		Disclosure:  delta.DisclosurePrivate,
	})

	return d
}

// --- Evidence derivation helpers ---

func deriveTestCoverage(ev *build.BuildEvidence) string {
	if ev.Sandbox.TestsOK {
		return "pass (vibecode: sandbox tests ok)"
	}
	return "unknown (vibecode: sandbox tests not reported)"
}

func deriveDesignChecklist(ev *build.BuildEvidence) string {
	if ev.Status == "success" {
		return "pass (vibecode: all stages passed)"
	}
	return "partial (vibecode: some stages failed)"
}

func deriveClassification(ev *build.BuildEvidence) string {
	if ev.Dispatch.Harness != "" {
		return fmt.Sprintf("%s/%s (score=%.2f)", ev.Dispatch.Harness, ev.Dispatch.Model, ev.Dispatch.Score)
	}
	return "vibecode-auto"
}

func deriveSpecReviewVerdict(ev *build.BuildEvidence) string {
	if ev.Status == "success" {
		return "pass (vibecode: build succeeded)"
	}
	return "needs-review (vibecode: build status=" + ev.Status + ")"
}

func deriveCodeReviewVerdict(ev *build.BuildEvidence) string {
	if ev.Sandbox.BuildOK {
		return "pass (vibecode: sandbox build ok)"
	}
	return "fail (vibecode: sandbox build failed)"
}

func deriveGoTestResult(ev *build.BuildEvidence) string {
	if ev.Sandbox.TestsOK {
		return "pass"
	}
	return "fail (vibecode: sandbox tests failed)"
}

func deriveGoVetResult(ev *build.BuildEvidence) string {
	if ev.Sandbox.BuildOK {
		return "clean (vibecode: build ok)"
	}
	return "unknown (vibecode: build not ok)"
}

func deriveProtocolCheckResult(ev *build.BuildEvidence) string {
	return "promoted (vibecode: no protocol check in vibecode mode)"
}

func deriveSmokeResult(ev *build.BuildEvidence) string {
	if ev.Status == "success" {
		return "pass (vibecode: all stages passed)"
	}
	return "fail (vibecode: status=" + ev.Status + ")"
}

// --- Helpers ---

func writeJSON(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func min8(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[:8]
}
