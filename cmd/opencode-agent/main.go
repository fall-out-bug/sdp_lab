package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"sdp_dev/internal/observability"
)

func emitObservability(phase string, status string, model string, startedAt time.Time, retryCount int, fallbackUsed bool, escalated bool) {
	issueID := strings.TrimSpace(os.Getenv("SDP_ISSUE_ID"))
	if issueID == "" {
		issueID = strings.TrimSpace(os.Getenv("SDP_ISSUE"))
	}
	evidenceLink := strings.TrimSpace(os.Getenv("SDP_EVIDENCE_CONTEXT_LINK"))
	prURL := strings.TrimSpace(os.Getenv("SDP_PR_URL"))
	if issueID != "" && evidenceLink == "" {
		evidenceLink = filepath.Join(".sdp", "evidence", issueID+".json")
	}
	if strings.TrimSpace(model) == "" {
		model = strings.TrimSpace(os.Getenv("SDP_MODEL"))
	}
	if strings.TrimSpace(model) == "" {
		model = "glm-4.7"
	}
	_ = observability.EmitIntakeRecords(os.Stderr, observability.IntakeEventInput{
		RunID:               strings.TrimSpace(os.Getenv("SDP_RUN_ID")),
		IssueID:             issueID,
		Phase:               phase,
		Status:              status,
		Component:           "opencode-agent",
		AgentRole:           "orchestrator",
		ModelName:           model,
		Elapsed:             time.Since(startedAt),
		RetryCount:          retryCount,
		FallbackUsed:        fallbackUsed,
		Escalated:           escalated,
		EvidenceContextLink: evidenceLink,
		PRURL:               prURL,
	})
}

func run(name string, args ...string) ([]byte, error) {
	return runWithModel("", name, args...)
}

func runWithModel(model string, name string, args ...string) ([]byte, error) {
	selectedModel := strings.TrimSpace(model)
	if selectedModel == "" {
		selectedModel = strings.TrimSpace(os.Getenv("SDP_MODEL"))
	}
	if selectedModel == "" {
		selectedModel = "glm-4.7"
	}
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), "SDP_RUNTIME=opencode", "SDP_MODEL="+selectedModel)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func runComponent(binary string, goPkg string) ([]byte, error) {
	return runComponentWithArgs(binary, goPkg)
}

func runComponentWithArgs(binary string, goPkg string, args ...string) ([]byte, error) {
	startedAt := time.Now()
	model := "glm-4.7"
	if binary == "swarm-reviewer" {
		model = "glm-5"
	}

	if os.Getenv("SDP_RUNTIME") == "opencode" && (binary == "swarm-worker" || binary == "swarm-reviewer") {
		if _, err := exec.LookPath("go"); err == nil {
			goArgs := append([]string{"run", goPkg}, args...)
			out, runErr := runWithModel(model, "go", goArgs...)
			emitObservability("execute", statusForError(runErr, false), model, startedAt, 0, true, false)
			return out, runErr
		}
	}

	if _, err := exec.LookPath(binary); err == nil {
		cmdArgs := append([]string{}, args...)
		out, runErr := runWithModel(model, binary, cmdArgs...)
		emitObservability("execute", statusForError(runErr, false), model, startedAt, 0, false, false)
		return out, runErr
	}
	goArgs := append([]string{"run", goPkg}, args...)
	out, runErr := runWithModel(model, "go", goArgs...)
	emitObservability("execute", statusForError(runErr, false), model, startedAt, 0, true, false)
	return out, runErr
}

func statusForError(err error, escalated bool) string {
	if err == nil {
		return "success"
	}
	if escalated {
		return "escalated"
	}
	return "failed"
}

func isTransientNetworkError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	needles := []string{
		"could not resolve host",
		"failed to connect",
		"i/o timeout",
		"no such host",
		"connection reset",
		"connection refused",
	}
	for _, needle := range needles {
		if strings.Contains(msg, needle) {
			return true
		}
	}
	return false
}

func preflightGitHubHealth() error {
	startedAt := time.Now()
	if _, err := run("gh", "auth", "status", "--hostname", "github.com"); err != nil {
		emitObservability("intake", "failed", "glm-4.7", startedAt, 0, false, true)
		return fmt.Errorf("preflight gh auth status: %w", err)
	}

	repoURL := os.Getenv("SDP_REPO_URL")
	if repoURL == "" {
		repoURL = "https://github.com/fall-out-bug/sdp_private.git"
	}
	if strings.HasPrefix(repoURL, "https://github.com/") {
		token := strings.TrimSpace(os.Getenv("GH_TOKEN"))
		if token == "" {
			token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
		}
		if token != "" {
			repoURL = "https://x-access-token:" + token + "@" + strings.TrimPrefix(repoURL, "https://")
		}
	}

	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if _, err := run("git", "ls-remote", "--exit-code", repoURL, "HEAD"); err == nil {
			emitObservability("intake", "success", "glm-4.7", startedAt, attempt-1, false, false)
			return nil
		} else {
			lastErr = err
			if !isTransientNetworkError(err) || attempt == 3 {
				break
			}
			emitObservability("intake", "retrying", "glm-4.7", startedAt, attempt, false, false)
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}

	emitObservability("intake", "escalated", "glm-4.7", startedAt, 3, false, true)
	return fmt.Errorf("preflight git ls-remote %s: %w", repoURL, lastErr)
}

