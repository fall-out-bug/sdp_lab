package evaluator

import (
	"errors"
	"testing"
)

func TestBuildContinuousImprovementPRLoopReportDeterministic(t *testing.T) {
	fixtures := DefaultTrialRunFixtures()
	plan := DefaultDeepThinkingSwarmPlan()
	rubric := DefaultOutcomeScoringRubric()
	thresholds := DefaultRecommendationQualityThresholds()

	calibration := BuildTrialRunCalibrationReport(fixtures, thresholds, plan, rubric)

	packet, err := BuildPersonaExecutionPacket("sdp_dev-hx0.1.6", plan)
	if err != nil {
		t.Fatalf("unexpected packet build error: %v", err)
	}

	runtimeReport := AssembleSwarmScoreReport(packet, fixtures[0].PersonaScores)
	ranks := RankImprovementOpportunities(rubric, fixtures[0].OpportunitySet)

	report, err := BuildContinuousImprovementPRLoopReport("sdp_dev-hx0.1.6", calibration, runtimeReport, ranks, DefaultPRLoopGuardrails())
	if err != nil {
		t.Fatalf("unexpected pr loop build error: %v", err)
	}

	if report.ContractVersion != ContinuousImprovementPRLoopContractVersion {
		t.Fatalf("unexpected contract version: got=%s want=%s", report.ContractVersion, ContinuousImprovementPRLoopContractVersion)
	}
	if !report.ReadyForPR {
		t.Fatalf("expected PR loop report to be ready, got %+v", report.GuardrailChecks)
	}
	if report.BacklogPlan.ContractVersion != BacklogInjectionPlanContractVersion {
		t.Fatalf("unexpected backlog contract version: got=%s want=%s", report.BacklogPlan.ContractVersion, BacklogInjectionPlanContractVersion)
	}
	if len(report.BacklogPlan.InjectedItems) != 1 {
		t.Fatalf("expected one injected item above minimum score, got %d", len(report.BacklogPlan.InjectedItems))
	}
	if report.BacklogPlan.InjectedItems[0].OpportunityID != "report-normalizer" {
		t.Fatalf("expected top ranked opportunity first, got %+v", report.BacklogPlan.InjectedItems)
	}
	if report.BacklogPlan.InjectedItems[0].TargetIssuePriority != 1 {
		t.Fatalf("unexpected priority mapping for top opportunity: %+v", report.BacklogPlan.InjectedItems[0])
	}
	if len(report.BacklogPlan.InjectedItems[0].SourceRecommendations) != 3 {
		t.Fatalf("expected deterministic recommendation slice, got %+v", report.BacklogPlan.InjectedItems[0].SourceRecommendations)
	}
}

func TestBuildContinuousImprovementPRLoopReportValidation(t *testing.T) {
	_, err := BuildContinuousImprovementPRLoopReport(
		"",
		TrialRunCalibrationReport{},
		SwarmScoreReport{},
		nil,
		DefaultPRLoopGuardrails(),
	)
	if !errors.Is(err, errPRLoopIssueIDRequired) {
		t.Fatalf("expected issue id required error, got %v", err)
	}
}

func TestBuildContinuousImprovementPRLoopReportBlocksFailedCalibration(t *testing.T) {
	fixtures := DefaultTrialRunFixtures()
	plan := DefaultDeepThinkingSwarmPlan()
	rubric := DefaultOutcomeScoringRubric()

	strictThresholds := RecommendationQualityThresholds{
		MinConsensusRatePercent:      100,
		MinAveragePersonaScore:       85,
		MinTopOpportunityScore:       95,
		MaxMissingPersonaResponses:   0,
		MinRunQualityPassRatePercent: 100,
	}
	calibration := BuildTrialRunCalibrationReport(fixtures, strictThresholds, plan, rubric)

	packet, err := BuildPersonaExecutionPacket("sdp_dev-hx0.1.6", plan)
	if err != nil {
		t.Fatalf("unexpected packet build error: %v", err)
	}
	runtimeReport := AssembleSwarmScoreReport(packet, fixtures[0].PersonaScores)
	ranks := RankImprovementOpportunities(rubric, fixtures[0].OpportunitySet)

	report, err := BuildContinuousImprovementPRLoopReport("sdp_dev-hx0.1.6", calibration, runtimeReport, ranks, DefaultPRLoopGuardrails())
	if err != nil {
		t.Fatalf("unexpected pr loop build error: %v", err)
	}
	if report.ReadyForPR {
		t.Fatalf("expected calibration failure to block pr loop readiness")
	}

	passedCalibrationCheck := true
	for _, check := range report.GuardrailChecks {
		if check.ID == "calibration-overall-gate" {
			passedCalibrationCheck = check.Passed
			break
		}
	}
	if passedCalibrationCheck {
		t.Fatalf("expected calibration guardrail check to fail")
	}
}

func TestBuildBacklogInjectionPlanEnforcesDeterministicGuardrails(t *testing.T) {
	guardrails := DefaultPRLoopGuardrails()
	ranks := []OpportunityRank{
		{OpportunityID: "needs-complete-rubric", NormalizedScore: 92, MissingDimensions: []string{"security"}},
		{OpportunityID: "unknown-rubric", NormalizedScore: 91, UnknownDimensions: []string{"privacy"}},
		{OpportunityID: "score-too-low", NormalizedScore: 79},
		{OpportunityID: "eligible-high", NormalizedScore: 90},
		{OpportunityID: "eligible-mid", NormalizedScore: 85},
		{OpportunityID: "eligible-limit-overflow", NormalizedScore: 84},
		{OpportunityID: "eligible-overflow-second", NormalizedScore: 83},
	}

	plan := BuildBacklogInjectionPlan(
		"sdp_dev-hx0.1.6",
		TrialRunCalibrationReport{ContractVersion: TrialRunCalibrationContractVersion},
		SwarmScoreReport{ContractVersion: PersonaExecutionPacketContractVersion, PriorityRecommendations: []string{"dx-expert: deterministic evidence", "sre: rollback confidence"}},
		ranks,
		guardrails,
	)

	if len(plan.InjectedItems) != 3 {
		t.Fatalf("expected max injected item guardrail (3), got %d", len(plan.InjectedItems))
	}
	if plan.InjectedItems[0].OpportunityID != "eligible-high" {
		t.Fatalf("expected deterministic descending order, got %+v", plan.InjectedItems)
	}

	reasonByOpportunity := make(map[string]string, len(plan.ExcludedOpportunities))
	for _, exclusion := range plan.ExcludedOpportunities {
		reasonByOpportunity[exclusion.OpportunityID] = exclusion.Reason
	}
	if reasonByOpportunity["needs-complete-rubric"] != "missing-rubric-dimensions" {
		t.Fatalf("missing-dimension guardrail not enforced: %+v", plan.ExcludedOpportunities)
	}
	if reasonByOpportunity["unknown-rubric"] != "unknown-rubric-dimensions" {
		t.Fatalf("unknown-dimension guardrail not enforced: %+v", plan.ExcludedOpportunities)
	}
	if reasonByOpportunity["score-too-low"] != "score-below-minimum" {
		t.Fatalf("score guardrail not enforced: %+v", plan.ExcludedOpportunities)
	}
	if reasonByOpportunity["eligible-overflow-second"] != "max-injected-items-reached" {
		t.Fatalf("max injected guardrail not enforced: %+v", plan.ExcludedOpportunities)
	}
}
