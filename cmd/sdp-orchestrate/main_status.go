package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/fall-out-bug/sdp_lab/internal/orchestrate"
)

type readyIssue struct {
	ID string `json:"id"`
}

type statusJSON struct {
	FeatureID   string                   `json:"feature_id"`
	Phase       string                   `json:"phase"`
	Workstreams []orchestrate.WSStatus   `json:"workstreams"`
	Pending     []string                 `json:"pending"`
	BeadsReady  string                   `json:"beads_ready"`
	NextAction  *orchestrate.NextAction  `json:"next_action"`
	Interrupted bool                     `json:"interrupted"`
	ResumeHint  string                   `json:"resume_hint,omitempty"`
}

func runStatus(projectRoot, featureID string, cp *orchestrate.Checkpoint, workstreams []string, jsonOutput bool) {
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

	// Detect interruption: build phase with in_progress workstreams
	interrupted := false
	if cp != nil && cp.Phase == orchestrate.PhaseBuild {
		for _, ws := range cp.Workstreams {
			if ws.Status == "in_progress" {
				interrupted = true
				break
			}
		}
	}

	resumeHint := ""
	if interrupted {
		resumeHint = fmt.Sprintf("Resume with: sdp-orchestrate --feature %s --resume --runtime opencode", featureID)
	}

	if jsonOutput {
		sj := statusJSON{
			FeatureID:   featureID,
			Phase:       cp.Phase,
			Workstreams: cp.Workstreams,
			Pending:     pending,
			BeadsReady:  beadsCount,
			NextAction:  action,
			Interrupted: interrupted,
			ResumeHint:  resumeHint,
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(sj); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Human-readable output
	fmt.Print(orchestrate.FormatCheckpointStatus(featureID, cp, workstreams, action))
	fmt.Println()
	fmt.Println("**Open beads (bd ready):**", beadsCount)
	if interrupted {
		fmt.Println()
		fmt.Println("**Interrupted!**", resumeHint)
	}
}
