package orchestrate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

// Exit codes for autonomous mode.
const (
	ExitAutonomousHumanRequired = 3
	ExitAutonomousStuckOrLimit  = 4
)

// Autonomous loop constants.
const (
	DefaultMaxIterations = 50
	MaxSameAction        = 3
)

// Sentinel errors returned by AutonomousDriver.Run.
var (
	ErrHumanRequired = errors.New("autonomous: human gate not in accept-gates allowlist")
	ErrLoopStuck     = errors.New("autonomous: loop stuck — same action repeated")
	ErrMaxIterations = errors.New("autonomous: max iterations exceeded")
)

// AutonomousConfig holds the configuration for the autonomous batch driver.
type AutonomousConfig struct {
	MaxIterations int
	AcceptGates   []string // e.g. ["review", "pr"]
	DryRun        bool
}

// AutonomousDriver drives the pull-FSM loop in batch mode. It calls the
// existing ComputeNextAction / Advance infrastructure on each iteration
// without spawning external processes (no opencode).
type AutonomousDriver struct {
	config     AutonomousConfig
	nextAction func(cp *Checkpoint, workstreams []string, projectRoot string) (*NextAction, error)
	advance    func(cp *Checkpoint, workstreams []string, result string) error
	save       func(dir string, cp *Checkpoint) error
	output     io.Writer // defaults to os.Stdout
}

// NewAutonomousDriver creates a driver with production defaults.
func NewAutonomousDriver(config AutonomousConfig) *AutonomousDriver {
	return &AutonomousDriver{
		config: config,
		nextAction: func(cp *Checkpoint, workstreams []string, projectRoot string) (*NextAction, error) {
			return ComputeNextAction(cp, workstreams, projectRoot)
		},
		advance: func(cp *Checkpoint, workstreams []string, result string) error {
			return AdvanceUnvalidated(cp, workstreams, result)
		},
		save: func(dir string, cp *Checkpoint) error {
			return SaveCheckpoint(dir, cp)
		},
		output: os.Stdout,
	}
}

// SetOutput sets the writer for autonomous loop messages (for testing).
func (d *AutonomousDriver) SetOutput(w io.Writer) {
	d.output = w
}

// SetAdvanceFn overrides the advance function (for testing stuck loops).
func (d *AutonomousDriver) SetAdvanceFn(fn func(cp *Checkpoint, workstreams []string, result string) error) {
	d.advance = fn
}

// Run executes the autonomous loop until done, error, or human-required.
// In dry-run mode, the checkpoint is cloned and advanced in-memory so that
// the full action sequence can be printed; the original checkpoint is not
// mutated and nothing is persisted to disk.
func (d *AutonomousDriver) Run(ctx context.Context, cp *Checkpoint, workstreams []string, projectRoot string, cpPath string) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Work on a copy in dry-run mode so the caller's checkpoint is untouched.
	var workCP *Checkpoint
	if d.config.DryRun {
		workCP = cloneCheckpoint(cp)
	} else {
		workCP = cp
	}

	history := make([]string, 0, d.config.MaxIterations)

	for i := 0; i < d.config.MaxIterations; i++ {
		select {
		case <-ctx.Done():
			// Best-effort save on interrupt (only in non-dry-run).
			if !d.config.DryRun {
				if saveErr := d.save(cpPath, workCP); saveErr != nil {
					slog.Error("failed to save checkpoint on shutdown", "error", saveErr)
				}
			}
			return ctx.Err()
		default:
		}

		action, err := d.nextAction(workCP, workstreams, projectRoot)
		if err != nil {
			return fmt.Errorf("iteration %d: compute next action: %w", i+1, err)
		}

		actionName := action.Action
		if actionName == "done" || actionName == "" {
			slog.Info("autonomous complete", "iterations", i+1)
			fmt.Fprintf(d.output, "autonomous: done after %d iterations\n", i+1)
			return nil
		}

		// Build a composite key for stuck detection: "action:wsid" so that
		// advancing different workstreams (build ws-01 vs build ws-02) is
		// not considered stuck. For non-build phases, the key is just the
		// action name since those phases don't repeat per-workstream.
		stuckKey := actionName
		if action.WSID != "" {
			stuckKey = actionName + ":" + action.WSID
		}

		// Gate allowlist check.
		if isHumanGate(actionName) && !inAllowlist(actionName, d.config.AcceptGates) {
			slog.Warn("autonomous: human gate not in allowlist", "action", actionName)
			fmt.Fprintf(d.output, "autonomous: stopped at human gate: %s\n", actionName)
			return ErrHumanRequired
		}

		// Stuck detection: count consecutive occurrences of the same composite key.
		history = append(history, stuckKey)
		if sameActionCount(history, stuckKey) >= MaxSameAction {
			slog.Error("autonomous: loop stuck", "action", actionName, "key", stuckKey, "count", MaxSameAction)
			fmt.Fprintf(d.output, "autonomous: loop stuck on action %q (%d consecutive)\n", stuckKey, MaxSameAction)
			return ErrLoopStuck
		}

		// In dry-run mode, advance state in-memory for sequence preview but
		// skip disk persistence.
		if d.config.DryRun {
			fmt.Fprintf(d.output, "[dry-run] action: %s\n", actionName)
			if err := d.advance(workCP, workstreams, ""); err != nil {
				return fmt.Errorf("iteration %d: dry-run advance (%s): %w", i+1, actionName, err)
			}
			continue
		}

		// Execute the advance.
		if err := d.advance(workCP, workstreams, ""); err != nil {
			return fmt.Errorf("iteration %d: advance (%s): %w", i+1, actionName, err)
		}

		// Persist checkpoint after each successful advance.
		if err := d.save(cpPath, workCP); err != nil {
			return fmt.Errorf("iteration %d: save checkpoint: %w", i+1, err)
		}

		slog.Info("autonomous: advanced", "action", actionName, "iteration", i+1, "phase", workCP.Phase)
		fmt.Fprintf(d.output, "[autonomous] %d/%d: %s -> %s\n", i+1, d.config.MaxIterations, actionName, workCP.Phase)
	}

	slog.Error("autonomous: max iterations exceeded", "limit", d.config.MaxIterations)
	fmt.Fprintf(d.output, "autonomous: max iterations (%d) exceeded\n", d.config.MaxIterations)
	return ErrMaxIterations
}

