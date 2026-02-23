package adapter

import (
	"context"
	"strings"
	"testing"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAgentRunReconciler_Reconcile_EmptyPhase_CreatesAnalystOnly(t *testing.T) {
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

	// Verify only analyst Task created (sequential pipeline)
	var taskList v1alpha1.TaskList
	if err := fakeClient.List(context.Background(), &taskList); err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(taskList.Items) != 1 {
		t.Errorf("expected 1 task (analyst), got %d", len(taskList.Items))
	}
	if taskList.Items[0].Labels["role"] != "analyst" {
		t.Errorf("expected analyst task, got role=%q", taskList.Items[0].Labels["role"])
	}

	// Verify AgentRun status
	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status.Phase != "Analyzing" {
		t.Errorf("expected Phase=Analyzing, got %q", run.Status.Phase)
	}
	if run.Status.WorkerTask != "run-1-analyst" {
		t.Errorf("expected WorkerTask=run-1-analyst, got %q", run.Status.WorkerTask)
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

func TestAgentRunReconciler_Reconcile_AnalystComplete_CreatesCoderWithHandoffAnnotation(t *testing.T) {
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
			Phase:      "AnalystComplete",
			WorkerTask: "run-1-analyst",
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
		t.Fatalf("expected 1 coder task, got %d", len(taskList.Items))
	}
	coder := taskList.Items[0]
	if coder.Labels["role"] != "coder" {
		t.Errorf("expected coder task, got role=%q", coder.Labels["role"])
	}
	expectedPath := ".sdp/handoff/i1/analyst.json"
	if got := coder.Annotations["sdp.dev/handoff-analyst"]; got != expectedPath {
		t.Errorf("expected sdp.dev/handoff-analyst=%q, got %q", expectedPath, got)
	}
	if !strings.Contains(coder.Spec.Prompt, expectedPath) {
		t.Errorf("expected prompt to contain handoff path %q, got: %s", expectedPath, coder.Spec.Prompt)
	}
}

func TestAgentRunReconciler_Reconcile_CoderComplete_CreatesReviewerWithDependsOn(t *testing.T) {
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
			Phase:      "CoderComplete",
			WorkerTask: "run-1-coder",
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
	if reviewer.Annotations["sdp.dev/handoff-analyst"] != ".sdp/handoff/i1/analyst.json" {
		t.Errorf("reviewer missing handoff-analyst annotation: %v", reviewer.Annotations)
	}
	if reviewer.Annotations["sdp.dev/handoff-coder"] != ".sdp/handoff/i1/coder.json" {
		t.Errorf("reviewer missing handoff-coder annotation: %v", reviewer.Annotations)
	}

	if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
		t.Fatalf("get run: %v", err)
	}
	if run.Status.Phase != "Reviewing" {
		t.Errorf("expected Phase=Reviewing, got %q", run.Status.Phase)
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

	// Analyst: no dependsOn, no handoff
	analyst := createTaskFromIntent(run, "analyst", intent, "", nil)
	if len(analyst.Spec.DependsOn) != 0 {
		t.Errorf("analyst DependsOn: %v, want empty", analyst.Spec.DependsOn)
	}

	// Coder: with dependsOn (analyst), handoff annotation
	coder := createTaskFromIntent(run, "coder", intent, "", map[string]string{"sdp.dev/handoff-analyst": ".sdp/handoff/i1/analyst.json"}, "run-1-analyst")
	if len(coder.Spec.DependsOn) != 1 || coder.Spec.DependsOn[0] != "run-1-analyst" {
		t.Errorf("coder DependsOn: %v", coder.Spec.DependsOn)
	}
	if coder.Annotations["sdp.dev/handoff-analyst"] != ".sdp/handoff/i1/analyst.json" {
		t.Errorf("coder missing handoff-analyst annotation: %v", coder.Annotations)
	}

	// Reviewer: with dependsOn, both handoff annotations
	reviewer := createTaskFromIntent(run, "reviewer", intent, "", map[string]string{"sdp.dev/handoff-analyst": ".sdp/handoff/i1/analyst.json", "sdp.dev/handoff-coder": ".sdp/handoff/i1/coder.json"}, "run-1-analyst", "run-1-coder")
	if len(reviewer.Spec.DependsOn) != 2 {
		t.Errorf("reviewer DependsOn: %v", reviewer.Spec.DependsOn)
	}
	if reviewer.Spec.DependsOn[0] != "run-1-analyst" || reviewer.Spec.DependsOn[1] != "run-1-coder" {
		t.Errorf("reviewer DependsOn: %v", reviewer.Spec.DependsOn)
	}
}

// TestAgentRunReconciler_SequentialPipeline_Integration runs the full analyst→coder→reviewer flow.
// 00-004-03: Integration test proving sequential pipeline works.
func TestAgentRunReconciler_SequentialPipeline_Integration(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	workDir := t.TempDir()
	run := &v1alpha1.AgentRun{
		TypeMeta:   metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "default"},
		Spec: v1alpha1.AgentRunSpec{
			IssueID: "i1",
			Model:   "glm-4.7",
		},
		Status: v1alpha1.AgentRunStatus{},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(run).
		WithStatusSubresource(run).
		WithStatusSubresource(&v1alpha1.Task{}).
		Build()

	r := NewAgentRunReconciler(fakeClient, scheme, AgentRunReconcilerOpts{
		IntentTranslator:  NewIntentTranslator(),
		PolicyGate:        NewPolicyGate(),
		WorkspaceResolver: func(string) string { return workDir },
		BeadsAvailable:    false,
	})

	reconcile := func() {
		_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(run)})
		if err != nil {
			t.Fatalf("Reconcile: %v", err)
		}
		if err := fakeClient.Get(context.Background(), client.ObjectKeyFromObject(run), run); err != nil {
			t.Fatalf("get run: %v", err)
		}
	}

	// Step 1: Initial reconcile creates analyst only
	reconcile()
	if run.Status.Phase != "Analyzing" {
		t.Fatalf("after init: phase=%q, want Analyzing", run.Status.Phase)
	}
	var tasks v1alpha1.TaskList
	if err := fakeClient.List(context.Background(), &tasks); err != nil {
		t.Fatal(err)
	}
	if len(tasks.Items) != 1 || tasks.Items[0].Labels["role"] != "analyst" {
		t.Fatalf("expected 1 analyst task, got %d: %v", len(tasks.Items), taskRoles(tasks.Items))
	}

	// Step 2: Simulate analyst completing - set Task to Succeeded
	analystTask := &tasks.Items[0]
	analystTask.Status.Phase = v1alpha1.TaskPhaseSucceeded
	if err := fakeClient.Status().Update(context.Background(), analystTask); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if run.Status.Phase != "AnalystComplete" {
		t.Fatalf("after analyst done: phase=%q, want AnalystComplete", run.Status.Phase)
	}

	// Step 3: Reconcile creates coder (not before analyst completes)
	reconcile()
	if run.Status.Phase != "Coding" {
		t.Fatalf("after coder create: phase=%q, want Coding", run.Status.Phase)
	}
	if err := fakeClient.List(context.Background(), &tasks); err != nil {
		t.Fatal(err)
	}
	var coderTask *v1alpha1.Task
	for i := range tasks.Items {
		if tasks.Items[i].Labels["role"] == "coder" {
			coderTask = &tasks.Items[i]
			break
		}
	}
	if coderTask == nil {
		t.Fatal("coder task not created")
	}
	if coderTask.Annotations["sdp.dev/handoff-analyst"] != ".sdp/handoff/i1/analyst.json" {
		t.Errorf("coder missing handoff-analyst: %v", coderTask.Annotations)
	}
	if !strings.Contains(coderTask.Spec.Prompt, ".sdp/handoff/i1/analyst.json") {
		t.Errorf("coder prompt should reference analyst handoff: %s", coderTask.Spec.Prompt)
	}

	// Step 4: Simulate coder completing
	coderTask.Status.Phase = v1alpha1.TaskPhaseSucceeded
	if err := fakeClient.Status().Update(context.Background(), coderTask); err != nil {
		t.Fatal(err)
	}
	reconcile()
	if run.Status.Phase != "CoderComplete" {
		t.Fatalf("after coder done: phase=%q, want CoderComplete", run.Status.Phase)
	}

	// Step 5: Reconcile creates reviewer with both handoff paths
	reconcile()
	if run.Status.Phase != "Reviewing" {
		t.Fatalf("after reviewer create: phase=%q, want Reviewing", run.Status.Phase)
	}
	if err := fakeClient.List(context.Background(), &tasks); err != nil {
		t.Fatal(err)
	}
	var reviewerTask *v1alpha1.Task
	for i := range tasks.Items {
		if tasks.Items[i].Labels["role"] == "reviewer" {
			reviewerTask = &tasks.Items[i]
			break
		}
	}
	if reviewerTask == nil {
		t.Fatal("reviewer task not created")
	}
	if reviewerTask.Annotations["sdp.dev/handoff-analyst"] != ".sdp/handoff/i1/analyst.json" {
		t.Errorf("reviewer missing handoff-analyst: %v", reviewerTask.Annotations)
	}
	if reviewerTask.Annotations["sdp.dev/handoff-coder"] != ".sdp/handoff/i1/coder.json" {
		t.Errorf("reviewer missing handoff-coder: %v", reviewerTask.Annotations)
	}

	// Step 6: Simulate reviewer approve -> AgentRun Succeeded
	reviewerTask.Status.Phase = v1alpha1.TaskPhaseSucceeded
	if err := fakeClient.Status().Update(context.Background(), reviewerTask); err != nil {
		t.Fatal(err)
	}
	reconcile() // Reviewing -> ReviewerComplete
	reconcile() // ReviewerComplete -> Succeeded
	if run.Status.Phase != "Succeeded" {
		t.Fatalf("after reviewer approve: phase=%q, want Succeeded", run.Status.Phase)
	}
}

func taskRoles(items []v1alpha1.Task) []string {
	roles := make([]string, len(items))
	for i := range items {
		roles[i] = items[i].Labels["role"]
	}
	return roles
}
