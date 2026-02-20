package artifact

import (
	"encoding/json"
	"sort"
	"strings"
)

type TransitionRequest struct {
	IssueID     string
	FromPhase   string
	ToPhase     string
	GateSignals map[string]GateSignalStatus
}

type GateSignalDecision struct {
	SignalID string
	Status   GateSignalStatus
	Passed   bool
}

type TransitionDecision struct {
	IssueID       string
	FromPhase     string
	ToPhase       string
	Allowed       bool
	ReasonCodes   []string
	GateDecisions []GateSignalDecision
}

type TransitionController struct {
	bus          *BusService
	policy       TransitionPolicyContract
	rulesByEdge  map[string]TransitionPolicyRule
	passStatuses map[GateSignalStatus]struct{}
}

func NewDefaultTransitionController(bus *BusService) *TransitionController {
	return NewTransitionController(bus, BuildTransitionPolicyContract())
}

func NewTransitionController(bus *BusService, policy TransitionPolicyContract) *TransitionController {
	controller := &TransitionController{
		bus:          bus,
		policy:       policy,
		rulesByEdge:  map[string]TransitionPolicyRule{},
		passStatuses: map[GateSignalStatus]struct{}{},
	}
	for _, rule := range policy.Rules {
		controller.rulesByEdge[rule.FromPhase+"->"+rule.ToPhase] = rule
	}
	for _, status := range policy.Aggregation.PassStatuses {
		controller.passStatuses[status] = struct{}{}
	}
	return controller
}

func (c *TransitionController) EvaluateTransition(req TransitionRequest) TransitionDecision {
	decision := TransitionDecision{
		IssueID:   req.IssueID,
		FromPhase: req.FromPhase,
		ToPhase:   req.ToPhase,
	}

	rule, ok := c.rulesByEdge[req.FromPhase+"->"+req.ToPhase]
	if !ok {
		decision.ReasonCodes = []string{transitionReasonCode(req.FromPhase, req.ToPhase, "policy-rule-missing")}
		return decision
	}
	if c.bus == nil {
		decision.ReasonCodes = []string{transitionReasonCode(req.FromPhase, req.ToPhase, "bus-unavailable")}
		return decision
	}

	gateMissing := false
	gateNotPassed := false
	for _, signalID := range rule.RequiredGateSignals {
		status, exists := req.GateSignals[signalID]
		if !exists || strings.TrimSpace(string(status)) == "" {
			status = GateSignalStatusMissing
		}
		_, passed := c.passStatuses[status]
		decision.GateDecisions = append(decision.GateDecisions, GateSignalDecision{
			SignalID: signalID,
			Status:   status,
			Passed:   passed,
		})
		if status == GateSignalStatusMissing {
			gateMissing = true
			continue
		}
		if !passed {
			gateNotPassed = true
		}
	}

	artifacts := c.bus.ListByIssue(req.IssueID)
	presentClasses := map[string]struct{}{}
	for _, envelope := range artifacts {
		presentClasses[envelope.ArtifactClass] = struct{}{}
	}
	missingArtifact := false
	for _, classID := range rule.RequiredArtifactClassIDs {
		if _, ok := presentClasses[classID]; !ok {
			missingArtifact = true
			break
		}
	}

	requiredClasses := map[string]struct{}{}
	for _, classID := range rule.RequiredArtifactClassIDs {
		requiredClasses[classID] = struct{}{}
	}
	payloadKeys := collectPayloadKeysByClass(artifacts, requiredClasses)
	missingProvenanceKey := false
	for _, key := range rule.RequiredProvenanceKeys {
		if _, ok := payloadKeys[key]; !ok {
			missingProvenanceKey = true
			break
		}
	}

	if gateMissing {
		decision.ReasonCodes = append(decision.ReasonCodes, denialReasonCode(rule, "missing-gate-signal"))
	}
	if gateNotPassed {
		decision.ReasonCodes = append(decision.ReasonCodes, denialReasonCode(rule, "gate-not-passed"))
	}
	if missingArtifact {
		decision.ReasonCodes = append(decision.ReasonCodes, denialReasonCode(rule, "missing-artifact"))
	}
	if missingProvenanceKey {
		decision.ReasonCodes = append(decision.ReasonCodes, denialReasonCode(rule, "missing-provenance-key"))
	}

	decision.Allowed = len(decision.ReasonCodes) == 0
	return decision
}

func denialReasonCode(rule TransitionPolicyRule, suffix string) string {
	wantSuffix := "-" + suffix
	for _, code := range rule.DenialReasonCodes {
		if strings.HasSuffix(code, wantSuffix) {
			return code
		}
	}
	return transitionReasonCode(rule.FromPhase, rule.ToPhase, suffix)
}

func collectPayloadKeysByClass(artifacts []ArtifactEnvelope, allowedClasses map[string]struct{}) map[string]struct{} {
	out := map[string]struct{}{}
	for _, envelope := range artifacts {
		if len(allowedClasses) > 0 {
			if _, ok := allowedClasses[envelope.ArtifactClass]; !ok {
				continue
			}
		}
		var payload map[string]any
		if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
			continue
		}
		for key, value := range payload {
			if !presentPayloadValue(value) {
				continue
			}
			out[key] = struct{}{}
		}
	}

	return out
}

func presentPayloadValue(v any) bool {
	switch typed := v.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case []string:
		return len(typed) > 0
	default:
		return true
	}
}

func (d TransitionDecision) SortedReasonCodes() []string {
	out := append([]string(nil), d.ReasonCodes...)
	sort.Strings(out)
	return out
}
