package omoclient

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"sdp_dev/internal/kernel"
)

// ServeInvoker implements kernel.RuntimeAdapter using OmO serve.
type ServeInvoker struct {
	client     *OmOServeClient
	supervisor *OmOSupervisor
}

// NewServeInvoker creates a new ServeInvoker
func NewServeInvoker(baseURL string) *ServeInvoker {
	return &ServeInvoker{
		client:     NewClient(baseURL),
		supervisor: NewOmOSupervisor(baseURL),
	}
}

// Invoke implements kernel.RuntimeAdapter.
func (s *ServeInvoker) Invoke(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	dir := req.WorkDir
	if dir == "" {
		dir = "."
	}

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	_, err := s.client.CreateSession(CreateSessionRequest{
		Project: dir,
		Session: sessionID,
	})
	if err != nil {
		return kernel.RuntimeResult{ExitCode: 1}, fmt.Errorf("create session: %v (serve API unavailable, use exec mode)", err)
	}
	defer func() {
		_ = s.client.DeleteSession(sessionID)
	}()

	resp, err := s.client.SendMessageStream(req.Prompt)
	if err != nil {
		return kernel.RuntimeResult{ExitCode: 1}, fmt.Errorf("send message stream: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check if response is HTML (SPA fallback) — means API not available
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/html") {
		return kernel.RuntimeResult{ExitCode: 1}, fmt.Errorf("serve returned HTML instead of SSE — API not available, use exec mode")
	}
	defer func() { _ = resp.Body.Close() }()

	events := ReadSSEStream(ctx, resp.Body)

	var output strings.Builder
	var exitCode int
	var lastError string

	for event := range events {
		slog.Debug("SSE event", "class", event.Class, "prefix", event.Prefix, "data", event.Data)

		switch event.Class {
		case EventToolStarted, EventToolCompleted:
		case EventCompletionSucceeded:
			exitCode = 0
		case EventCompletionFailed:
			exitCode = 1
			if event.Data != "" {
				lastError = event.Data
			}
		case EventWarning:
			slog.Warn("warning from serve", "data", event.Data)
		case EventUnknown:
			if event.Prefix != "" {
				output.WriteString(event.Prefix)
			}
			if event.Data != "" {
				output.WriteString(event.Data)
				output.WriteString("\n")
			}
		}
	}

	if ctx.Err() != nil {
		return kernel.RuntimeResult{ExitCode: 130}, fmt.Errorf("invoke cancelled: %w", ctx.Err())
	}

	result := output.String()
	if result == "" && lastError != "" {
		result = lastError
	}

	if exitCode == 0 && result == "" {
		return kernel.RuntimeResult{}, nil
	}

	return kernel.RuntimeResult{Output: result, ExitCode: exitCode}, nil
}

// StartSupervisor starts the opencode serve subprocess
func (s *ServeInvoker) StartSupervisor(ctx context.Context) error {
	return s.supervisor.Start(ctx)
}

// WaitReady waits for the supervisor to be ready
func (s *ServeInvoker) WaitReady(ctx context.Context, timeout time.Duration) error {
	return s.supervisor.WaitReady(ctx, timeout)
}

// StopSupervisor stops the supervisor gracefully
func (s *ServeInvoker) StopSupervisor(gracePeriod time.Duration) error {
	return s.supervisor.Stop(gracePeriod)
}

// Status returns the supervisor status
// Status checks if opencode serve is reachable.
// First checks internal supervisor state, then falls back to TCP probe.
func (s *ServeInvoker) Status() (running bool, ready bool) {
	r, re := s.supervisor.Status()
	if r {
		return r, re
	}
	// Fallback: TCP probe to check if serve is running externally
	return s.httpProbe()
}

// httpProbe checks if the serve URL is reachable via TCP connection.
func (s *ServeInvoker) httpProbe() (running bool, ready bool) {
	baseURL := strings.TrimPrefix(s.client.baseURL, "http://")
	baseURL = strings.TrimPrefix(baseURL, "https://")
	// Strip path, keep host:port
	if idx := strings.Index(baseURL, "/"); idx >= 0 {
		baseURL = baseURL[:idx]
	}
	conn, err := net.DialTimeout("tcp", baseURL, 3*time.Second)
	if err != nil {
		return false, false
	}
	_ = conn.Close()
	return true, true
}
