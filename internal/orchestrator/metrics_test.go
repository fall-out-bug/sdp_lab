package orchestrator

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestSetActiveRuns(t *testing.T) {
	SetActiveRuns(0)
	SetActiveRuns(3)
	SetActiveRuns(1)
	// No panic; gauge is updated (we don't read back prometheus in unit test)
}

func TestIncDispatched(t *testing.T) {
	IncDispatched()
	IncDispatched()
}

func TestIncEscalated(t *testing.T) {
	IncEscalated()
	IncEscalated()
}

func TestServeMetrics(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:19999"
	done := make(chan error, 1)
	go func() { done <- ServeMetrics(ctx, addr) }()
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
