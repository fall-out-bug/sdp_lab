package orchestrator

import (
	"context"
	"testing"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestActivePhases(t *testing.T) {
	if !ActivePhases["Running"] || !ActivePhases[""] || !ActivePhases["ReviewerPending"] {
		t.Error("expected active phases to include Running, empty, ReviewerPending")
	}
	if ActivePhases["Succeeded"] || ActivePhases["Failed"] {
		t.Error("Succeeded and Failed should not be active")
	}
}

func TestCountActiveAgentRuns(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar1", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: "Running"},
		},
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar2", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: "Succeeded"},
		},
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar3", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: ""},
		},
	).Build()
	ctx := context.Background()
	n, err := CountActiveAgentRuns(ctx, k8s, "ns")
	if err != nil {
		t.Fatalf("CountActiveAgentRuns: %v", err)
	}
	if n != 2 {
		t.Errorf("CountActiveAgentRuns = %d, want 2", n)
	}
}

func TestCountActiveAgentRuns_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()
	n, err := CountActiveAgentRuns(ctx, k8s, "ns")
	if err != nil {
		t.Fatalf("CountActiveAgentRuns: %v", err)
	}
	if n != 0 {
		t.Errorf("CountActiveAgentRuns = %d, want 0", n)
	}
}
