package executil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts exec.Command for testability.
// orchestrate, evidence, and guard use this instead of direct exec.
type CommandRunner interface {
	// Output runs the command and returns stdout. Stderr is discarded.
	Output(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// CombinedOutput runs the command and returns stdout+stderr.
	CombinedOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error)
	// Run runs the command without capturing output. Returns error on non-zero exit.
	Run(ctx context.Context, dir, name string, args ...string) error
}

// DefaultRunner uses os/exec.
var DefaultRunner CommandRunner = &defaultRunner{}

type defaultRunner struct{}

func (r *defaultRunner) Output(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func (r *defaultRunner) CombinedOutput(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return out, nil
}

func (r *defaultRunner) Run(ctx context.Context, dir, name string, args ...string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
