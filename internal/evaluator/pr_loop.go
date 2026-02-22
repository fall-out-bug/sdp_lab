package evaluator

import (
	"errors"
	"sort"
	"strings"
)

const ContinuousImprovementPRLoopContractVersion = "deep-thinking-improvement-pr-loop/v1"
const BacklogInjectionPlanContractVersion = "deep-thinking-backlog-injection-plan/v1"

var errPRLoopIssueIDRequired = errors.New("issue id is required")

type PRLoopGuardrails struct {
	RequireCalibrationPass      bool
	RequireConsensus            bool
	RequireNoMissingPersonas    bool
	MinOpportunityScore         int
	MaxInjectedItems            int
	MaxPersonaRecommendations   int
	RequireCompleteRubricScores bool
}

type PRLoopGuardrailCheck struct {
	ID      string
	Passed  bool
	Details string
}

type BacklogInjectionExclusion struct {
	OpportunityID string
	Reason        string
}

type BacklogInjectionItem struct {
	OpportunityID           string
	NormalizedScore         int
	TargetIssueTitle        string
	TargetIssueType         string
	TargetIssuePriority     int
	TargetIssueLabels       []string
	SourceRecommendations   []string
	RequiredEvidenceSignals []string
}

type BacklogInjectionPlan struct {
	ContractVersion        string
	SourceIssueID          string
	SourceContractVersions []string
	EligibleOpportunityIDs []string
	InjectedItems          []BacklogInjectionItem
	ExcludedOpportunities  []BacklogInjectionExclusion
}

type ContinuousImprovementPRLoopReport struct {
	ContractVersion string
	IssueID         string
	ReadyForPR      bool
	GuardrailChecks []PRLoopGuardrailCheck
	BacklogPlan     BacklogInjectionPlan
}

func DefaultPRLoopGuardrails() PRLoopGuardrails {
	return PRLoopGuardrails{
		RequireCalibrationPass:      true,
		RequireConsensus:            true,
		RequireNoMissingPersonas:    true,
		MinOpportunityScore:         80,
		MaxInjectedItems:            3,
		MaxPersonaRecommendations:   3,
		RequireCompleteRubricScores: true,
	}
}

func BuildContinuousImprovementPRLoopReport(issueID string, calibration TrialRunCalibrationReport, scoreReport SwarmScoreReport, ranks []OpportunityRank, guardrails PRLoopGuardrails) (ContinuousImprovementPRLoopReport, error) {
	if issueID == "" {
		return ContinuousImprovementPRLoopReport{}, errPRLoopIssueIDRequired
	}

	checks := make([]PRLoopGuardrailCheck, 0, 4)
	if guardrails.RequireCalibrationPass {
		checks = append(checks, PRLoopGuardrailCheck{
			ID:      "calibration-overall-gate",
			Passed:  calibration.OverallGatePassed,
			Details: "trial-run calibration aggregate threshold evidence must pass",
		})
	}
	if guardrails.RequireConsensus {
		checks = append(checks, PRLoopGuardrailCheck{
			ID:      "runtime-consensus",
			Passed:  scoreReport.ConsensusReached,
			Details: "swarm runtime report must reach consensus",
		})
	}
	if guardrails.RequireNoMissingPersonas {
		checks = append(checks, PRLoopGuardrailCheck{
			ID:      "persona-coverage",
			Passed:  len(scoreReport.MissingPersonaIDs) == 0,
			Details: "all configured personas must respond before backlog injection",
		})
	}

	plan := BuildBacklogInjectionPlan(issueID, calibration, scoreReport, ranks, guardrails)
	checks = append(checks, PRLoopGuardrailCheck{
		ID:      "backlog-injection-availability",
		Passed:  len(plan.InjectedItems) > 0,
		Details: "at least one deterministic backlog item must be eligible",
	})

	ready := true
	for _, check := range checks {
		if !check.Passed {
			ready = false
			break
		}
	}

	return ContinuousImprovementPRLoopReport{
		ContractVersion: ContinuousImprovementPRLoopContractVersion,
		IssueID:         issueID,
		ReadyForPR:      ready,
		GuardrailChecks: checks,
		BacklogPlan:     plan,
	}, nil
}

