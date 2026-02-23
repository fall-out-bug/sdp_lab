package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	buildPhaseTimeout  = 30 * time.Minute
	reviewPhaseTimeout = 15 * time.Minute
	prPhaseTimeout     = 10 * time.Minute
)

// CurrentBranch returns the current git branch.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureRunFile creates the initial run file for a feature (atomic write).
func EnsureRunFile(dir, featureID, branch string) error {
	if err := validateFeatureID(featureID); err != nil {
		return err
	}
	runID := fmt.Sprintf("oneshot-%s-%s", featureID, time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, runID+".json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir runs dir: %w", err)
	}
	body := fmt.Sprintf(`{
  "run_id": "%s",
  "feature_id": "%s",
  "orchestrator": "sdp-orchestrate",
  "branch": "%s",
  "started_at": "%s",
  "events": [{"at": "%s", "phase": "init", "state": "ok"}],
  "last_phase": "init",
  "last_state": "ok"
}
`, runID, featureID, branch,
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339))
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write run file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename run file: %w", err)
	}
	return nil
}

// RunPRPhase executes git push and gh pr create with timeout.
func RunPRPhase(ctx context.Context, projectRoot, featureID string, cp *Checkpoint) error {
	phaseCtx, cancel := context.WithTimeout(ctx, prPhaseTimeout)
	defer cancel()
	push := exec.CommandContext(phaseCtx, "git", "push", "origin", "HEAD")
	push.Dir = projectRoot
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	title := fmt.Sprintf("feat(%s): oneshot outer loop", strings.TrimPrefix(featureID, "F"))
	create := exec.CommandContext(phaseCtx, "gh", "pr", "create", "--base", "master", "--head", cp.Branch, "--title", title, "--body", "Autonomous execution via sdp orchestrate")
	create.Dir = projectRoot
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}
	return nil
}

// ErrNoPR is returned when no PR exists for the current branch.
var ErrNoPR = errors.New("no PR found for current branch")

// GetPRInfo returns PR number and URL for the current branch.
func GetPRInfo() (int, string, error) {
	branch, err := CurrentBranch()
	if err != nil {
		return 0, "", err
	}
	out, err := exec.Command("gh", "pr", "list", "--head", branch, "--json", "number,url").Output()
	if err != nil {
		return 0, "", err
	}
	if len(out) == 0 {
		return 0, "", ErrNoPR
	}
	var arr []struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(out, &arr); err != nil {
		return 0, "", err
	}
	if len(arr) == 0 {
		return 0, "", ErrNoPR
	}
	return arr[0].Number, arr[0].URL, nil
}

// RunCILoop invokes sdp-ci-loop for the given PR (respects ctx cancellation).
func RunCILoop(ctx context.Context, pr int, featureID, checkpointDir, runsDir string) error {
	path, err := exec.LookPath("sdp-ci-loop")
	if err != nil {
		path = "sdp-ci-loop"
	}
	cmd := exec.CommandContext(ctx, path, "--pr", fmt.Sprintf("%d", pr), "--feature", featureID, "--checkpoint-dir", checkpointDir, "--runs-dir", runsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
