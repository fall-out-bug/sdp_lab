// adapter-controller is a placeholder for the kubeopencode adapter controller.
// It watches Task/Agent CRDs and drives the adapter layer (intent translator,
// lifecycle reconciler, evidence projector, policy gate).
//
// Full implementation requires:
// - kubeopencode client-go types
// - informer/watch on Task and Agent CRDs
// - reconciliation loop that calls adapter components
//
// Run: go run ./cmd/adapter-controller
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"sdp_dev/internal/adapter"
	"sdp_dev/internal/beads"
)

func main() {
	workDir, _ := os.Getwd()
	if d := os.Getenv("SDP_WORK_DIR"); d != "" {
		workDir = d
	}

	translator := adapter.NewIntentTranslator()
	lockMgr := adapter.NewRunLockManager(filepath.Join(os.TempDir(), "sdp-adapter-locks"))
	policyGate := adapter.NewPolicyGate()
	projector := adapter.NewEvidenceProjector(workDir)

	// Demo: translate a Beads issue (requires bd and .beads)
	bdAdapter := beads.NewAdapter(workDir)
	issues, err := bdAdapter.Ready([]string{"autonomy"}, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bd ready: %v\n", err)
		os.Exit(1)
	}
	if len(issues) == 0 {
		fmt.Println("No ready issues; adapter-controller is a no-op")
		return
	}

	issue := &issues[0]
	intent, err := translator.Translate(issue, issue.ID+"-1")
	if err != nil {
		fmt.Fprintf(os.Stderr, "translate: %v\n", err)
		os.Exit(1)
	}

	// Policy gate
	gr := policyGate.PreDispatchModelAllowlist("glm-4.7")
	if !gr.Passed {
		fmt.Fprintf(os.Stderr, "policy gate: %s\n", gr.Reason)
		os.Exit(1)
	}

	// Run lock
	runID, acquired, err := lockMgr.TryAcquire(issue.ID, intent.RunID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lock: %v\n", err)
		os.Exit(1)
	}
	if !acquired {
		fmt.Println("Issue already locked; skipping")
		return
	}
	defer func() {
		if err := lockMgr.Release(issue.ID); err != nil {
			fmt.Fprintf(os.Stderr, "release lock: %v\n", err)
		}
	}()

	// Evidence projection
	path, err := projector.ProjectFromIntent(intent, map[string]string{"coder": "placeholder"}, runID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "project: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Evidence projected to %s\n", path)
}
