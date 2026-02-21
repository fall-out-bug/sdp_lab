package bus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"

	"sdp_dev/internal/bus"
)

func TestBusIntegration(t *testing.T) {
	opts := &server.Options{
		Port: -1,
		JetStream: true,
	}
	ns, err := server.NewServer(opts)
	if err != nil {
		t.Fatalf("start embedded NATS: %v", err)
	}
	go ns.Start()
	defer ns.Shutdown()
	if !ns.ReadyForConnections(2 * time.Second) {
		t.Fatal("NATS not ready")
	}

	url := ns.ClientURL()
	ctx := context.Background()
	b, err := bus.ConnectAndProvision(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	received := make(chan bus.Envelope, 1)
	_, err = b.Subscribe("sdp.test.>", "test-queue", func(env bus.Envelope) {
		received <- env
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{"hello": "world"})
	env := bus.Envelope{
		IssueID:       "test-1",
		ArtifactID:    "artifact-1",
		ArtifactClass: "test",
		Phase:         "test",
		Role:          "test",
		Payload:       payload,
	}
	if err := b.Publish("sdp.test.foo", env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-received:
		if got.IssueID != "test-1" {
			t.Errorf("got IssueID %q", got.IssueID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
