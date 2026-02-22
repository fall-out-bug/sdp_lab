package bus

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/propagation"
)

// natsHeaderCarrier adapts nats.Header to propagation.TextMapCarrier for W3C trace context.
type natsHeaderCarrier struct {
	h nats.Header
}

func (c natsHeaderCarrier) Get(key string) string {
	return c.h.Get(key)
}

func (c natsHeaderCarrier) Set(key, value string) {
	c.h.Set(key, value)
}

func (c natsHeaderCarrier) Keys() []string {
	if c.h == nil {
		return nil
	}
	keys := make([]string, 0, len(c.h))
	for k := range c.h {
		keys = append(keys, k)
	}
	return keys
}

// defaultPropagator is the W3C Trace Context propagator.
var defaultPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// injectTraceContext injects traceparent/tracestate from ctx into h.
func injectTraceContext(ctx context.Context, h nats.Header) {
	if h == nil {
		return
	}
	defaultPropagator.Inject(ctx, natsHeaderCarrier{h: h})
}

// extractTraceContext extracts trace context from h into ctx.
func extractTraceContext(ctx context.Context, h nats.Header) context.Context {
	if h == nil || len(h) == 0 {
		return ctx
	}
	return defaultPropagator.Extract(ctx, natsHeaderCarrier{h: h})
}
