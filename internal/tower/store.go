package tower

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// CardView is a card displayed on the board.
type CardView struct {
	ID            string
	Title         string
	Description   string
	Priority      int
	Labels        []string
	Status        string
	Phase         string
	ExecutorState string
	Project       string
	// Evidence preview (loaded in bulk for board view)
	Evidence *EvidencePreview
}

// EvidencePreview is a compact evidence summary shown on board cards.
type EvidencePreview struct {
	Clarified    bool
	NeedsClarify bool
	Questions    int
	BuildStatus  string // "", "success", "failed"
	BuildAgent   string
	BuildCommit  string
	EvalVerdict  string // "", "pass", "fail", "needs_review", "blocked"
	EvalScore    float64
	EvalCriteria map[string]bool
}

// Column represents a kanban column.
type Column struct {
	Name        string
	NameEN      string // for CSS classes
	Icon        string
	Color       string // tailwind border color
	Cards       []CardView
}

// BoardData is the full board view.
type BoardData struct {
	Columns    []Column
	Projects   []string // unique project labels for filter
	TotalCards int
}

// CardDetail is the full card detail with all evidence.
type CardDetail struct {
	ID            string
	Title         string
	Description   string
	Priority      int
	Labels        []string
	Status        string
	Phase         string
	ExecutorState string
	Project       string
	Clarification *EvidenceClarification
	Build         *EvidenceBuild
	Evaluation    *EvidenceEval
	Provenance    *EvidenceProvenance
}

type EvidenceClarification struct {
	Status       string   `json:"status"`
	Questions    []string `json:"questions,omitempty"`
	RawFeedback  string   `json:"raw_feedback,omitempty"`
	NormalizedIntent string `json:"normalized_intent,omitempty"`
	Intent       string
}

type EvidenceBuild struct {
	ExitCode  int             `json:"exit_code"`
	Executor  string          `json:"executor"`
	Status    string          `json:"status"`
	Summary   string          `json:"summary,omitempty"`
	Artifacts []BuildArtifact `json:"artifacts,omitempty"`
	Findings  any             `json:"findings,omitempty"`
	Timestamp string          `json:"timestamp,omitempty"`
}

type BuildArtifact struct {
	Type        string `json:"type"`
	Reference   string `json:"reference"`
	Description string `json:"description"`
}

type EvidenceEval struct {
	Verdict    string          `json:"verdict"`
	Score      float64         `json:"score"`
	Passed     map[string]bool `json:"passed"`
	Findings   []string        `json:"findings,omitempty"`
	RawFeedback string         `json:"raw_feedback,omitempty"`
	Timestamp  string          `json:"timestamp,omitempty"`
}

type EvidenceProvenance struct {
	Timestamp      string            `json:"timestamp"`
	ContractHash   string            `json:"contract_hash"`
	PacketHash     string            `json:"packet_hash"`
	PromptHash     string            `json:"prompt_hash"`
	ContextSources []ProvenanceSource `json:"context_sources"`
}

type ProvenanceSource struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// BoardFilter for query params.
type BoardFilter struct {
	Project string
	Phase   string
	Query   string
}

