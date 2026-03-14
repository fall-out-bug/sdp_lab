package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const statusSpecVersion = "v1.0"

type BeadsItem struct {
	ID        string
	Title     string
	Status    string
	Priority  int
	BlockedBy []string
	Labels    []string
}

type StatusView struct {
	SpecVersion     string         `json:"spec_version"`
	Timestamp       string         `json:"timestamp"`
	ReadyCount      int            `json:"ready_count"`
	BlockedCount    int            `json:"blocked_count"`
	InProgressCount int            `json:"in_progress_count,omitempty"`
	Items           []StatusItem   `json:"items"`
	NextAction      StatusNextStep `json:"next_action"`
}

type StatusItem struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Status    string   `json:"status"`
	Priority  int      `json:"priority"`
	BlockedBy []string `json:"blocked_by,omitempty"`
	Labels    []string `json:"labels,omitempty"`
}

type StatusNextStep struct {
	Recommended    string   `json:"recommended"`
	Reason         string   `json:"reason"`
	Command        string   `json:"command,omitempty"`
	BlockingIssues []string `json:"blocking_issues,omitempty"`
}

func NewStatusViewFromBeads(beadsItems []BeadsItem) *StatusView {
	items := make([]StatusItem, 0, len(beadsItems))
	for _, src := range beadsItems {
		status := normalizeStatus(src.Status, len(src.BlockedBy) > 0)
		items = append(items, StatusItem{
			ID:        src.ID,
			Title:     src.Title,
			Status:    status,
			Priority:  src.Priority,
			BlockedBy: src.BlockedBy,
			Labels:    src.Labels,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Status != items[j].Status {
			return statusOrder(items[i].Status) < statusOrder(items[j].Status)
		}
		if items[i].Priority != items[j].Priority {
			return items[i].Priority < items[j].Priority
		}
		return items[i].ID < items[j].ID
	})

	view := &StatusView{
		SpecVersion: statusSpecVersion,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Items:       items,
	}

	for _, item := range items {
		switch item.Status {
		case "ready":
			view.ReadyCount++
		case "blocked":
			view.BlockedCount++
		case "in_progress":
			view.InProgressCount++
		}
	}

	view.NextAction = chooseNextAction(items)
	return view
}

func (s *StatusView) RenderText() string {
	var b strings.Builder
	b.WriteString("SDP Ready Status\n")
	b.WriteString(fmt.Sprintf("Ready: %d | Blocked: %d | In Progress: %d\n\n", s.ReadyCount, s.BlockedCount, s.InProgressCount))

	for _, item := range s.Items {
		line := fmt.Sprintf("- [%s] P%d %s: %s", item.Status, item.Priority, item.ID, item.Title)
		if len(item.BlockedBy) > 0 {
			line = line + " (blocked_by: " + strings.Join(item.BlockedBy, ", ") + ")"
		}
		b.WriteString(line + "\n")
	}

	b.WriteString("\nNext action:\n")
	b.WriteString("- Recommendation: " + s.NextAction.Recommended + "\n")
	b.WriteString("- Reason: " + s.NextAction.Reason + "\n")
	if s.NextAction.Command != "" {
		b.WriteString("- Command: " + s.NextAction.Command + "\n")
	}
	if len(s.NextAction.BlockingIssues) > 0 {
		b.WriteString("- Blocking issues: " + strings.Join(s.NextAction.BlockingIssues, ", ") + "\n")
	}

	return strings.TrimSpace(b.String())
}

func (s *StatusView) RenderJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func chooseNextAction(items []StatusItem) StatusNextStep {
	inProgress := make([]StatusItem, 0)
	ready := make([]StatusItem, 0)
	blocked := make([]StatusItem, 0)

	for _, item := range items {
		switch item.Status {
		case "in_progress":
			inProgress = append(inProgress, item)
		case "ready":
			ready = append(ready, item)
		case "blocked":
			blocked = append(blocked, item)
		}
	}

	if len(inProgress) > 0 {
		current := inProgress[0]
		return StatusNextStep{
			Recommended: fmt.Sprintf("Continue %s", current.ID),
			Reason:      fmt.Sprintf("%s is already in progress and should be completed before starting new work", current.ID),
			Command:     "bd show " + current.ID,
		}
	}

	if len(ready) > 0 {
		next := ready[0]
		return StatusNextStep{
			Recommended: fmt.Sprintf("Start %s", next.ID),
			Reason:      fmt.Sprintf("%s has the highest priority among ready issues", next.ID),
			Command:     "bd update " + next.ID + " --status in_progress",
		}
	}

	if len(blocked) > 0 {
		issueIDs := make([]string, 0, len(blocked))
		blockers := make([]string, 0)
		seen := make(map[string]struct{})
		for _, item := range blocked {
			issueIDs = append(issueIDs, item.ID)
			for _, blocker := range item.BlockedBy {
				if _, ok := seen[blocker]; ok {
					continue
				}
				seen[blocker] = struct{}{}
				blockers = append(blockers, blocker)
			}
		}
		sort.Strings(blockers)
		return StatusNextStep{
			Recommended:    "Resolve blockers",
			Reason:         fmt.Sprintf("No ready work: %d issue(s) are blocked (%s)", len(blocked), strings.Join(issueIDs, ", ")),
			Command:        "bd blocked",
			BlockingIssues: blockers,
		}
	}

	return StatusNextStep{
		Recommended: "No action required",
		Reason:      "No ready, blocked, or in-progress issues were found",
		Command:     "bd ready",
	}
}

func statusOrder(status string) int {
	switch status {
	case "in_progress":
		return 0
	case "ready":
		return 1
	case "blocked":
		return 2
	default:
		return 3
	}
}

func normalizeStatus(status string, hasBlockers bool) string {
	switch status {
	case "in_progress":
		return "in_progress"
	case "blocked":
		return "blocked"
	case "ready":
		return "ready"
	default:
		if hasBlockers {
			return "blocked"
		}
		return "ready"
	}
}
