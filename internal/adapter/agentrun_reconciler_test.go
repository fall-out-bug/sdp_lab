package adapter

import (
	"context"
	"testing"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAgentRunReconciler_Reconcile_EmptyPhase_CreatesTasks(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	run := &v1alpha1.AgentRun{
		TypeMeta:   metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: v1alpha1.AgentRunSpec{
			IssueID: "i1",
			Model:   "glm-4.7",
		},
		Status: v1alpha1.AgentRunStatus{},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()

	r := NewAgentRunReconciler(fakeClient, scheme, AgentRunReconcilerOpts{
		IntentTranslator:  NewIntentTranslator(),
		PolicyGate:        NewPolicyGate(),
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		BeadsAvailable:    true,
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	// Verify Tasks created
	var taskList v1alpha1.TaskList
	if err := fakeClient.List(context.Background(), &taskList); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(taskList.Items) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(taskList.Items))
	}

	// Verify AgentRun status updated
	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status.Phase != "Running" {
		t.Errorf("expected Phase=Running, got %q", run.Status.Phase)
	}
	if run.Status.WorkerTask == "" {
		t.Error("expected WorkerTask set")
	}
}

func TestAgentRunReconciler_Reconcile_PropagatesProjectLabel(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	run := &v1alpha1.AgentRun{
		TypeMeta: metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "run-1",
			Namespace: "default",
			Labels:    map[string]string{LabelProject: "proj-a"},
		},
		Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", Model: "glm-4.7"},
		Status: v1alpha1.AgentRunStatus{},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()

	r := NewAgentRunReconciler(fakeClient, scheme, AgentRunReconcilerOpts{
		IntentTranslator:  NewIntentTranslator(),
		PolicyGate:        NewPolicyGate(),
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		BeadsAvailable:    false,
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var taskList v1alpha1.TaskList
	if err := fakeClient.List(context.Background(), &taskList); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	for _, task := range taskList.Items {
		if got := task.Labels[LabelProject]; got != "proj-a" {
			t.Errorf("task %s: sdp.project = %q, want proj-a", task.Name, got)
		}
	}
}

func TestAgentRunReconciler_Reconcile_PolicyGateDeny(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	run := &v1alpha1.AgentRun{
		TypeMeta:   metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: v1alpha1.AgentRunSpec{
			IssueID: "i1",
			Model:   "denied-model",
		},
		Status: v1alpha1.AgentRunStatus{},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()

	r := NewAgentRunReconciler(fakeClient, scheme, AgentRunReconcilerOpts{
		IntentTranslator:  NewIntentTranslator(),
		PolicyGate:        NewPolicyGate(),
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
	})

	result, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status.Phase != "Failed" {
		t.Errorf("expected Phase=Failed, got %q", run.Status.Phase)
	}
}

func TestAgentRunReconciler_Reconcile_ReviewerPending_CreatesReviewerWithDependsOn(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	run := &v1alpha1.AgentRun{
		TypeMeta:   metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: v1alpha1.AgentRunSpec{
			IssueID: "i1",
			Model:   "glm-4.7",
		},
		Status: v1alpha1.AgentRunStatus{
			Phase:      "ReviewerPending",
			WorkerTask: "run-1-analyst,run-1-coder",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()

	r := NewAgentRunReconciler(fakeClient, scheme, AgentRunReconcilerOpts{
		IntentTranslator:  NewIntentTranslator(),
		PolicyGate:        NewPolicyGate(),
		WorkspaceResolver: NewWorkspaceResolver(t.TempDir()),
		BeadsAvailable:    false,
	})

	_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
	if err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	var taskList v1alpha1.TaskList
	if err := fakeClient.List(context.Background(), &taskList); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(taskList.Items) != 1 {
		t.Fatalf("expected 1 reviewer task, got %d", len(taskList.Items))
	}
	reviewer := taskList.Items[0]
	if len(reviewer.Spec.DependsOn) != 2 {
		t.Errorf("expected DependsOn [run-1-analyst, run-1-coder], got %v", reviewer.Spec.DependsOn)
	}
	if reviewer.Labels["role"] != "reviewer" {
		t.Errorf("expected role=reviewer, got %q", reviewer.Labels["role"])
	}

	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status.Phase != "ReviewerRunning" {
		t.Errorf("expected Phase=ReviewerRunning, got %q", run.Status.Phase)
	}
	if run.Status.ReviewerTask != "run-1-reviewer" {
		t.Errorf("expected ReviewerTask=run-1-reviewer, got %q", run.Status.ReviewerTask)
	}
}

func TestCreateTaskFromIntent_DependsOn(t *testing.T) {
	run := &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec:       v1alpha1.AgentRunSpec{IssueID: "i1", Model: "glm-4.7"},
	}
	intent := &TaskIntent{IssueID: "i1", Prompt: "p", Objective: "o", RunID: "r1"}

	// Analyst/coder: no dependsOn
	analyst := createTaskFromIntent(run, "analyst", intent, "")
	if len(analyst.Spec.DependsOn) != 0 {
		t.Errorf("analyst DependsOn: %v, want empty", analyst.Spec.DependsOn)
	}

	// Reviewer: with dependsOn
	reviewer := createTaskFromIntent(run, "reviewer", intent, "", "run-1-analyst", "run-1-coder")
	if len(reviewer.Spec.DependsOn) != 2 {
		t.Errorf("reviewer DependsOn: %v", reviewer.Spec.DependsOn)
	}
	if reviewer.Spec.DependsOn[0] != "run-1-analyst" || reviewer.Spec.DependsOn[1] != "run-1-coder" {
		t.Errorf("reviewer DependsOn: %v", reviewer.Spec.DependsOn)
	}
}
