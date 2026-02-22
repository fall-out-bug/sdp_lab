package orchestrator

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/bus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// mockBus records Publish calls for tests.
type mockBus struct {
	mu        sync.Mutex
	publishes []struct {
		subject string
		env     bus.Envelope
	}
}

func (m *mockBus) Publish(subject string, envelope bus.Envelope) error {
	m.mu.Lock()
	m.publishes = append(m.publishes, struct {
		subject string
		env     bus.Envelope
	}{subject, envelope})
	m.mu.Unlock()
	return nil
}

func (m *mockBus) PublishWithContext(_ context.Context, subject string, envelope bus.Envelope) error {
	return m.Publish(subject, envelope)
}

func (m *mockBus) Subscribe(_, _ string, _ func(bus.Envelope)) (bus.Subscription, error) {
	return nil, nil
}
func (m *mockBus) SubscribeWithContext(_, _ string, _ func(context.Context, bus.Envelope)) (bus.Subscription, error) {
	return nil, nil
}
func (m *mockBus) Request(_ string, _ bus.Envelope, _ time.Duration) (bus.Envelope, error) {
	return bus.Envelope{}, nil
}
func (m *mockBus) JetStream() nats.JetStreamContext { return nil }
func (m *mockBus) Close()                 {}

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

func TestEscalateTimedOut_PublishesToBus(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	old := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	run := &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ar1",
			Namespace:         "ns",
			CreationTimestamp: old,
			Labels:            map[string]string{"project": "p1"},
		},
		Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 60},
		Status: v1alpha1.AgentRunStatus{Phase: "Running"},
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).WithStatusSubresource(run).Build()
	b := &mockBus{}
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, b, "ns")
	b.mu.Lock()
	n := len(b.publishes)
	subj := ""
	var env bus.Envelope
	if n > 0 {
		subj = b.publishes[0].subject
		env = b.publishes[0].env
	}
	b.mu.Unlock()
	if n != 1 {
		t.Fatalf("expected 1 Publish call, got %d", n)
	}
	if subj != "sdp.lifecycle.agentrun.escalated" {
		t.Errorf("subject = %q", subj)
	}
	if env.Phase != "escalated" || env.IssueID != "i1" || env.ProjectID != "p1" {
		t.Errorf("envelope: %+v", env)
	}
}

func TestEscalateTimedOut_SkipsCreatedZero(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	run := &v1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "ar1",
			Namespace:         "ns",
			CreationTimestamp: metav1.Time{},
		},
		Spec:   v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 60},
		Status: v1alpha1.AgentRunStatus{Phase: "Running"},
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(run).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	got := &v1alpha1.AgentRun{}
	_ = k8s.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "ar1"}, got)
	if got.Status.Phase != "Running" {
		t.Errorf("expected Running when CreationTimestamp zero, got %s", got.Status.Phase)
	}
}

func TestEscalateTimedOut_SkipsEscalatedPhase(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	old := metav1.NewTime(time.Now().Add(-2 * time.Hour))
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&v1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "ar1", Namespace: "ns", CreationTimestamp: old},
			Spec:       v1alpha1.AgentRunSpec{IssueID: "i1", TimeoutSec: 60},
			Status:     v1alpha1.AgentRunStatus{Phase: "Escalated"},
		},
	).Build()
	ctx := context.Background()
	escalateTimedOut(ctx, k8s, nil, "ns")
	run := &v1alpha1.AgentRun{}
	_ = k8s.Get(ctx, client.ObjectKey{Namespace: "ns", Name: "ar1"}, run)
	if run.Status.Phase != "Escalated" {
		t.Errorf("expected Escalated unchanged, got %s", run.Status.Phase)
	}
}

func TestMonitorAgentRunTimeouts_OneTick(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	k8s := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	MonitorAgentRunTimeouts(ctx, k8s, nil, "ns", 10*time.Millisecond)
	// No panic; at least one tick runs before context expires
}
