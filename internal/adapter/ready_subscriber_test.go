package adapter

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"

	"sdp_dev/api/v1alpha1"
	"sdp_dev/internal/beads"
	"sdp_dev/internal/bus"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReadySubscriber_Handle_EmptyPayload(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	sub := &ReadySubscriber{
		Client:          c,
		Namespace:       "default",
		IntentTranslator: NewIntentTranslator(),
		PolicyGate:      NewPolicyGate(),
		LockManager:     NewRunLockManager(t.TempDir()),
	}
	env := bus.Envelope{Payload: []byte("{}")}
	sub.Handle(context.Background(), env)
}

func TestReadySubscriber_Handle_OneIssue(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	lockDir := t.TempDir()
	sub := &ReadySubscriber{
		Client:          c,
		Namespace:       "default",
		IntentTranslator: NewIntentTranslator(),
		PolicyGate:      NewPolicyGate(),
		LockManager:     NewRunLockManager(lockDir),
	}
	payload := ReadyPayload{
		ProjectID: "proj1",
		Issues: []beads.Issue{
			{ID: "issue-1", Title: "Fix bug", Description: "desc", AcceptanceCriteria: "AC"},
		},
		Count: 1,
	}
	raw, _ := json.Marshal(payload)
	env := bus.Envelope{Payload: raw, ProjectID: "proj1"}
	sub.Handle(context.Background(), env)
	var list v1alpha1.TaskList
	if err := c.List(context.Background(), &list); err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("expected 1 Task, got %d", len(list.Items))
	}
	if len(list.Items) > 0 && list.Items[0].Labels["beads.issue"] != "issue-1" {
		t.Errorf("Task beads.issue = %q, want issue-1", list.Items[0].Labels["beads.issue"])
	}
}

func TestDnsName(t *testing.T) {
	if got := dnsName("sdp_dev-4pg"); got != "sdp-dev-4pg" {
		t.Errorf("dnsName(sdp_dev-4pg) = %q, want sdp-dev-4pg", got)
	}
	if got := dnsName("UPPER"); got != "upper" {
		t.Errorf("dnsName(UPPER) = %q, want upper", got)
	}
}

// TestReadySubscriber_NATS_Integration verifies that a ready event published to
// sdp.beads.*.ready is received by a subscriber and creates a Task (embedded NATS).
func TestReadySubscriber_NATS_Integration(t *testing.T) {
	opts := &server.Options{Port: -1, JetStream: true}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("start NATS: %v", err)
	}
	go ns.Start()
	defer ns.Shutdown()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS not ready")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	b, err := bus.ConnectAndProvision(ctx, ns.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	lockDir := t.TempDir()
	readySub := &ReadySubscriber{
		Client:           c,
		Namespace:        "default",
		IntentTranslator: NewIntentTranslator(),
		PolicyGate:       NewPolicyGate(),
		LockManager:      NewRunLockManager(lockDir),
	}
	_, err = b.Subscribe("sdp.beads.*.ready", "adapter-test", func(env bus.Envelope) {
		readySub.Handle(context.Background(), env)
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	payload := ReadyPayload{
		ProjectID: "proj1",
		Issues: []beads.Issue{
			{ID: "nat-1", Title: "NATS test", AcceptanceCriteria: "AC"},
		},
		Count: 1,
	}
	raw, _ := json.Marshal(payload)
	env := bus.Envelope{
		IssueID:       "proj1",
		ArtifactID:    "ready-snapshot",
		ArtifactClass: "beads",
		Phase:         "ready",
		Payload:       raw,
		ProjectID:     "proj1",
	}
	if err := b.Publish("sdp.beads.proj1.ready", env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	var list v1alpha1.TaskList
	if err := c.List(context.Background(), &list); err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 {
		t.Errorf("expected 1 Task after ready event, got %d", len(list.Items))
	}
	if len(list.Items) > 0 {
		if list.Items[0].Labels["beads.issue"] != "nat-1" {
			t.Errorf("Task beads.issue = %q, want nat-1", list.Items[0].Labels["beads.issue"])
		}
		if list.Items[0].Labels["sdp.project"] != "proj1" {
			t.Errorf("Task sdp.project = %q, want proj1", list.Items[0].Labels["sdp.project"])
		}
	}
}
