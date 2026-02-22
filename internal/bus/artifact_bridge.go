package bus

import (
	"encoding/json"
	"log/slog"
	"sync"

	"sdp_dev/internal/artifact"
)

// ArtifactBridge subscribes to NATS artifact subjects and ingests into BusService.
// Maintains hash-chain provenance in the artifact bus for all received envelopes.
type ArtifactBridge struct {
	client    *Client
	bus       *artifact.BusService
	sub       Subscription
	subject   string
	queue     string
	mu        sync.Mutex
	ingested  map[string]struct{} // issueID:artifactID for dedup
}

// NewArtifactBridge creates a bridge that ingests NATS messages into the artifact bus.
func NewArtifactBridge(client *Client, bus *artifact.BusService, subject, queue string) *ArtifactBridge {
	return &ArtifactBridge{
		client:   client,
		bus:      bus,
		subject:  subject,
		queue:    queue,
		ingested: map[string]struct{}{},
	}
}

// Start subscribes to the subject and begins ingesting. Blocks until Stop is called.
func (b *ArtifactBridge) Start() error {
	sub := NewSubscriber(b.client)
	s, err := sub.Subscribe(b.subject, b.queue, b.handle)
	if err != nil {
		return err
	}
	b.sub = s
	return nil
}

// Stop unsubscribes and stops ingesting.
func (b *ArtifactBridge) Stop() {
	if b.sub != nil {
		_ = b.sub.Unsubscribe()
		b.sub = nil
	}
}

func (b *ArtifactBridge) handle(env Envelope) {
	key := env.IssueID + ":" + env.ArtifactID
	b.mu.Lock()
	if _, ok := b.ingested[key]; ok {
		b.mu.Unlock()
		return
	}
	b.mu.Unlock()

	req := env.ToIngestRequest()
	_, err := b.bus.Ingest(req)
	if err != nil {
		slog.Warn("artifact bridge ingest failed", "issue", env.IssueID, "artifact", env.ArtifactID, "err", err)
		return
	}

	b.mu.Lock()
	b.ingested[key] = struct{}{}
	b.mu.Unlock()

	slog.Debug("artifact bridge ingested", "issue", env.IssueID, "artifact", env.ArtifactID)
}

// IngestFromBytes parses a NATS message payload and ingests into the bus.
// Used when processing messages outside the bridge's subscription.
func (b *ArtifactBridge) IngestFromBytes(data []byte) error {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return err
	}
	req := env.ToIngestRequest()
	_, err := b.bus.Ingest(req)
	return err
}
