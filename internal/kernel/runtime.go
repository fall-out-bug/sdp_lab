package kernel

import "context"

type RuntimeInvocation struct {
	WorkDir string `json:"work_dir,omitempty"`
	Agent   string `json:"agent"`
	Prompt  string `json:"prompt"`
}

type RuntimeResult struct {
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code"`
}

type RuntimeAdapter interface {
	Invoke(ctx context.Context, req RuntimeInvocation) (RuntimeResult, error)
}
