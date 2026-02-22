package agent

import (
	"sdp_dev/internal/observability"
	"sdp_dev/internal/policy"
)

// RoleDefaultModels is deprecated. Use policy.RoleDefaultModel(role) for config-driven lookup.
// Kept for backward compatibility when policy config not loaded.
var RoleDefaultModels = map[string]string{
	"analyst": "glm-5", "coder": "glm-4.7", "reviewer": "glm-5", "retro": "glm-5",
	"orchestrator": "glm-5",
}

// PolicyContext wraps policy.Decide with role-based defaults.
type PolicyContext struct {
	Role         string
	DefaultModel string
}

// NewPolicyContext creates a PolicyContext for the given role.
// DefaultModel comes from policy config (roles.primary) when loaded, else RoleDefaultModels.
func NewPolicyContext(role string) *PolicyContext {
	model := policy.RoleDefaultModel(role)
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
		Role:           p.Role,
		PreferredModel: preferredModel,
		ChangedPaths:   changedPaths,
	}
	if preferredModel == "" {
		req.PreferredModel = p.DefaultModel
	}
	decision = policy.Decide(req)
	tier := policy.TierForModel(decision.FallbackChain, decision.SelectedModel)
	reason := "preferred"
	if len(decision.Reasons) > 0 {
		reason = decision.Reasons[0]
		if len(reason) > 32 {
			reason = reason[:32]
		}
	}
	observability.IncModelSelection(p.Role, tier, decision.SelectedModel, reason)
	return decision.SelectedModel, decision
}

// SelectModelSimple returns model for role when no paths or preference.
func (p *PolicyContext) SelectModelSimple() string {
	_, dec := p.SelectModel(p.DefaultModel, nil)
	return dec.SelectedModel
}