func syncWorkspace() error {
	if _, err := os.Stat(filepath.Join(".", ".git")); err != nil {
		return nil
	}
	branch := os.Getenv("SDP_REPO_BRANCH")
	if branch == "" {
		branch = "master"
	}

	currentRaw, err := run("git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return err
	}
	current := strings.TrimSpace(string(currentRaw))

	dirtyRaw, err := run("git", "status", "--porcelain")
	if err != nil {
		return err
	}
	dirty := strings.TrimSpace(string(dirtyRaw)) != ""

	if dirty {
		namesRaw, err := run("git", "diff", "--name-only")
		if err != nil {
			return err
		}
		cachedRaw, err := run("git", "diff", "--cached", "--name-only")
		if err != nil {
			return err
		}
		nameSet := make(map[string]struct{})
		for _, n := range strings.Fields(strings.TrimSpace(string(namesRaw))) {
			nameSet[n] = struct{}{}
		}
		for _, n := range strings.Fields(strings.TrimSpace(string(cachedRaw))) {
			nameSet[n] = struct{}{}
		}
		names := make([]string, 0, len(nameSet))
		for n := range nameSet {
			names = append(names, n)
		}
		if len(names) > 0 {
			onlyBeadsSyncNoise := true
			for _, n := range names {
				if n != ".beads/metadata.json" && n != ".beads/issues.jsonl" {
					onlyBeadsSyncNoise = false
					break
				}
			}
			if onlyBeadsSyncNoise {
				_, _ = run("git", "reset", "HEAD", "--", ".beads/metadata.json", ".beads/issues.jsonl")
				if _, err := run("git", "checkout", "--", ".beads/metadata.json", ".beads/issues.jsonl"); err != nil {
					return err
				}
				dirty = false
			}
		}
		if len(names) == 1 && names[0] == ".beads/metadata.json" {
			if _, err := run("git", "checkout", "--", ".beads/metadata.json"); err != nil {
				return err
			}
			dirty = false
		}
	}

	if current != branch {
		if dirty {
			return fmt.Errorf("workspace dirty on branch %s; cannot switch to %s", current, branch)
		}
		if _, err := run("git", "checkout", branch); err != nil {
			return err
		}
	}

	if dirty {
		return nil
	}
	if _, err := run("git", "fetch", "origin", branch); err != nil {
		return err
	}
	if _, err := run("git", "rebase", "FETCH_HEAD"); err != nil {
		return err
	}
	return nil
}

func runCycle() error {
	cycleStart := time.Now()
	emitObservability("plan", "running", strings.TrimSpace(os.Getenv("SDP_MODEL")), cycleStart, 0, false, false)
	if err := syncWorkspace(); err != nil {
		emitObservability("plan", "blocked", strings.TrimSpace(os.Getenv("SDP_MODEL")), cycleStart, 0, false, true)
		return err
	}

	if err := preflightGitHubHealth(); err != nil {
		emitObservability("intake", "failed", strings.TrimSpace(os.Getenv("SDP_MODEL")), cycleStart, 0, false, true)
		return err
	}

	if _, err := run("bd", "sync", "--import-only"); err != nil {
		emitObservability("intake", "failed", strings.TrimSpace(os.Getenv("SDP_MODEL")), cycleStart, 0, false, true)
		return err
	}

	swarmWorkerArgs := []string{}
	if issueID := strings.TrimSpace(os.Getenv("SDP_ISSUE")); issueID != "" {
		swarmWorkerArgs = append(swarmWorkerArgs, "--issue", issueID)
	}
	out, err := runComponentWithArgs("swarm-worker", "./cmd/swarm-worker", swarmWorkerArgs...)
	if err != nil {
		return err
	}
	fmt.Print(string(out))

	if out, err := runComponent("swarm-reviewer", "./cmd/swarm-reviewer"); err != nil {
		emitObservability("review", "failed", "glm-5", cycleStart, 0, false, true)
		return err
	} else {
		fmt.Print(string(out))
	}

	emitObservability("publish", "success", strings.TrimSpace(os.Getenv("SDP_MODEL")), cycleStart, 0, false, false)
	return nil
}

func buildOpencodeObservabilityRecords(issueID string, model string, status string, retryCount int, fallbackUsed bool, escalated bool, evidenceContextLink string, prURL string, elapsed time.Duration) []map[string]any {
	return observability.BuildIntakeRecords(observability.IntakeEventInput{
		RunID:               "opencode-run-test",
		IssueID:             issueID,
		Phase:               "execute",
		Status:              status,
		Component:           "opencode-agent",
		AgentRole:           "orchestrator",
		ModelName:           model,
		Elapsed:             elapsed,
		RetryCount:          retryCount,
		FallbackUsed:        fallbackUsed,
		Escalated:           escalated,
		EvidenceContextLink: evidenceContextLink,
		PRURL:               prURL,
	})
}

func main() {
	loop := flag.Bool("loop", false, "Run continuously")
	interval := flag.Duration("interval", 30*time.Second, "Loop interval")
	flag.Parse()

	if !*loop {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	for {
		if err := runCycle(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(*interval)
	}
}
