package evaluator

import (
	"errors"
	"sort"
	"strings"

	"sdp_dev/internal/beads"
)

const PersonaExecutionPacketContractVersion = "deep-thinking-evaluator-runtime/v1"

type PersonaExecutionUnit struct {
	PersonaID        string
	DecisionLens     string
	PrimaryQuestion  string
	RequiredEvidence []string
	EscalationTarget string
	PhaseFocus       []string
	EntryGateSignals []string
}

type PersonaExecutionPacket struct {
	ContractVersion string
	IssueID         string
	Cadence         string
	PhaseOrder      []string
	Units           []PersonaExecutionUnit
}

type PersonaScore struct {
	PersonaID      string
	Score          int
	Summary        string
	Risks          []string
	Recommendation string
}

type SwarmScoreReport struct {
	ContractVersion         string
	IssueID                 string
	PersonaCount            int
	RespondedPersonaCount   int
	MissingPersonaIDs       []string
	UnknownPersonaIDs       []string
	AverageScore            int
	MinScore                int
	MaxScore                int
	ConsensusReached        bool
	DissentingPersonaIDs    []string
	PriorityRecommendations []string
}

var errIssueIDRequired = errors.New("issue id is required")
var errSwarmPlanIncomplete = errors.New("swarm plan must include trigger signals, phases, and roles")

func BuildPersonaExecutionPacket(issueID string, plan DeepThinkingSwarmPlan) (PersonaExecutionPacket, error) {
	if issueID == "" {
		return PersonaExecutionPacket{}, errIssueIDRequired
	}
	if len(plan.TriggerSignals) == 0 || len(plan.Phases) == 0 || len(plan.Roles) == 0 {
		return PersonaExecutionPacket{}, errSwarmPlanIncomplete
	}

	phaseOrder := make([]string, 0, len(plan.Phases))
	for _, phase := range plan.Phases {
		phaseOrder = append(phaseOrder, phase.ID)
	}
	sort.Strings(phaseOrder)

	entryGateSignals := append([]string(nil), plan.TriggerSignals...)
	sort.Strings(entryGateSignals)

	roles := append([]PersonaRole(nil), plan.Roles...)
	sort.Slice(roles, func(i, j int) bool { return roles[i].ID < roles[j].ID })

	units := make([]PersonaExecutionUnit, 0, len(roles))
	for _, role := range roles {
		requiredEvidence := append([]string(nil), role.RequiredEvidence...)
		sort.Strings(requiredEvidence)

		units = append(units, PersonaExecutionUnit{
			PersonaID:        role.ID,
			DecisionLens:     role.DecisionLens,
			PrimaryQuestion:  role.PrimaryQuestion,
			RequiredEvidence: requiredEvidence,
			EscalationTarget: role.EscalationTarget,
			PhaseFocus:       append([]string(nil), phaseOrder...),
			EntryGateSignals: append([]string(nil), entryGateSignals...),
		})
	}

	return PersonaExecutionPacket{
		ContractVersion: PersonaExecutionPacketContractVersion,
		IssueID:         issueID,
		Cadence:         plan.Cadence,
		PhaseOrder:      phaseOrder,
		Units:           units,
	}, nil
}

