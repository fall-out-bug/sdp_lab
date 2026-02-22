package bus

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
)

// Bus is the main interface for SDP message bus operations.
type Bus interface {
	Publish(subject string, envelope Envelope) error
	Subscribe(subject, queue string, handler func(Envelope)) (Subscription, error)
	Request(subject string, envelope Envelope, timeout time.Duration) (Envelope, error)
	JetStream() nats.JetStreamContext
	Close()
}

// natsBus implements Bus using NATS client.
type natsBus struct {
	client    *Client
	publisher *Publisher
	subscriber *Subscriber
}

// NewBus creates a Bus from a connected Client.
func NewBus(client *Client) Bus {
	return &natsBus{
		client:     client,
		publisher:  NewPublisher(client),
		subscriber: NewSubscriber(client),
	}
}

// Publish publishes envelope to subject.
func (b *natsBus) Publish(subject string, envelope Envelope) error {
	return b.publisher.Publish(subject, envelope)
}

// Subscribe subscribes to subject with optional queue group.
func (b *natsBus) Subscribe(subject, queue string, handler func(Envelope)) (Subscription, error) {
	return b.subscriber.Subscribe(subject, queue, handler)
}

// Request sends request and waits for reply.
func (b *natsBus) Request(subject string, envelope Envelope, timeout time.Duration) (Envelope, error) {
	return Request(b.client, subject, envelope, timeout)
}

// JetStream returns the JetStream context.
func (b *natsBus) JetStream() nats.JetStreamContext {
	return b.client.JetStream()
}

// Close closes the underlying connection.
func (b *natsBus) Close() {
	b.subscriber.UnsubscribeAll()
	b.client.Close()
}

// ConnectAndProvision connects to NATS, provisions JetStream streams, and returns a Bus.
func ConnectAndProvision(ctx context.Context, url string) (Bus, error) {
	client := NewClient(url)
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	if err := ProvisionStreamsDefault(ctx, client); err != nil {
		client.Close()
		return nil, err
	}
	return NewBus(client), nil
}
