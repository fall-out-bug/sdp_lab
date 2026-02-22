package evaluator

import (
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

const DeepThinkingSwarmPlanContractVersion = "deep-thinking-evaluator-swarm/v1"

type PersonaRole struct {
	ID               string
	DecisionLens     string
	PrimaryQuestion  string
	RequiredEvidence []string
	EscalationTarget string
}

type EvaluationPhase struct {
	ID              string
	Objective       string
	RequiredSignals []string
	OutputArtifacts []string
}

type CollaborationProtocol struct {
	ConflictRule       string
	ConsensusThreshold string
	ResolutionFallback string
}

type DeepThinkingSwarmPlan struct {
	ContractVersion string
	Cadence         string
	TriggerSignals  []string
	Phases          []EvaluationPhase
	Roles           []PersonaRole
	Collaboration   CollaborationProtocol
}

type SwarmPlanReadinessResult struct {
	Passed                bool
	MissingTriggerSignals []string
	MissingRoleIDs        []string
	MissingPhaseIDs       []string
}

var deepThinkingTriggerSignals = []string{
	"issue-selected",
	"dependencies-clear",
	"scope-baseline-defined",
	"gate-command-declared",
	"callback-contract-available",
}

var deepThinkingPhases = []EvaluationPhase{
	{
		ID:              "framing",
		Objective:       "Restate the target issue, constraints, and success criteria before role analysis starts.",
		RequiredSignals: []string{"intent-brief", "dependency-map"},
		OutputArtifacts: []string{"evaluation-frame", "risk-register"},
	},
	{
		ID:              "persona-analysis",
		Objective:       "Run role-specific critique against architecture, operations, security, developer experience, and product outcomes.",
		RequiredSignals: []string{"evaluation-frame", "artifact-evidence-bundle"},
		OutputArtifacts: []string{"persona-findings"},
	},
	{
		ID:              "adversarial-review",
		Objective:       "Force cross-persona challenge rounds to surface contradictions and hidden risk.",
		RequiredSignals: []string{"persona-findings"},
		OutputArtifacts: []string{"conflict-matrix", "challenge-transcript"},
	},
	{
		ID:              "consensus-synthesis",
		Objective:       "Converge on ranked recommendations with explicit tradeoffs and dissent markers.",
		RequiredSignals: []string{"conflict-matrix", "challenge-transcript"},
		OutputArtifacts: []string{"prioritized-recommendations", "dissent-log"},
	},
	{
		ID:              "publish-handoff",
		Objective:       "Attach accepted recommendations to implementation backlog and callback evidence.",
		RequiredSignals: []string{"prioritized-recommendations"},
		OutputArtifacts: []string{"implementation-brief", "trace-link"},
	},
}

var deepThinkingRoles = []PersonaRole{
	{
		ID:               "systems-architect",
		DecisionLens:     "System cohesion, dependency boundaries, and long-term maintainability.",
		PrimaryQuestion:  "Does the change preserve architecture integrity under expected roadmap growth?",
		RequiredEvidence: []string{"boundary-map", "dependency-graph", "upgrade-path"},
		EscalationTarget: "product-strategist",
	},
	{
		ID:               "sre",
		DecisionLens:     "Reliability, operability, failure isolation, and incident response speed.",
		PrimaryQuestion:  "Can this behavior survive production-like stress without paging instability?",
		RequiredEvidence: []string{"slo-impact", "runbook-delta", "rollback-plan"},
		EscalationTarget: "systems-architect",
	},
	{
		ID:               "security-reviewer",
		DecisionLens:     "Abuse resistance, data exposure paths, and policy compliance.",
		PrimaryQuestion:  "What is the worst realistic abuse path and is it detected and contained?",
		RequiredEvidence: []string{"threat-model", "secret-handling-proof", "policy-check-results"},
		EscalationTarget: "sre",
	},
	{
		ID:               "dx-expert",
		DecisionLens:     "Operator ergonomics, clarity of contracts, and iteration speed.",
		PrimaryQuestion:  "Can a maintainer execute and verify this flow without hidden context?",
		RequiredEvidence: []string{"contract-examples", "cli-runbook", "verification-latency"},
		EscalationTarget: "systems-architect",
	},
	{
		ID:               "product-strategist",
		DecisionLens:     "Outcome alignment, user value, and roadmap sequencing.",
		PrimaryQuestion:  "Does this recommendation maximize user impact for the next planning horizon?",
		RequiredEvidence: []string{"outcome-hypothesis", "adoption-signal", "opportunity-cost"},
		EscalationTarget: "systems-architect",
	},
}

// personaRegistryYAML is the schema for specs/persona-registry.yaml
type personaRegistryYAML struct {
	Personas []struct {
		ID               string   `yaml:"id"`
		DecisionLens     string   `yaml:"decision_lens"`
		PrimaryQuestion  string   `yaml:"primary_question"`
		RequiredEvidence []string `yaml:"required_evidence"`
		EscalationTarget string   `yaml:"escalation_target"`
		Model            string   `yaml:"model"`
	} `yaml:"personas"`
}

// LoadSwarmPlanFromRegistry loads roles from specs/persona-registry.yaml when present.
// Falls back to DefaultDeepThinkingSwarmPlan() if file missing or invalid.
func LoadSwarmPlanFromRegistry(workDir string) DeepThinkingSwarmPlan {
	path := filepath.Join(workDir, "specs", "persona-registry.yaml")
	b, err := os.ReadFile(path)
	if err != nil {
		return DefaultDeepThinkingSwarmPlan()
	}
	var reg personaRegistryYAML
	if err := yaml.Unmarshal(b, &reg); err != nil {
		return DefaultDeepThinkingSwarmPlan()
	}
	if len(reg.Personas) == 0 {
		return DefaultDeepThinkingSwarmPlan()
	}
	roles := make([]PersonaRole, 0, len(reg.Personas))
	for _, p := range reg.Personas {
		if p.ID == "" {
			continue
		}
		roles = append(roles, PersonaRole{
			ID:               p.ID,
			DecisionLens:     p.DecisionLens,
			PrimaryQuestion:  p.PrimaryQuestion,
			RequiredEvidence: p.RequiredEvidence,
			EscalationTarget: p.EscalationTarget,
		})
	}
	if len(roles) == 0 {
		return DefaultDeepThinkingSwarmPlan()
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })
	plan := DefaultDeepThinkingSwarmPlan()
	plan.Roles = roles
	return plan
}