// cloneCheckpoint creates a deep copy of a Checkpoint via JSON round-trip.
func cloneCheckpoint(cp *Checkpoint) *Checkpoint {
	data, err := json.Marshal(cp)
	if err != nil {
		// Should never happen with a valid Checkpoint.
		return cp
	}
	var copy Checkpoint
	if err := json.Unmarshal(data, &copy); err != nil {
		return cp
	}
	return &copy
}

// isHumanGate returns true if the given action requires human intervention
// in the autonomous loop. Actions like "review", "pr", "ci-loop", and "qa"
// are considered human gates because they typically require external approval
// or manual oversight. The "build" and "init" actions are always automated.
func isHumanGate(action string) bool {
	switch action {
	case "review", "pr", "ci-loop", "qa":
		return true
	default:
		return false
	}
}

// inAllowlist checks if an action is in the accept-gates allowlist.
func inAllowlist(action string, allowlist []string) bool {
	for _, a := range allowlist {
		if strings.EqualFold(a, action) {
			return true
		}
	}
	return false
}

// sameActionCount counts how many times the given action key appears consecutively
// at the tail of the history slice. It walks backward from history[len-1] and stops
// at the first mismatch. This is used for stuck-loop detection: if the autonomous
// driver emits the same action (e.g. "build:ws-01") MaxSameAction (3) times in a row
// without any intervening different action, the FSM is considered stuck and the loop
// aborts with ErrLoopStuck.
func sameActionCount(history []string, action string) int {
	count := 0
	for i := len(history) - 1; i >= 0; i-- {
		if history[i] == action {
			count++
		} else {
			break
		}
	}
	return count
}

// RunAutonomous is the high-level entry point called from main.go.
// It sets up the AutonomousDriver and runs the loop, mapping errors to
// the appropriate exit codes.
func RunAutonomous(ctx context.Context, config AutonomousConfig, projectRoot, featureID, cpPath string, cp *Checkpoint, workstreams []string) error {
	driver := NewAutonomousDriver(config)
	err := driver.Run(ctx, cp, workstreams, projectRoot, cpPath)
	if err == nil {
		return nil
	}

	// Map sentinel errors to structured exit codes.
	switch {
	case errors.Is(err, ErrHumanRequired):
		fmt.Fprintf(os.Stderr, "EXIT %d: %v\n", ExitAutonomousHumanRequired, err)
		return &AutonomousExitError{Code: ExitAutonomousHumanRequired, Err: err}
	case errors.Is(err, ErrLoopStuck), errors.Is(err, ErrMaxIterations):
		fmt.Fprintf(os.Stderr, "EXIT %d: %v\n", ExitAutonomousStuckOrLimit, err)
		return &AutonomousExitError{Code: ExitAutonomousStuckOrLimit, Err: err}
	default:
		return err
	}
}

// AutonomousExitError wraps an error with a specific exit code for the CLI.
type AutonomousExitError struct {
	Code int
	Err  error
}

func (e *AutonomousExitError) Error() string {
	return fmt.Sprintf("autonomous exit %d: %v", e.Code, e.Err)
}

func (e *AutonomousExitError) Unwrap() error {
	return e.Err
}

// ExitCode returns the exit code from an AutonomousExitError, or 1 for other errors.
func ExitCode(err error) int {
	var aerr *AutonomousExitError
	if errors.As(err, &aerr) {
		return aerr.Code
	}
	return 1
}
