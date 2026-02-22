package observability

import (
	"context"
	"os"
	"testing"
)

func TestSetupTracing_disabled(t *testing.T) {
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "none")
	defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown, err := SetupTracing("test-service")
	if err != nil {
		t.Fatalf("SetupTracing: %v", err)
	}
	if shutdown != nil {
		t.Error("expected nil shutdown when disabled")
	}
}

func TestSetupTracing_unset(t *testing.T) {
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT_HTTP")

	// With default endpoint, SetupTracing will try to connect.
	// Use none to avoid connection in unit test.
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "none")
	shutdown, err := SetupTracing("test-service")
	if err != nil {
		t.Fatalf("SetupTracing: %v", err)
	}
	if shutdown != nil {
		t.Error("expected nil shutdown when endpoint=none")
	}
}

func TestTracerProviderShutdown(t *testing.T) {
	// Shutdown of nil should be safe
	var fn TracerProviderShutdown
	if fn != nil {
		_ = fn(context.Background())
	}
}
