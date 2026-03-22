package control

import (
	"fmt"
	"slices"
	"strings"
)

// ExecutionPacket represents a task packet dispatched by the orchestrator to an executor.
// This is the contract for orchestrator -> executor communication.
//
// Based on ORCHESTRATOR_BEADS_OPERATING_MODEL.md section 7
type ExecutionPacket struct {
	// BeadsTaskID is the ID of the Beads issue being executed
	BeadsTaskID string `json:"beads_task_id"`

	// ParentFeatureID is the FeatureCard ID if this task is part of a feature
	ParentFeatureID string `json:"parent_feature_id,omitempty"`

	// ProjectID identifies the project this task belongs to
	ProjectID string `json:"project_id"`

	// TargetRepo identifies the target repository for this task
	TargetRepo string `json:"target_repo"`

	// ExecutorRole is the role/agent type that should execute this task
	ExecutorRole string `json:"executor_role"`

	// Objective describes the goal of this task
	Objective string `json:"objective"`

	// ScopeIn lists what's included in the task scope
	ScopeIn []string `json:"scope_in,omitempty"`

	// ScopeOut lists expected outputs from this task
	ScopeOut []string `json:"scope_out,omitempty"`

	// Constraints lists constraints the executor must respect
	Constraints []string `json:"constraints,omitempty"`

	// RequiredArtifacts lists artifacts that must be produced
	RequiredArtifacts []string `json:"required_artifacts,omitempty"`

	// RequiredChecks lists verifications that must pass
	RequiredChecks []string `json:"required_checks,omitempty"`

	// NextHandoffTarget is the role or stage to hand off to after completion
	NextHandoffTarget string `json:"next_handoff_target,omitempty"`
}

// ExecutorRole represents the type of executor that should handle a task.
// These are canonical executor roles based on ORCHESTRATOR_BEADS_OPERATING_MODEL.md section 6.
type ExecutorRole string

const (
	// ExecutorRoleOmOImplementation routes to OmO implementation agents
	ExecutorRoleOmOImplementation ExecutorRole = "omo-implementation"

	// ExecutorRoleClarification routes to orchestrator clarification or planner/analyst
	ExecutorRoleClarification ExecutorRole = "clarification"

	// ExecutorRoleReview routes to reviewer role
	ExecutorRoleReview ExecutorRole = "review"

	// ExecutorRoleReleaseCheck routes to release-check role
	ExecutorRoleReleaseCheck ExecutorRole = "release-check"

	// ExecutorRoleDocsTranslation routes to repo-local docs/translator/triage
	ExecutorRoleDocsTranslation ExecutorRole = "docs-translation"

	// ExecutorRoleHumanAdmin routes to human/admin feedback loop
	ExecutorRoleHumanAdmin ExecutorRole = "human-admin"

	// ExecutorRoleRepoLocalArchitecture routes to repo-local architecture agent
	ExecutorRoleRepoLocalArchitecture ExecutorRole = "repo-local-architecture"
)

// RouteToExecutor determines the appropriate executor role for a FeatureCard.
func (s *Store) RouteToExecutor(card *FeatureCard) (ExecutorRole, error) {
	if card == nil {
		return "", fmt.Errorf("nil card")
	}

	// Check for escalation/uncertainty/risk threshold
	if card.RiskLevel == "high" && (len(card.DecisionRequired) > 0 || len(card.AdminActionRequired) > 0) {
		return ExecutorRoleHumanAdmin, nil
	}

	// Check for review stage
	if card.Status == "reviewing" {
		return ExecutorRoleReview, nil
	}

	// Check for clarification/ambiguity needs
	if card.Status == "clarifying" || card.Status == "needs_input" {
		return ExecutorRoleClarification, nil
	}

	// Check for release/migration sensitivity
	if card.RiskLevel == "high" && containsAny(card.ScopeIn, "release", "migration", "deployment") {
		return ExecutorRoleReleaseCheck, nil
	}

	// Check for docs/translation needs
	if containsAny(card.ScopeIn, "docs", "translation", "triage") {
		return ExecutorRoleDocsTranslation, nil
	}

	// Check for repo architecture questions
	if card.TargetArea == "architecture" || containsAny(card.ScopeIn, "architecture", "design", "structure") {
		return ExecutorRoleRepoLocalArchitecture, nil
	}

	// Default to OmO implementation for generic implementation work
	return ExecutorRoleOmOImplementation, nil
}

// BuildExecutionPacket constructs an ExecutionPacket for a FeatureCard.
// This is called when the orchestrator wants to dispatch a card to an executor.
func (s *Store) BuildExecutionPacket(projectID, cardID string) (*ExecutionPacket, error) {
	card, err := s.LoadCard(projectID, cardID)
	if err != nil {
		return nil, err
	}

	executorRole, err := s.RouteToExecutor(card)
	if err != nil {
		return nil, fmt.Errorf("route executor: %w", err)
	}

	packet := &ExecutionPacket{
		BeadsTaskID:       deriveBeadsTaskID(card),
		ParentFeatureID:   card.ID,
		ProjectID:         card.ProjectID,
		TargetRepo:        card.TargetRepo,
		ExecutorRole:      string(executorRole),
		Objective:         deriveObjective(card),
		ScopeIn:           card.ScopeIn,
		ScopeOut:          card.ScopeOut,
		Constraints:       deriveConstraints(card),
		RequiredArtifacts: card.RequiredArtifacts,
		RequiredChecks:    card.RequiredChecks,
		NextHandoffTarget: deriveNextHandoff(card),
	}

	return packet, nil
}

func deriveBeadsTaskID(card *FeatureCard) string {
	if len(card.LinkedBeadsIDs) > 0 {
		return card.LinkedBeadsIDs[0]
	}
	return ""
}

func deriveObjective(card *FeatureCard) string {
	if card.NormalizedIntent != "" {
		return card.NormalizedIntent
	}
	return card.RawRequest
}

func deriveConstraints(card *FeatureCard) []string {
	constraints := []string{}

	if card.RiskLevel != "" {
		constraints = append(constraints, "risk_level: "+card.RiskLevel)
	}

	if card.ExecutionMode != "" {
		constraints = append(constraints, "execution_mode: "+card.ExecutionMode)
	}

	if len(card.NonGoals) > 0 {
		constraints = append(constraints, "non_goals: exclude from implementation")
	}

	return constraints
}

func deriveNextHandoff(card *FeatureCard) string {
	if card.RecommendedNext != "" {
		return card.RecommendedNext
	}

	if card.Status == "ready" {
		return "executor"
	}

	return "orchestrator"
}

func containsAny(items []string, targets ...string) bool {
	lowerItems := make([]string, len(items))
	for i, item := range items {
		lowerItems[i] = strings.ToLower(item)
	}

	for _, target := range targets {
		if slices.Contains(lowerItems, strings.ToLower(target)) {
			return true
		}
	}
	return false
}
