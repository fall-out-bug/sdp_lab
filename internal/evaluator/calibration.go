package evaluator

import "sort"

const TrialRunCalibrationContractVersion = "deep-thinking-evaluator-calibration/v1"

type TrialRunFixture struct {
	RunID          string
	IssueID        string
	PersonaScores  []PersonaScore
	OpportunitySet []OpportunityEvaluation
}

type RecommendationQualityThresholds struct {
	MinConsensusRatePercent      int
	MinAveragePersonaScore       int
	MinTopOpportunityScore       int
	MaxMissingPersonaResponses   int
	MinRunQualityPassRatePercent int
}

type RunCalibrationResult struct {
	RunID               string
	ConsensusReached    bool
	AveragePersonaScore int
	MissingPersonaCount int
	TopOpportunityID    string
	TopOpportunityScore int
	QualityGatePassed   bool
	FailedChecks        []string
}

type CalibrationThresholdEvidence struct {
	Metric    string
	Observed  int
	Threshold int
	Passed    bool
}

type TrialRunCalibrationReport struct {
	ContractVersion           string
	Thresholds                RecommendationQualityThresholds
	RunResults                []RunCalibrationResult
	RunCount                  int
	ConsensusRatePercent      int
	RunQualityPassRatePercent int
	OverallGatePassed         bool
	ThresholdEvidence         []CalibrationThresholdEvidence
}

func DefaultRecommendationQualityThresholds() RecommendationQualityThresholds {
	return RecommendationQualityThresholds{
		MinConsensusRatePercent:      75,
		MinAveragePersonaScore:       74,
		MinTopOpportunityScore:       80,
		MaxMissingPersonaResponses:   1,
		MinRunQualityPassRatePercent: 75,
	}
}

func DefaultTrialRunFixtures() []TrialRunFixture {
	fixtures := []TrialRunFixture{
		{
			RunID:   "trial-alpha",
			IssueID: "sdp_dev-hx0.1.7.alpha",
			PersonaScores: []PersonaScore{
				{PersonaID: "dx-expert", Score: 86, Recommendation: "capture command transcript snippets in artifacts"},
				{PersonaID: "product-strategist", Score: 78, Recommendation: "prioritize highest leverage recommendation only"},
				{PersonaID: "security-reviewer", Score: 90, Recommendation: "attach deterministic policy-check evidence"},
				{PersonaID: "sre", Score: 82, Recommendation: "document rollback simulation in run report"},
				{PersonaID: "systems-architect", Score: 88, Recommendation: "keep recommendation scope bounded by component"},
			},
			OpportunitySet: []OpportunityEvaluation{
				{OpportunityID: "report-normalizer", DimensionScores: map[string]int{"delivery": 84, "developer-experience": 83, "reliability": 90, "security": 88}},
				{OpportunityID: "artifact-indexing", DimensionScores: map[string]int{"delivery": 79, "developer-experience": 80, "reliability": 82, "security": 77}},
			},
		},
		{
			RunID:   "trial-bravo",
			IssueID: "sdp_dev-hx0.1.7.bravo",
			PersonaScores: []PersonaScore{
				{PersonaID: "dx-expert", Score: 77, Recommendation: "fail fast when fixture evidence is incomplete"},
				{PersonaID: "product-strategist", Score: 68, Recommendation: "defer low-signal recommendation candidates"},
				{PersonaID: "security-reviewer", Score: 81, Recommendation: "add explicit abuse-case verification rows"},
				{PersonaID: "sre", Score: 74, Recommendation: "attach triage note for dissenting persona"},
				{PersonaID: "systems-architect", Score: 79, Recommendation: "limit recommendation set to top-ranked items"},
			},
			OpportunitySet: []OpportunityEvaluation{
				{OpportunityID: "calibration-threshold-evidence", DimensionScores: map[string]int{"delivery": 78, "developer-experience": 75, "reliability": 86, "security": 84}},
				{OpportunityID: "persona-retry-guidance", DimensionScores: map[string]int{"delivery": 76, "developer-experience": 79, "reliability": 80, "security": 72}},
			},
		},
		{
			RunID:   "trial-charlie",
			IssueID: "sdp_dev-hx0.1.7.charlie",
			PersonaScores: []PersonaScore{
				{PersonaID: "dx-expert", Score: 73, Recommendation: "publish artifact schema examples"},
				{PersonaID: "security-reviewer", Score: 72, Recommendation: "track unresolved threat-model deltas"},
				{PersonaID: "sre", Score: 69, Recommendation: "capture stabilization debt as follow-up issue"},
				{PersonaID: "systems-architect", Score: 74, Recommendation: "tighten recommendation acceptance checklist"},
			},
			OpportunitySet: []OpportunityEvaluation{
				{OpportunityID: "consensus-escalation-template", DimensionScores: map[string]int{"delivery": 70, "developer-experience": 72, "reliability": 76, "security": 74}},
				{OpportunityID: "runbook-coverage", DimensionScores: map[string]int{"delivery": 68, "developer-experience": 71, "reliability": 73, "security": 70}},
			},
		},
		{
			RunID:   "trial-delta",
			IssueID: "sdp_dev-hx0.1.7.delta",
			PersonaScores: []PersonaScore{
				{PersonaID: "dx-expert", Score: 80, Recommendation: "add deterministic evidence links into final summary"},
				{PersonaID: "product-strategist", Score: 75, Recommendation: "prefer recommendations with immediate user impact"},
				{PersonaID: "security-reviewer", Score: 84, Recommendation: "require security rubric coverage per recommendation"},
				{PersonaID: "sre", Score: 76, Recommendation: "record rollback confidence rating per run"},
				{PersonaID: "systems-architect", Score: 81, Recommendation: "enforce single-owner follow-through per recommendation"},
			},
			OpportunitySet: []OpportunityEvaluation{
				{OpportunityID: "recommendation-quality-gate", DimensionScores: map[string]int{"delivery": 82, "developer-experience": 80, "reliability": 85, "security": 81}},
				{OpportunityID: "evidence-diff-summary", DimensionScores: map[string]int{"delivery": 78, "developer-experience": 82, "reliability": 79, "security": 77}},
			},
		},
	}

	sort.Slice(fixtures, func(i, j int) bool { return fixtures[i].RunID < fixtures[j].RunID })
	return fixtures
}

