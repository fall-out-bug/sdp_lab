package orchestrate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// runFileJSON is the initial run file schema (safe JSON marshal, no quote injection).
type runFileJSON struct {
	RunID        string            `json:"run_id"`
	FeatureID   string            `json:"feature_id"`
	Orchestrator string           `json:"orchestrator"`
	Branch      string            `json:"branch"`
	StartedAt   string            `json:"started_at"`
	Events      []runFileEventJSON `json:"events"`
	LastPhase   string            `json:"last_phase"`
	LastState   string            `json:"last_state"`
}

type runFileEventJSON struct {
	At    string `json:"at"`
	Phase string `json:"phase"`
	State string `json:"state"`
}

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
	now := time.Now().UTC().Format(time.RFC3339)
	runID := fmt.Sprintf("oneshot-%s-%s", featureID, time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, runID+".json")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir runs dir: %w", err)
	}
	rf := runFileJSON{
		RunID:        runID,
		FeatureID:   featureID,
		Orchestrator: "sdp-orchestrate",
		Branch:      branch,
		StartedAt:   now,
		Events:      []runFileEventJSON{{At: now, Phase: "init", State: "ok"}},
		LastPhase:   "init",
		LastState:   "ok",
	}
	body, err := json.MarshalIndent(rf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal run file: %w", err)
	}
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, body, 0o644); err != nil {
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
	head, err := CurrentBranch()
	if err != nil {
		return fmt.Errorf("current branch: %w", err)
	}
	title := fmt.Sprintf("feat(%s): oneshot outer loop", strings.TrimPrefix(featureID, "F"))
	create := exec.CommandContext(phaseCtx, "gh", "pr", "create", "--base", "master", "--head", head, "--title", title, "--body", "Autonomous execution via sdp orchestrate")
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
	if err := json.NewDecoder(io.LimitReader(bytes.NewReader(out), maxJSONDecodeBytes)).Decode(&arr); err != nil {
		return 0, "", err
	}
	if len(arr) == 0 {
		return 0, "", ErrNoPR
	}
	return arr[0].Number, arr[0].URL, nil
}

// AdvancePRPhase runs PR phase (push, create PR), fetches PR info, updates checkpoint to PhaseCI.
func AdvancePRPhase(ctx context.Context, projectRoot, featureID, cpPath string, cp *Checkpoint) error {
	if err := RunPRPhase(ctx, projectRoot, featureID, cp); err != nil {
		return err
	}
	prNum, prURL, err := GetPRInfo()
	if err != nil {
		return err
	}
	cp.PRNumber = &prNum
	cp.PRURL = prURL
	cp.Phase = PhaseCI
	return SaveCheckpoint(cpPath, cp)
}

// AdvanceCIPhase runs CI loop if PR exists, then sets checkpoint to PhaseDone.
func AdvanceCIPhase(ctx context.Context, projectRoot, featureID, cpPath, runsPath string, cp *Checkpoint) error {
	cpFilePath := filepath.Join(cpPath, featureID+".json")
	env := HookEnv{FeatureID: featureID, Phase: PhaseCI, CheckpointPath: cpFilePath}
	if err := RunHooks(ctx, projectRoot, "ci", "pre", env, func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	}); err != nil {
		return err
	}
	pr := 0
	if cp.PRNumber != nil {
		pr = *cp.PRNumber
	}
	if pr == 0 {
		prNum, _, err := GetPRInfo()
		if err != nil {
			return err
		}
		pr = prNum
	}
	if pr > 0 {
		if err := RunCILoop(ctx, pr, featureID, cpPath, runsPath); err != nil {
			return err
		}
	}
	if err := RunHooks(ctx, projectRoot, "ci", "post", env, func(msg string) {
		fmt.Fprintln(os.Stderr, msg)
	}); err != nil {
		return err
	}
	cp.Phase = PhaseDone
	return SaveCheckpoint(cpPath, cp)
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
