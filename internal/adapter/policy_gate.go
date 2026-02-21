package adapter

import (
	"fmt"
	"strings"

	"sdp_dev/internal/policy"
)

// GateResult holds the result of a policy gate check.
type GateResult struct {
	GateID   string
	Passed   bool
	Reason   string
	DenyCode string
}

// PolicyGate enforces model/risk/publish controls.
type PolicyGate struct{}

// NewPolicyGate returns a new policy gate.
func NewPolicyGate() *PolicyGate {
	return &PolicyGate{}
}

// PreDispatchModelAllowlist checks that the model is in the allowlist.
func (g *PolicyGate) PreDispatchModelAllowlist(model string) GateResult {
	modelID := policy.NormalizeModel(model)
	if !policy.AllowedModel(modelID) {
		return GateResult{
			GateID:   "pre_dispatch_model_allowlist",
			Passed:   false,
			Reason:   fmt.Sprintf("model %q not in allowlist", model),
			DenyCode: "policy_denied",
		}
	}
	return GateResult{GateID: "pre_dispatch_model_allowlist", Passed: true}
}

// PreCloseRiskThreshold checks risk before closing. For now always passes.
func (g *PolicyGate) PreCloseRiskThreshold(riskClass string) GateResult {
	risk := strings.ToLower(riskClass)
	if risk == "critical" {
		return GateResult{
			GateID:   "pre_close_risk_threshold",
			Passed:   false,
			Reason:   "critical risk requires manual security signoff",
			DenyCode: "policy_denied",
		}
	}
	return GateResult{GateID: "pre_close_risk_threshold", Passed: true}
}

// PrePublishGoNoGo checks before PR publication. For now always passes.
func (g *PolicyGate) PrePublishGoNoGo(evidenceComplete bool) GateResult {
	if !evidenceComplete {
		return GateResult{
			GateID:   "pre_publish_go_no_go",
			Passed:   false,
			Reason:   "evidence incomplete",
			DenyCode: "verification_failed",
		}
	}
	return GateResult{GateID: "pre_publish_go_no_go", Passed: true}
}
