// Package router implements L1 Project Routing for the SDP orchestration stack.
//
// The SDP architecture layers are:
//
//	L0 Intent -> L1 ROUTING -> L2 PLANNING -> L3 DISPATCH -> L4 EXECUTION -> L5 DATA
//
// L1 routes an incoming intent (from OpenClaw, Kanban, Beads, or manual input)
// to a specific project, rig, and entry phase before handing off to L2/L3.
package router

import (
	"fmt"
	"strings"
)

// Intent represents an incoming request from OpenClaw/Kanban/Beads.
type Intent struct {
	ID          string            `json:"id"`
	Source      string            `json:"source"`             // "openclaw", "kanban", "beads", "manual"
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	TaskType    string            `json:"task_type,omitempty"` // "feature", "bugfix", "hotfix", "refactor"
	Labels      []string          `json:"labels,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RoutingDecision is the output of the Project Router.
type RoutingDecision struct {
	ProjectRoot string `json:"project_root"`
	Rig         string `json:"rig"`         // "sdp-full", "sdp-lite", "manual"
	EntryPhase  string `json:"entry_phase"` // "discovery", "design", "build"
	Reason      string `json:"reason"`
}

// ProjectConfig describes a known project for routing purposes.
type ProjectConfig struct {
	Name        string   `json:"name"`
	Root        string   `json:"root"` // absolute path
	Languages   []string `json:"languages"`
	DefaultRig  string   `json:"default_rig"`
	Patterns    []string `json:"patterns"` // keywords that match to this project
}

// Router routes intents to projects.
type Router struct {
	Projects []ProjectConfig
}

// Route selects the project, rig, and entry phase for the given intent.
func (r *Router) Route(intent Intent) (*RoutingDecision, error) {
	project, err := r.matchProject(intent)
	if err != nil {
		return nil, err
	}

	rig := SelectRig(intent.TaskType, project.DefaultRig)

	// Determine available signals from metadata.
	hasRequirements := intent.Metadata["has_requirements"] == "true"
	hasDesign := intent.Metadata["has_design"] == "true"
	entryPhase := InferEntryPhase(intent.TaskType, hasRequirements, hasDesign)

	return &RoutingDecision{
		ProjectRoot: project.Root,
		Rig:         rig,
		EntryPhase:  entryPhase,
		Reason:      fmt.Sprintf("matched project %q via patterns; task_type=%s", project.Name, intent.TaskType),
	}, nil
}

// matchProject finds the best matching project for the intent by checking
// labels and title against project patterns.
func (r *Router) matchProject(intent Intent) (*ProjectConfig, error) {
	var bestMatch *ProjectConfig
	bestScore := 0

	for i := range r.Projects {
		score := r.scoreProject(&r.Projects[i], intent)
		if score > bestScore {
			bestScore = score
			bestMatch = &r.Projects[i]
		}
	}

	if bestMatch == nil {
		return nil, fmt.Errorf("no project matched intent %q (labels=%v)", intent.Title, intent.Labels)
	}
	return bestMatch, nil
}

// scoreProject returns a match score for a project against an intent.
// Higher is better; 0 means no match.
func (r *Router) scoreProject(project *ProjectConfig, intent Intent) int {
	score := 0
	titleLower := strings.ToLower(intent.Title)

	for _, pattern := range project.Patterns {
		p := strings.ToLower(pattern)

		// Check labels
		for _, label := range intent.Labels {
			if strings.ToLower(label) == p {
				score += 2
			}
		}

		// Check title
		if strings.Contains(titleLower, p) {
			score++
		}
	}

	return score
}
