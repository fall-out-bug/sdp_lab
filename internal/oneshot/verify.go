package oneshot

import (
	"fmt"
	"sort"
	"strings"
)

type RoleEvidence struct {
	TaskID              string   `json:"task_id"`
	Role                string   `json:"role"`
	Status              string   `json:"status"`
	ArtifactIDs         []string `json:"artifact_ids"`
	ConsumedArtifactIDs []string `json:"consumed_artifact_ids"`
}

type VerificationReport struct {
	OK                     bool                `json:"ok"`
	MissingTaskEvidence    []string            `json:"missing_task_evidence,omitempty"`
	DuplicateTaskEvidence  []string            `json:"duplicate_task_evidence,omitempty"`
	UnknownEvidenceTasks   []string            `json:"unknown_evidence_tasks,omitempty"`
	RoleMismatches         []string            `json:"role_mismatches,omitempty"`
	InvalidStatuses        []string            `json:"invalid_statuses,omitempty"`
	ReviewerDependencyGaps map[string][]string `json:"reviewer_dependency_gaps,omitempty"`
}

func VerifyRoleEvidence(manifest ExecutionManifest, evidence []RoleEvidence) VerificationReport {
	allowedStatuses := map[string]bool{"ok": true, "needs_changes": true}
	tasks := make(map[string]ExecutionTask, len(manifest.Tasks))
	for _, task := range manifest.Tasks {
		tasks[task.ID] = task
	}

	report := VerificationReport{ReviewerDependencyGaps: map[string][]string{}}
	seenByTask := make(map[string]RoleEvidence, len(evidence))

	for _, item := range evidence {
		taskID := strings.TrimSpace(item.TaskID)
		if taskID == "" {
			report.UnknownEvidenceTasks = append(report.UnknownEvidenceTasks, "<empty>")
			continue
		}
		task, ok := tasks[taskID]
		if !ok {
			report.UnknownEvidenceTasks = append(report.UnknownEvidenceTasks, taskID)
			continue
		}
		if _, duplicate := seenByTask[taskID]; duplicate {
			report.DuplicateTaskEvidence = append(report.DuplicateTaskEvidence, taskID)
			continue
		}

		role := strings.TrimSpace(item.Role)
		if role != task.Role {
			report.RoleMismatches = append(report.RoleMismatches, fmt.Sprintf("%s expected %s got %s", taskID, task.Role, role))
		}

		status := strings.TrimSpace(item.Status)
		if !allowedStatuses[status] {
			report.InvalidStatuses = append(report.InvalidStatuses, fmt.Sprintf("%s=%s", taskID, status))
		}

		item.ArtifactIDs = uniqueSorted(item.ArtifactIDs)
		item.ConsumedArtifactIDs = uniqueSorted(item.ConsumedArtifactIDs)
		seenByTask[taskID] = item
	}

	for _, task := range manifest.Tasks {
		if _, ok := seenByTask[task.ID]; !ok {
			report.MissingTaskEvidence = append(report.MissingTaskEvidence, task.ID)
		}
	}

	for _, task := range manifest.Tasks {
		if task.Role != "reviewer" || len(task.DependsOn) == 0 {
			continue
		}
		review, ok := seenByTask[task.ID]
		if !ok {
			continue
		}
		missingDeps := make([]string, 0)
		consumedSet := toSet(review.ConsumedArtifactIDs)
		for _, depID := range task.DependsOn {
			depEvidence, found := seenByTask[depID]
			if !found {
				missingDeps = append(missingDeps, depID)
				continue
			}
			if !intersects(consumedSet, depEvidence.ArtifactIDs) {
				missingDeps = append(missingDeps, depID)
			}
		}
		if len(missingDeps) > 0 {
			sort.Strings(missingDeps)
			report.ReviewerDependencyGaps[task.ID] = missingDeps
		}
	}

	report.MissingTaskEvidence = uniqueSorted(report.MissingTaskEvidence)
	report.DuplicateTaskEvidence = uniqueSorted(report.DuplicateTaskEvidence)
	report.UnknownEvidenceTasks = uniqueSorted(report.UnknownEvidenceTasks)
	report.RoleMismatches = uniqueSorted(report.RoleMismatches)
	report.InvalidStatuses = uniqueSorted(report.InvalidStatuses)
	if len(report.ReviewerDependencyGaps) == 0 {
		report.ReviewerDependencyGaps = nil
	}

	report.OK = len(report.MissingTaskEvidence) == 0 &&
		len(report.DuplicateTaskEvidence) == 0 &&
		len(report.UnknownEvidenceTasks) == 0 &&
		len(report.RoleMismatches) == 0 &&
		len(report.InvalidStatuses) == 0 &&
		len(report.ReviewerDependencyGaps) == 0

	return report
}

type RecoveryPlan struct {
	RequeueTaskIDs []string `json:"requeue_task_ids"`
	Reason         string   `json:"reason"`
}

func PlanFailureRecovery(manifest ExecutionManifest, failedTaskIDs []string) (RecoveryPlan, error) {
	if len(failedTaskIDs) == 0 {
		return RecoveryPlan{}, fmt.Errorf("at least one failed task is required")
	}

	tasks := make(map[string]ExecutionTask, len(manifest.Tasks))
	dependents := make(map[string][]string)
	for _, task := range manifest.Tasks {
		tasks[task.ID] = task
		for _, dep := range task.DependsOn {
			dependents[dep] = append(dependents[dep], task.ID)
		}
	}

	queue := make([]string, 0, len(failedTaskIDs))
	selected := make(map[string]struct{})
	for _, id := range failedTaskIDs {
		taskID := strings.TrimSpace(id)
		if _, ok := tasks[taskID]; !ok {
			return RecoveryPlan{}, fmt.Errorf("unknown failed task: %s", taskID)
		}
		if _, exists := selected[taskID]; exists {
			continue
		}
		selected[taskID] = struct{}{}
		queue = append(queue, taskID)
	}

	for i := 0; i < len(queue); i++ {
		current := queue[i]
		for _, dep := range dependents[current] {
			if _, exists := selected[dep]; exists {
				continue
			}
			selected[dep] = struct{}{}
			queue = append(queue, dep)
		}
	}

	requeue := make([]string, 0, len(selected))
	for taskID := range selected {
		requeue = append(requeue, taskID)
	}
	sort.Strings(requeue)

	return RecoveryPlan{
		RequeueTaskIDs: requeue,
		Reason:         "requeue failed tasks and downstream dependents",
	}, nil
}

func uniqueSorted(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		seen[trimmed] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for item := range seen {
		out = append(out, item)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func toSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func intersects(consumed map[string]struct{}, provided []string) bool {
	for _, candidate := range provided {
		if _, ok := consumed[candidate]; ok {
			return true
		}
	}
	return false
}
