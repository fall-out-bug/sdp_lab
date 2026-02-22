package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

// Subscription represents an active NATS subscription.
type Subscription interface {
	Unsubscribe() error
}

// subImpl wraps nats.Subscription.
type subImpl struct {
	sub *nats.Subscription
}

func (s *subImpl) Unsubscribe() error {
	return s.sub.Unsubscribe()
}

// Subscriber subscribes to NATS subjects and invokes handlers.
type Subscriber struct {
	client *Client
	mu     sync.Mutex
	subs   []*nats.Subscription
}

// NewSubscriber creates a subscriber for the given client.
func NewSubscriber(client *Client) *Subscriber {
	return &Subscriber{client: client}
}

// Subscribe subscribes to subject with optional queue group and invokes handler for each message.
func (s *Subscriber) Subscribe(subject, queue string, handler func(Envelope)) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nc := s.client.Conn()

	cb := func(msg *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			return
		}
		handler(env)
	}

	var sub *nats.Subscription
	var err error
	if queue != "" {
		sub, err = nc.QueueSubscribe(subject, queue, cb)
	} else {
		sub, err = nc.Subscribe(subject, cb)
	}
	if err != nil {
		return nil, fmt.Errorf("subscribe %q: %w", subject, err)
	}

	s.subs = append(s.subs, sub)
	return &subImpl{sub: sub}, nil
}

// SubscribeWithContext subscribes and runs handler with context containing extracted W3C trace context.
func (s *Subscriber) SubscribeWithContext(subject, queue string, handler func(context.Context, Envelope)) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	nc := s.client.Conn()

	cb := func(msg *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			return
		}
		ctx := extractTraceContext(context.Background(), msg.Header)
		handler(ctx, env)
	}

	var sub *nats.Subscription
	var err error
	if queue != "" {
		sub, err = nc.QueueSubscribe(subject, queue, cb)
	} else {
		sub, err = nc.Subscribe(subject, cb)
	}
	if err != nil {
		return nil, fmt.Errorf("subscribe %q: %w", subject, err)
	}

	s.subs = append(s.subs, sub)
	return &subImpl{sub: sub}, nil
}

// SubscribeJetStream subscribes to a JetStream stream consumer.
// subject is the stream subject filter (e.g. "sdp.artifact.>").
func (s *Subscriber) SubscribeJetStream(subject, stream, consumer, queue string, handler func(Envelope)) (Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	js := s.client.JetStream()

	opts := []nats.SubOpt{
		nats.Durable(consumer),
		nats.ManualAck(),
		nats.BindStream(stream),
	}

	cb := func(msg *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			_ = msg.Nak()
			return
		}
		handler(env)
		_ = msg.Ack()
	}

	var sub *nats.Subscription
	var err error
	if queue != "" {
		sub, err = js.QueueSubscribe(subject, queue, cb, opts...)
	} else {
		sub, err = js.Subscribe(subject, cb, opts...)
	}
	if err != nil {
		return nil, fmt.Errorf("jetstream subscribe: %w", err)
	}

	s.subs = append(s.subs, sub)
	return &subImpl{sub: sub}, nil
}

// UnsubscribeAll unsubscribes all subscriptions created by this subscriber.
func (s *Subscriber) UnsubscribeAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sub := range s.subs {
		_ = sub.Unsubscribe()
	}
	s.subs = nil
}
