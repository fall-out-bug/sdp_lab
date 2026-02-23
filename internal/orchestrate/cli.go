package orchestrate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	buildPhaseTimeout  = 30 * time.Minute
	reviewPhaseTimeout = 15 * time.Minute
)

// CurrentBranch returns the current git branch.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// EnsureRunFile creates the initial run file for a feature.
func EnsureRunFile(dir, featureID, branch string) {
	runID := fmt.Sprintf("oneshot-%s-%s", featureID, time.Now().UTC().Format("20060102T150405Z"))
	path := filepath.Join(dir, runID+".json")
	_ = os.MkdirAll(dir, 0o755)
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
	_ = os.WriteFile(path, []byte(body), 0o644)
}

// RunPRPhase executes git push and gh pr create.
func RunPRPhase(projectRoot, featureID string, cp *Checkpoint) error {
	push := exec.Command("git", "push", "origin", "HEAD")
	push.Dir = projectRoot
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	title := fmt.Sprintf("feat(%s): oneshot outer loop", strings.TrimPrefix(featureID, "F"))
	create := exec.Command("gh", "pr", "create", "--base", "master", "--head", cp.Branch, "--title", title, "--body", "Autonomous execution via sdp orchestrate")
	create.Dir = projectRoot
	create.Stdout = os.Stdout
	create.Stderr = os.Stderr
	if err := create.Run(); err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}
	return nil
}

// GetPRInfo returns PR number and URL for the current branch.
func GetPRInfo() (int, string, error) {
	branch, err := CurrentBranch()
	if err != nil {
		return 0, "", err
	}
	out, err := exec.Command("gh", "pr", "list", "--head", branch, "--json", "number,url").Output()
	if err != nil || len(out) == 0 {
		return 0, "", err
	}
	var arr []struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	}
	if err := json.Unmarshal(out, &arr); err != nil || len(arr) == 0 {
		return 0, "", err
	}
	return arr[0].Number, arr[0].URL, nil
}

// RunCILoop invokes sdp-ci-loop for the given PR.
func RunCILoop(pr int, featureID, checkpointDir, runsDir string) error {
	path, err := exec.LookPath("sdp-ci-loop")
	if err != nil {
		path = "sdp-ci-loop"
	}
	cmd := exec.Command(path, "--pr", fmt.Sprintf("%d", pr), "--feature", featureID, "--checkpoint-dir", checkpointDir, "--runs-dir", runsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RunOpenCodeLoop drives the full workflow using opencode as the inner loop.
func RunOpenCodeLoop(projectRoot, featureID, cpPath, runsPath string, cp *Checkpoint, workstreams []string) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "shutdown: %v\n", ctx.Err())
			os.Exit(1)
		default:
		}

		action, err := ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		switch action.Action {
		case "build":
			phaseCtx, cancel := context.WithTimeout(ctx, buildPhaseTimeout)
			commit, err := RunBuildPhase(phaseCtx, projectRoot, action.WSID)
			cancel()
			if err != nil {
				fmt.Fprintf(os.Stderr, "opencode build failed: %v\n", err)
				os.Exit(1)
			}
			if err := Advance(cp, workstreams, commit); err != nil {
				fmt.Fprintf(os.Stderr, "error: advance: %v\n", err)
				os.Exit(1)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "review":
			phaseCtx, cancel := context.WithTimeout(ctx, reviewPhaseTimeout)
			approved, err := RunReviewPhase(phaseCtx, projectRoot, action.Feature)
			cancel()
			if err != nil || !approved {
				fmt.Fprintf(os.Stderr, "opencode review failed or not approved: %v\n", err)
				os.Exit(1)
			}
			if err := Advance(cp, workstreams, ""); err != nil {
				fmt.Fprintf(os.Stderr, "error: advance: %v\n", err)
				os.Exit(1)
			}
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "pr":
			if err := RunPRPhase(projectRoot, featureID, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			prNum, prURL, _ := GetPRInfo()
			cp.PRNumber = &prNum
			cp.PRURL = prURL
			cp.Phase = PhaseCI
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "ci-loop":
			pr := 0
			if cp.PRNumber != nil {
				pr = *cp.PRNumber
			}
			if pr == 0 {
				pr, _, _ = GetPRInfo()
			}
			if pr > 0 {
				if err := RunCILoop(pr, featureID, cpPath, runsPath); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
			}
			cp.Phase = PhaseDone
			if err := SaveCheckpoint(cpPath, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: save checkpoint: %v\n", err)
				os.Exit(1)
			}
		case "done":
			fmt.Println("CI GREEN - @oneshot complete")
			return
		}
	}
}
