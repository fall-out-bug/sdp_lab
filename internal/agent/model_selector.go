package agent

import (
	"sdp_dev/internal/policy"
)

// RoleDefaultModels maps role to default model when no override is set.
var RoleDefaultModels = map[string]string{
	"analyst":      "glm-5",
	"coder":        "glm-4.7",
	"reviewer":     "glm-5",
	"retro":        "glm-5",
	"orchestrator": "glm-5",
}

// PolicyContext wraps policy.Decide with role-based defaults.
type PolicyContext struct {
	Role         string
	DefaultModel string
}

// NewPolicyContext creates a PolicyContext for the given role.
func NewPolicyContext(role string) *PolicyContext {
	model := RoleDefaultModels[role]
	if model == "" {
		model = policy.DefaultModel()
	}
	return &PolicyContext{
		Role:         role,
		DefaultModel: model,
	}
}

// SelectModel returns the model to use. Prefers preferredModel if allowed, else role default.
func (p *PolicyContext) SelectModel(preferredModel string, changedPaths []string) (model string, decision policy.DecisionResponse) {
	req := policy.DecisionRequest{
		IssueID:        "",
		Title:          "",
		Lane:           "commit",
		PreferredModel: preferredModel,
		ChangedPaths:   changedPaths,
	}
	if preferredModel == "" {
		req.PreferredModel = p.DefaultModel
	}
	decision = policy.Decide(req)
	return decision.SelectedModel, decision
}

// SelectModelSimple returns model for role when no paths or preference.
func (p *PolicyContext) SelectModelSimple() string {
	_, dec := p.SelectModel(p.DefaultModel, nil)
	return dec.SelectedModel
}