func BuildBacklogInjectionPlan(issueID string, calibration TrialRunCalibrationReport, scoreReport SwarmScoreReport, ranks []OpportunityRank, guardrails PRLoopGuardrails) BacklogInjectionPlan {
	eligible := make([]string, 0, len(ranks))
	injected := make([]BacklogInjectionItem, 0, len(ranks))
	excluded := make([]BacklogInjectionExclusion, 0)

	sourceRecommendations := selectPersonaRecommendations(scoreReport.PriorityRecommendations, guardrails.MaxPersonaRecommendations)

	limit := guardrails.MaxInjectedItems
	if limit <= 0 {
		limit = len(ranks)
	}

	sorted := append([]OpportunityRank(nil), ranks...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].NormalizedScore == sorted[j].NormalizedScore {
			return sorted[i].OpportunityID < sorted[j].OpportunityID
		}
		return sorted[i].NormalizedScore > sorted[j].NormalizedScore
	})

	for _, rank := range sorted {
		reason := ""
		if rank.NormalizedScore < guardrails.MinOpportunityScore {
			reason = "score-below-minimum"
		} else if guardrails.RequireCompleteRubricScores && len(rank.MissingDimensions) > 0 {
			reason = "missing-rubric-dimensions"
		} else if guardrails.RequireCompleteRubricScores && len(rank.UnknownDimensions) > 0 {
			reason = "unknown-rubric-dimensions"
		}

		if reason != "" {
			excluded = append(excluded, BacklogInjectionExclusion{OpportunityID: rank.OpportunityID, Reason: reason})
			continue
		}

		eligible = append(eligible, rank.OpportunityID)
		if len(injected) >= limit {
			excluded = append(excluded, BacklogInjectionExclusion{OpportunityID: rank.OpportunityID, Reason: "max-injected-items-reached"})
			continue
		}

		injected = append(injected, BacklogInjectionItem{
			OpportunityID:           rank.OpportunityID,
			NormalizedScore:         rank.NormalizedScore,
			TargetIssueTitle:        "improvement: " + strings.ReplaceAll(rank.OpportunityID, "-", " "),
			TargetIssueType:         "task",
			TargetIssuePriority:     priorityFromScore(rank.NormalizedScore),
			TargetIssueLabels:       []string{"autonomy", "strict-evidence", "workstream:self-improvement-evaluator"},
			SourceRecommendations:   append([]string(nil), sourceRecommendations...),
			RequiredEvidenceSignals: []string{"trial-run-calibration", "rubric-ranked-opportunities", "swarm-score-report"},
		})
	}

	sort.Strings(eligible)
	sort.Slice(excluded, func(i, j int) bool {
		if excluded[i].OpportunityID == excluded[j].OpportunityID {
			return excluded[i].Reason < excluded[j].Reason
		}
		return excluded[i].OpportunityID < excluded[j].OpportunityID
	})

	return BacklogInjectionPlan{
		ContractVersion: BacklogInjectionPlanContractVersion,
		SourceIssueID:   issueID,
		SourceContractVersions: []string{
			calibration.ContractVersion,
			scoreReport.ContractVersion,
			OutcomeScoringRubricContractVersion,
		},
		EligibleOpportunityIDs: eligible,
		InjectedItems:          injected,
		ExcludedOpportunities:  excluded,
	}
}

func selectPersonaRecommendations(recommendations []string, limit int) []string {
	if limit <= 0 || len(recommendations) == 0 {
		return nil
	}
	if limit > len(recommendations) {
		limit = len(recommendations)
	}
	out := append([]string(nil), recommendations[:limit]...)
	return out
}

func priorityFromScore(score int) int {
	switch {
	case score >= 90:
		return 0
	case score >= 85:
		return 1
	case score >= 80:
		return 2
	default:
		return 3
	}
}
