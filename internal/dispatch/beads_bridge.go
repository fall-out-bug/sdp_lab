package dispatch

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"sdp_dev/internal/executil"
)

// BeadContext holds task metadata extracted from the beads issue tracker.
type BeadContext struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	DependsOn   []string `json:"depends_on"`
	BlockedBy   []string `json:"blocked_by"`
	Labels      []string `json:"labels"`
	Priority    int      `json:"priority"`
	Type        string   `json:"type"` // task, bug, feature
	Notes       string   `json:"notes"`
}

// BeadsReader extracts task context from the beads issue tracker.
type BeadsReader interface {
	ReadBead(ctx context.Context, beadID string) (*BeadContext, error)
	ReadDependencies(ctx context.Context, beadID string) ([]string, error)
}

// bdShowOutput models the JSON structure returned by `bd show <id> --json`.
// The command returns a JSON array with a single element containing the issue.
type bdShowOutput struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Notes       string   `json:"notes"`
	Status      string   `json:"status"`
	Priority    int      `json:"priority"`
	Type        string   `json:"issue_type"`
	Labels      []string `json:"labels"`
	Dependencies []struct {
		ID             string `json:"id"`
		DependencyType string `json:"dependency_type"`
	} `json:"dependencies"`
}

// ShellBeadsReader implements BeadsReader by shelling out to `bd show --json`.
type ShellBeadsReader struct {
	// ProjectRoot is the working directory for bd commands.
	ProjectRoot string
	// BdShow runs `bd show <id> --json` and returns raw output.
	// Overridable for testing. Defaults to real shell execution via executil.
	BdShow func(ctx context.Context, projectRoot, beadID string) (string, error)
}

// NewShellBeadsReader creates a ShellBeadsReader with default shell execution.
func NewShellBeadsReader(projectRoot string) *ShellBeadsReader {
	return &ShellBeadsReader{
		ProjectRoot: projectRoot,
		BdShow:      defaultBdShow,
	}
}

// defaultBdShow runs `bd show <id> --json` via executil.DefaultRunner.
func defaultBdShow(ctx context.Context, projectRoot, beadID string) (string, error) {
	out, err := executil.DefaultRunner.Output(ctx, projectRoot, "bd", "show", beadID, "--json")
	if err != nil {
		return "", fmt.Errorf("bd show %s: %w", beadID, err)
	}
	return string(out), nil
}

// ReadBead extracts BeadContext by running `bd show <id> --json` and parsing output.
// Returns an error if bd show fails. Missing fields default to zero values.
func (r *ShellBeadsReader) ReadBead(ctx context.Context, beadID string) (*BeadContext, error) {
	if r.BdShow == nil {
		return nil, fmt.Errorf("beads bridge: BdShow function not configured")
	}

	raw, err := r.BdShow(ctx, r.ProjectRoot, beadID)
	if err != nil {
		return nil, fmt.Errorf("beads bridge read %s: %w", beadID, err)
	}

	var issues []bdShowOutput
	if err := json.Unmarshal([]byte(raw), &issues); err != nil {
		return nil, fmt.Errorf("beads bridge parse %s: %w", beadID, err)
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("beads bridge: no issue returned for %s", beadID)
	}

	issue := issues[0]

	var dependsOn []string
	var blockedBy []string
	for _, dep := range issue.Dependencies {
		switch dep.DependencyType {
		case "depends_on":
			dependsOn = append(dependsOn, dep.ID)
		case "blocks":
			blockedBy = append(blockedBy, dep.ID)
		}
	}

	return &BeadContext{
		ID:          issue.ID,
		Title:       issue.Title,
		Description: issue.Description,
		Status:      issue.Status,
		DependsOn:   dependsOn,
		BlockedBy:   blockedBy,
		Labels:      issue.Labels,
		Priority:    issue.Priority,
		Type:        issue.Type,
		Notes:       issue.Notes,
	}, nil
}

// ReadDependencies returns bead IDs that the given bead depends on.
// Returns an empty (non-nil) slice if there are no dependencies.
func (r *ShellBeadsReader) ReadDependencies(ctx context.Context, beadID string) ([]string, error) {
	bead, err := r.ReadBead(ctx, beadID)
	if err != nil {
		return nil, err
	}

	deps := bead.DependsOn
	if deps == nil {
		deps = []string{}
	}
	return deps, nil
}

// parseCommaList splits a comma-separated string into trimmed, non-empty strings.
func parseCommaList(s string) []string {
	if s == "" || s == "(none)" {
		return nil
	}
	var result []string
	for _, part := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
