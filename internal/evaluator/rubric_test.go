package evaluator

import "testing"

func TestDefaultOutcomeScoringRubricDeterministicAndWeighted(t *testing.T) {
	rubric := DefaultOutcomeScoringRubric()

	if rubric.ContractVersion != OutcomeScoringRubricContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", rubric.ContractVersion, OutcomeScoringRubricContractVersion)
	}
	if len(rubric.Dimensions) != 4 {
		t.Fatalf("expected 4 dimensions, got %d", len(rubric.Dimensions))
	}

	totalWeight := 0
	for i, dimension := range rubric.Dimensions {
		if dimension.ID == "" || dimension.Description == "" {
			t.Fatalf("dimension %d has empty required field: %+v", i, dimension)
		}
		if dimension.Weight <= 0 {
			t.Fatalf("dimension %s must have positive weight", dimension.ID)
		}
		totalWeight += dimension.Weight
		if i > 0 && rubric.Dimensions[i-1].ID > dimension.ID {
			t.Fatalf("dimensions not sorted: %q before %q", rubric.Dimensions[i-1].ID, dimension.ID)
		}
	}

	if totalWeight != 100 {
		t.Fatalf("expected total weight 100, got %d", totalWeight)
	}
}

func TestRankImprovementOpportunitiesOrdersByNormalizedScore(t *testing.T) {
	rubric := DefaultOutcomeScoringRubric()

	ranks := RankImprovementOpportunities(rubric, []OpportunityEvaluation{
		{
			OpportunityID: "op-b",
			DimensionScores: map[string]int{
				"delivery":             78,
				"developer-experience": 75,
				"reliability":          80,
				"security":             82,
			},
		},
		{
			OpportunityID: "op-a",
			DimensionScores: map[string]int{
				"delivery":             94,
				"developer-experience": 90,
				"reliability":          92,
				"security":             96,
			},
		},
	})

	if len(ranks) != 2 {
		t.Fatalf("expected 2 ranked opportunities, got %d", len(ranks))
	}
	if ranks[0].OpportunityID != "op-a" {
		t.Fatalf("expected op-a first, got %s", ranks[0].OpportunityID)
	}
	if ranks[1].OpportunityID != "op-b" {
		t.Fatalf("expected op-b second, got %s", ranks[1].OpportunityID)
	}
	if ranks[0].NormalizedScore <= ranks[1].NormalizedScore {
		t.Fatalf("expected descending normalized score order: %+v", ranks)
	}
}

func TestRankImprovementOpportunitiesTracksMissingUnknownAndClamp(t *testing.T) {
	rubric := DefaultOutcomeScoringRubric()

	ranks := RankImprovementOpportunities(rubric, []OpportunityEvaluation{
		{
			OpportunityID: "op-c",
			DimensionScores: map[string]int{
				"delivery":             110,
				"developer-experience": 65,
				"security":             -4,
				"unknown-dimension":    50,
			},
		},
	})

	if len(ranks) != 1 {
		t.Fatalf("expected one rank result, got %d", len(ranks))
	}
	rank := ranks[0]

	wantMissing := []string{"reliability"}
	if len(rank.MissingDimensions) != len(wantMissing) {
		t.Fatalf("missing dimensions mismatch: got=%v want=%v", rank.MissingDimensions, wantMissing)
	}
	if rank.MissingDimensions[0] != wantMissing[0] {
		t.Fatalf("missing dimension mismatch: got=%v want=%v", rank.MissingDimensions, wantMissing)
	}

	wantUnknown := []string{"unknown-dimension"}
	if len(rank.UnknownDimensions) != len(wantUnknown) {
		t.Fatalf("unknown dimensions mismatch: got=%v want=%v", rank.UnknownDimensions, wantUnknown)
	}
	if rank.UnknownDimensions[0] != wantUnknown[0] {
		t.Fatalf("unknown dimension mismatch: got=%v want=%v", rank.UnknownDimensions, wantUnknown)
	}

	if rank.WeightedScore != 2975 {
		t.Fatalf("weighted score mismatch: got=%d want=%d", rank.WeightedScore, 2975)
	}
	if rank.NormalizedScore != 29 {
		t.Fatalf("normalized score mismatch: got=%d want=%d", rank.NormalizedScore, 29)
	}
}
