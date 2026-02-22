package quality

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"sdp_dev/internal/agent"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/llm"
)

// RunTests runs `go test ./...` in workDir. Returns true if all tests pass.
func RunTests(workDir string) (bool, error) {
	cmd := exec.Command("go", "test", "./...")
	cmd.Dir = workDir
	err := cmd.Run()
	return err == nil, nil
}

// EvidenceConfig configures evidence collection.
type EvidenceConfig struct {
	WorkDir       string
	IssueID       string
	Branch        string
	RiskClass     string
	Model         string
	Role          string
	Boundary      llm.BoundarySpec
	ChangedFiles  []string
	ModelUsed     string
	TestsPassed   bool
	BoundaryViolation error
}

// CollectEvidence initializes evidence file and optionally updates execution section.
// If InitOnly is true, only Initialize is called; otherwise UpdateExecution is called with CollectResult from cfg.
func CollectEvidence(cfg EvidenceConfig) (path string, err error) {
	ec := agent.NewEvidenceCollector(cfg.WorkDir)
	path = filepath.Join(cfg.WorkDir, ".sdp", "evidence", cfg.IssueID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	riskClass := cfg.RiskClass
	if riskClass == "" {
		riskClass = "medium"
	}
	if _, err := ec.Initialize(cfg.IssueID, cfg.Branch, riskClass, cfg.Model, cfg.Role, cfg.Boundary); err != nil {
		return "", fmt.Errorf("evidence init: %w", err)
	}
	res := agent.CollectResult{
		ChangedFiles:      cfg.ChangedFiles,
		ModelUsed:         cfg.ModelUsed,
		TestsPassed:       cfg.TestsPassed,
		BoundaryViolation: cfg.BoundaryViolation,
	}
	if err := ec.UpdateExecution(cfg.IssueID, res); err != nil {
		return "", fmt.Errorf("evidence update: %w", err)
	}
	return path, nil
}

// UpdateEvidence updates the evidence file with execution results (e.g. after RunTests).
func UpdateEvidence(workDir, issueID string, result agent.CollectResult) error {
	return agent.NewEvidenceCollector(workDir).UpdateExecution(issueID, result)
}

// ProvenanceConfig configures provenance signing.
type ProvenanceConfig struct {
	AgentID       string
	Role          string
	IssueID       string
	ArtifactID    string
	Phase         string
	Payload       any
	ModelUsed     string
	TraceLink     string
	EvidenceLink  string
}

// SignProvenance produces a signed envelope for the given config.
func SignProvenance(cfg ProvenanceConfig) (bus.Envelope, error) {
	signer := agent.NewProvenanceSigner(cfg.AgentID, cfg.Role)
	return signer.Sign(agent.SignInput{
		IssueID:       cfg.IssueID,
		ArtifactID:    cfg.ArtifactID,
		ArtifactClass: "artifact",
		Phase:         cfg.Phase,
		Payload:       cfg.Payload,
		ModelUsed:     cfg.ModelUsed,
		TraceLink:     cfg.TraceLink,
		EvidenceLink:  cfg.EvidenceLink,
	})
}

// RunPRGate runs `pr-gate --prepublish --issue <issueID>` in workDir.
// Requires pr-gate to be in PATH (e.g. installed or go run ./cmd/pr-gate from repo root).
func RunPRGate(issueID, workDir string) error {
	cmd := exec.Command("pr-gate", "--prepublish", "--issue", issueID)
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pr-gate: %w: %s", err, string(out))
	}
	return nil
}

// TransitionFSM runs `beads-fsm --issue <issueID> --to <targetState> --apply` in workDir.
// Requires beads-fsm to be in PATH (e.g. installed or go run ./cmd/beads-fsm from repo root).
func TransitionFSM(issueID, targetState, workDir string) error {
	cmd := exec.Command("beads-fsm", "--issue", issueID, "--to", targetState, "--apply")
	cmd.Dir = workDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("beads-fsm: %w: %s", err, string(out))
	}
	return nil
}

// PublishConfig configures commit and pr-publish.
type PublishConfig struct {
	WorkDir   string
	IssueID   string
	Title     string
	Changed   []string
	BaseBranch string
}

// BaseBranch returns SDP_REPO_BRANCH or "master".
func BaseBranch() string {
	if b := os.Getenv("SDP_REPO_BRANCH"); b != "" {
		return b
	}
	return "master"
}

// CommitAndPublish stages, commits, pushes worker branch and runs pr-publish.
func CommitAndPublish(cfg PublishConfig) (string, error) {
	base := cfg.BaseBranch
	if base == "" {
		base = BaseBranch()
	}
	args := append([]string{"add"}, cfg.Changed...)
	args = append(args, ".beads/issues.jsonl", ".beads/metadata.json")
	cmd := exec.Command("git", args...)
	cmd.Dir = cfg.WorkDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git add: %w: %s", err, string(out))
	}
	commitCmd := exec.Command("git", "commit", "-m", "worker: implement "+cfg.IssueID, "-m", "SDP swarm quality pipeline")
	commitCmd.Dir = cfg.WorkDir
	if out, err := commitCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git commit: %w: %s", err, string(out))
	}
	pushCmd := exec.Command("git", "push", "-u", "origin", "worker/"+cfg.IssueID)
	pushCmd.Dir = cfg.WorkDir
	if out, err := pushCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git push: %w: %s", err, string(out))
	}
	bodyPath := filepath.Join(cfg.WorkDir, ".sdp", "pr-body-"+cfg.IssueID+".md")
	_ = os.MkdirAll(filepath.Dir(bodyPath), 0o755)
	body := "## Summary\n\n- SDP swarm pipeline execution for " + cfg.IssueID + "\n- " + cfg.Title + "\n"
	_ = os.WriteFile(bodyPath, []byte(body), 0o644)
	prTitle := "Worker: " + cfg.Title
	if prTitle == "Worker: " {
		prTitle = "Worker: " + cfg.IssueID
	}
	prCmd := exec.Command("pr-publish", "--issue", cfg.IssueID, "--title", prTitle, "--head", "worker/"+cfg.IssueID, "--base", base, "--body-file", bodyPath)
	prCmd.Dir = cfg.WorkDir
	if out, err := prCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("pr-publish: %w: %s", err, string(out))
	}
	return "worker/" + cfg.IssueID, nil
}
