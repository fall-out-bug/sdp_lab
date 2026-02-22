package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"sdp_dev/internal/artifact"
)

func TestFromArtifactEnvelope(t *testing.T) {
	ae := artifact.ArtifactEnvelope{
		IssueID: "i1", ArtifactID: "a1", ArtifactClass: "class", Phase: "p", Role: "r",
		CapturedAt: "2026-01-01T00:00:00Z", Payload: json.RawMessage(`{}`),
	}
	env := FromArtifactEnvelope(ae, "run-1", "proj1")
	if env.IssueID != ae.IssueID || env.RunID != "run-1" || env.ProjectID != "proj1" {
		t.Errorf("FromArtifactEnvelope: got %+v", env)
	}
}

func TestEnvelope_Timestamp(t *testing.T) {
	e := Envelope{CapturedAt: "2026-01-15T12:00:00Z"}
	if got := e.Timestamp(); got != "2026-01-15T12:00:00Z" {
		t.Errorf("Timestamp with CapturedAt: got %q", got)
	}
	e2 := Envelope{}
	got2 := e2.Timestamp()
	if got2 == "" || len(got2) < 20 {
		t.Errorf("Timestamp empty CapturedAt: got %q", got2)
	}
}

func TestEnvelopeRoundTrip(t *testing.T) {
	env := Envelope{
		IssueID:       "test-1",
		ArtifactID:    "a1",
		ArtifactClass: "analysis",
		Phase:         "execute",
		Role:          "analyst",
		CapturedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Payload:       json.RawMessage(`{"summary":"test"}`),
		RunID:         "run-1",
		ProjectID:     "proj1",
	}

	req := env.ToIngestRequest()
	if req.IssueID != env.IssueID {
		t.Errorf("IssueID: got %s", req.IssueID)
	}
	if req.Payload == nil {
		t.Error("Payload should not be nil")
	}

	ae := env.ToArtifactEnvelope()
	if ae.IssueID != env.IssueID {
		t.Errorf("ArtifactEnvelope IssueID: got %s", ae.IssueID)
	}
	_ = ae
}

func TestArtifactBridgeIngest(t *testing.T) {
	bus := artifact.NewBusService()
	req := artifact.IngestRequest{
		IssueID:       "test-1",
		ArtifactID:    "a1",
		ArtifactClass: "analysis",
		Phase:         "execute",
		Role:          "analyst",
		CapturedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		Payload:       map[string]any{"summary": "test"},
	}
	_, err := bus.Ingest(req)
	if err != nil {
		t.Fatal(err)
	}
	env, ok := bus.GetByIssueArtifactID("test-1", "a1")
	if !ok {
		t.Fatal("artifact not found")
	}
	if env.ArtifactID != "a1" {
		t.Errorf("got %s", env.ArtifactID)
	}
}

func TestDefaultStreams(t *testing.T) {
	configs := DefaultStreams()
	if len(configs) != 5 {
		t.Errorf("expected 5 streams, got %d", len(configs))
	}
	names := map[string]bool{}
	for _, c := range configs {
		names[c.Name] = true
	}
	for _, n := range []string{StreamIntake, StreamArtifacts, StreamReviews, StreamLifecycle, StreamRetro} {
		if !names[n] {
			t.Errorf("missing stream %s", n)
		}
	}
}

func TestClientConnect(t *testing.T) {
	// Use in-memory server for test - skip if nats-server not available
	// For now just test that Client can be created
	client := NewClient("nats://localhost:4222", WithConnectTimeout(100*time.Millisecond))
	defer client.Close()
	ctx := context.Background()
	err := client.Connect(ctx)
	if err != nil {
		// Expected if no NATS server
		t.Logf("Connect failed (expected without server): %v", err)
		return
	}
	if !client.IsConnected() {
		t.Error("expected connected")
	}
}
