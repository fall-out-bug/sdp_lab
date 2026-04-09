package kernel

import "context"

// RuntimeInvocation contains the parameters for invoking an AI runtime.
type RuntimeInvocation struct {
	WorkDir string `json:"work_dir,omitempty"`
	Agent   string `json:"agent"`
	Prompt  string `json:"prompt"`
}

// RuntimeResult contains the output from an AI runtime invocation.
type RuntimeResult struct {
	Output   string `json:"output,omitempty"`
	ExitCode int    `json:"exit_code"`
}

// RuntimeAdapter abstracts the invocation of an AI coding runtime.
// Implementations may invoke runtimes via subprocess, API, or other mechanisms.
type RuntimeAdapter interface {
	Invoke(ctx context.Context, req RuntimeInvocation) (RuntimeResult, error)
}
