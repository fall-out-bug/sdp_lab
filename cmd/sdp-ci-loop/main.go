package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sdp_dev/internal/ciloop"
)

const execTimeout = 30 * time.Second
// exitCodes matches WS AC.
const (
	exitGreen    = 0
	exitEscalate = 1
	exitMaxIter  = 2
)

func main() {
	prNum := flag.Int("pr", 0, "PR number to poll")
	feature := flag.String("feature", "", "Feature ID (e.g. F014)")
	maxIter := flag.Int("max-iter", 5, "Max fix iterations before exit 2")
	checkpointDir := flag.String("checkpoint-dir", ".sdp/checkpoints", "Directory containing checkpoint files")
	runsDir := flag.String("runs-dir", ".sdp/runs", "Directory containing run files")
	pollDelay := flag.Duration("poll-delay", 60*time.Second, "Delay between polls")
	retryDelay := flag.Duration("retry-delay", 60*time.Second, "Delay when checks are pending")
	flag.Parse()

	// Resolve PR number and branch: flags take precedence, then checkpoint.
	if *prNum == 0 && *feature != "" {
		cp, err := ciloop.LoadCheckpoint(*checkpointDir, *feature)
		if err != nil {
			slog.Debug("cannot load checkpoint", "error", err, "feature", *feature)
		} else if cp.PRNumber != nil {
			*prNum = *cp.PRNumber
		}
	}

	if *prNum == 0 {
		fmt.Fprintln(os.Stderr, "error: --pr is required (or set pr_number in checkpoint)")
		flag.Usage()
		os.Exit(exitEscalate)
	}

	runner := &execRunner{}
	poller := ciloop.NewPoller(runner)

	onEscalate := func(checks []ciloop.CheckResult) error {
		names := make([]string, len(checks))
		for i, c := range checks {
			names[i] = c.Name
		}
		title := fmt.Sprintf("CI BLOCKED: %s (PR #%d)", strings.Join(names, ", "), *prNum)
		slog.Warn("escalating", "title", title, "checks", names, "pr", *prNum)
		cmd := exec.Command("bd", "create", "--title", title, "--priority", "0", "--labels", fmt.Sprintf("ci-finding,%s", *feature))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			slog.Warn("bd create failed", "error", err, "title", title)
		}
		return nil
	}

	fixer := ciloop.NewFixer(ciloop.FixerOptions{
		PRNumber:  *prNum,
		FeatureID: *feature,
		Committer: &gitCommitter{},
		LogFetcher: &ghLogFetcher{runner: runner},
		DecisionLogger: func(decision, rationale string) error {
			fmt.Printf("DECISION: %s — %s\n", decision, rationale)
			return nil
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	opts := ciloop.LoopOptions{Context: ctx, PRNumber: *prNum, MaxIter: *maxIter,
		MaxPendingRetries: ciloop.DefaultMaxPendingRetries, PollDelay: *pollDelay, RetryDelay: *retryDelay,
		Poller: poller, OnEscalate: onEscalate, Fixer: fixer}

	result, err := ciloop.RunLoop(opts)
	if err != nil {
		slog.Error("ci-loop failed", "error", err, "pr", *prNum, "feature", *feature)
		os.Exit(exitEscalate)
	}

	switch result {
	case ciloop.ResultGreen:
		fmt.Println("CI GREEN")
		if *feature != "" {
			if err := updateArtifacts(*checkpointDir, *runsDir, *feature); err != nil {
				slog.Error("update artifacts failed", "error", err, "feature", *feature)
				os.Exit(exitEscalate)
			}
		}
		os.Exit(exitGreen)

	case ciloop.ResultEscalated:
		slog.Warn("CI escalated", "pr", *prNum, "feature", *feature)
		os.Exit(exitEscalate)

	case ciloop.ResultMaxIter:
		slog.Warn("CI max iterations exceeded", "max_iter", *maxIter, "pr", *prNum)
		os.Exit(exitMaxIter)
	}
}

func updateArtifacts(checkpointDir, runsDir, featureID string) error {
	cp, err := ciloop.LoadCheckpoint(checkpointDir, featureID)
	if err == nil {
		if saveErr := ciloop.SaveCheckpoint(checkpointDir, cp); saveErr != nil {
			return fmt.Errorf("save checkpoint: %w", saveErr)
		}
	}
	if err := ciloop.AppendRunEvent(runsDir, featureID, "ci", "ok", ""); err != nil {
		return fmt.Errorf("append run event: %w", err)
	}
	return nil
}

// execRunner implements CommandRunner via os/exec with a 30s timeout.
type execRunner struct{}

func (e *execRunner) Run(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), execTimeout)
	defer cancel()
	return exec.CommandContext(ctx, name, args...).Output()
}

// gitCommitter implements Committer via git CLI.
type gitCommitter struct{}

func (g *gitCommitter) Commit(msg string) error {
	add := exec.Command("git", "add", ".sdp/ci-fixes/")
	add.Stdout = os.Stdout
	add.Stderr = os.Stderr
	if err := add.Run(); err != nil {
		return err
	}
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (g *gitCommitter) Push() error {
	cmd := exec.Command("git", "push")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// ghLogFetcher implements LogFetcher via gh CLI (uses CommandRunner interface: v9w3).
type ghLogFetcher struct {
	runner ciloop.CommandRunner
}

func (g *ghLogFetcher) FailedLogs(prNumber int) (string, error) {
	runID, err := g.runner.Run("gh", "run", "list",
		"--branch", currentBranch(),
		"--json", "databaseId,conclusion",
		"--jq", `.[] | select(.conclusion == "failure") | .databaseId`,
	)
	if err != nil {
		return "", fmt.Errorf("list failed runs: %w", err)
	}
	id := strings.TrimSpace(string(runID))
	if id == "" {
		return "", fmt.Errorf("no failed run found for PR #%d", prNumber)
	}
	if nl := strings.Index(id, "\n"); nl > 0 {
		id = id[:nl]
	}
	out, err := g.runner.Run("gh", "run", "view", id, "--log-failed")
	if err != nil {
		return "", fmt.Errorf("fetch run logs: %w", err)
	}
	return string(out), nil
}

func currentBranch() string {
	out, err := exec.Command("git", "branch", "--show-current").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
