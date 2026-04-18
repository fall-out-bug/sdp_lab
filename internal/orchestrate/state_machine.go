package orchestrate

import (
	"fmt"
	"path/filepath"
	"strings"
)

// NextAction describes what the agent should do next.
type NextAction struct {
	Action  string `json:"action"` // build, review, pr, ci-loop, done
	WSID    string `json:"ws_id,omitempty"`
	Feature string `json:"feature,omitempty"`
	PR      int    `json:"pr,omitempty"`
}

// FormatNextAction returns a human-readable string for a NextAction.
func FormatNextAction(a *NextAction) string {
	switch a.Action {
	case "init":
		return "Phase: init -- run advance to begin build"
	case "build":
		return fmt.Sprintf("Phase: build -- next workstream: %s (feature %s)", a.WSID, a.Feature)
	case "review":
		return fmt.Sprintf("Phase: review -- feature %s ready for review", a.Feature)
	case "pr":
		return fmt.Sprintf("Phase: pr -- create/update pull request for %s", a.Feature)
	case "ci-loop":
		if a.PR > 0 {
			return fmt.Sprintf("Phase: ci -- run CI loop for PR #%d (feature %s)", a.PR, a.Feature)
		}
		return fmt.Sprintf("Phase: ci -- run CI loop for %s (no PR yet)", a.Feature)
	case "qa":
		return fmt.Sprintf("Phase: qa -- run QA for feature %s", a.Feature)
	case "done":
		return "Phase: done -- all phases complete"
	default:
		return fmt.Sprintf("Phase: %s", a.Action)
	}
}

// FormatCheckpointStatus returns a human-readable status summary for a checkpoint.
func FormatCheckpointStatus(featureID string, cp *Checkpoint, workstreams []string, action *NextAction) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "## Feature Status: %s\n\n", featureID)

	// Phase
	phase := "unknown"
	if cp != nil {
		phase = cp.Phase
	}
	fmt.Fprintf(&sb, "**Phase:** %s\n\n", phase)

	// Workstream table
	sb.WriteString("**Workstreams:**\n\n")
	if cp != nil && len(cp.Workstreams) > 0 {
		sb.WriteString("| # | Workstream | Status |\n")
		sb.WriteString("|---|------------|--------|\n")
		done := 0
		for i, ws := range cp.Workstreams {
			var marker string
			switch ws.Status {
			case "done":
				marker = "done"
				done++
			case "in_progress":
				marker = "in-progress"
			case "pending":
				marker = "pending"
			default:
				marker = ws.Status
			}
			fmt.Fprintf(&sb, "| %d | %s | %s |\n", i+1, ws.ID, marker)
		}
fmt.Fprintf(&sb, "\nProgress: %d/%d workstreams done\n", done, len(cp.Workstreams))	} else {
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
		if len(pending) == 0 {
			sb.WriteString("(all done)\n")
		} else {
			sb.WriteString(strings.Join(pending, ", ") + "\n")
		}
	}

	// Next action
	sb.WriteString("\n**Next action:** ")
	if action != nil {
		sb.WriteString(FormatNextAction(action))
	} else {
		sb.WriteString("(unknown)")
	}
	sb.WriteString("\n")

	return sb.String()
}

// ComputeNextAction returns the next action based on checkpoint state.
func ComputeNextAction(cp *Checkpoint, workstreams []string, projectRoot string) (*NextAction, error) {
	switch cp.Phase {
	case PhaseInit:
		return &NextAction{Action: "init"}, nil
	case PhaseBuild:
		for _, ws := range cp.Workstreams {
			if ws.Status != "done" {
				// Use ws.ID directly instead of indexing workstreams slice
				// to avoid index out of bounds panic if lengths differ
				return &NextAction{Action: "build", WSID: ws.ID, Feature: cp.FeatureID}, nil
			}
		}
		return &NextAction{Action: "review", Feature: cp.FeatureID}, nil
	case PhaseReview:
		return &NextAction{Action: "review", Feature: cp.FeatureID}, nil
	case PhasePR:
		return &NextAction{Action: "pr", Feature: cp.FeatureID}, nil
	case PhaseCI:
		pr := 0
		if cp.PRNumber != nil {
			pr = *cp.PRNumber
		}
		return &NextAction{Action: "ci-loop", Feature: cp.FeatureID, PR: pr}, nil
	case PhaseQA:
		return &NextAction{Action: "qa", Feature: cp.FeatureID}, nil
	case PhaseDone:
		return &NextAction{Action: "done"}, nil
	default:
		return nil, fmt.Errorf("unknown phase %q", cp.Phase)
	}
}

