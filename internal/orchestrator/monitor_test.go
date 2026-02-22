package orchestrator

import (
	"context"
	"testing"
	"time"

	"sdp_dev/api/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestEscalateTimedOut_Empty(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	// No panic, no error
}

func TestEscalateTimedOut_SkipsSucceeded(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	old := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ar1",
				Namespace:         "ns",
				CreationTimestamp: old,
			},
			Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 60},
			Status: v1alpha1.AgentRunStatus{Phase: "Succeeded"},
		},
	).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	run := &v1alpha1.AgentRun{}
	_ = k8s.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "ar1"}, run)
	if run.Status.Phase != "Succeeded" {
		t.Errorf("expected Succeeded unchanged, got %s", run.Status.Phase)
	}
}

func TestEscalateTimedOut_EscalatesTimedOut(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	old := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	run := &v1alpha1.AgentRun{
		TypeMeta: metav1.TypeMeta{Kind: "AgentRun", APIVersion: "sdp.dev/v1alpha1"},
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ar1",
			Namespace:         "ns",
			CreationTimestamp: old,
		},
		Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 60},
		Status: v1alpha1.AgentRunStatus{Phase: "Running"},
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	got := &v1alpha1.AgentRun{}
	if err := k8s.Get(ctx, client.ObjectKeyFromObject(run), got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status.Phase != "Escalated" {
		t.Errorf("expected Escalated, got %s", got.Status.Phase)
	}
	if got.Status.LastError != "timeout exceeded" {
		t.Errorf("expected LastError timeout exceeded, got %s", got.Status.LastError)
	}
}

func TestEscalateTimedOut_DoesNotEscalateWithinTimeout(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	recent := metav1.NewTime(time.Now().Add(-30 * time.Second))
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "ar1",
				Namespace:         "ns",
				CreationTimestamp: recent,
			},
			Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 3600},
			Status: v1alpha1.AgentRunStatus{Phase: "Running"},
		},
	).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	run := &v1alpha1.AgentRun{}
	_ = k8s.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "ar1"}, run)
	if run.Status.Phase != "Running" {
		t.Errorf("expected Running unchanged, got %s", run.Status.Phase)
	}
}
