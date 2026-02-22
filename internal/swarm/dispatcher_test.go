package swarm

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats.go"

	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"
	"sdp_dev/internal/federation"
)

func TestDispatcher_nilBus(t *testing.T) {
	svc := Dispatcher(nil)
	if svc == nil {
		t.Fatal("Dispatcher(nil) returned nil")
	}
	task := federation.FederatedTask{ProjectID: "p1"}
	if err := svc.Dispatch(task, "coder"); err != nil {
		t.Errorf("Dispatch with nil bus: %v", err)
	}
}

// mockBus records PublishWithContext calls for DispatchWithContext tests.
type mockBus struct {
	subject string
	env     bus.Envelope
	err     error
}

func (m *mockBus) Publish(subject string, env bus.Envelope) error { return m.err }
func (m *mockBus) PublishWithContext(_ context.Context, subject string, env bus.Envelope) error {
	m.subject = subject
	m.env = env
	return m.err
}
func (m *mockBus) Subscribe(_, _ string, _ func(bus.Envelope)) (bus.Subscription, error) { return nil, nil }
func (m *mockBus) SubscribeWithContext(_, _ string, _ func(context.Context, bus.Envelope)) (bus.Subscription, error) {
	return nil, nil
}
func (m *mockBus) Request(_ string, _ bus.Envelope, _ time.Duration) (bus.Envelope, error) {
	return bus.Envelope{}, nil
}
func (m *mockBus) JetStream() nats.JetStreamContext { return nil }
func (m *mockBus) Close() {}

func TestDispatchWithContext_PublishesToCorrectSubject(t *testing.T) {
	mb := &mockBus{}
	svc := Dispatcher(mb)
	task := federation.FederatedTask{
		ProjectID: "proj1",
		Issue:     beads.Issue{ID: "issue-1"},
	}
	if err := svc.DispatchWithContext(context.Background(), task, "coder"); err != nil {
		t.Fatalf("DispatchWithContext: %v", err)
	}
	if mb.subject != "sdp.dispatch.proj1.coder" {
		t.Errorf("subject = %q, want sdp.dispatch.proj1.coder", mb.subject)
	}
	if mb.env.IssueID != "issue-1" || mb.env.Phase != "coder" {
		t.Errorf("envelope = %+v", mb.env)
	}
	var decoded federation.FederatedTask
	if err := json.Unmarshal(mb.env.Payload, &decoded); err != nil {
		t.Fatalf("payload: %v", err)
	}
	if decoded.ProjectID != "proj1" || decoded.Issue.ID != "issue-1" {
		t.Errorf("payload task = %+v", decoded)
	}
}
