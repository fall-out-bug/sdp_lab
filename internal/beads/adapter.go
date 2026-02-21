// Package beads provides a typed Go API over the bd CLI for SDP integration.
package beads

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Issue represents a Beads issue as returned by bd show/ready.
type Issue struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Notes            string   `json:"notes"`
	Status           string   `json:"status"`
	Priority         int      `json:"priority"`
	IssueType        string   `json:"issue_type"`
	Labels           []string `json:"labels"`
	AcceptanceCriteria string `json:"acceptance_criteria"`
	SpecID           string   `json:"spec_id"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
	ClosedAt         string   `json:"closed_at,omitempty"`
	CloseReason      string   `json:"close_reason,omitempty"`
}

// CreateOpts holds options for creating a new issue.
type CreateOpts struct {
	Title            string
	Type             string // task, feature, epic, etc.
	Priority         int
	Description      string
	Acceptance       string
	SpecID           string
	Labels           []string
	ParentID         string
}

// Adapter wraps bd CLI with typed operations.
type Adapter struct {
	workDir string
}

// NewAdapter returns an adapter for the given working directory (repo root).
func NewAdapter(workDir string) *Adapter {
	return &Adapter{workDir: workDir}
}

func (a *Adapter) run(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	cmd.Dir = a.workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

// Ready returns ready issues matching the given labels.
func (a *Adapter) Ready(labels []string, limit int) ([]Issue, error) {
	args := []string{"ready", "--json"}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	if limit > 0 {
		args = append(args, "--limit", fmt.Sprintf("%d", limit))
	}
	out, err := a.run(args...)
	if err != nil {
		return nil, err
	}
	return parseIssueList(out)
}

// Show returns a single issue by ID.
func (a *Adapter) Show(id string) (*Issue, error) {
	out, err := a.run("show", id, "--json")
	if err != nil {
		return nil, err
	}
	list, err := parseIssueList(out)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("issue %s not found", id)
	}
	return &list[0], nil
}

// Claim sets issue status to in_progress.
func (a *Adapter) Claim(id string) error {
	_, err := a.run("update", id, "--status", "in_progress")
	return err
}

// Close closes an issue with the given reason.
func (a *Adapter) Close(id string, reason string) error {
	args := []string{"close", id}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	_, err := a.run(args...)
	return err
}

// Sync runs bd sync. If importOnly is true, only imports from JSONL (no commit/push).
func (a *Adapter) Sync(importOnly bool) error {
	args := []string{"sync"}
	if importOnly {
		args = append(args, "--import-only")
	}
	_, err := a.run(args...)
	return err
}

// DepAdd adds a dependency: blocked depends on blocker (blocker must complete before blocked).
func (a *Adapter) DepAdd(blockedID, blockerID string) error {
	_, err := a.run("dep", "add", blockedID, blockerID)
	return err
}

// Create creates a new issue and returns its ID.
func (a *Adapter) Create(opts CreateOpts) (string, error) {
	args := []string{"create", opts.Title, "-t", opts.Type, "-p", fmt.Sprintf("%d", opts.Priority), "--json"}
	if opts.Description != "" {
		args = append(args, "--description", opts.Description)
	}
	if opts.Acceptance != "" {
		args = append(args, "--acceptance", opts.Acceptance)
	}
	if opts.SpecID != "" {
		args = append(args, "--spec-id", opts.SpecID)
	}
	if len(opts.Labels) > 0 {
		args = append(args, "--labels", strings.Join(opts.Labels, ","))
	}
	if opts.ParentID != "" {
		args = append(args, "--parent", opts.ParentID)
	}
	out, err := a.run(args...)
	if err != nil {
		return "", err
	}
	var issue Issue
	if err := json.Unmarshal(out, &issue); err != nil {
		return "", fmt.Errorf("parse bd create output: %w", err)
	}
	return issue.ID, nil
}

func parseIssueList(out []byte) ([]Issue, error) {
	// bd may prefix JSON with warnings; find first [ or {
	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil, nil
	}
	start := -1
	for i, ch := range trimmed {
		if ch == '[' || ch == '{' {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no JSON array or object in bd output")
	}
	payload := []byte(trimmed[start:])
	// Try array first
	var list []Issue
	if err := json.Unmarshal(payload, &list); err == nil {
		return list, nil
	}
	// Single object
	var single Issue
	if err := json.Unmarshal(payload, &single); err != nil {
		return nil, fmt.Errorf("parse bd json: %w", err)
	}
	return []Issue{single}, nil
}
