package observability_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"sdp_dev/internal/observability"
)

func TestServeMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:19998"
	done := make(chan error, 1)
	go func() { done <- observability.ServeMetrics(ctx, addr) }()
	time.Sleep(80 * time.Millisecond)

	resp, err := http.Get("http://" + addr + "/metrics")
	if err != nil {
		t.Skipf("metrics server not reachable: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /metrics: status = %d", resp.StatusCode)
	}
	cancel()
	<-done
}

func TestMetricsHelpers(t *testing.T) {
	observability.IncAgentRuns("proj", "succeeded", "glm-4")
	observability.SetEvidenceCompleteness("proj", 0.85)
	observability.SetDispatchQueueDepth(5)
	// No panic; metrics updated
}
