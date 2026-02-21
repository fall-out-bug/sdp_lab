package adapter

import (
	"context"
	"testing"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestTaskReconciler_Reconcile_Pending(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			AgentRef: v1alpha1.AgentRef{Model: "glm-4.7"},
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhasePending},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1"}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(task).Build()

	lockMgr := NewRunLockManager(t.TempDir())
	policyGate := NewPolicyGate()
	beadsAdapter := beads.NewAdapter(t.TempDir())
	projector := NewEvidenceProjector(t.TempDir())
	lifecycleReconciler := NewLifecycleReconciler()

	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkDir:             t.TempDir(),
		LockManager:         lockMgr,
		PolicyGate:          policyGate,
		BeadsAdapter:        beadsAdapter,
		EvidenceProjector:   projector,
		LifecycleReconciler: lifecycleReconciler,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_NotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}