// CurrentBuildWS returns the workstream ID being built (first non-done) when in build phase.
func CurrentBuildWS(cp *Checkpoint) string {
	if cp.Phase != PhaseBuild {
		return ""
	}
	for _, ws := range cp.Workstreams {
		if ws.Status != "done" {
			return ws.ID
		}
	}
	return ""
}

// Advance transitions the checkpoint to the next phase.
// For build phase, result is the commit hash of the completed workstream.
func Advance(cp *Checkpoint, workstreams []string, result string) error {
	// Validate the transition before mutating any state
	if err := ValidateAdvance(cp, workstreams); err != nil {
		return err
	}
	return advanceCheckpoint(cp, workstreams, result)
}

// AdvanceUnvalidated performs the same state transitions as Advance but skips
// FSM pre-validation. This is needed for batch/autonomous mode where the
// pre-validation logic in computeNextPhase is overly strict for the last
// workstream in the build phase (it computes the post-advance target phase
// but then validates against the pre-advance state).
func AdvanceUnvalidated(cp *Checkpoint, workstreams []string, result string) error {
	return advanceCheckpoint(cp, workstreams, result)
}

// advanceCheckpoint contains the shared state transition logic used by both
// Advance (with validation) and AdvanceUnvalidated (without validation).
func advanceCheckpoint(cp *Checkpoint, workstreams []string, result string) error {
	switch cp.Phase {
	case PhaseInit:
		cp.Phase = PhaseBuild
		cp.Workstreams = make([]WSStatus, len(workstreams))
		for i, ws := range workstreams {
			cp.Workstreams[i] = WSStatus{ID: ws, Status: "pending"}
		}
		return nil
	case PhaseBuild:
		for i := range cp.Workstreams {
			if cp.Workstreams[i].Status != "done" {
				cp.Workstreams[i].Status = "done"
				if result != "" {
					cp.Workstreams[i].Commit = result
				}
				cp.Workstreams[i].Attempts++
				break
			}
		}
		allDone := true
		for _, ws := range cp.Workstreams {
			if ws.Status != "done" {
				allDone = false
				break
			}
		}
		if allDone {
			cp.Phase = PhaseReview
			if cp.Review == nil {
				cp.Review = &ReviewStatus{Iteration: 0, Status: "pending"}
			}
		}
		return nil
	case PhaseReview:
		cp.Phase = PhasePR
		if cp.Review != nil {
			cp.Review.Status = "approved"
		}
		return nil
	case PhasePR:
		cp.Phase = PhaseCI
		return nil
	case PhaseCI:
		cp.Phase = PhaseQA
		if cp.QA == nil {
			cp.QA = &QAStatus{Iteration: 0, Status: "pending"}
		}
		cp.QA.Iteration++
		return nil
	case PhaseQA:
		cp.Phase = PhaseDone
		if cp.QA != nil {
			cp.QA.Status = "passed"
		}
		return nil
	case PhaseDone:
		return nil
	default:
		return fmt.Errorf("unknown phase %q", cp.Phase)
	}
}

// CreateInitialCheckpoint builds a new checkpoint for a feature.
func CreateInitialCheckpoint(featureID, branch string, workstreams []string) *Checkpoint {
	ws := make([]WSStatus, len(workstreams))
	for i, id := range workstreams {
		ws[i] = WSStatus{ID: id, Status: "pending"}
	}
	return &Checkpoint{
		Schema:      "1.0",
		FeatureID:   featureID,
		Branch:      branch,
		Phase:       PhaseInit,
		Workstreams: ws,
		Review:      &ReviewStatus{Iteration: 0, Status: "pending"},
		QA:          &QAStatus{Iteration: 0, Status: "pending"},
	}
}

// FindProjectRoot walks up from dir to find a directory containing docs/workstreams.
func FindProjectRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; d != "" && d != "/"; d = filepath.Dir(d) {
		check := filepath.Join(d, "docs", "workstreams", "backlog")
		if ents, err := filepath.Glob(filepath.Join(check, "*.md")); err == nil && len(ents) > 0 {
			return d, nil
		}
	}
	return "", fmt.Errorf("project root not found (no docs/workstreams/backlog)")
}
