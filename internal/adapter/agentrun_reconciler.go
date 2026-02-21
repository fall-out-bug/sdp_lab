package adapter

import (
	"context"
	"fmt"
	"strings"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AgentRunReconciler reconciles AgentRun CRDs and creates worker Tasks.
type AgentRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	IntentTranslator *IntentTranslator
	PolicyGate       *PolicyGate
	BeadsAdapter     *beads.Adapter
}

// NewAgentRunReconciler returns an AgentRunReconciler.
func NewAgentRunReconciler(c client.Client, scheme *runtime.Scheme, opts AgentRunReconcilerOpts) *AgentRunReconciler {
	return &AgentRunReconciler{
		Client:           c,
		Scheme:           scheme,
		IntentTranslator: opts.IntentTranslator,
		PolicyGate:       opts.PolicyGate,
		BeadsAdapter:     opts.BeadsAdapter,
	}
}

// AgentRunReconcilerOpts holds optional dependencies.
type AgentRunReconcilerOpts struct {
	IntentTranslator *IntentTranslator
	PolicyGate       *PolicyGate
	BeadsAdapter     *beads.Adapter
}

// Reconcile handles AgentRun reconciliation.
func (r *AgentRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	run := &v1alpha1.AgentRun{}
	if err := r.Get(ctx, req.NamespacedName, run); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	phase := run.Status.Phase
	model := run.Spec.Model
	if model == "" {
		model = "glm-4.7"
	}

	switch phase {
	case "":
		if r.PolicyGate != nil {
			gr := r.PolicyGate.PreDispatchModelAllowlist(model)
			if !gr.Passed {
				return r.setFailed(ctx, run, gr.Reason)
			}
		}

		issue := &beads.Issue{ID: run.Spec.IssueID, Title: "AgentRun " + run.Name, AcceptanceCriteria: "Implement task"}
		if r.BeadsAdapter != nil {
			if iss, err := r.BeadsAdapter.Show(run.Spec.IssueID); err == nil {
				issue = iss
			}
		}

		intent, err := r.IntentTranslator.Translate(issue, run.Name)
		if err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("translate: %v", err))
		}

		analystTask := r.createTaskFromIntent(run, "analyst", intent)
		coderTask := r.createTaskFromIntent(run, "coder", intent)

		if err := r.Create(ctx, analystTask); err != nil {
			return ctrl.Result{}, fmt.Errorf("create analyst task: %w", err)
		}
		if err := r.Create(ctx, coderTask); err != nil {
			return ctrl.Result{}, fmt.Errorf("create coder task: %w", err)
		}

		run.Status.Phase = "Running"
		run.Status.WorkerTask = analystTask.Name + "," + coderTask.Name
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created worker tasks", "analyst", analystTask.Name, "coder", coderTask.Name)

	case "Running":
		workerNames := strings.Split(run.Status.WorkerTask, ",")
		if len(workerNames) < 2 {
			return ctrl.Result{}, nil
		}

		allTerminal := true
		anyFailed := false
		for _, name := range workerNames {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			task := &v1alpha1.Task{}
			if err := r.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: name}, task); err != nil {
				allTerminal = false
				break
			}
			p := task.Status.Phase
			if p != v1alpha1.TaskPhaseSucceeded && p != v1alpha1.TaskPhaseCompleted && p != v1alpha1.TaskPhaseFailed {
				allTerminal = false
				break
			}
			if p == v1alpha1.TaskPhaseFailed {
				anyFailed = true
			}
		}

		if allTerminal {
			if anyFailed {
				return r.setFailed(ctx, run, "worker task failed")
			}
			run.Status.Phase = "ReviewerPending"
			if err := r.Status().Update(ctx, run); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("workers complete, reviewer pending")
		}

	case "ReviewerPending":
		// TODO WS-002-01 full: create reviewer Task with aggregated context
		// For minimal: transition to ReviewerRunning with placeholder, or Succeeded
		run.Status.Phase = "Succeeded"
		run.Status.ReviewerTask = ""
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("reviewer skipped (minimal), run succeeded")

	case "ReviewerRunning":
		// Check reviewer Task status
		reviewerName := strings.TrimSpace(run.Status.ReviewerTask)
		if reviewerName == "" {
			return ctrl.Result{}, nil
		}
		task := &v1alpha1.Task{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: reviewerName}, task); err != nil {
			return ctrl.Result{}, nil
		}
		p := task.Status.Phase
		if p == v1alpha1.TaskPhaseSucceeded || p == v1alpha1.TaskPhaseCompleted {
			run.Status.Phase = "Succeeded"
		} else if p == v1alpha1.TaskPhaseFailed {
			return r.setFailed(ctx, run, "reviewer task failed")
		} else {
			return ctrl.Result{}, nil
		}
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *AgentRunReconciler) setFailed(ctx context.Context, run *v1alpha1.AgentRun, reason string) (ctrl.Result, error) {
	run.Status.Phase = "Failed"
	run.Status.LastError = reason
	if err := r.Status().Update(ctx, run); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *AgentRunReconciler) createTaskFromIntent(run *v1alpha1.AgentRun, role string, intent *TaskIntent) *v1alpha1.Task {
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

// SetupWithManager registers the reconciler with the Manager.
func (r *AgentRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AgentRun{}).
		Owns(&v1alpha1.Task{}).
		Complete(r)
}
