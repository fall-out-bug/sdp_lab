package adapter

import (
	"context"
	"testing"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"

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
		IntentTranslator: NewIntentTranslator(),
		PolicyGate:       NewPolicyGate(),
		BeadsAdapter:    beads.NewAdapter(t.TempDir()),
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
		IntentTranslator: NewIntentTranslator(),
		PolicyGate:       NewPolicyGate(),
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
