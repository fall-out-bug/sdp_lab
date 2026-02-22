package observability

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// TracerProviderShutdown is a function that shuts down the tracer provider.
type TracerProviderShutdown func(context.Context) error

// SetupTracing initializes OTLP trace export. Set OTEL_EXPORTER_OTLP_ENDPOINT=none to disable.
// Returns nil, nil if disabled. Otherwise returns shutdown func.
// Dual-write: JSONL emission is preserved; OTLP adds distributed traces.
func SetupTracing(serviceName string) (TracerProviderShutdown, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT_HTTP")
	}
	if endpoint == "none" {
		return nil, nil
	}
	if endpoint == "" {
		endpoint = "otel-collector.sdp-observability.svc.cluster.local:4318"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	host := endpoint
	if idx := strings.Index(endpoint, "://"); idx >= 0 {
		host = endpoint[idx+3:]
	}
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(host),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}
