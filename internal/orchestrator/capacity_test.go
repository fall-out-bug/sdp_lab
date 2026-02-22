package orchestrator

import (
	"context"
	"testing"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCountActiveAgentRuns_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	n, err := CountActiveAgentRuns(context.Background(), k8s, "ns")
	if err != nil {
		t.Fatalf("CountActiveAgentRuns: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0, got %d", n)
	}
}

func TestCountActiveAgentRuns_CountsActive(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar1", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: "Running"},
		},
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar2", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: "Pending"},
		},
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar3", Namespace: "ns"},
			Status:     v1alpha1.AgentRunStatus{Phase: "Succeeded"},
		},
	).Build()
	n, err := CountActiveAgentRuns(context.Background(), k8s, "ns")
	if err != nil {
		t.Fatalf("CountActiveAgentRuns: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 active (Running, Pending), got %d", n)
	}
}

func TestActivePhases(t *testing.T) {
	for _, phase := range []string{"", "Running", "Pending", "ReviewerPending", "ReviewerRunning", "Escalated"} {
		if !ActivePhases[phase] {
			t.Errorf("phase %q should be active", phase)
		}
	}
	if ActivePhases["Succeeded"] || ActivePhases["Failed"] {
		t.Error("Succeeded/Failed should not be active")
	}
}
