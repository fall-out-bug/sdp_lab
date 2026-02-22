package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"sdp_dev/internal/observability"
	"sdp_dev/internal/orchestrator"
)

func main() {
	_, _ = observability.SetupTracing("orchestrator")

	host := flag.String("host", "", "SSH host (user@ip) for remote dispatch")
	port := flag.String("port", "22", "SSH port")
	issue := flag.String("issue", "", "Specific issue ID (optional; if empty, pick from ready)")
	feature := flag.String("feature", "", "Feature/epic ID to decompose and orchestrate e2e")
	workDir := flag.String("work-dir", "", "Working directory (default: cwd)")
	inCluster := flag.Bool("in-cluster", false, "Use in-cluster dispatch (kubectl exec)")
	loop := flag.Bool("loop", false, "Run continuously")
	interval := flag.Duration("interval", 30*time.Second, "Poll interval when looping")
	namespace := flag.String("namespace", "sdp-workers", "Kubernetes namespace for opencode-agent")
	flag.Parse()

	wd := *workDir
	if wd == "" {
		var err error
		wd, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	tracker, err := orchestrator.RunTrackerFromWorkDir(wd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := tracker.EnsureRunsDir(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	labels := []string{"autonomy", "strict-evidence"}
	lockDir := os.Getenv("SDP_LOCK_DIR") // persistent path in K8s (e.g. PVC) so locks survive restarts
	scheduler := orchestrator.NewScheduler(wd, lockDir, labels, 10)

	var runOne func() error
	if *feature != "" {
		runOne = func() error {
			return runFeature(wd, *feature, *host, *port, *inCluster, *namespace, scheduler, tracker)
		}
	} else if *issue != "" {
		runOne = func() error {
			return runIssue(wd, *issue, *host, *port, *inCluster, *namespace, scheduler, tracker)
		}
	} else {
		runOne = func() error {
			if err := scheduler.Adapter().Sync(true); err != nil {
				return fmt.Errorf("bd sync: %w", err)
			}
			id, err := scheduler.PickOne()
			if err != nil {
				return err
			}
			if id == "" {
				return nil // no work
			}
			defer scheduler.Unlock(id)
			return runIssue(wd, id, *host, *port, *inCluster, *namespace, scheduler, tracker)
		}
	}

	if !*loop {
		if err := runOne(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	for {
		if err := runOne(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
		time.Sleep(*interval)
	}
}

func runIssue(workDir, issueID, host, port string, inCluster bool, namespace string, sched *orchestrator.Scheduler, tracker *orchestrator.RunTracker) error {
	// Short-circuit if already closed
	iss, err := sched.Adapter().Show(issueID)
	if err != nil {
		return fmt.Errorf("show issue: %w", err)
	}
	if iss.Status == "closed" {
		return nil
	}

	runID := orchestrator.RunID(issueID)
	_, err = tracker.Create(runID, issueID, "cmd/orchestrator", host, 10, 300)
	if err != nil {
		return fmt.Errorf("create run: %w", err)
	}

	var disp orchestrator.Dispatcher
	if inCluster {
		disp = orchestrator.NewInClusterDispatcher(workDir)
	} else {
		if host == "" {
			return fmt.Errorf("--host required for SSH dispatch")
		}
		disp = orchestrator.NewSSHDispatcher(workDir)
	}

	cfg := orchestrator.DispatchConfig{
		Runtime:   orchestrator.SelectRuntime(iss.Labels),
		Host:      host,
		Port:      port,
		IssueID:   issueID,
		Namespace: namespace,
		InCluster: inCluster,
	}

	_ = tracker.AppendPhase(runID, "dispatch", "running", "triggering agent cycle", "")
	if err := disp.Dispatch(cfg); err != nil {
		_ = tracker.AppendPhase(runID, "dispatch", "failed", err.Error(), "")
		return err
	}
	_ = tracker.AppendPhase(runID, "dispatch", "ok", "agent cycle triggered", "")

	// Write run file path for compatibility with orchestrate script consumers
	runPath := filepath.Join(workDir, ".sdp", "runs", runID+".json")
	fmt.Printf("[orchestrator] run_id=%s run_file=%s issue=%s\n", runID, runPath, issueID)
	return nil
}

func runFeature(workDir, featureID, host, port string, inCluster bool, namespace string, sched *orchestrator.Scheduler, tracker *orchestrator.RunTracker) error {
	// Short-circuit if already closed
	iss, err := sched.Adapter().Show(featureID)
	if err != nil {
		return fmt.Errorf("show feature: %w", err)
	}
	if iss.Status == "closed" {
		return nil
	}
	if err := sched.Adapter().Sync(true); err != nil {
		return fmt.Errorf("bd sync: %w", err)
	}
	created, err := orchestrator.Decompose(context.Background(), sched.Adapter(), *iss, workDir, "glm-5")
	if err != nil {
		return fmt.Errorf("decompose: %w", err)
	}
	if err := sched.Adapter().Sync(false); err != nil {
		return fmt.Errorf("bd sync after decompose: %w", err)
	}
	for _, c := range created {
		if err := runIssue(workDir, c.ID, host, port, inCluster, namespace, sched, tracker); err != nil {
			fmt.Fprintf(os.Stderr, "[orchestrator] subtask %s: %v\n", c.ID, err)
		}
	}
	return nil
}
