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
	"sdp_dev/internal/safeid"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// AgentRunReconciler reconciles AgentRun CRDs and creates worker Tasks.
// Sequential pipeline: analyst → coder → reviewer (F004).
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
// Sequential phases: "" → Analyzing → AnalystComplete → Coding → CoderComplete → Reviewing → ReviewerComplete → Succeeded/Failed
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
		// Phase "" → create only analyst Task (sequential pipeline)
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
		analystTask := createTaskFromIntent(run, "analyst", intent, providerHint, nil)

		observability.IncProviderUsed(projectID, providerHint, model)
		if providerHint == "anthropic_direct" || providerHint == "openai_direct" {
			observability.AddCostSaved(projectID, providerHint, 0.001)
		}

		if err := r.Create(ctx, analystTask); err != nil {
			return ctrl.Result{}, fmt.Errorf("create analyst task: %w", err)
		}

		run.Status.Phase = "Analyzing"
		run.Status.WorkerTask = analystTask.Name
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created analyst task", "analyst", analystTask.Name)

	case "Analyzing":
		// Wait for analyst to complete
		allTerminal, anyFailed := workerTasksTerminal(ctx, r.Client, run)
		if !allTerminal {
			return ctrl.Result{}, nil
		}
		if anyFailed {
			return r.setFailed(ctx, run, "analyst task failed")
		}
		run.Status.Phase = "AnalystComplete"
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("analyst complete, transitioning to coder")

	case "AnalystComplete":
		// Create coder Task (handoff path injection in 00-004-02)
		if err := safeid.ValidateIssueID(run.Spec.IssueID); err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("invalid issue_id: %v", err))
		}
		issue := &beads.Issue{ID: run.Spec.IssueID, Title: "AgentRun " + run.Name, AcceptanceCriteria: "Implement task"}
		if beadsAdapter != nil {
			if iss, err := beadsAdapter.Show(run.Spec.IssueID); err == nil {
				issue = iss
			}
		}
		intent, err := r.IntentTranslator.Translate(issue, run.Name)
		if err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("translate coder: %v", err))
		}
		// Inject handoff path and instruction for analyst artifact (00-004-02)
		analystPath := HandoffPath(run.Spec.IssueID, "analyst")
		intent.Prompt = intent.Prompt + "\n\nRead the analyst handoff at " + analystPath + " and incorporate its recommendations."
		providerHint := policy.ResolveProviderForModel(model, r.ProviderHealthChecker)
		handoffAnnots := map[string]string{"sdp.dev/handoff-analyst": analystPath}
		coderTask := createTaskFromIntent(run, "coder", intent, providerHint, handoffAnnots, run.Status.WorkerTask)
		if err := r.Create(ctx, coderTask); err != nil {
			return ctrl.Result{}, fmt.Errorf("create coder task: %w", err)
		}
		run.Status.Phase = "Coding"
		run.Status.WorkerTask = coderTask.Name
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created coder task", "coder", coderTask.Name)

	case "Coding":
		// Wait for coder to complete
		allTerminal, anyFailed := workerTasksTerminal(ctx, r.Client, run)
		if !allTerminal {
			return ctrl.Result{}, nil
		}
		if anyFailed {
			return r.setFailed(ctx, run, "coder task failed")
		}
		run.Status.Phase = "CoderComplete"
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("coder complete, transitioning to reviewer")

	case "CoderComplete":
		// Create reviewer Task with dependsOn [analyst, coder]
		if err := safeid.ValidateIssueID(run.Spec.IssueID); err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("invalid issue_id: %v", err))
		}
		issue := &beads.Issue{ID: run.Spec.IssueID, Title: "AgentRun " + run.Name, AcceptanceCriteria: "Review analyst and coder outputs"}
		if beadsAdapter != nil {
			if iss, err := beadsAdapter.Show(run.Spec.IssueID); err == nil {
				issue = iss
			}
		}
		intent, err := r.IntentTranslator.Translate(issue, run.Name)
		if err != nil {
			return r.setFailed(ctx, run, fmt.Sprintf("translate reviewer: %v", err))
		}
		// Inject handoff paths and instruction for both artifacts (00-004-02)
		analystPath := HandoffPath(run.Spec.IssueID, "analyst")
		coderPath := HandoffPath(run.Spec.IssueID, "coder")
		providerHint := policy.ResolveProviderForModel(model, r.ProviderHealthChecker)
		analystName := run.Name + "-analyst"
		coderName := run.Name + "-coder"
		reviewerIntent := *intent
		reviewerIntent.Prompt = "Review analyst and coder outputs. Read the handoff files at " + analystPath + " and " + coderPath + " and incorporate their findings. " + intent.Prompt
		handoffAnnots := map[string]string{"sdp.dev/handoff-analyst": analystPath, "sdp.dev/handoff-coder": coderPath}
		reviewerTask := createTaskFromIntent(run, "reviewer", &reviewerIntent, providerHint, handoffAnnots, analystName, coderName)
		if err := r.Create(ctx, reviewerTask); err != nil {
			return ctrl.Result{}, fmt.Errorf("create reviewer task: %w", err)
		}
		run.Status.Phase = "Reviewing"
		run.Status.ReviewerTask = reviewerTask.Name
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("created reviewer task", "reviewer", reviewerTask.Name)

	case "Reviewing":
		// Wait for reviewer to complete
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
			run.Status.Phase = "ReviewerComplete"
			if err := r.Status().Update(ctx, run); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("reviewer complete, transitioning to final")
		} else if p == v1alpha1.TaskPhaseFailed {
			return r.setFailed(ctx, run, "reviewer task failed")
		} else {
			return ctrl.Result{}, nil
		}

	case "ReviewerComplete":
		// Transition to Succeeded or Failed based on reviewer verdict
		reviewerName := strings.TrimSpace(run.Status.ReviewerTask)
		if reviewerName == "" {
			return ctrl.Result{}, nil
		}
		task := &v1alpha1.Task{}
		if err := r.Get(ctx, client.ObjectKey{Namespace: run.Namespace, Name: reviewerName}, task); err != nil {
			return ctrl.Result{}, nil
		}
		verdict := reviewerVerdict(task)
		if verdict == "approve" {
			run.Status.Phase = "Succeeded"
			r.recordAgentRunComplete(run, "Succeeded")
			r.closeBeadsOnSuccess(ctx, run.Spec.IssueID, "AgentRun completed", beadsAdapter)
			r.publishLifecycle("sdp.lifecycle.agentrun.completed", run.Spec.IssueID, projectID, "completed")
		} else {
			run.Status.Phase = "Failed"
			run.Status.LastError = "reviewer verdict: " + verdict
			r.recordAgentRunComplete(run, "Failed")
			r.publishLifecycle("sdp.lifecycle.agentrun.failed", run.Spec.IssueID, projectID, "failed")
		}
		if err := r.Status().Update(ctx, run); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// reviewerVerdict extracts approve/needs_changes from reviewer Task.
// Checks annotations and output for verdict.
func reviewerVerdict(task *v1alpha1.Task) string {
	if task.Annotations != nil {
		if v := task.Annotations["sdp.dev/reviewer-verdict"]; v != "" {
			return v
		}
	}
	// Fallback: Succeeded/Completed = approve, Failed = needs_changes
	if task.Status.Phase == v1alpha1.TaskPhaseSucceeded || task.Status.Phase == v1alpha1.TaskPhaseCompleted {
		return "approve"
	}
	if task.Status.Phase == v1alpha1.TaskPhaseFailed {
		return "needs_changes"
	}
	return "unknown"
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
