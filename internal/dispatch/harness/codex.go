package harness

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// CodexHarness wraps the codex CLI tool as an agent harness.
type CodexHarness struct{}

// NewCodexHarness returns a new CodexHarness.
func NewCodexHarness() *CodexHarness {
	return &CodexHarness{}
}

// Name returns the harness identifier.
func (h *CodexHarness) Name() string {
	return "codex"
}

// SupportedProviders returns the list of provider names this harness supports.
func (h *CodexHarness) SupportedProviders() []string {
	return []string{"openai"}
}

// Available reports whether the codex binary can be found on PATH.
func (h *CodexHarness) Available() bool {
	_, err := exec.LookPath("codex")
	return err == nil
}

// Spawn starts a codex process for the given opts and returns a Process.
//
// Command: codex exec --full-auto -q "prompt" [--model model]
func (h *CodexHarness) Spawn(ctx context.Context, opts SpawnOpts) (*Process, error) {
	args := []string{"exec", "--full-auto", "-q", opts.Prompt}
	if opts.Model != "" {
		args = append(args, "--model", opts.Model)
	}
	args = append(args, opts.ExtraArgs...)

	var outBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "codex", args...)
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
