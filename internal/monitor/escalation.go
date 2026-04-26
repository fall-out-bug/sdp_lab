// Package monitor provides agent health monitoring capabilities.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// WitnessEvent represents a Gas Town Witness stuck detection event
type WitnessEvent struct {
	AgentID       string            `json:"agent_id"`
	TaskID        string            `json:"task_id"`
	LastAction    string            `json:"last_action"`
	StuckDuration time.Duration     `json:"stuck_duration"`
	DetectedAt    time.Time         `json:"detected_at"`
	Context       map[string]string `json:"context"`
}

// EscalationBridge handles Witness → Beads escalation
type EscalationBridge struct {
	bdBinary    string
	beadsDir    string
	wispTimeout time.Duration
}

// NewEscalationBridge creates a new escalation bridge
func NewEscalationBridge() *EscalationBridge {
	bdBinary := "bd"
	if path := os.Getenv("BD_BINARY"); path != "" {
		bdBinary = path
	}

	beadsDir := ".beads"
	if path := os.Getenv("BEADS_DIR"); path != "" {
		beadsDir = path
	}

	return &EscalationBridge{
		bdBinary:    bdBinary,
		beadsDir:    beadsDir,
		wispTimeout: 24 * time.Hour,
	}
}

// HandleStuckAgent creates a Beads wisp for a stuck agent
func (b *EscalationBridge) HandleStuckAgent(event WitnessEvent) error {
	// Use existing escalation handler
	handler := newEscalationHandler(escalationConfig{
		CreateWisp: true,
	})

	return handler.escalate(context.Background(), event.AgentID, event.DetectedAt)
}

// RunGTWitnessCommand runs the gt witness command to get stuck agents
func (b *EscalationBridge) RunGTWitnessCommand() ([]WitnessEvent, error) {
	cmd := exec.Command("gt", "witness", "stuck", "--json")
	output, err := cmd.Output()
	if err != nil {
		// If gt command is not available, return empty list
		// This is expected in development/testing environments
		if strings.Contains(err.Error(), "executable file not found") {
			return []WitnessEvent{}, nil
		}
		return nil, fmt.Errorf("failed to run gt witness stuck: %w", err)
	}

	var events []WitnessEvent
	if err := json.Unmarshal(output, &events); err != nil {
		return nil, fmt.Errorf("failed to parse witness events JSON: %w", err)
	}

	return events, nil
}

// isValidNotifyCommand validates that a notify command is safe to execute.

// isValidNotifyCommand validates that a notify command is safe to execute.
// It checks against a whitelist of allowed commands and prevents shell injection.
func isValidNotifyCommand(cmd string) bool {
	if cmd == "" {
		return false
	}

	if strings.ContainsAny(cmd, ";|&`$<>") {
		return false
	}

	// Whitelist of allowed base commands
	allowed := map[string]bool{
		"bd":     true,
		"notify": true,
		"echo":   true,
	}

	parts, err := parseNotifyCommand(cmd)
	if err != nil || len(parts) == 0 {
		return false
	}
	base := parts[0]

	// Check against whitelist
	return allowed[base]
}

// parseNotifyCommand splits a command string into command and arguments safely.
// It handles quoted arguments and prevents shell injection.
func parseNotifyCommand(cmd string) ([]string, error) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return nil, fmt.Errorf("empty command")
	}

	parts := make([]string, 0, 4)
	var current strings.Builder
	var quote rune
	escaped := false

	for _, r := range cmd {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}

		if r == '\\' {
			escaped = true
			continue
		}

		if quote != 0 {
			if r == quote {
				quote = 0
			} else {
				current.WriteRune(r)
			}
			continue
		}

		switch r {
		case '\'', '"':
			quote = r
		case ' ', '\t', '\n', '\r':
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}

	if escaped {
		return nil, fmt.Errorf("invalid trailing escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid command")
	}

	return parts, nil
}

// escalationHandler handles escalation of stuck agents.
type escalationHandler struct {
	createWisp bool
	notifyCmd  string
	onEscalate func(sessionID string, lastEvent time.Time)
}

// escalationConfig configures escalation handler.
type escalationConfig struct {
	// CreateWisp determines if a Beads wisp should be created.
	CreateWisp bool

	// NotifyCommand is a command to run on escalation.
	// The command receives SESSION_ID and LAST_EVENT environment variables.
	NotifyCommand string

	// OnEscalate is called when an escalation occurs.
	OnEscalate func(sessionID string, lastEvent time.Time)
}

// newEscalationHandler creates a new escalation handler.
func newEscalationHandler(cfg escalationConfig) *escalationHandler {
	return &escalationHandler{
		createWisp: cfg.CreateWisp,
		notifyCmd:  cfg.NotifyCommand,
		onEscalate: cfg.OnEscalate,
	}
}

// escalate handles an escalation event.
func (eh *escalationHandler) escalate(ctx context.Context, sessionID string, lastEvent time.Time) error {
	if ctx == nil {
		ctx = context.Background()
	}

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
			return fmt.Errorf("run notify command: %w", err)
		}
	}

	return nil
}

// createBeadsWisp creates an ephemeral Beads issue for stuck agent.
func (eh *escalationHandler) createBeadsWisp(ctx context.Context, sessionID string, lastEvent time.Time) error {
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
func (eh *escalationHandler) runNotifyCommand(ctx context.Context, sessionID string, lastEvent time.Time) error {
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
