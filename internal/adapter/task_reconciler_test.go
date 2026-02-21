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

func schemeAndClient(t *testing.T, objs ...client.Object) (*runtime.Scheme, client.Client) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
	return scheme, c
}

func TestTaskReconciler_Reconcile_Pending(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			AgentRef: v1alpha1.AgentRef{Model: "glm-4.7"},
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhasePending},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1"}

	scheme, fakeClient := schemeAndClient(t, task)

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
	scheme, fakeClient := schemeAndClient(t)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "nonexistent", Namespace: "default"}})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_NoBeadsIssue(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Status:     v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhasePending},
	}
	task.Labels = map[string]string{"sdp.run_id": "run-1"}

	scheme, fakeClient := schemeAndClient(t, task)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		LockManager: NewRunLockManager(t.TempDir()),
		PolicyGate:  NewPolicyGate(),
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_PolicyGateDeny(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec:      v1alpha1.TaskSpec{AgentRef: v1alpha1.AgentRef{Model: "denied-model"}},
		Status:    v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhasePending},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1"}

	scheme, fakeClient := schemeAndClient(t, task)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		LockManager: NewRunLockManager(t.TempDir()),
		PolicyGate:  NewPolicyGate(),
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_LockManagerDeny(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-2", Namespace: "default"},
		Status:     v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhasePending},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-2"}

	scheme, fakeClient := schemeAndClient(t, task)
	lockMgr := NewRunLockManager(t.TempDir())
	lockMgr.TryAcquire("i1", "run-1")

	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		LockManager: lockMgr,
		PolicyGate:  NewPolicyGate(),
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_Running(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Status:     v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhaseRunning},
	}
	task.Labels = map[string]string{"beads.issue": "i1"}

	scheme, fakeClient := schemeAndClient(t, task)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_Succeeded(t *testing.T) {
	dir := t.TempDir()
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			Prompt:    "test",
			Objective: "test obj",
			AgentRef:  v1alpha1.AgentRef{Model: "glm-4.7"},
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhaseSucceeded},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1"}

	scheme, fakeClient := schemeAndClient(t, task)
	projector := NewEvidenceProjector(dir)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkDir:           dir,
		EvidenceProjector: projector,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_Completed(t *testing.T) {
	dir := t.TempDir()
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			Prompt:    "test",
			Objective: "obj",
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhaseCompleted},
	}
	task.Labels = map[string]string{"beads.issue": "i1"}

	scheme, fakeClient := schemeAndClient(t, task)
	projector := NewEvidenceProjector(dir)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkDir:           dir,
		EvidenceProjector: projector,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestTaskReconciler_Reconcile_Failed(t *testing.T) {
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Status: v1alpha1.TaskStatus{
			Phase:          v1alpha1.TaskPhaseFailed,
			TerminalReason: &v1alpha1.TerminalReason{Code: "RetryExhausted"},
		},
	}
	task.Labels = map[string]string{"beads.issue": "i1"}

	scheme, fakeClient := schemeAndClient(t, task)
	lockMgr := NewRunLockManager(t.TempDir())
	lockMgr.TryAcquire("i1", "run-1")
	lifecycleReconciler := NewLifecycleReconciler()

	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkDir:             t.TempDir(),
		LockManager:         lockMgr,
		LifecycleReconciler: lifecycleReconciler,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
	if lockMgr.IsLocked("i1") {
		t.Error("lock should be released after Failed")
	}
}
