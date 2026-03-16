package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sdp_dev/internal/orchestrate"
)

type readyIssue struct {
	ID string `json:"id"`
}

func runStatus(projectRoot, featureID string, cp *orchestrate.Checkpoint, workstreams []string) {
	var pending []string
	if cp != nil {
		for _, ws := range cp.Workstreams {
			if ws.Status != "done" {
				pending = append(pending, ws.ID)
			}
		}
	} else {
		pending = workstreams
	}

	beadsCount := "N/A"
	if path, err := exec.LookPath("bd"); err == nil {
		cmd := exec.Command(path, "ready", "--json", "-n", "0")
		cmd.Dir = projectRoot
		out, err := cmd.Output()
		if err == nil {
			var issues []readyIssue
			if err := json.Unmarshal(out, &issues); err == nil {
				beadsCount = fmt.Sprintf("%d", len(issues))
			}
		}
	}

	action, err := orchestrate.ComputeNextAction(cp, workstreams, projectRoot)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	actionJSON, _ := json.Marshal(action)

	fmt.Println("## Feature Status:", featureID)
	fmt.Println()
	fmt.Println("**Pending workstreams:**", strings.Join(pending, ", "))
	if len(pending) == 0 && cp != nil && cp.Phase != orchestrate.PhaseInit {
		fmt.Println("(all done)")
	}
	fmt.Println()
	fmt.Println("**Open beads (bd ready):**", beadsCount)
	fmt.Println()
	fmt.Println("**Next action:**")
	fmt.Println("```json")
	fmt.Println(string(actionJSON))
	fmt.Println("```")
	fmt.Println()
	fmt.Println("**Run:** `go run ./cmd/sdp-orchestrate --feature", featureID, "--next-action`")
}
