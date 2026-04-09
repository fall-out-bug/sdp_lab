package omoclient

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

// OmOSupervisor manages opencode serve subprocess lifecycle
type OmOSupervisor struct {
	cmd     *exec.Cmd
	mu      sync.Mutex
	running bool
	ready   bool
	baseURL string
}

// NewOmOSupervisor creates a new supervisor instance
func NewOmOSupervisor(baseURL string) *OmOSupervisor {
	return &OmOSupervisor{
		baseURL: baseURL,
	}
}

// Start launches the opencode serve subprocess
func (s *OmOSupervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("supervisor already running")
	}

	s.cmd = exec.CommandContext(ctx, "opencode", "serve", "--listen", s.baseURL)
	s.cmd.Stdout = os.Stdout
	s.cmd.Stderr = os.Stderr

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start opencode serve: %w", err)
	}

	s.running = true
	s.ready = false
	slog.Info("opencode serve started", "url", s.baseURL)
	return nil
}

// WaitReady waits for the serve process to be ready for connections
func (s *OmOSupervisor) WaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Until(deadline)):
			return fmt.Errorf("timeout waiting for opencode serve to be ready")
		case <-ticker.C:
			s.mu.Lock()
			if s.ready {
				s.mu.Unlock()
				return nil
			}

			if s.cmd.Process == nil {
				s.mu.Unlock()
				return fmt.Errorf("opencode serve process not started")
			}

			if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
				s.mu.Unlock()
				return fmt.Errorf("opencode serve process exited unexpectedly")
			}
			s.mu.Unlock()

			// Probe the HTTP endpoint to verify the server is actually ready
			probeURL := fmt.Sprintf("%s/session", s.baseURL)
			req, err := http.NewRequestWithContext(ctx, "GET", probeURL, nil)
			if err != nil {
				// Context canceled or invalid URL, try again next tick
				continue
			}

			resp, err := client.Do(req)
			if err != nil {
				// Server not ready yet, try again next tick
				continue
			}
			resp.Body.Close()

			// Any response means the server is ready
			s.mu.Lock()
			s.ready = true
			s.mu.Unlock()
			return nil
		}
	}
}

// Stop gracefully terminates the serve process with a grace period
func (s *OmOSupervisor) Stop(gracePeriod time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	if s.cmd.Process == nil {
		s.running = false
		return nil
	}

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case err := <-done:
		s.running = false
		return fmt.Errorf("wait for process exit: %w", err)
	case <-time.After(gracePeriod):
		if err := s.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("kill opencode serve: %w", err)
		}
		s.running = false
		return nil
	}
}

// Status returns current supervisor status
func (s *OmOSupervisor) Status() (running bool, ready bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running, s.ready
}
