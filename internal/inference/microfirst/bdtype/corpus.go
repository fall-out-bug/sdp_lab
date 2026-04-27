// Package bdtype provides an embedding-based micro-classifier for issue type
// (bug / task / feature). It mirrors the bdseverity package structure.
package bdtype

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// rawIssue is the JSON shape for .beads/issues.jsonl entries.
type rawIssue struct {
	Type        string `json:"_type"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IssueType   string `json:"issue_type"`
}

// CorpusEntry is a single training/eval record.
type CorpusEntry struct {
	ID    string
	Text  string // Title + " " + Description
	Label string // "bug", "task", or "feature"
}

// Corpus holds train and eval splits.
type Corpus struct {
	Train []CorpusEntry
	Eval  []CorpusEntry
}

// normalizeType maps chore→task and returns ("", false) for epic (exclude).
func normalizeType(t string) (string, bool) {
	switch strings.ToLower(t) {
	case "bug":
		return "bug", true
	case "task":
		return "task", true
	case "feature":
		return "feature", true
	case "chore":
		return "task", true // normalise chore → task
	case "epic":
		return "", false // exclude from corpus
	default:
		return "", false
	}
}

// LoadCorpus reads issues from path, filters by status==closed and allowed types,
// normalises chore→task, excludes epic, and splits: last 30 → eval, rest → train.
func LoadCorpus(path string) (*Corpus, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("bdtype: open corpus: %w", err)
	}
	defer f.Close()
	return parseCorpus(f)
}

func parseCorpus(r io.Reader) (*Corpus, error) {
	var entries []CorpusEntry

	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var issue rawIssue
		if err := json.Unmarshal([]byte(line), &issue); err != nil {
			continue // skip malformed lines
		}
		if issue.Status != "closed" {
			continue
		}
		label, ok := normalizeType(issue.IssueType)
		if !ok {
			continue
		}
		text := strings.TrimSpace(issue.Title + " " + issue.Description)
		entries = append(entries, CorpusEntry{
			ID:    issue.ID,
			Text:  text,
			Label: label,
		})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("bdtype: scan corpus: %w", err)
	}

	// Split: last 30 → eval, rest → train.
	evalStart := len(entries) - 30
	if evalStart < 0 {
		evalStart = 0
	}
	return &Corpus{
		Train: entries[:evalStart],
		Eval:  entries[evalStart:],
	}, nil
}
