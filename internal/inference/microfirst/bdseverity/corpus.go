package bdseverity

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// BdIssue is a minimal parsed record from .beads/issues.jsonl.
type BdIssue struct {
	ID          string
	Title       string
	Description string
	Status      string // "open" | "closed"
	Priority    string // "P0" | "P1" | "P2" | "P3" | "" (unset)
	IssueType   string // "bug" | "task" | "feature" | "epic"
	CreatedAt   string
}

// rawIssue mirrors the JSON structure of a .beads/issues.jsonl record.
type rawIssue struct {
	Type        string `json:"_type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	IssueType   string `json:"issue_type"`
	CreatedAt   string `json:"created_at"`
}

// LoadCorpus reads a .beads/issues.jsonl-formatted file and returns:
//   - train: all qualifying issues except the last 30 (sorted by created_at ascending)
//   - eval:  the last 30 qualifying issues (sorted by created_at ascending)
//
// Qualifying issues are: status=="closed", priority!="", issue_type!="epic".
func LoadCorpus(path string) (train, eval []BdIssue, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("bdseverity: open corpus: %w", err)
	}
	defer f.Close()

	var issues []BdIssue
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		var raw rawIssue
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			// Skip malformed lines.
			continue
		}
		if raw.Status != "closed" {
			continue
		}
		if raw.Priority == "" {
			continue
		}
		if raw.IssueType == "epic" {
			continue
		}
		issues = append(issues, BdIssue{
			ID:          raw.ID,
			Title:       raw.Title,
			Description: raw.Description,
			Status:      raw.Status,
			Priority:    raw.Priority,
			IssueType:   raw.IssueType,
			CreatedAt:   raw.CreatedAt,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("bdseverity: scan corpus: %w", err)
	}

	// Sort by created_at ascending (lexicographic — ISO-8601 sorts correctly).
	sort.Slice(issues, func(i, j int) bool {
		return issues[i].CreatedAt < issues[j].CreatedAt
	})

	const evalSize = 30
	if len(issues) <= evalSize {
		return nil, issues, nil
	}

	split := len(issues) - evalSize
	return issues[:split], issues[split:], nil
}
