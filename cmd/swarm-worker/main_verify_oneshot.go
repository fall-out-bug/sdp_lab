package main

import (
	"encoding/json"
	"strings"

	"sdp_dev/internal/oneshot"
)

type oneShotVerificationResult struct {
	Report        oneshot.VerificationReport
	RecoveryPlan  *oneshot.RecoveryPlan
	FailedTaskIDs []string
	RoleEvidence  []oneshot.RoleEvidence
}

func evaluateOneShotVerification(changedFiles []string, testsPassed bool) (oneShotVerificationResult, error) {
	manifest, err := oneshot.BuildExecutionManifest(oneshot.PlannerGraph{Nodes: []oneshot.PlannerNode{
		{ID: "plan", Owner: "analyst", Artifacts: []string{"manifest:plan"}, ContractID: "handoff-plan"},
		{ID: "build", Owner: "coder", DependsOn: []string{"plan"}, Artifacts: []string{"diff:worker", "tests:go-test"}, ContractID: "handoff-build"},
		{ID: "review", Owner: "reviewer", DependsOn: []string{"build"}, Artifacts: []string{"verdict:review"}, ContractID: "handoff-review"},
	}})
	if err != nil {
		return oneShotVerificationResult{}, err
	}

	hasTestChange := false
	for _, path := range changedFiles {
		if strings.HasSuffix(path, "_test.go") {
			hasTestChange = true
			break
		}
	}

	buildStatus := "ok"
	reviewStatus := "ok"
	reviewerConsumed := []string{"diff:worker"}
	if !testsPassed || !hasTestChange {
		buildStatus = "needs_changes"
		reviewStatus = "needs_changes"
		reviewerConsumed = nil
	}

	evidence := []oneshot.RoleEvidence{
		{TaskID: "plan", Role: "analyst", Status: "ok", ArtifactIDs: []string{"manifest:plan"}},
		{TaskID: "build", Role: "coder", Status: buildStatus, ArtifactIDs: []string{"diff:worker", "tests:go-test"}},
		{TaskID: "review", Role: "reviewer", Status: reviewStatus, ArtifactIDs: []string{"verdict:review"}, ConsumedArtifactIDs: reviewerConsumed},
	}

	report := oneshot.VerifyRoleEvidence(manifest, evidence)
	failedTaskIDs := make([]string, 0)
	for _, item := range evidence {
		if item.Status != "ok" {
			failedTaskIDs = append(failedTaskIDs, item.TaskID)
		}
	}
	failedTaskIDs = uniqueStrings(failedTaskIDs)

	var recoveryPlan *oneshot.RecoveryPlan
	if len(failedTaskIDs) > 0 || !report.OK {
		seed := failedTaskIDs
		if len(seed) == 0 {
			if len(report.MissingTaskEvidence) > 0 {
				seed = append(seed, report.MissingTaskEvidence...)
			}
			for taskID := range report.ReviewerDependencyGaps {
				seed = append(seed, taskID)
			}
			seed = uniqueStrings(seed)
		}
		if len(seed) > 0 {
			plan, err := oneshot.PlanFailureRecovery(manifest, seed)
			if err != nil {
				return oneShotVerificationResult{}, err
			}
			recoveryPlan = &plan
		}
	}

	return oneShotVerificationResult{
		Report:        report,
		RecoveryPlan:  recoveryPlan,
		FailedTaskIDs: failedTaskIDs,
		RoleEvidence:  evidence,
	}, nil
}

func applyOneShotVerification(payload map[string]any, runPacket map[string]any, changedFiles []string, testsPassed bool) (string, error) {
	result, err := evaluateOneShotVerification(changedFiles, testsPassed)
	if err != nil {
		return "", err
	}

	verification, _ := payload["verification"].(map[string]any)
	if verification == nil {
		verification = map[string]any{}
		payload["verification"] = verification
	}
	verification["oneshot"] = map[string]any{
		"evidence_ok":     result.Report.OK,
		"failed_task_ids": result.FailedTaskIDs,
		"report":          result.Report,
		"role_evidence":   result.RoleEvidence,
	}
	if result.RecoveryPlan != nil {
		ones, _ := verification["oneshot"].(map[string]any)
		ones["recovery_plan"] = result.RecoveryPlan
	}

	if runPacket != nil {
		runPacket["oneshot_verification"] = map[string]any{
			"evidence_ok":       result.Report.OK,
			"failed_task_count": len(result.FailedTaskIDs),
		}
		if result.RecoveryPlan != nil {
			runPacket["oneshot_recovery"] = result.RecoveryPlan
		}
	}

	note := map[string]any{
		"kind":             "oneshot_verify",
		"evidence_ok":      result.Report.OK,
		"failed_task_ids":  result.FailedTaskIDs,
		"missing_evidence": result.Report.MissingTaskEvidence,
		"dependency_gaps":  result.Report.ReviewerDependencyGaps,
		"invalid_statuses": result.Report.InvalidStatuses,
	}
	if result.RecoveryPlan != nil {
		note["requeue_task_ids"] = result.RecoveryPlan.RequeueTaskIDs
	}
	b, err := json.Marshal(note)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
