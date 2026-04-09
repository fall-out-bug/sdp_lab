package omoclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"
)

// skipIfNoServer skips if opencode serve is not running.
func skipIfNoServer(t *testing.T) string {
	port := os.Getenv("OPENCODE_SERVE_PORT")
	if port == "" {
		port = "4096"
	}
	url := fmt.Sprintf("http://127.0.0.1:%s", port)
	resp, err := http.Get(url + "/session/status")
	if err != nil {
		t.Skipf("opencode serve not running at %s: %v", url, err)
	}
	_ = resp.Body.Close()
	return url
}

func TestIntegration_CreateAndListSession(t *testing.T) {
t.Skip("Integration test requires REST API not available on opencode serve")
	baseURL := skipIfNoServer(t)
	client := NewClient(baseURL)

	session, err := client.CreateSession(CreateSessionRequest{
		Project: "bridge-integration-test",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session == nil || session.ID == "" {
		t.Fatal("expected non-nil session with ID")
	}
	t.Logf("Created session: %s", session.ID)
	defer func() { _ = client.DeleteSession(session.ID) }()

	// List sessions and verify ours is there
	sessions, err := client.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	found := false
	for _, s := range sessions {
		if s.ID == session.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created session not found in list")
	}
}

func TestIntegration_WarmStartLatency(t *testing.T) {
	baseURL := skipIfNoServer(t)
	client := NewClient(baseURL)

	n := 5
	times := make([]time.Duration, n)
	var ids []string

	for i := 0; i < n; i++ {
		start := time.Now()
		session, err := client.CreateSession(CreateSessionRequest{
			Project: fmt.Sprintf("latency-test-%d", i),
		})
		if err != nil {
			t.Fatalf("CreateSession %d: %v", i, err)
		}
		times[i] = time.Since(start)
		ids = append(ids, session.ID)
		t.Logf("Session %d: created in %v", i, times[i])
	}

	// Cleanup
	for _, id := range ids {
		_ = client.DeleteSession(id)
	}

	avg := times[0]
	for _, d := range times[1:] {
		avg += d
	}
	avg /= time.Duration(n)

	min := times[0]
	max := times[0]
	for _, d := range times[1:] {
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	t.Logf("Warm session creation: avg=%v min=%v max=%v", avg, min, max)

	if avg > 2*time.Second {
		t.Errorf("Warm start avg %v exceeds 2s threshold", avg)
	}
}

func TestIntegration_SendMessageSSE(t *testing.T) {
	baseURL := skipIfNoServer(t)
	client := NewClient(baseURL)

	session, err := client.CreateSession(CreateSessionRequest{
		Project: "bridge-sse-test",
	})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	defer func() { _ = client.DeleteSession(session.ID) }()
	t.Logf("Session: %s", session.ID)

	resp, err := client.SendMessageStream("Reply with exactly: HELLO_BRIDGE_TEST")
	if err != nil {
		t.Fatalf("SendMessageStream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	events := ReadSSEStream(ctx, resp.Body)

	timeout := time.After(90 * time.Second)
	eventCount := 0
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Logf("SSE stream closed after %d events", eventCount)
				return
			}
			eventCount++
			if eventCount <= 5 || ev.Class == EventCompletionSucceeded || ev.Class == EventCompletionFailed {
				t.Logf("Event #%d: class=%s data=%q", eventCount, ev.Class, ev.Data)
			}
			// Try to detect completion
			var parsed map[string]any
			if json.Unmarshal([]byte(ev.Data), &parsed) == nil {
				if ev.Class == EventCompletionSucceeded || ev.Class == EventCompletionFailed {
					t.Logf("Completion event received after %d events", eventCount)
					return
				}
			}
		case <-timeout:
			t.Fatalf("timeout after %d events", eventCount)
		}
	}
}
