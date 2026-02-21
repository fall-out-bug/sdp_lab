package adapter

import (
	"context"

	"sdp_dev/api/v1alpha1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TaskReconciler reconciles Task CRDs and drives the adapter pipeline.
// WS-001-02 implements full reconcile logic; this stub satisfies Manager registration.
type TaskReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// NewTaskReconciler returns a TaskReconciler.
func NewTaskReconciler(client client.Client, scheme *runtime.Scheme) *TaskReconciler {
	return &TaskReconciler{Client: client, Scheme: scheme}
}

// Reconcile handles Task reconciliation.
// Full implementation in WS-001-02: Pending→claim, Running→heartbeat, Succeeded→evidence, Failed→FSM.
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	// Stub: WS-001-02 implements full logic
	return ctrl.Result{}, nil
}

// SetupWithManager registers the reconciler with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Task{}).
		Complete(r)
}
