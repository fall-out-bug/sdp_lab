package bus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
)

// Publisher publishes envelopes to NATS.
type Publisher struct {
	client *Client
}

// NewPublisher creates a publisher for the given client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish serializes the envelope and publishes to the subject (no trace headers).
func (p *Publisher) Publish(subject string, envelope Envelope) error {
	envelope.Subject = subject
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	nc := p.client.Conn()
	return nc.Publish(subject, data)
}

// PublishWithContext publishes with W3C trace context in NATS headers when ctx has span.
func (p *Publisher) PublishWithContext(ctx context.Context, subject string, envelope Envelope) error {
	envelope.Subject = subject
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  make(nats.Header),
	}
	injectTraceContext(ctx, msg.Header)

	nc := p.client.Conn()
	return nc.PublishMsg(msg)
}

// PublishJetStream publishes to a JetStream stream (durable).
func (p *Publisher) PublishJetStream(subject string, envelope Envelope) error {
	envelope.Subject = subject
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	js := p.client.JetStream()
	_, err = js.Publish(subject, data)
	return err
}