// LoadBoard builds the kanban from bd list output.
func LoadBoard(issuesJSON []byte, filter BoardFilter) (*BoardData, error) {
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

	projectSet := map[string]bool{}
	board := &BoardData{
		Columns: []Column{
			{Name: "Backlog", NameEN: "backlog", Icon: "📥", Color: "border-gray-600"},
			{Name: "Ready", NameEN: "ready", Icon: "🏁", Color: "border-blue-600"},
			{Name: "In Progress", NameEN: "in-progress", Icon: "⚡", Color: "border-yellow-600"},
			{Name: "Review", NameEN: "review", Icon: "🔍", Color: "border-purple-600"},
			{Name: "Done", NameEN: "done", Icon: "✅", Color: "border-green-600"},
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

		for _, l := range r.Labels {
			if strings.HasPrefix(l, "sdp:phase-") {
				cv.Phase = strings.TrimPrefix(l, "sdp:phase-")
			}
			if strings.HasPrefix(l, "sdp:project-") {
				cv.Project = strings.TrimPrefix(l, "sdp:project-")
				projectSet[cv.Project] = true
			}
		}

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

		// Apply filters
		if filter.Project != "" && cv.Project != filter.Project {
			continue
		}
		if filter.Phase != "" {
			if filter.Phase == "open" && cv.Status != "open" {
				continue
			}
			if filter.Phase != "open" && cv.Phase != filter.Phase {
				continue
			}
		}
		if filter.Query != "" {
			q := strings.ToLower(filter.Query)
			if !strings.Contains(strings.ToLower(cv.Title), q) &&
				!strings.Contains(strings.ToLower(cv.Description), q) &&
				!strings.Contains(strings.ToLower(cv.ID), q) {
				continue
			}
		}

		colIdx := columnFor(cv)
		board.Columns[colIdx].Cards = append(board.Columns[colIdx].Cards, cv)
		board.TotalCards++
	}

	// Build project list
	for p := range projectSet {
		board.Projects = append(board.Projects, p)
	}
	slices.Sort(board.Projects)

	return board, nil
}

// LoadEvidencePreview loads compact evidence for board cards.
func LoadEvidencePreview(cardID, projectRoot string) *EvidencePreview {
	artDir := filepath.Join(projectRoot, ".sdp", "artifacts", cardID)
	ep := &EvidencePreview{}

	if data, err := os.ReadFile(filepath.Join(artDir, "clarification.json")); err == nil {
		var ev struct {
			Status    string `json:"status"`
			Questions []any  `json:"questions"`
		}
		if json.Unmarshal(data, &ev) == nil {
			ep.NeedsClarify = ev.Status == "needs_clarification"
			ep.Clarified = ev.Status == "clarified"
			ep.Questions = len(ev.Questions)
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "build.json")); err == nil {
		var ev EvidenceBuild
		if json.Unmarshal(data, &ev) == nil {
			ep.BuildStatus = ev.Status
			ep.BuildAgent = ev.Executor
			for _, a := range ev.Artifacts {
				if a.Type == "commit" {
					ep.BuildCommit = a.Reference
					break
				}
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "evaluation.json")); err == nil {
		var ev EvidenceEval
		if json.Unmarshal(data, &ev) == nil {
			ep.EvalVerdict = ev.Verdict
			ep.EvalScore = ev.Score
			ep.EvalCriteria = ev.Passed
		}
	}

	return ep
}

func columnFor(cv CardView) int {
	if cv.Status == "closed" {
		return 4
	}
	switch cv.Phase {
	case "build":
		return 2
	case "clarify", "plan":
		return 1
	case "evaluate":
		return 3
	}
	if slices.Contains(cv.Labels, "sdp:gate:human") {
		return 3
	}
	return 0
}

// LoadCardDetail loads full card detail from bd show + evidence files.
func LoadCardDetail(showJSON []byte, cardID, projectRoot string) (*CardDetail, error) {
	var issues []map[string]any
	if err := json.Unmarshal(showJSON, &issues); err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, fmt.Errorf("card not found")
	}

	issue := issues[0]
	detail := &CardDetail{
		ID:        cardID,
		Title:     strVal(issue, "title"),
		Description: strVal(issue, "description"),
		Status:    strVal(issue, "status"),
		Priority:  intVal(issue, "priority"),
	}

	if labels, ok := issue["labels"].([]any); ok {
		for _, l := range labels {
			detail.Labels = append(detail.Labels, fmt.Sprint(l))
		}
	}
	for _, l := range detail.Labels {
		if strings.HasPrefix(l, "sdp:phase-") {
			detail.Phase = strings.TrimPrefix(l, "sdp:phase-")
		}
		if strings.HasPrefix(l, "sdp:project-") {
			detail.Project = strings.TrimPrefix(l, "sdp:project-")
		}
	}

	// Load evidence
	artDir := filepath.Join(projectRoot, ".sdp", "artifacts", cardID)

	if data, err := os.ReadFile(filepath.Join(artDir, "clarification.json")); err == nil {
		var ev struct {
			Status      string `json:"status"`
			Questions   []string `json:"questions"`
			RawFeedback string `json:"raw_feedback"`
			Card        struct {
				NormalizedIntent string `json:"normalized_intent"`
			} `json:"card"`
		}
		if json.Unmarshal(data, &ev) == nil {
			detail.Clarification = &EvidenceClarification{
				Status:            ev.Status,
				Questions:         ev.Questions,
				RawFeedback:       stripANSI(ev.RawFeedback),
				NormalizedIntent:  ev.Card.NormalizedIntent,
			}
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "build.json")); err == nil {
		var ev EvidenceBuild
		if json.Unmarshal(data, &ev) == nil {
			detail.Build = &ev
		}
	}

	if data, err := os.ReadFile(filepath.Join(artDir, "evaluation.json")); err == nil {
		var ev EvidenceEval
		if json.Unmarshal(data, &ev) == nil {
			ev.RawFeedback = stripANSI(ev.RawFeedback)
			detail.Evaluation = &ev
		}
	}

	provPath := filepath.Join(projectRoot, ".sdp", "dispatch-provenance-"+cardID+".json")
	if data, err := os.ReadFile(provPath); err == nil {
		var ev EvidenceProvenance
		if json.Unmarshal(data, &ev) == nil {
			detail.Provenance = &ev
		}
	}

	return detail, nil
}

func strVal(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprint(int(val))
	default:
		return fmt.Sprint(val)
	}
}

func intVal(m map[string]any, key string) int {
	v, _ := m[key]
	switch val := v.(type) {
	case float64:
		return int(val)
	default:
		return 0
	}
}

// stripANSI removes ANSI escape codes.
func stripANSI(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && !((s[j] >= 'A' && s[j] <= 'Z') || (s[j] >= 'a' && s[j] <= 'z')) {
				j++
			}
			i = j + 1
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
