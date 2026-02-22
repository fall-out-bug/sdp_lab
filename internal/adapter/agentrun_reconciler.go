package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/policy"
	"sdp_dev/internal/observability"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AgentRunReconciler reconciles AgentRun CRDs and creates worker Tasks.
type AgentRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	IntentTranslator      *IntentTranslator
	PolicyGate            *PolicyGate
	WorkspaceResolver     WorkspaceResolver
	BeadsAvailable        bool
	ProviderHealthChecker policy.ProviderHealthChecker
	Bus                   bus.Bus
}

// NewAgentRunReconciler returns an AgentRunReconciler.
func NewAgentRunReconciler(c client.Client, scheme *runtime.Scheme, opts AgentRunReconcilerOpts) *AgentRunReconciler {
	return &AgentRunReconciler{
		Client:                c,
		Scheme:                scheme,
		IntentTranslator:      opts.IntentTranslator,
		PolicyGate:            opts.PolicyGate,
		WorkspaceResolver:     opts.WorkspaceResolver,
		BeadsAvailable:        opts.BeadsAvailable,
		ProviderHealthChecker: opts.ProviderHealthChecker,
		Bus:                   opts.Bus,
	}
}

// AgentRunReconcilerOpts holds optional dependencies.
type AgentRunReconcilerOpts struct {
	IntentTranslator       *IntentTranslator
	PolicyGate             *PolicyGate
	WorkspaceResolver      WorkspaceResolver
	BeadsAvailable         bool
	ProviderHealthChecker  policy.ProviderHealthChecker
	Bus                    bus.Bus
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

	projectID := ProjectIDFromLabels(run.Labels)
	workDir := "."
	if r.WorkspaceResolver != nil {
		workDir = r.WorkspaceResolver(projectID)
		if workDir == "" {
			workDir = "."
		}
	}
	var beadsAdapter *beads.Adapter
	if r.BeadsAvailable {
		beadsAdapter = beads.NewAdapter(workDir)
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
		if beadsAdapter != nil {
			if iss, err := beadsAdapter.Show(run.Spec.IssueID); err == nil {
				issue = iss
			}
		}

		intent, err := r.IntentTranslator.Translate(issue, run.Name)
		if err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("translate: %v", err))
		}

		providerHint := policy.ResolveProviderForModel(run.Spec.Model, r.ProviderHealthChecker)
		analystTask := createTaskFromIntent(run, "analyst", intent, providerHint)
		coderTask := createTaskFromIntent(run, "coder", intent, providerHint)

		observability.IncProviderUsed(projectID, providerHint, model)
		if providerHint == "anthropic_direct" || providerHint == "openai_direct" {
			observability.AddCostSaved(projectID, providerHint, 0.001)
		}

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
		allTerminal, anyFailed := workerTasksTerminal(ctx, r.Client, run)
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
		r.recordAgentRunComplete(run, "Succeeded")
		r.closeBeadsOnSuccess(ctx, run.Spec.IssueID, "AgentRun completed", beadsAdapter)
		r.publishLifecycle("sdp.lifecycle.agentrun.completed", run.Spec.IssueID, projectID, "completed")
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
			r.recordAgentRunComplete(run, "Succeeded")
			r.closeBeadsOnSuccess(ctx, run.Spec.IssueID, "AgentRun completed", beadsAdapter)
			r.publishLifecycle("sdp.lifecycle.agentrun.completed", run.Spec.IssueID, projectID, "completed")
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
	r.recordAgentRunComplete(run, "Failed")
	r.publishLifecycle("sdp.lifecycle.agentrun.failed", run.Spec.IssueID, ProjectIDFromLabels(run.Labels), "failed")
	return ctrl.Result{}, nil
}

func (r *AgentRunReconciler) recordAgentRunComplete(run *v1alpha1.AgentRun, status string) {
	proj := ProjectIDFromLabels(run.Labels)
	model := run.Spec.Model
	if model == "" {
		model = "glm-4.7"
	}
	observability.IncAgentRuns(proj, status, model)
	if !run.CreationTimestamp.IsZero() {
		d := time.Since(run.CreationTimestamp.Time)
		observability.ObserveAgentRunDuration(proj, "agentrun", d)
	}
}

// closeBeadsOnSuccess closes the beads issue when AgentRun succeeds.
func (r *AgentRunReconciler) closeBeadsOnSuccess(ctx context.Context, issueID, reason string, beadsAdapter *beads.Adapter) {
	if beadsAdapter == nil || issueID == "" {
		return
	}
	if err := beadsAdapter.Close(issueID, reason); err != nil {
		log.FromContext(ctx).Info("beads close failed (may be already closed)", "issue", issueID, "err", err)
	}
}
func (r *AgentRunReconciler) publishLifecycle(subject, issueID, projectID, phase string) {
	if r.Bus == nil {
		return
	}
	pl, _ := json.Marshal(map[string]string{"issue_id": issueID, "phase": phase})
	_ = r.Bus.Publish(subject, bus.Envelope{
		IssueID: issueID, ProjectID: projectID, Phase: phase, Payload: pl,
	})
}

// SetupWithManager registers the reconciler with the Manager.
func (r *AgentRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.AgentRun{}).
		Owns(&v1alpha1.Task{}).
		Complete(r)
}
