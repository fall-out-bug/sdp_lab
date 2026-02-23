package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/orchestrate"
)

func main() {
	feature := flag.String("feature", "", "Feature ID (e.g. F016)")
	nextAction := flag.Bool("next-action", false, "Output next action as JSON")
	advance := flag.Bool("advance", false, "Advance to next phase after current action")
	result := flag.String("result", "", "Result for advance (e.g. commit hash for build phase)")
	resume := flag.Bool("resume", false, "Resume from existing checkpoint")
	checkpointDir := flag.String("checkpoint-dir", ".sdp/checkpoints", "Checkpoint directory")
	runsDir := flag.String("runs-dir", ".sdp/runs", "Runs directory")
	runtime := flag.String("runtime", "", "Runtime for LLM phases: opencode (invokes opencode run as subprocess)")
	flag.Parse()

	if *feature == "" {
		fmt.Fprintln(os.Stderr, "error: --feature is required")
		flag.Usage()
		os.Exit(1)
	}

	featureID := strings.ToUpper(*feature)
	if !strings.HasPrefix(featureID, "F") {
		featureID = "F" + featureID
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	projectRoot, err := orchestrate.FindProjectRoot(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	workstreams, err := orchestrate.DiscoverWorkstreams(projectRoot, featureID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	cpPath := filepath.Join(projectRoot, *checkpointDir)
	runsPath := filepath.Join(projectRoot, *runsDir)

	cp, err := orchestrate.LoadCheckpoint(cpPath, featureID)
	if err != nil {
		if *resume || !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		branch, err := currentBranch()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		cp = orchestrate.CreateInitialCheckpoint(featureID, branch, workstreams)
		cp.CreatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := os.MkdirAll(cpPath, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := orchestrate.SaveCheckpoint(cpPath, cp); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		ensureRunFile(runsPath, featureID, cp.Branch)
	}

	if *nextAction {
		action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(action); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if *runtime == "opencode" {
		runOpenCodeLoop(projectRoot, featureID, cpPath, runsPath, cp, workstreams)
		return
	}

	if *advance {
		if err := orchestrate.Advance(cp, workstreams, *result); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := orchestrate.SaveCheckpoint(cpPath, cp); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if cp.Phase == orchestrate.PhasePR {
			if err := runPRPhase(projectRoot, featureID, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
		prNum, prURL, _ := getPRInfo()
		cp.PRNumber = &prNum
		cp.PRURL = prURL
			cp.Phase = orchestrate.PhaseCI
			orchestrate.SaveCheckpoint(cpPath, cp)
		}
		if cp.Phase == orchestrate.PhaseCI {
			pr := 0
			if cp.PRNumber != nil {
				pr = *cp.PRNumber
			}
			if pr == 0 {
				pr, _, _ = getPRInfo()
			}
			if pr > 0 {
				if err := runCILoop(pr, featureID, cpPath, runsPath); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
				cp.Phase = orchestrate.PhaseDone
				orchestrate.SaveCheckpoint(cpPath, cp)
			}
		}
		return
	}

	action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	switch action.Action {
	case "build":
		fmt.Printf("INVOKE: @build %s\n", action.WSID)
	case "review":
		fmt.Printf("INVOKE: @review %s\n", action.Feature)
	case "pr":
		fmt.Println("INVOKE: git push && gh pr create")
	case "ci-loop":
		fmt.Printf("INVOKE: sdp-ci-loop --pr %d --feature %s\n", action.PR, action.Feature)
	case "done":
		fmt.Println("CI GREEN - @oneshot complete")
	}
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func ensureRunFile(dir, featureID, branch string) {
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

func runPRPhase(projectRoot, featureID string, cp *orchestrate.Checkpoint) error {
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

func getPRInfo() (int, string, error) {
	branch, err := currentBranch()
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

func runOpenCodeLoop(projectRoot, featureID, cpPath, runsPath string, cp *orchestrate.Checkpoint, workstreams []string) {
	ctx := context.Background()
	for {
		action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		switch action.Action {
		case "build":
			commit, err := orchestrate.RunBuildPhase(ctx, projectRoot, action.WSID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "opencode build failed: %v\n", err)
				os.Exit(1)
			}
			_ = orchestrate.Advance(cp, workstreams, commit)
			_ = orchestrate.SaveCheckpoint(cpPath, cp)
		case "review":
			approved, err := orchestrate.RunReviewPhase(ctx, projectRoot, action.Feature)
			if err != nil || !approved {
				fmt.Fprintf(os.Stderr, "opencode review failed or not approved: %v\n", err)
				os.Exit(1)
			}
			_ = orchestrate.Advance(cp, workstreams, "")
			_ = orchestrate.SaveCheckpoint(cpPath, cp)
		case "pr":
			if err := runPRPhase(projectRoot, featureID, cp); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			prNum, prURL, _ := getPRInfo()
			cp.PRNumber = &prNum
			cp.PRURL = prURL
			cp.Phase = orchestrate.PhaseCI
			_ = orchestrate.SaveCheckpoint(cpPath, cp)
		case "ci-loop":
			pr := 0
			if cp.PRNumber != nil {
				pr = *cp.PRNumber
			}
			if pr == 0 {
				pr, _, _ = getPRInfo()
			}
			if pr > 0 {
				if err := runCILoop(pr, featureID, cpPath, runsPath); err != nil {
					fmt.Fprintf(os.Stderr, "error: %v\n", err)
					os.Exit(1)
				}
			}
			cp.Phase = orchestrate.PhaseDone
			_ = orchestrate.SaveCheckpoint(cpPath, cp)
		case "done":
			fmt.Println("CI GREEN - @oneshot complete")
			return
		}
	}
}

func runCILoop(pr int, featureID, checkpointDir, runsDir string) error {
	path, err := exec.LookPath("sdp-ci-loop")
	if err != nil {
		path = "sdp-ci-loop"
	}
	cmd := exec.Command(path, "--pr", fmt.Sprintf("%d", pr), "--feature", featureID, "--checkpoint-dir", checkpointDir, "--runs-dir", runsDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

