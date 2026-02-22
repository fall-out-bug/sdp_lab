package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/agent"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/evidence"
	"sdp_dev/internal/quality"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const heartbeatInterval = 60 * time.Second

// TaskReconciler reconciles Task CRDs and drives the adapter pipeline.
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	WorkspaceResolver   WorkspaceResolver
	BeadsAvailable      bool
	LockManager         *RunLockManager
	PolicyGate          *PolicyGate
	LifecycleReconciler *LifecycleReconciler
	TraceEmitter        *agent.TraceEmitter // nil if no bus; fallback for single-project only
	Bus                 bus.Bus           // nil if no NATS; publish terminal status
}

// NewTaskReconciler returns a TaskReconciler.
func NewTaskReconciler(c client.Client, scheme *runtime.Scheme, opts TaskReconcilerOpts) *TaskReconciler {
	return &TaskReconciler{
		Client:              c,
		Scheme:              scheme,
		WorkspaceResolver:   opts.WorkspaceResolver,
		BeadsAvailable:      opts.BeadsAvailable,
		LockManager:         opts.LockManager,
		PolicyGate:          opts.PolicyGate,
		LifecycleReconciler: opts.LifecycleReconciler,
		TraceEmitter:        opts.TraceEmitter,
		Bus:                 opts.Bus,
	}
}

// TaskReconcilerOpts holds optional dependencies for TaskReconciler.
type TaskReconcilerOpts struct {
	WorkspaceResolver   WorkspaceResolver
	BeadsAvailable      bool
	LockManager         *RunLockManager
	PolicyGate          *PolicyGate
	LifecycleReconciler *LifecycleReconciler
	TraceEmitter        *agent.TraceEmitter
	Bus                 bus.Bus
}

// Reconcile handles Task reconciliation.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx, span := otel.Tracer("adapter").Start(ctx, "TaskReconcile")
	defer span.End()
	span.SetAttributes(
		attribute.String("task", req.NamespacedName.String()),
	)

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

	projectID := ProjectIDFromLabels(task.Labels)
	workDir := "."
	if r.WorkspaceResolver != nil {
		workDir = r.WorkspaceResolver(projectID)
		if workDir == "" {
			workDir = "."
		}
	}

	phase := CRDPhase(task.Status.Phase)
	span.SetAttributes(attribute.String("phase", string(phase)), attribute.String("issue", issueID))
	log.V(1).Info("reconcile trace", "task", req.NamespacedName, "phase", phase, "issue", issueID, "workDir", workDir)
	runID := task.Name
	if rid := task.Labels["sdp.run_id"]; rid != "" {
		runID = rid
	}

	model := task.Spec.AgentRef.Model
	if model == "" {
		model = "glm-4.7"
	}

	var beadsAdapter *beads.Adapter
	if r.BeadsAvailable {
		beadsAdapter = beads.NewAdapter(workDir)
	}
	projector := NewEvidenceProjector(workDir)

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
		if beadsAdapter != nil {
			if err := beadsAdapter.Claim(issueID); err != nil {
				log.Error(err, "beads claim failed", "issue", issueID)
			}
		}

	case PhaseRunning:
		if r.TraceEmitter != nil {
			_ = r.TraceEmitter.EmitHeartbeatIfDue(heartbeatInterval)
		}

	case PhaseSucceeded, PhaseCompleted:
		intent := taskToIntent(task, runID)
		roleOutputs := map[string]string{"coder": "placeholder"}
		evPath, err := projector.ProjectFromIntent(intent, roleOutputs, runID)
		if err != nil {
			log.Error(err, "evidence projection failed", "issue", issueID)
			return ctrl.Result{}, nil
		}
		traceEvts := evidence.LoadTraceEventsFromRunFile(workDir, runID)
		if traceEvts == nil && r.TraceEmitter != nil && projectID == "" {
			evts := r.TraceEmitter.Events()
			traceEvts = make([]evidence.TraceEvent, len(evts))
			for i, e := range evts {
				traceEvts[i] = evidence.TraceEvent{At: e.At, Phase: e.Phase}
			}
		}
		if traceEvts != nil && len(traceEvts) > 0 {
			tvRes := evidence.ValidateTraceChain(traceEvts)
			if !tvRes.OK {
				for _, w := range tvRes.Warnings {
					log.Info("trace validation warning", "issue", issueID, "warning", w)
				}
			}
			if err := evidence.AddTraceValidationToEvidence(evPath, tvRes); err != nil {
				log.V(1).Info("could not add trace_validation to evidence", "issue", issueID, "err", err)
			}
		}
		res, err := evidence.ValidateStrictFile(evPath, false)
		if err != nil {
			log.Error(err, "evidence validation failed", "issue", issueID)
			return ctrl.Result{}, nil
		}
		if !res.OK {
			log.Info("evidence invalid, blocking FSM", "issue", issueID, "reason", res.Reason)
			return ctrl.Result{}, nil
		}
		if beadsAdapter != nil {
			if testsPassed, _ := quality.RunTests(workDir); !testsPassed {
				log.V(1).Info("quality.RunTests failed (may be no go mod)", "issue", issueID)
			} else if err := quality.RunPRGate(issueID, workDir); err != nil {
				log.V(1).Info("quality.RunPRGate failed (may be pr-gate not in PATH)", "issue", issueID, "err", err)
			}
		}
		if err := runBeadsFSM(workDir, issueID, "review"); err != nil {
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
			if beadsAdapter != nil {
				iss, _ := beadsAdapter.Show(issueID)
				if iss != nil {
					currentFSM = FSMState(iss.Status)
				}
			}
			targetFSM, _, err := r.LifecycleReconciler.ReconcilePhase(currentFSM, PhaseFailed, reason)
			if err == nil {
				_ = runBeadsFSM(workDir, issueID, string(targetFSM))
			}
		}
		if r.LockManager != nil {
			_ = r.LockManager.Release(issueID)
		}
	}

	if r.Bus != nil && (phase == PhaseSucceeded || phase == PhaseCompleted || phase == PhaseFailed) {
		publishTerminalStatus(r.Bus, projectID, issueID, string(phase))
	}

	return ctrl.Result{}, nil
}

func publishTerminalStatus(b bus.Bus, projectID, issueID, phase string) {
	if projectID == "" {
		projectID = "default"
	}
	subject := "sdp.status." + projectID + "." + issueID
	payload, _ := json.Marshal(map[string]string{"phase": phase, "issue_id": issueID})
	env := bus.Envelope{
		IssueID:       issueID,
		ArtifactID:    "status",
		ArtifactClass: "status",
		Phase:         phase,
		Payload:       json.RawMessage(payload),
		ProjectID:     projectID,
	}
	_ = b.Publish(subject, env)
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
	if !BeadsFSMAvailable() {
		return nil // no-op when beads-fsm not in PATH
	}
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