func DefaultDeepThinkingSwarmPlan() DeepThinkingSwarmPlan {
	triggers := append([]string(nil), deepThinkingTriggerSignals...)
	sort.Strings(triggers)

	phases := append([]EvaluationPhase(nil), deepThinkingPhases...)
	sort.Slice(phases, func(i, j int) bool { return phases[i].ID < phases[j].ID })

	roles := append([]PersonaRole(nil), deepThinkingRoles...)
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })

	return DeepThinkingSwarmPlan{
		ContractVersion: DeepThinkingSwarmPlanContractVersion,
		Cadence:         "weekly-or-change-triggered",
		TriggerSignals:  triggers,
		Phases:          phases,
		Roles:           roles,
		Collaboration: CollaborationProtocol{
			ConflictRule:       "adversarial-double-pass",
			ConsensusThreshold: "4-of-5-persona-majority",
			ResolutionFallback: "escalate-to-issue-owner-with-dissent-log",
		},
	}
}

func EvaluateSwarmPlanReadiness(triggerSignals map[string]bool, roleSignals map[string]bool, phaseSignals map[string]bool) SwarmPlanReadinessResult {
	plan := DefaultDeepThinkingSwarmPlan()

	missingTriggers := make([]string, 0, len(plan.TriggerSignals))
	for _, signal := range plan.TriggerSignals {
		if !triggerSignals[signal] {
			missingTriggers = append(missingTriggers, signal)
		}
	}

	missingRoles := make([]string, 0, len(plan.Roles))
	for _, role := range plan.Roles {
		if !roleSignals[role.ID] {
			missingRoles = append(missingRoles, role.ID)
		}
	}

	missingPhases := make([]string, 0, len(plan.Phases))
	for _, phase := range plan.Phases {
		if !phaseSignals[phase.ID] {
			missingPhases = append(missingPhases, phase.ID)
		}
	}

	return SwarmPlanReadinessResult{
		Passed:                len(missingTriggers) == 0 && len(missingRoles) == 0 && len(missingPhases) == 0,
		MissingTriggerSignals: missingTriggers,
		MissingRoleIDs:        missingRoles,
		MissingPhaseIDs:       missingPhases,
	}
}
