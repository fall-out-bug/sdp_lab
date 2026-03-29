package harness

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// ClaudeHarness wraps the claude CLI tool as an agent harness.
type ClaudeHarness struct{}

// NewClaudeHarness returns a new ClaudeHarness.
func NewClaudeHarness() *ClaudeHarness {
	return &ClaudeHarness{}
}

// Name returns the harness identifier.
func (h *ClaudeHarness) Name() string {
	return "claude"
}

// SupportedProviders returns the list of provider names this harness supports.
func (h *ClaudeHarness) SupportedProviders() []string {
	return []string{"anthropic", "zai"}
}

// Available reports whether the claude binary can be found on PATH.
func (h *ClaudeHarness) Available() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}

// Spawn starts a claude process for the given opts and returns a Process.
//
// Command: claude -p "prompt" --output-format text [--model model]
func (h *ClaudeHarness) Spawn(ctx context.Context, opts SpawnOpts) (*Process, error) {
	args := []string{"-p", opts.Prompt, "--output-format", "text"}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	args = append(args, opts.ExtraArgs...)

	var outBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "claude", args...)
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
