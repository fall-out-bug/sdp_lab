package evaluator

import "sort"

const PeriodicComponentAuditProtocolContractVersion = "deep-thinking-component-audit-protocol/v1"

type AuditDecisionCheckpoint struct {
	ID                  string
	DecisionQuestion    string
	RequiredInputs      []string
	PassCondition       string
	EscalationOnFailure []string
}

type PeriodicComponentAuditProtocol struct {
	ContractVersion     string
	Cadence             string
	RequiredInputs      []string
	DecisionCheckpoints []AuditDecisionCheckpoint
	EscalationPath      string
}

type PeriodicAuditDecisionResult struct {
	Passed              bool
	MissingInputs       []string
	FailedCheckpointIDs []string
	EscalationTargets   []string
	EscalationRequired  bool
}

func DefaultPeriodicComponentAuditProtocol() PeriodicComponentAuditProtocol {
	plan := DefaultDeepThinkingSwarmPlan()

	inputs := []string{
		"component-boundary-map",
		"component-change-log",
		"dependency-risk-delta",
		"incident-signal-digest",
		"outcome-metric-delta",
		"persona-score-report",
	}
	sort.Strings(inputs)

	checkpoints := []AuditDecisionCheckpoint{
		{
			ID:               "entry-gate",
			DecisionQuestion: "Are cadence trigger signals complete so the audit run is allowed to start?",
			RequiredInputs: []string{
				"component-change-log",
				"dependency-risk-delta",
			},
			PassCondition:       "all-intake-triggers-pass",
			EscalationOnFailure: []string{"systems-architect"},
		},
		{
			ID:               "evidence-gate",
			DecisionQuestion: "Is evidence complete enough across reliability, security, DX, and delivery outcomes?",
			RequiredInputs: []string{
				"component-boundary-map",
				"incident-signal-digest",
				"outcome-metric-delta",
			},
			PassCondition:       "all-persona-evidence-present",
			EscalationOnFailure: []string{"sre", "security-reviewer", "dx-expert"},
		},
		{
			ID:               "decision-gate",
			DecisionQuestion: "Does the run reach recommendation consensus with explicit dissent handling?",
			RequiredInputs: []string{
				"persona-score-report",
			},
			PassCondition:       "consensus-threshold-met-or-dissent-published",
			EscalationOnFailure: []string{"product-strategist", "systems-architect"},
		},
		{
			ID:               "handoff-gate",
			DecisionQuestion: "Can accepted recommendations be converted into bounded implementation tasks?",
			RequiredInputs: []string{
				"component-boundary-map",
				"persona-score-report",
			},
			PassCondition:       "backlog-items-created-with-trace-link",
			EscalationOnFailure: []string{"product-strategist"},
		},
	}

	for i := range checkpoints {
		sort.Strings(checkpoints[i].RequiredInputs)
		sort.Strings(checkpoints[i].EscalationOnFailure)
	}
	sort.Slice(checkpoints, func(i, j int) bool { return checkpoints[i].ID < checkpoints[j].ID })

	return PeriodicComponentAuditProtocol{
		ContractVersion:     PeriodicComponentAuditProtocolContractVersion,
		Cadence:             plan.Cadence,
		RequiredInputs:      inputs,
		DecisionCheckpoints: checkpoints,
		EscalationPath:      "checkpoint-owner -> issue-owner-with-dissent-log",
	}
}

func EvaluatePeriodicAuditDecision(protocol PeriodicComponentAuditProtocol, providedInputs map[string]bool, checkpointPass map[string]bool) PeriodicAuditDecisionResult {
	missingInputs := make([]string, 0, len(protocol.RequiredInputs))
	for _, input := range protocol.RequiredInputs {
		if !providedInputs[input] {
			missingInputs = append(missingInputs, input)
		}
	}

	failedCheckpoints := make([]string, 0, len(protocol.DecisionCheckpoints))
	escalationSet := map[string]struct{}{}
	for _, checkpoint := range protocol.DecisionCheckpoints {
		if checkpointPass[checkpoint.ID] {
			continue
		}
		failedCheckpoints = append(failedCheckpoints, checkpoint.ID)
		for _, escalationTarget := range checkpoint.EscalationOnFailure {
			escalationSet[escalationTarget] = struct{}{}
		}
	}

	escalationTargets := make([]string, 0, len(escalationSet))
	for target := range escalationSet {
		escalationTargets = append(escalationTargets, target)
	}
	sort.Strings(escalationTargets)

	passed := len(missingInputs) == 0 && len(failedCheckpoints) == 0
	return PeriodicAuditDecisionResult{
		Passed:              passed,
		MissingInputs:       missingInputs,
		FailedCheckpointIDs: failedCheckpoints,
		EscalationTargets:   escalationTargets,
		EscalationRequired:  !passed,
	}
}
