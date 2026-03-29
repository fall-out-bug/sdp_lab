package harness

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// CursorHarness wraps the cursor CLI tool as an agent harness.
type CursorHarness struct{}

// NewCursorHarness returns a new CursorHarness.
func NewCursorHarness() *CursorHarness {
	return &CursorHarness{}
}

// Name returns the harness identifier.
func (h *CursorHarness) Name() string {
	return "cursor"
}

// SupportedProviders returns the list of provider names this harness supports.
func (h *CursorHarness) SupportedProviders() []string {
	return []string{"cursor"}
}

// Available reports whether the cursor binary can be found on PATH.
func (h *CursorHarness) Available() bool {
	_, err := exec.LookPath("cursor")
	return err == nil
}

// Spawn starts a cursor agent process for the given opts and returns a Process.
//
// Command: cursor agent -p "prompt"
func (h *CursorHarness) Spawn(ctx context.Context, opts SpawnOpts) (*Process, error) {
	args := []string{"agent", "-p", opts.Prompt}
	args = append(args, opts.ExtraArgs...)

	var outBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "cursor", args...)
	cmd.Dir = opts.Worktree
	cmd.Stdout = &outBuf

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	done := make(chan Result, 1)
	proc := &Process{
		HarnessName: h.Name(),
		PID:         cmd.Process.Pid,
		Worktree:    opts.Worktree,
		StartedAt:   time.Now(),
		Done:        done,
	}

	go func() {
		start := proc.StartedAt
		err := cmd.Wait()
		code := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				code = exitErr.ExitCode()
			}
		}
		done <- Result{
			ExitCode: code,
			Duration: time.Since(start),
			Output:   outBuf.String(),
		}
	}()

	return proc, nil
}
