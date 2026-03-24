package omoclient

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// ServeInvoker implements orchestrate.LLMInvoker using OmO serve
type ServeInvoker struct {
	client     *OmOServeClient
	supervisor *OmOSupervisor
	logger     *log.Logger
}

// NewServeInvoker creates a new ServeInvoker
func NewServeInvoker(baseURL string, logger *log.Logger) *ServeInvoker {
	return &ServeInvoker{
		client:     NewClient(baseURL, logger),
		supervisor: NewOmOSupervisor(baseURL, logger),
		logger:     logger,
	}
}

// Invoke implements orchestrate.LLMInvoker
func (s *ServeInvoker) Invoke(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	if dir == "" {
		dir = "."
	}

	sessionID := fmt.Sprintf("session-%d", time.Now().UnixNano())

	_, err := s.client.CreateSession(CreateSessionRequest{
		Project: dir,
		Session: sessionID,
	})
	if err != nil {
		return "", 1, fmt.Errorf("create session: %w", err)
	}
	defer func() {
		_ = s.client.DeleteSession(sessionID)
	}()

	resp, err := s.client.SendMessageStream(prompt)
	if err != nil {
		return "", 1, fmt.Errorf("send message stream: %w", err)
	}
	defer resp.Body.Close()

	events := ReadSSEStream(ctx, resp.Body, s.logger)

	var output strings.Builder
	var exitCode int
	var lastError string

	for event := range events {
		s.logger.Printf("Event: %s, Prefix: %s, Data: %s", event.Class, event.Prefix, event.Data)

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
			s.logger.Printf("Warning: %s", event.Data)
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
		return "", 130, fmt.Errorf("invoke cancelled: %w", ctx.Err())
	}

	result := output.String()
	if result == "" && lastError != "" {
		result = lastError
	}

	if exitCode == 0 && result == "" {
		return "", 0, nil
	}

	return result, exitCode, nil
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
func (s *ServeInvoker) Status() (running bool, ready bool) {
	return s.supervisor.Status()
}
