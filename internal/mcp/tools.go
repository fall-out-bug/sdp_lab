package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
)

// CommandExecutor abstracts running CLI commands. Production code uses
// realExecutor; tests inject mockExecutor.
type CommandExecutor interface {
	// Run executes the sdp CLI with the given subcommand and arguments.
	Run(ctx context.Context, args ...string) ([]byte, error)

	// RunCustom executes a named binary with the given arguments.
	// This is used for non-sdp CLIs like sdp-dispatch or bd.
	RunCustom(ctx context.Context, binary string, args ...string) ([]byte, error)
}

// realExecutor runs actual CLI commands via exec.CommandContext.
type realExecutor struct {
	binaryPath string
}

func (r *realExecutor) Run(ctx context.Context, args ...string) ([]byte, error) {
	return r.RunCustom(ctx, r.binaryPath, args...)
}

func (r *realExecutor) RunCustom(ctx context.Context, binary string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", binary, args, err, stderr.String())
	}
	return stdout.Bytes(), nil
}
