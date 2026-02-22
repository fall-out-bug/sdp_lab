package adapter

import (
	"context"
	"strings"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// createTaskFromIntent builds a Task from an AgentRun and TaskIntent.
func createTaskFromIntent(run *v1alpha1.AgentRun, role string, intent *TaskIntent) *v1alpha1.Task {
	name := run.Name + "-" + role
	labels := map[string]string{
		"beads.issue": intent.IssueID,
		"sdp.run_id":  intent.RunID,
		"agentrun":    run.Name,
		"role":        role,
	}
	for k, v := range intent.Labels {
		labels[k] = v
	}
	labels["role"] = role

	return &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: run.Namespace,
			Labels:    labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: run.APIVersion,
					Kind:       run.Kind,
					Name:       run.Name,
					UID:        run.UID,
				},
			},
		},
		Spec: v1alpha1.TaskSpec{
			Prompt:    intent.Prompt,
			Objective: intent.Objective,
			AgentRef:  v1alpha1.AgentRef{Model: run.Spec.Model},
		},
		Status: v1alpha1.TaskStatus{
			Phase: v1alpha1.TaskPhasePending,
		},
	}
}

// workerTasksTerminal checks if all worker Tasks are in a terminal phase.
// Returns (allTerminal, anyFailed).
func workerTasksTerminal(ctx context.Context, r client.Reader, run *v1alpha1.AgentRun) (allTerminal, anyFailed bool) {
	workerNames := strings.Split(run.Status.WorkerTask, ",")
	if len(workerNames) < 2 {
		return false, false
	}
	for _, name := range workerNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		task := &v1alpha1.Task{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: name}, task); err != nil {
			return false, false
		}
		p := task.Status.Phase
		if p != v1alpha1.TaskPhaseSucceeded && p != v1alpha1.TaskPhaseCompleted && p != v1alpha1.TaskPhaseFailed {
			return false, false
		}
		if p == v1alpha1.TaskPhaseFailed {
			anyFailed = true
		}
	}
	return true, anyFailed
}
