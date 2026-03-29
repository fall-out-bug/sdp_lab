package dispatch

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

// GastownInfra implements Infrastructure by delegating to the "gt" CLI.
type GastownInfra struct{}

// Name returns "gastown".
func (g *GastownInfra) Name() string { return "gastown" }

// CreateConvoy runs: gt convoy create {name} {issues...} --json
// It expects the JSON response to contain a top-level "id" string field.
func (g *GastownInfra) CreateConvoy(name string, issues []string) (string, error) {
	args := append([]string{"convoy", "create", name}, issues...)
	args = append(args, "--json")
	out, err := exec.Command("gt", args...).Output()
	if err != nil {
		return "", fmt.Errorf("gt convoy create: %w", err)
	}
	var payload struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("gt convoy create: parse response: %w", err)
	}
	return payload.ID, nil
}

// ConvoyStatus runs: gt convoy show {id} --json
// It expects the JSON response to contain a top-level "status" string field.
func (g *GastownInfra) ConvoyStatus(id string) (string, error) {
	out, err := exec.Command("gt", "convoy", "show", id, "--json").Output()
	if err != nil {
		return "", fmt.Errorf("gt convoy show: %w", err)
	}
	var payload struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		return "", fmt.Errorf("gt convoy show: parse response: %w", err)
	}
	return payload.Status, nil
}

// Sling runs: gt sling {issue} {rig} --agent {agent}
func (g *GastownInfra) Sling(issue, rig, agent string) error {
	cmd := exec.Command("gt", "sling", issue, rig, "--agent", agent)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("gt sling: %w: %s", err, out)
	}
	return nil
}
