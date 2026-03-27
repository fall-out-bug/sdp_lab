package tower

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// CardView is a simplified card for the board.
type CardView struct {
	ID            string
	Title         string
	Description   string
	Priority      int
	Labels        []string
	Status        string // open/closed
	Phase         string // build, clarify, plan, evaluate, ""
	ExecutorState string // running, completed, ""
	Project       string // extracted from sdp:project-* label
}

// Column represents a kanban column.
type Column struct {
	Name  string
	Cards []CardView
}

// BoardData is the full board view.
type BoardData struct {
	Columns []Column
}

// CardDetail extends CardView with evidence.
type CardDetail struct {
	CardView
	Clarification *EvidenceClarification `json:"clarification,omitempty"`
	Build         *EvidenceBuild         `json:"build,omitempty"`
	Evaluation    *EvidenceEval          `json:"evaluation,omitempty"`
}

type EvidenceClarification struct {
	Status    string   `json:"status"`
	Questions []string `json:"questions,omitempty"`
}

type EvidenceBuild struct {
	ExitCode int      `json:"exit_code"`
	Executor string   `json:"executor"`
	Status   string   `json:"status"`
	Commit   string   `json:"commit,omitempty"`
	Artifacts []map[string]any `json:"artifacts,omitempty"`
}

type EvidenceEval struct {
	Verdict string            `json:"verdict"`
	Score   float64           `json:"score"`
	Passed  map[string]bool   `json:"passed"`
	Findings []string         `json:"findings,omitempty"`
}

// LoadBoard builds the kanban board from bd list output.
func LoadBoard(issuesJSON []byte) (*BoardData, error) {
	var raw []struct {
		ID          string   `json:"id"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Priority    int      `json:"priority"`
		Labels      []string `json:"labels"`
		Status      string   `json:"status"`
		Metadata    any      `json:"metadata"`
	}
	if err := json.Unmarshal(issuesJSON, &raw); err != nil {
		return nil, err
	}

	board := &BoardData{
		Columns: []Column{
			{Name: "Backlog"},
			{Name: "Ready"},
			{Name: "In Progress"},
			{Name: "Review"},
			{Name: "Done"},
		},
	}

	for _, r := range raw {
		cv := CardView{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Priority:    r.Priority,
			Labels:      r.Labels,
			Status:      r.Status,
		}

		// Extract phase from labels
		for _, l := range r.Labels {
			if strings.HasPrefix(l, "sdp:phase-") {
				cv.Phase = strings.TrimPrefix(l, "sdp:phase-")
			}
			if strings.HasPrefix(l, "sdp:project-") {
				cv.Project = strings.TrimPrefix(l, "sdp:project-")
			}
		}

		// Extract executor state from metadata
		if r.Metadata != nil {
			if m, ok := r.Metadata.(map[string]any); ok {
				if sdp, ok := m["sdp"].(map[string]any); ok {
					if exec, ok := sdp["executor"].(map[string]any); ok {
						if state, ok := exec["state"].(string); ok {
							cv.ExecutorState = state
						}
					}
				}
			}
		}

		// Assign to column
		colIdx := columnFor(cv)
		board.Columns[colIdx].Cards = append(board.Columns[colIdx].Cards, cv)
	}

	return board, nil
}

func columnFor(cv CardView) int {
	if cv.Status == "closed" {
		return 4 // Done
	}
	switch cv.Phase {
	case "build":
		if cv.ExecutorState == "running" {
			return 2 // In Progress
		}
		return 2
	case "clarify", "plan":
		return 1 // Ready
	case "evaluate":
		return 3 // Review
	}
	// Check for gate:human label
	if slices.Contains(cv.Labels, "sdp:gate:human") {
		return 3 // Review
	}
	return 0 // Backlog
}

// LoadCardDetail loads evidence from artifact files.
func LoadCardDetail(cardID, projectRoot string) (*CardDetail, error) {
	detail := &CardDetail{CardView: CardView{ID: cardID}}

	artDir := filepath.Join(projectRoot, ".sdp", "artifacts", cardID)

	if data, err := os.ReadFile(filepath.Join(artDir, "clarification.json")); err == nil {
		var ev EvidenceClarification
		if json.Unmarshal(data, &ev) == nil {
			detail.Clarification = &ev
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "build.json")); err == nil {
		var ev EvidenceBuild
		if json.Unmarshal(data, &ev) == nil {
			detail.Build = &ev
			// Extract commit from artifacts
			for _, a := range ev.Artifacts {
				if ref, ok := a["reference"].(string); ok {
					ev.Commit = ref
					break
				}
			}
			detail.Build = &ev
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "evaluation.json")); err == nil {
		var ev EvidenceEval
		if json.Unmarshal(data, &ev) == nil {
			detail.Evaluation = &ev
		}
	}

	return detail, nil
}
