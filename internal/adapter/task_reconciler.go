package adapter

import (
	"context"
	"fmt"
	"os/exec"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/agent"
	"sdp_dev/internal/beads"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TaskReconciler reconciles Task CRDs and drives the adapter pipeline.
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	WorkDir         string
	LockManager     *RunLockManager
	PolicyGate      *PolicyGate
	BeadsAdapter    *beads.Adapter
	EvidenceProjector *EvidenceProjector
	LifecycleReconciler *LifecycleReconciler
	TraceEmitter    *agent.TraceEmitter // nil if no bus
}

// NewTaskReconciler returns a TaskReconciler.
func NewTaskReconciler(c client.Client, scheme *runtime.Scheme, opts TaskReconcilerOpts) *TaskReconciler {
	return &TaskReconciler{
		Client:              c,
		Scheme:              scheme,
		WorkDir:             opts.WorkDir,
		LockManager:         opts.LockManager,
		PolicyGate:          opts.PolicyGate,
		BeadsAdapter:        opts.BeadsAdapter,
		EvidenceProjector:   opts.EvidenceProjector,
		LifecycleReconciler: opts.LifecycleReconciler,
		TraceEmitter:        opts.TraceEmitter,
	}
}

// TaskReconcilerOpts holds optional dependencies for TaskReconciler.
type TaskReconcilerOpts struct {
	WorkDir             string
	LockManager         *RunLockManager
	PolicyGate          *PolicyGate
	BeadsAdapter        *beads.Adapter
	EvidenceProjector   *EvidenceProjector
	LifecycleReconciler *LifecycleReconciler
	TraceEmitter        *agent.TraceEmitter
}

// Reconcile handles Task reconciliation.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	task := &v1alpha1.Task{}
	if err := r.Get(ctx, req.NamespacedName, task); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	issueID := task.Labels["beads.issue"]
	if issueID == "" {
		return ctrl.Result{}, nil
	}

	phase := CRDPhase(task.Status.Phase)
	runID := task.Name
	if rid := task.Labels["sdp.run_id"]; rid != "" {
		runID = rid
	}

	model := task.Spec.AgentRef.Model
	if model == "" {
		model = "glm-4.7"
	}

	switch phase {
	case PhasePending:
		if r.LockManager != nil {
			_, acquired, err := r.LockManager.TryAcquire(issueID, runID)
			if err != nil || !acquired {
				return ctrl.Result{}, nil
			}
		}
		if r.PolicyGate != nil {
			gr := r.PolicyGate.PreDispatchModelAllowlist(model)
			if !gr.Passed {
				// TODO: set Task annotation "sdp.deny" = gr.Reason
				return ctrl.Result{}, nil
			}
		}
		if r.BeadsAdapter != nil {
			if err := r.BeadsAdapter.Claim(issueID); err != nil {
				log.Error(err, "beads claim failed", "issue", issueID)
			}
		}

	case PhaseRunning:
		if r.TraceEmitter != nil {
			_ = r.TraceEmitter.EmitPhase("heartbeat", "ok", "task "+task.Name)
		}

	case PhaseSucceeded, PhaseCompleted:
		if r.EvidenceProjector != nil {
			intent := taskToIntent(task, runID)
			roleOutputs := map[string]string{"coder": "placeholder"}
			if _, err := r.EvidenceProjector.ProjectFromIntent(intent, roleOutputs, runID); err != nil {
				log.Error(err, "evidence projection failed", "issue", issueID)
			}
		}
		if err := runBeadsFSM(r.WorkDir, issueID, "review"); err != nil {
			log.Error(err, "beads-fsm review failed", "issue", issueID)
		}

	case PhaseFailed:
		if r.LifecycleReconciler != nil {
			reason := ""
			if task.Status.TerminalReason != nil {
				reason = task.Status.TerminalReason.Code
				if reason == "" {
					reason = task.Status.TerminalReason.Message
				}
			}
			currentFSM := FSMInProgress
			if r.BeadsAdapter != nil {
				iss, _ := r.BeadsAdapter.Show(issueID)
				if iss != nil {
					currentFSM = FSMState(iss.Status)
				}
			}
			targetFSM, _, err := r.LifecycleReconciler.ReconcilePhase(currentFSM, PhaseFailed, reason)
			if err == nil {
				_ = runBeadsFSM(r.WorkDir, issueID, string(targetFSM))
			}
		}
		if r.LockManager != nil {
			_ = r.LockManager.Release(issueID)
		}
	}

	return ctrl.Result{}, nil
}

func taskToIntent(task *v1alpha1.Task, runID string) *TaskIntent {
	labels := make(map[string]string)
	for k, v := range task.Labels {
		labels[k] = v
	}
	return &TaskIntent{
		RunID:     runID,
		IssueID:   task.Labels["beads.issue"],
		Prompt:    task.Spec.Prompt,
		Objective: task.Spec.Objective,
		AgentRef:  task.Spec.AgentRef.Model,
		Labels:    labels,
		SpecHash:  "",
	}
}

func runBeadsFSM(workDir, issueID, target string) error {
	if workDir == "" {
		workDir = "."
	}
	cmd := exec.Command("beads-fsm", "--issue", issueID, "--to", target, "--apply")
	cmd.Dir = workDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("beads-fsm: %w: %s", err, string(out))
	}
	return nil
}

// SetupWithManager registers the reconciler with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Task{}).
		Complete(r)
}
