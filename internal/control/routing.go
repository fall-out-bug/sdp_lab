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

// ExecutorResultPacket represents a result packet returned by an executor to the orchestrator.
// This is contract for executor -> orchestrator communication.
//
// Based on ORCHESTRATOR_BEADS_OPERATING_MODEL.md section 8
type ExecutorResultPacket struct {
	// BeadsTaskID is the ID of the Beads task this result corresponds to
	BeadsTaskID string `json:"beads_task_id"`

	// ParentFeatureID is the FeatureCard ID this result corresponds to
	ParentFeatureID string `json:"parent_feature_id"`

	// ExecutorRole is the role that produced this result
	ExecutorRole string `json:"executor_role"`

	// Status indicates the outcome of the execution
	Status ExecutorResultStatus `json:"status"`

	// Summary provides a short human-readable summary of the result
	Summary string `json:"summary"`

	// Artifacts lists artifact references produced by the executor
	Artifacts []ExecutorArtifact `json:"artifacts,omitempty"`

	// Findings lists follow-up findings or issues discovered during execution
	Findings []string `json:"findings,omitempty"`

	// OpenRisks lists any risks that remain after execution
	OpenRisks []string `json:"open_risks,omitempty"`

	// RecommendedNextStep suggests the next action for the orchestrator
	RecommendedNextStep string `json:"recommended_next_step,omitempty"`
}

// ExecutorResultStatus represents the status of an executor result
type ExecutorResultStatus string

const (
	// ResultStatusSuccess indicates the executor completed successfully
	ResultStatusSuccess ExecutorResultStatus = "success"

	// ResultStatusBlocked indicates the executor was blocked
	ResultStatusBlocked ExecutorResultStatus = "blocked"

	// ResultStatusNeedsReview indicates the executor requires human/admin review
	ResultStatusNeedsReview ExecutorResultStatus = "needs_review"

	// ResultStatusNeedsInput indicates the executor needs additional input
	ResultStatusNeedsInput ExecutorResultStatus = "needs_input"

	// ResultStatusFailed indicates the executor failed
	ResultStatusFailed ExecutorResultStatus = "failed"
)

// ExecutorArtifact represents an artifact produced by an executor
type ExecutorArtifact struct {
	// Type is the type of artifact (e.g., "code", "test", "doc", "review")
	Type string `json:"type"`

	// Reference is a reference to the artifact (path, URL, etc.)
	Reference string `json:"reference"`

	// Description provides additional context about the artifact
	Description string `json:"description,omitempty"`
}
