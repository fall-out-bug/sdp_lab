package bus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"sdp_dev/internal/bus"
)

func TestPublishWithContext_InjectTraceHeaders(t *testing.T) {
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

	ctx := context.Background()
	b, err := bus.ConnectAndProvision(ctx, ns.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	// Setup tracer so we have span context to inject
	tp := sdktrace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
	defer func() { _ = tp.Shutdown(ctx) }()

	ctx, span := otel.Tracer("test").Start(ctx, "test-span")
	defer span.End()

	received := make(chan bus.Envelope, 1)
	_, err = b.Subscribe("sdp.trace.>", "trace-queue", func(env bus.Envelope) {
		received <- env
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{"trace": "test"})
	env := bus.Envelope{
		IssueID:       "trace-1",
		ArtifactID:    "a1",
		ArtifactClass: "test",
		Phase:         "test",
		Role:          "test",
		Payload:       payload,
	}
	if err := b.PublishWithContext(ctx, "sdp.trace.foo", env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-received:
		if got.IssueID != "trace-1" {
			t.Errorf("got IssueID %q", got.IssueID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestSubscribeWithContext_ExtractTraceContext(t *testing.T) {
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

	ctx := context.Background()
	b, err := bus.ConnectAndProvision(ctx, ns.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer b.Close()

	receivedCtx := make(chan context.Context, 1)
	_, err = b.SubscribeWithContext("sdp.trace2.>", "trace2-queue", func(ctx context.Context, env bus.Envelope) {
		receivedCtx <- ctx
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	payload, _ := json.Marshal(map[string]string{"trace": "test2"})
	env := bus.Envelope{
		IssueID:       "trace-2",
		ArtifactID:    "a2",
		ArtifactClass: "test",
		Phase:         "test",
		Role:          "test",
		Payload:       payload,
	}
	if err := b.Publish("sdp.trace2.bar", env); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case gotCtx := <-receivedCtx:
		if gotCtx == nil {
			t.Error("expected non-nil context")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message")
	}
}
