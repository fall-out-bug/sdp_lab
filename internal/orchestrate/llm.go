package orchestrate

import (
	"context"
	"log/slog"
)

// LLMInvoker invokes an LLM/agent (e.g. opencode) with a prompt.
// RunBuildPhase and RunReviewPhase use this for testability.
type LLMInvoker interface {
	Invoke(ctx context.Context, dir, agent, prompt string) (output string, exitCode int, err error)
}

// DefaultLLMInvoker uses InvokeOpenCode (opencode CLI subprocess).
// Switch to serve mode via SetDefaultInvoker.
var DefaultLLMInvoker LLMInvoker = openCodeInvoker{}

type openCodeInvoker struct{}

func (openCodeInvoker) Invoke(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	return InvokeOpenCode(ctx, dir, agent, prompt)
}

// SetDefaultInvoker replaces the global DefaultLLMInvoker.
// Call with omoclient.NewServeInvoker(...) to switch from subprocess to serve mode.
func SetDefaultInvoker(inv LLMInvoker) {
	DefaultLLMInvoker = inv
	slog.Info("LLM invoker switched")
}
