package evaluator

import "testing"

func TestDefaultTrialRunFixturesDeterministic(t *testing.T) {
	fixtures := DefaultTrialRunFixtures()

	if len(fixtures) != 4 {
		t.Fatalf("expected 4 deterministic trial fixtures, got %d", len(fixtures))
	}
	for i, fixture := range fixtures {
		if fixture.RunID == "" || fixture.IssueID == "" {
			t.Fatalf("fixture %d missing run metadata: %+v", i, fixture)
		}
		if len(fixture.PersonaScores) == 0 {
			t.Fatalf("fixture %s missing persona scores", fixture.RunID)
		}
		if len(fixture.OpportunitySet) == 0 {
			t.Fatalf("fixture %s missing opportunity set", fixture.RunID)
		}
		if i > 0 && fixtures[i-1].RunID > fixture.RunID {
			t.Fatalf("fixtures not sorted by run id: %q before %q", fixtures[i-1].RunID, fixture.RunID)
		}
	}
}

func TestBuildTrialRunCalibrationReportDefaultThresholds(t *testing.T) {
	fixtures := DefaultTrialRunFixtures()
	report := BuildTrialRunCalibrationReport(
		fixtures,
		DefaultRecommendationQualityThresholds(),
		DefaultDeepThinkingSwarmPlan(),
		DefaultOutcomeScoringRubric(),
	)

	if report.ContractVersion != TrialRunCalibrationContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", report.ContractVersion, TrialRunCalibrationContractVersion)
	}
	if report.RunCount != 4 {
		t.Fatalf("expected run count 4, got %d", report.RunCount)
	}
	if report.ConsensusRatePercent != 75 {
		t.Fatalf("consensus rate mismatch: got=%d want=%d", report.ConsensusRatePercent, 75)
	}
	if report.RunQualityPassRatePercent != 75 {
		t.Fatalf("run quality pass rate mismatch: got=%d want=%d", report.RunQualityPassRatePercent, 75)
	}
	if !report.OverallGatePassed {
		t.Fatalf("expected default threshold calibration pass, got %+v", report.ThresholdEvidence)
	}

	var failedRuns int
	for _, result := range report.RunResults {
		if result.RunID == "trial-charlie" {
			if result.QualityGatePassed {
				t.Fatalf("expected trial-charlie to fail quality gate, got %+v", result)
			}
			if result.TopOpportunityScore != 73 {
				t.Fatalf("unexpected trial-charlie top opportunity score: %d", result.TopOpportunityScore)
			}
		}
		if !result.QualityGatePassed {
			failedRuns++
		}
	}
	if failedRuns != 1 {
		t.Fatalf("expected exactly one run quality gate failure, got %d", failedRuns)
	}
}

func TestBuildTrialRunCalibrationReportStrictThresholdsFail(t *testing.T) {
	fixtures := DefaultTrialRunFixtures()
	thresholds := RecommendationQualityThresholds{
		MinConsensusRatePercent:      90,
		MinAveragePersonaScore:       80,
		MinTopOpportunityScore:       90,
		MaxMissingPersonaResponses:   0,
		MinRunQualityPassRatePercent: 100,
	}

	report := BuildTrialRunCalibrationReport(
		fixtures,
		thresholds,
		DefaultDeepThinkingSwarmPlan(),
		DefaultOutcomeScoringRubric(),
	)

	if report.OverallGatePassed {
		t.Fatalf("expected strict threshold calibration to fail, got %+v", report)
	}
	if len(report.ThresholdEvidence) != 2 {
		t.Fatalf("expected 2 threshold evidence rows, got %d", len(report.ThresholdEvidence))
	}
	for _, row := range report.ThresholdEvidence {
		if row.Passed {
			t.Fatalf("expected strict threshold evidence to fail for %s, got %+v", row.Metric, row)
		}
	}
}
