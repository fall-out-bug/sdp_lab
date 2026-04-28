package orchestrate

import (
	"context"
	"log/slog"
	"sync"

	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

// LLMInvoker invokes an LLM/agent (e.g. opencode) with a prompt.
// RunBuildPhase and RunReviewPhase use this for testability.
type LLMInvoker = kernel.RuntimeAdapter

var (
	defaultLLMInvoker LLMInvoker = openCodeInvoker{}
	defaultLLMMutex   sync.RWMutex
)

type openCodeInvoker struct{}

func (openCodeInvoker) Invoke(ctx context.Context, req kernel.RuntimeInvocation) (kernel.RuntimeResult, error) {
	output, exitCode, err := InvokeOpenCode(ctx, req.WorkDir, req.Agent, req.Prompt)
	return kernel.RuntimeResult{Output: output, ExitCode: exitCode}, err
}

// GetDefaultInvoker returns the current DefaultLLMInvoker in a thread-safe manner.
func GetDefaultInvoker() LLMInvoker {
	defaultLLMMutex.RLock()
	defer defaultLLMMutex.RUnlock()
	return defaultLLMInvoker
}

// SetDefaultInvoker replaces the global DefaultLLMInvoker in a thread-safe manner.
// Call with omoclient.NewServeInvoker(...) to switch from subprocess to serve mode.
func SetDefaultInvoker(inv kernel.RuntimeAdapter) {
	defaultLLMMutex.Lock()
	defer defaultLLMMutex.Unlock()
	defaultLLMInvoker = inv
	slog.Info("LLM invoker switched")
}
