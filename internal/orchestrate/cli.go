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

	"sdp_dev/internal/sdputil"
)

type prInfo struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

const (
	buildPhaseTimeout  = 30 * time.Minute
	reviewPhaseTimeout = 15 * time.Minute
	prPhaseTimeout     = 10 * time.Minute
	qaPhaseTimeout     = 15 * time.Minute
)

const cliExecTimeout = 30 * time.Second

// CurrentBranch returns the current git branch. Uses ctx for cancellation.
func CurrentBranch(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, cliExecTimeout)
	defer cancel()
	out, err := exec.CommandContext(runCtx, "git", "branch", "--show-current").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// RunPRPhase executes git push and gh pr create with timeout.
func RunPRPhase(ctx context.Context, projectRoot, featureID string, cp *Checkpoint) error {
	return EnsureDraftPR(ctx, projectRoot, featureID, cp)
}

func EnsureDraftPR(ctx context.Context, projectRoot, featureID string, cp *Checkpoint) error {
	phaseCtx, cancel := context.WithTimeout(ctx, prPhaseTimeout)
	defer cancel()
	if prNum, prURL, err := GetPRInfo(phaseCtx); err == nil {
		cp.PRNumber = &prNum
		cp.PRURL = prURL
		return nil
	} else if !errors.Is(err, ErrNoPR) {
		return err
	}
	push := exec.CommandContext(phaseCtx, "git", "push", "origin", "HEAD")
	push.Dir = projectRoot
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	head, err := CurrentBranch(phaseCtx)
	if err != nil {
		return fmt.Errorf("current branch: %w", err)
	}
	title := fmt.Sprintf("F%s: oneshot outer loop", strings.TrimPrefix(featureID, "F"))
	create := exec.CommandContext(phaseCtx, "gh", "pr", "create", "--draft", "--base", "dev", "--head", head, "--title", title, "--body", "Autonomous execution via sdp orchestrate")
	create.Dir = projectRoot
	out, err := create.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}
	if prURL := extractPRURL(string(out)); prURL != "" {
		cp.PRURL = prURL
	}
	if prNum, prURL, err := GetPRInfo(phaseCtx); err == nil {
		cp.PRNumber = &prNum
		cp.PRURL = prURL
	}
	return nil
}

// ErrNoPR is returned when no PR exists for the current branch.
var ErrNoPR = errors.New("no PR found for current branch")

// GetPRInfo returns PR number and URL for the current branch. Uses ctx for cancellation.
func GetPRInfo(ctx context.Context) (int, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	branch, err := CurrentBranch(ctx)
	if err != nil {
		return 0, "", err
	}
	runCtx, cancel := context.WithTimeout(ctx, cliExecTimeout)
	defer cancel()
	out, err := exec.CommandContext(runCtx, "gh", "pr", "list", "--head", branch, "--json", "number,url").Output()
	if err != nil {
		return 0, "", err
	}
	if len(out) == 0 {
		return 0, "", ErrNoPR
	}
	var arr []prInfo
	if err := json.NewDecoder(io.LimitReader(bytes.NewReader(out), sdputil.MaxJSONDecodeBytes)).Decode(&arr); err != nil {
		return 0, "", err
	}
	if len(arr) == 0 {
		return 0, "", ErrNoPR
	}
	return arr[0].Number, arr[0].URL, nil
}

func extractPRURL(output string) string {
	for _, field := range strings.Fields(output) {
		if strings.HasPrefix(field, "https://") && strings.Contains(field, "/pull/") {
			return field
		}
	}
	return ""
}

// AdvancePRPhase runs PR phase (push, create PR), fetches PR info, updates checkpoint to PhaseCI.
func AdvancePRPhase(ctx context.Context, projectRoot, featureID, cpPath string, cp *Checkpoint) error {
	if err := RunPRPhase(ctx, projectRoot, featureID, cp); err != nil {
		return err
	}
	prNum, prURL, err := GetPRInfo(ctx)
	if err != nil {
		return err
	}
	cp.PRNumber = &prNum
	cp.PRURL = prURL
	cp.Phase = PhaseCI
	return SaveCheckpoint(cpPath, cp)
}

// AdvanceCIPhase runs CI loop if PR exists, then sets checkpoint to PhaseQA.
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
		prNum, _, err := GetPRInfo(ctx)
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
	cp.Phase = PhaseQA
	if cp.QA == nil {
		cp.QA = &QAStatus{Iteration: 0, Status: "pending"}
	}
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
