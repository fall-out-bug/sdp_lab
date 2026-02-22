package bus

import (
	"encoding/json"
	"fmt"
)

// Publisher publishes envelopes to NATS.
type Publisher struct {
	client *Client
}

// NewPublisher creates a publisher for the given client.
func NewPublisher(client *Client) *Publisher {
	return &Publisher{client: client}
}

// Publish serializes the envelope and publishes to the subject.
func (p *Publisher) Publish(subject string, envelope Envelope) error {
	envelope.Subject = subject
	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	nc := p.client.Conn()
	return nc.Publish(subject, data)
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
