package evaluator

import "sort"

const OutcomeScoringRubricContractVersion = "deep-thinking-outcome-rubric/v1"

type OutcomeDimension struct {
	ID          string
	Weight      int
	Description string
}

type OutcomeScoringRubric struct {
	ContractVersion string
	Dimensions      []OutcomeDimension
}

type OpportunityEvaluation struct {
	OpportunityID   string
	DimensionScores map[string]int
}

type OpportunityRank struct {
	OpportunityID     string
	WeightedScore     int
	NormalizedScore   int
	MissingDimensions []string
	UnknownDimensions []string
}

func DefaultOutcomeScoringRubric() OutcomeScoringRubric {
	dimensions := []OutcomeDimension{
		{
			ID:          "delivery",
			Weight:      20,
			Description: "Lead-time and throughput impact on shipping improvements.",
		},
		{
			ID:          "developer-experience",
			Weight:      15,
			Description: "Operator and maintainer ergonomics for repeatable execution.",
		},
		{
			ID:          "reliability",
			Weight:      40,
			Description: "Resilience, observability, and rollback confidence under stress.",
		},
		{
			ID:          "security",
			Weight:      25,
			Description: "Exploit resistance, data handling safety, and policy alignment.",
		},
	}

	sort.Slice(dimensions, func(i, j int) bool { return dimensions[i].ID < dimensions[j].ID })

	return OutcomeScoringRubric{
		ContractVersion: OutcomeScoringRubricContractVersion,
		Dimensions:      dimensions,
	}
}

func RankImprovementOpportunities(rubric OutcomeScoringRubric, evaluations []OpportunityEvaluation) []OpportunityRank {
	if len(evaluations) == 0 {
		return nil
	}

	dimensionWeights := make(map[string]int, len(rubric.Dimensions))
	dimensionIDs := make([]string, 0, len(rubric.Dimensions))
	totalWeight := 0
	for _, dimension := range rubric.Dimensions {
		dimensionWeights[dimension.ID] = dimension.Weight
		dimensionIDs = append(dimensionIDs, dimension.ID)
		totalWeight += dimension.Weight
	}
	sort.Strings(dimensionIDs)

	ranks := make([]OpportunityRank, 0, len(evaluations))
	for _, evaluation := range evaluations {
		weightedScore := 0
		missing := make([]string, 0, len(dimensionIDs))
		for _, dimensionID := range dimensionIDs {
			score, ok := evaluation.DimensionScores[dimensionID]
			if !ok {
				missing = append(missing, dimensionID)
				continue
			}
			weightedScore += clampScore(score) * dimensionWeights[dimensionID]
		}

		unknown := make([]string, 0, len(evaluation.DimensionScores))
		for dimensionID := range evaluation.DimensionScores {
			if _, ok := dimensionWeights[dimensionID]; !ok {
				unknown = append(unknown, dimensionID)
			}
		}
		sort.Strings(unknown)

		normalized := 0
		if totalWeight > 0 {
			normalized = weightedScore / totalWeight
		}

		ranks = append(ranks, OpportunityRank{
			OpportunityID:     evaluation.OpportunityID,
			WeightedScore:     weightedScore,
			NormalizedScore:   normalized,
			MissingDimensions: missing,
			UnknownDimensions: unknown,
		})
	}

	sort.Slice(ranks, func(i, j int) bool {
		if ranks[i].NormalizedScore == ranks[j].NormalizedScore {
			return ranks[i].OpportunityID < ranks[j].OpportunityID
		}
		return ranks[i].NormalizedScore > ranks[j].NormalizedScore
	})

	return ranks
}