func AssembleSwarmScoreReport(packet PersonaExecutionPacket, scores []PersonaScore) SwarmScoreReport {
	scoresByPersona := make(map[string]PersonaScore, len(scores))
	unknownSet := map[string]struct{}{}

	knownPersonas := make(map[string]struct{}, len(packet.Units))
	personaOrder := make([]string, 0, len(packet.Units))
	for _, unit := range packet.Units {
		knownPersonas[unit.PersonaID] = struct{}{}
		personaOrder = append(personaOrder, unit.PersonaID)
	}
	sort.Strings(personaOrder)

	input := append([]PersonaScore(nil), scores...)
	sort.Slice(input, func(i, j int) bool {
		if input[i].PersonaID == input[j].PersonaID {
			return input[i].Score > input[j].Score
		}
		return input[i].PersonaID < input[j].PersonaID
	})

	for _, score := range input {
		if _, ok := knownPersonas[score.PersonaID]; !ok {
			unknownSet[score.PersonaID] = struct{}{}
			continue
		}
		if _, exists := scoresByPersona[score.PersonaID]; exists {
			continue
		}
		scoresByPersona[score.PersonaID] = PersonaScore{
			PersonaID:      score.PersonaID,
			Score:          clampScore(score.Score),
			Summary:        score.Summary,
			Risks:          append([]string(nil), score.Risks...),
			Recommendation: score.Recommendation,
		}
	}

	missing := make([]string, 0, len(packet.Units))
	dissenting := make([]string, 0, len(packet.Units))
	total := 0
	minScore := 101
	maxScore := -1
	responded := 0
	recommendations := make([]PersonaScore, 0, len(scoresByPersona))

	for _, personaID := range personaOrder {
		score, ok := scoresByPersona[personaID]
		if !ok {
			missing = append(missing, personaID)
			continue
		}
		responded++
		total += score.Score
		if score.Score < minScore {
			minScore = score.Score
		}
		if score.Score > maxScore {
			maxScore = score.Score
		}
		if score.Score < 70 {
			dissenting = append(dissenting, personaID)
		}
		if score.Recommendation != "" {
			recommendations = append(recommendations, score)
		}
	}

	average := 0
	if responded > 0 {
		average = total / responded
	}
	if responded == 0 {
		minScore = 0
		maxScore = 0
	}

	unknown := make([]string, 0, len(unknownSet))
	for personaID := range unknownSet {
		unknown = append(unknown, personaID)
	}
	sort.Strings(unknown)

	sort.Slice(recommendations, func(i, j int) bool {
		if recommendations[i].Score == recommendations[j].Score {
			return recommendations[i].PersonaID < recommendations[j].PersonaID
		}
		return recommendations[i].Score > recommendations[j].Score
	})

	priority := make([]string, 0, len(recommendations))
	for _, recommendation := range recommendations {
		priority = append(priority, recommendation.PersonaID+": "+recommendation.Recommendation)
	}

	consensusThreshold := requiredConsensus(len(packet.Units))
	consensusReached := responded-len(dissenting) >= consensusThreshold && len(missing) == 0

	return SwarmScoreReport{
		ContractVersion:         PersonaExecutionPacketContractVersion,
		IssueID:                 packet.IssueID,
		PersonaCount:            len(packet.Units),
		RespondedPersonaCount:   responded,
		MissingPersonaIDs:       missing,
		UnknownPersonaIDs:       unknown,
		AverageScore:            average,
		MinScore:                minScore,
		MaxScore:                maxScore,
		ConsensusReached:        consensusReached,
		DissentingPersonaIDs:    dissenting,
		PriorityRecommendations: priority,
	}
}

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func requiredConsensus(personaCount int) int {
	if personaCount <= 0 {
		return 0
	}
	return (4*personaCount + 4) / 5
}

// RecommendationsToBeadsTasks creates Beads tasks from priority recommendations.
// Each recommendation is "persona_id: recommendation text".
func RecommendationsToBeadsTasks(workDir string, recommendations []string, maxItems int) ([]string, error) {
	if maxItems <= 0 {
		maxItems = 3
	}
	adapter := beads.NewAdapter(workDir)
	var created []string
	for i, rec := range recommendations {
		if i >= maxItems {
			break
		}
		title := rec
		if idx := strings.Index(rec, ": "); idx >= 0 {
			title = strings.TrimSpace(rec[idx+2:])
		}
		if len(title) > 80 {
			title = title[:77] + "..."
		}
		id, err := adapter.Create(beads.CreateOpts{
			Title:   title,
			Type:    "task",
			Priority: 2,
			Labels:  []string{"autonomy", "strict-evidence", "workstream:evaluator-recommendation"},
		})
		if err != nil {
			continue
		}
		created = append(created, id)
	}
	return created, nil
}
