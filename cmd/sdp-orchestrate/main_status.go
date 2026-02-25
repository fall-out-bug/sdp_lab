package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"sdp_dev/internal/orchestrate"
)

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
		cmd := exec.Command(path, "ready")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			n := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "---") {
					n++
				}
			}
			beadsCount = fmt.Sprintf("%d", n)
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
