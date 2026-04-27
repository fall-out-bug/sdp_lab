package harness

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// OpenCodeHarness wraps the opencode CLI tool as an agent harness.
type OpenCodeHarness struct{}

// NewOpenCodeHarness returns a new OpenCodeHarness.
func NewOpenCodeHarness() *OpenCodeHarness {
	return &OpenCodeHarness{}
}

// Name returns the harness identifier.
func (h *OpenCodeHarness) Name() string {
	return "opencode"
}

// SupportedProviders returns the list of provider names this harness supports.
func (h *OpenCodeHarness) SupportedProviders() []string {
	return []string{"zai", "openai", "anthropic"}
}

// Available reports whether the opencode binary can be found on PATH.
func (h *OpenCodeHarness) Available() bool {
	_, err := exec.LookPath("opencode")
	return err == nil
}

// buildArgs constructs the opencode CLI args for the given opts.
// When opts.Model is non-empty, -m is appended in opencode's required
// "provider/model" form. If the caller passes a bare model id without a
// provider prefix, "ollama/" is auto-prepended (Ollama is the canonical
// local-tier provider in F145). Extracted for testability.
func (h *OpenCodeHarness) buildArgs(opts SpawnOpts) []string {
	args := []string{"run", "--agent", opts.Agent}
	if opts.Model != "" {
		model := opts.Model
		if !strings.Contains(model, "/") {
			model = "ollama/" + model
		}
		args = append(args, "-m", model)
	}
	args = append(args, opts.ExtraArgs...)
	return args
}

// Spawn starts an opencode process for the given opts and returns a Process.
//
// Command: opencode run --agent <agent> [-m <provider/model>]
// The prompt is passed via stdin.
func (h *OpenCodeHarness) Spawn(ctx context.Context, opts SpawnOpts) (*Process, error) {
	args := h.buildArgs(opts)

	var outBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, "opencode", args...)
	cmd.Dir = opts.Worktree
	cmd.Stdin = strings.NewReader(opts.Prompt)
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
