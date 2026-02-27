package orchestrate

import "context"

// LLMInvoker invokes an LLM/agent (e.g. opencode) with a prompt.
// RunBuildPhase and RunReviewPhase use this for testability.
type LLMInvoker interface {
	Invoke(ctx context.Context, dir, agent, prompt string) (output string, exitCode int, err error)
}

// DefaultLLMInvoker uses InvokeOpenCode (opencode CLI).
var DefaultLLMInvoker LLMInvoker = openCodeInvoker{}

type openCodeInvoker struct{}

func (openCodeInvoker) Invoke(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	return InvokeOpenCode(ctx, dir, agent, prompt)
}