func BuildTrialRunCalibrationReport(fixtures []TrialRunFixture, thresholds RecommendationQualityThresholds, plan DeepThinkingSwarmPlan, rubric OutcomeScoringRubric) TrialRunCalibrationReport {
	runResults := make([]RunCalibrationResult, 0, len(fixtures))
	consensusCount := 0
	runQualityPassCount := 0

	for _, fixture := range fixtures {
		packet, err := BuildPersonaExecutionPacket(fixture.IssueID, plan)
		if err != nil {
			continue
		}

		scoreReport := AssembleSwarmScoreReport(packet, fixture.PersonaScores)
		ranked := RankImprovementOpportunities(rubric, fixture.OpportunitySet)

		topOpportunityID := ""
		topOpportunityScore := 0
		if len(ranked) > 0 {
			topOpportunityID = ranked[0].OpportunityID
			topOpportunityScore = ranked[0].NormalizedScore
		}

		checks := make([]string, 0, 4)
		if !scoreReport.ConsensusReached {
			checks = append(checks, "consensus")
		}
		if scoreReport.AverageScore < thresholds.MinAveragePersonaScore {
			checks = append(checks, "average-persona-score")
		}
		if len(scoreReport.MissingPersonaIDs) > thresholds.MaxMissingPersonaResponses {
			checks = append(checks, "missing-persona-coverage")
		}
		if topOpportunityScore < thresholds.MinTopOpportunityScore {
			checks = append(checks, "top-opportunity-score")
		}

		if scoreReport.ConsensusReached {
			consensusCount++
		}
		if len(checks) == 0 {
			runQualityPassCount++
		}

		runResults = append(runResults, RunCalibrationResult{
			RunID:               fixture.RunID,
			ConsensusReached:    scoreReport.ConsensusReached,
			AveragePersonaScore: scoreReport.AverageScore,
			MissingPersonaCount: len(scoreReport.MissingPersonaIDs),
			TopOpportunityID:    topOpportunityID,
			TopOpportunityScore: topOpportunityScore,
			QualityGatePassed:   len(checks) == 0,
			FailedChecks:        checks,
		})
	}

	sort.Slice(runResults, func(i, j int) bool { return runResults[i].RunID < runResults[j].RunID })

	consensusRate := 0
	runQualityPassRate := 0
	if len(runResults) > 0 {
		consensusRate = consensusCount * 100 / len(runResults)
		runQualityPassRate = runQualityPassCount * 100 / len(runResults)
	}

	evidence := []CalibrationThresholdEvidence{
		{
			Metric:    "consensus-rate-percent",
			Observed:  consensusRate,
			Threshold: thresholds.MinConsensusRatePercent,
			Passed:    consensusRate >= thresholds.MinConsensusRatePercent,
		},
		{
			Metric:    "run-quality-pass-rate-percent",
			Observed:  runQualityPassRate,
			Threshold: thresholds.MinRunQualityPassRatePercent,
			Passed:    runQualityPassRate >= thresholds.MinRunQualityPassRatePercent,
		},
	}

	overallPass := true
	for _, entry := range evidence {
		if !entry.Passed {
			overallPass = false
			break
		}
	}

	return TrialRunCalibrationReport{
		ContractVersion:           TrialRunCalibrationContractVersion,
		Thresholds:                thresholds,
		RunResults:                runResults,
		RunCount:                  len(runResults),
		ConsensusRatePercent:      consensusRate,
		RunQualityPassRatePercent: runQualityPassRate,
		OverallGatePassed:         overallPass,
		ThresholdEvidence:         evidence,
	}
}
