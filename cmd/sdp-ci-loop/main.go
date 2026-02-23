package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
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
			fmt.Fprintf(os.Stderr, "warning: cannot load checkpoint: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "ESCALATE: %s\n", title)
		// Create beads issue for the blocker.
		cmd := exec.Command("bd", "create",
			"--title", title,
			"--priority", "0",
			"--labels", fmt.Sprintf("ci-finding,%s", *feature),
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: bd create failed: %v\n", err)
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

	opts := ciloop.LoopOptions{
		PRNumber:          *prNum,
		MaxIter:           *maxIter,
		MaxPendingRetries: ciloop.DefaultMaxPendingRetries,
		PollDelay:         *pollDelay,
		RetryDelay:        *retryDelay,
		Poller:            poller,
		OnEscalate:        onEscalate,
		Fixer:             fixer,
	}

	result, err := ciloop.RunLoop(opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ci-loop error: %v\n", err)
		os.Exit(exitEscalate)
	}

	switch result {
	case ciloop.ResultGreen:
		fmt.Println("CI GREEN")
		if *feature != "" {
			updateArtifacts(*checkpointDir, *runsDir, *feature)
		}
		os.Exit(exitGreen)

	case ciloop.ResultEscalated:
		fmt.Fprintln(os.Stderr, "CI ESCALATED — see beads issue")
		os.Exit(exitEscalate)

	case ciloop.ResultMaxIter:
		fmt.Fprintf(os.Stderr, "CI max iterations (%d) exceeded\n", *maxIter)
		os.Exit(exitMaxIter)
	}
}

func updateArtifacts(checkpointDir, runsDir, featureID string) {
	cp, err := ciloop.LoadCheckpoint(checkpointDir, featureID)
	if err == nil {
		if saveErr := ciloop.SaveCheckpoint(checkpointDir, cp); saveErr != nil {
			fmt.Fprintf(os.Stderr, "warning: save checkpoint: %v\n", saveErr)
		}
	}
	if err := ciloop.AppendRunEvent(runsDir, featureID, "ci", "ok", ""); err != nil {
		fmt.Fprintf(os.Stderr, "warning: append run event: %v\n", err)
	}
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
	cmd := exec.Command("git", "commit", "-am", msg)
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

// ghLogFetcher implements LogFetcher via gh CLI.
type ghLogFetcher struct {
	runner *execRunner
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
	// Take only first line if multiple.
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
