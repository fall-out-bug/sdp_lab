package omoclient

import (
	"context"
	"fmt"
	"log"
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
	logger  *log.Logger
}

// NewOmOSupervisor creates a new supervisor instance
func NewOmOSupervisor(baseURL string, logger *log.Logger) *OmOSupervisor {
	return &OmOSupervisor{
		baseURL: baseURL,
		logger:  logger,
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
	s.cmd.Stdout = s.logger.Writer()
	s.cmd.Stderr = s.logger.Writer()

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("start opencode serve: %w", err)
	}

	s.running = true
	s.ready = false
	return nil
}

// WaitReady waits for the serve process to be ready for connections
func (s *OmOSupervisor) WaitReady(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(deadline.Sub(time.Now())):
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
		return err
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
