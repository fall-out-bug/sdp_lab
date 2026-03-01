// Package monitor provides agent health monitoring capabilities.
package monitor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// EscalationHandler handles escalation of stuck agents.
type EscalationHandler struct {
	createWisp bool
	notifyCmd  string
	onEscalate func(sessionID string, lastEvent time.Time)
}

// EscalationConfig configures the escalation handler.
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

// createBeadsWisp creates an ephemeral Beads issue for the stuck agent.
func (eh *EscalationHandler) createBeadsWisp(ctx context.Context, sessionID string, lastEvent time.Time) error {
	title := fmt.Sprintf("STUCK: Agent session %s", sessionID)
	_ = title // Use in bd create when available

	description := fmt.Sprintf(
		"Agent session %s has been inactive since %s (> 5 minutes).\n\n"+
			"This is an automatically generated escalation. Investigate the session logs.",
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
	cmd := exec.CommandContext(ctx, "sh", "-c", eh.notifyCmd)
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
