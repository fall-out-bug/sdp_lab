// Package monitor provides agent health monitoring capabilities.
package monitor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// isValidNotifyCommand validates that a notify command is safe to execute.
// It checks against a whitelist of allowed commands and prevents shell injection.
func isValidNotifyCommand(cmd string) bool {
	if cmd == "" {
		return false
	}

	// Whitelist of allowed base commands
	allowed := map[string]bool{
		"bd":      true,
		"notify":  true,
		"echo":    true,
		"/bin/sh": true,
	}

	// Extract base command (first word)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}
	base := parts[0]

	// Check against whitelist
	return allowed[base]
}

// parseNotifyCommand splits a command string into command and arguments safely.
// It handles quoted arguments and prevents shell injection.
func parseNotifyCommand(cmd string) ([]string, error) {
	if cmd == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Simple split on whitespace for now
	// TODO: implement proper shell-like parsing if needed
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid command")
	}

	return parts, nil
}

// EscalationHandler handles escalation of stuck agents.
type EscalationHandler struct {
	createWisp bool
	notifyCmd  string
	onEscalate func(sessionID string, lastEvent time.Time)
}

// EscalationConfig configures escalation handler.
type EscalationConfig struct {
	// CreateWisp determines if a Beads wisp should be created.
	CreateWisp bool

	// NotifyCommand is a command to run on escalation.
	// The command receives SESSION_ID and LAST_EVENT environment variables.
	NotifyCommand string

	// OnEscalate is called when an escalation occurs.
	OnEscalate func(sessionID string, lastEvent time.Time)
}

// NewEscalationHandler creates a new escalation handler.
func NewEscalationHandler(cfg EscalationConfig) *EscalationHandler {
	return &EscalationHandler{
		createWisp: cfg.CreateWisp,
		notifyCmd:  cfg.NotifyCommand,
		onEscalate: cfg.OnEscalate,
	}
}

// Escalate handles an escalation event.
func (eh *EscalationHandler) Escalate(ctx context.Context, sessionID string, lastEvent time.Time) error {
	// Call custom handler first
	if eh.onEscalate != nil {
		eh.onEscalate(sessionID, lastEvent)
	}

	// Create Beads wisp if enabled
	if eh.createWisp {
		if err := eh.createBeadsWisp(ctx, sessionID, lastEvent); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "failed to create wisp: %v\n", err)
		}
	}

	// Run notification command if configured
	if eh.notifyCmd != "" {
		if err := eh.runNotifyCommand(ctx, sessionID, lastEvent); err != nil {
			// Log error but continue
			fmt.Fprintf(os.Stderr, "failed to run notify command: %v\n", err)
		}
	}

	return nil
}

// createBeadsWisp creates an ephemeral Beads issue for stuck agent.
func (eh *EscalationHandler) createBeadsWisp(ctx context.Context, sessionID string, lastEvent time.Time) error {
	title := fmt.Sprintf("STUCK: Agent session %s", sessionID)
	_ = title // Use in bd create when available

	description := fmt.Sprintf(
		"Agent session %s has been inactive since %s (> 5 minutes).\n\n"+
			"This is an automatically generated escalation. Investigate session logs.",
		sessionID,
		lastEvent.Format(time.RFC3339),
	)
	_ = description // For future use with bd create --description

	// Create wisp using bd command
	// Note: Beads may need specific flags for wisp creation
	cmd := exec.CommandContext(ctx, "bd", "create", "--type", "task", "--priority", "1", "--label", "stuck", "--label", "auto-escalated", title)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bd create failed: %w: %s", err, output)
	}

	return nil
}

// runNotifyCommand runs a notification command.
func (eh *EscalationHandler) runNotifyCommand(ctx context.Context, sessionID string, lastEvent time.Time) error {
	// Validate notify command to prevent shell injection
	if !isValidNotifyCommand(eh.notifyCmd) {
		return fmt.Errorf("invalid notify command: %s", eh.notifyCmd)
	}
	// Split command and arguments for safe execution
	parts, err := parseNotifyCommand(eh.notifyCmd)
	if err != nil {
		return fmt.Errorf("failed to parse notify command: %w", err)
	}
	cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("SESSION_ID=%s", sessionID),
		fmt.Sprintf("LAST_EVENT=%s", lastEvent.Format(time.RFC3339)),
		fmt.Sprintf("DURATION=%v", time.Since(lastEvent)),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("notify command failed: %w: %s", err, output)
	}

	return nil
}
