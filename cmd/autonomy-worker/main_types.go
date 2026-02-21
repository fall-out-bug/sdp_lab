package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// workstreamConfig is the schema for specs/workstream-config.yaml
type workstreamConfig struct {
	Workstreams []struct {
		Label        string   `yaml:"label"`
		PathPrefixes []string `yaml:"path_prefixes"`
	} `yaml:"workstreams"`
}

// supportedWorkstreams lists workstream labels that autonomy-worker can claim.
// Loaded from specs/workstream-config.yaml when present; otherwise fallback.
var supportedWorkstreams = []string{
	"workstream:policy-slugify-trim",
	"workstream:model-chain-default-fallback",
	"workstream:policy-k8s-risk-high",
	"workstream:handoff-validation",
	"workstream:generic",
	"workstream:self-improvement",
	"workstream:evaluator-recommendation",
}

func loadWorkstreamConfig(root string) {
	path := filepath.Join(root, "specs", "workstream-config.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg workstreamConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return
	}
	if len(cfg.Workstreams) > 0 {
		labels := make([]string, 0, len(cfg.Workstreams))
		for _, w := range cfg.Workstreams {
			if w.Label != "" {
				labels = append(labels, w.Label)
			}
		}
		if len(labels) > 0 {
			supportedWorkstreams = labels
		}
	}
}

type dep struct {
	IssueID        string `json:"issue_id"`
	DependsOnID    string `json:"depends_on_id"`
	ID             string `json:"id"`
	Type           string `json:"type"`
	DependencyType string `json:"dependency_type"`
	IssueType      string `json:"issue_type"`
	Status         string `json:"status"`
}

func (d dep) refID() string {
	if d.DependsOnID != "" {
		return d.DependsOnID
	}
	return d.ID
}

func (d dep) kind() string {
	if d.Type != "" {
		return d.Type
	}
	if d.DependencyType != "" {
		return d.DependencyType
	}
	if d.IssueType == "epic" || d.IssueType == "feature" {
		return "parent-child"
	}
	return d.DependencyType
}

type issue struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	SpecID       string   `json:"spec_id"`
	IssueType    string   `json:"issue_type"`
	Status       string   `json:"status"`
	Priority     int      `json:"priority"`
	Labels       []string `json:"labels"`
	Dependencies []dep    `json:"dependencies"`
	CreatedAt    string   `json:"created_at"`
}

func runBD(args ...string) ([]byte, error) {
	cmd := exec.Command("bd", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("bd %s: %w: %s", strings.Join(args, " "), err, string(out))
	}
	return out, nil
}

func extractJSON(out []byte) []byte {
	for i, b := range out {
		if b == '[' || b == '{' {
			return out[i:]
		}
	}
	return out
}
