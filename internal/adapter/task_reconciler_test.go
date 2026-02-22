package adapter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/evidence"

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
	baseDir := t.TempDir()
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
	lifecycleReconciler := NewLifecycleReconciler()

	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver:   NewWorkspaceResolver(baseDir),
		BeadsAvailable:     true,
		LockManager:        lockMgr,
		PolicyGate:         policyGate,
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
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
	})

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
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		LockManager:       NewRunLockManager(t.TempDir()),
		PolicyGate:        NewPolicyGate(),
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
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		LockManager:       NewRunLockManager(t.TempDir()),
		PolicyGate:        NewPolicyGate(),
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
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		LockManager:       lockMgr,
		PolicyGate:        NewPolicyGate(),
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
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
	})

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
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(dir),
		BeadsAvailable:     false,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	// Empty projectID -> default workspace: dir/default
	evPath := filepath.Join(dir, "default", ".sdp", "evidence", "i1.json")
	res, err := evidence.ValidateStrictFile(evPath, false)
	if err != nil {
		t.Fatalf("evidence file missing or unreadable: %v", err)
	}
	if !res.OK {
		t.Errorf("evidence invalid: %s", res.Reason)
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
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(dir),
		BeadsAvailable:     false,
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
		WorkspaceResolver:   NewWorkspaceResolver(t.TempDir()),
		LockManager:        lockMgr,
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

func TestTaskReconciler_Reconcile_PerProjectWorkspace(t *testing.T) {
	baseDir := t.TempDir()
	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			Prompt:    "test",
			Objective: "obj",
			AgentRef:  v1alpha1.AgentRef{Model: "glm-4.7"},
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhaseSucceeded},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1", LabelProject: "proj-a"}

	scheme, fakeClient := schemeAndClient(t, task)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(baseDir),
		BeadsAvailable:     false,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	// Per-project: evidence in baseDir/proj-a/.sdp/evidence/
	evPath := filepath.Join(baseDir, "proj-a", ".sdp", "evidence", "i1.json")
	res, err := evidence.ValidateStrictFile(evPath, false)
	if err != nil {
		t.Fatalf("evidence file missing: %v", err)
	}
	if !res.OK {
		t.Errorf("evidence invalid: %s", res.Reason)
	}
}

// TestTaskReconciler_Reconcile_Succeeded_WithTraceValidation is an integration test
// for FR-017: trace validation in adapter reconcile. When a run file exists with
// trace events, Reconcile(PhaseSucceeded) loads them, runs ValidateTraceChain,
// and adds trace_validation to the evidence file.
func TestTaskReconciler_Reconcile_Succeeded_WithTraceValidation(t *testing.T) {
	baseDir := t.TempDir()
	// Default project workspace: baseDir/default
	workDir := filepath.Join(baseDir, "default")
	runsDir := filepath.Join(workDir, ".sdp", "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	runFile := filepath.Join(runsDir, "run-1.json")
	runPayload := map[string]any{
		"events": []map[string]string{
			{"at": "2025-01-01T10:00:00Z", "phase": "execute"},
			{"at": "2025-01-01T10:01:00Z", "phase": "verify"},
			{"at": "2025-01-01T10:02:00Z", "phase": "review"},
		},
	}
	b, _ := json.MarshalIndent(runPayload, "", "  ")
	if err := os.WriteFile(runFile, b, 0o644); err != nil {
		t.Fatal(err)
	}

	task := &v1alpha1.Task{
		ObjectMeta: metav1.ObjectMeta{Name: "task-1", Namespace: "default"},
		Spec: v1alpha1.TaskSpec{
			Prompt:    "trace test",
			Objective: "obj",
			AgentRef:  v1alpha1.AgentRef{Model: "glm-4.7"},
		},
		Status: v1alpha1.TaskStatus{Phase: v1alpha1.TaskPhaseSucceeded},
	}
	task.Labels = map[string]string{"beads.issue": "i1", "sdp.run_id": "run-1"}

	scheme, fakeClient := schemeAndClient(t, task)
	r := NewTaskReconciler(fakeClient, scheme, TaskReconcilerOpts{
		WorkspaceResolver: NewWorkspaceResolver(baseDir),
		BeadsAvailable:     false,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(task)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	evPath := filepath.Join(workDir, ".sdp", "evidence", "i1.json")
	res, err := evidence.ValidateStrictFile(evPath, false)
	if err != nil {
		t.Fatalf("evidence file missing: %v", err)
	}
	if !res.OK {
		t.Errorf("evidence invalid: %s", res.Reason)
	}

	// Assert trace_validation was added by reconcile
	data, err := os.ReadFile(evPath)
	if err != nil {
		t.Fatalf("read evidence: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse evidence: %v", err)
	}
	tv, ok := doc["trace_validation"].(map[string]any)
	if !ok {
		t.Fatal("evidence missing trace_validation section")
	}
	if v, _ := tv["ok"].(bool); !v {
		t.Errorf("trace_validation.ok = false, want true; missing=%v", tv["missing"])
	}
}
