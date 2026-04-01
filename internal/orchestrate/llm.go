package orchestrate

import (
	"context"
	"log/slog"

	"sdp_dev/internal/kernel"
)

// LLMInvoker invokes an LLM/agent (e.g. opencode) with a prompt.
// RunBuildPhase and RunReviewPhase use this for testability.
type LLMInvoker = kernel.RuntimeAdapter

// DefaultLLMInvoker uses InvokeOpenCode (opencode CLI subprocess).
// Switch to serve mode via SetDefaultInvoker.
var DefaultLLMInvoker LLMInvoker = openCodeInvoker{}

type openCodeInvoker struct{}

func (openCodeInvoker) Invoke(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	output, exitCode, err := InvokeOpenCode(ctx, req.WorkDir, req.Agent, req.Prompt)
	return kernel.RuntimeResult{Output: output, ExitCode: exitCode}, err
}

// SetDefaultInvoker replaces the global DefaultLLMInvoker.
// Call with omoclient.NewServeInvoker(...) to switch from subprocess to serve mode.
func SetDefaultInvoker(inv kernel.RuntimeAdapter) {
	DefaultLLMInvoker = inv
	slog.Info("LLM invoker switched")
}
