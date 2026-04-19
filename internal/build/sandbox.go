package build

import (
	"context"
	"fmt"
	"time"
)

// Sandbox defines the interface for build sandboxing.
type Sandbox interface {
	// Build compiles the project in the given directory.
	Build(ctx context.Context, dir string) (*SandboxResult, error)
	// Test runs the project tests in the given directory.
	Test(ctx context.Context, dir string) (*SandboxResult, error)
	// Cleanup releases any sandbox resources.
	Cleanup() error
}

// SandboxResult holds sandbox execution results.
type SandboxResult struct {
	Success  bool          `json:"success"`
	ExitCode int           `json:"exit_code"`
	Stdout   string        `json:"stdout"`
	Stderr   string        `json:"stderr"`
	Duration time.Duration `json:"duration"`
}

// NewSandbox creates a sandbox implementation based on the sandbox type string.
func NewSandbox(sandboxType string, cgo bool) (Sandbox, error) {
	switch sandboxType {
	case "none", "":
		return NewNoneSandboxWithCGO(cgo), nil
	case "docker":
		return NewDockerSandbox(DockerSandboxConfig{
			CGO: cgo,
		})
	case "testcontainers":
		return nil, fmt.Errorf("testcontainers sandbox not yet fully implemented (F135-02 partial)")
	default:
		return nil, fmt.Errorf("build: unknown sandbox type %q (use docker, testcontainers, or none)", sandboxType)
	}
}

// NoneSandbox runs build/test directly via exec.Command without sandboxing.
type NoneSandbox struct {
	cgo bool
}

// NewNoneSandbox creates a sandbox that runs commands locally without isolation.
func NewNoneSandbox() *NoneSandbox {
	return &NoneSandbox{}
}

// NewNoneSandboxWithCGO creates a sandbox that runs commands locally with CGO enabled.
func NewNoneSandboxWithCGO(cgo bool) *NoneSandbox {
	return &NoneSandbox{cgo: cgo}
}

// Build runs go build ./... in the given directory.
func (s *NoneSandbox) Build(ctx context.Context, dir string) (*SandboxResult, error) {
	return runGoCommand(ctx, dir, s.cgo, "build", "./...")
}

// Test runs go test ./... in the given directory.
func (s *NoneSandbox) Test(ctx context.Context, dir string) (*SandboxResult, error) {
	return runGoCommand(ctx, dir, s.cgo, "test", "./...")
}

// Cleanup is a no-op for the none sandbox.
func (s *NoneSandbox) Cleanup() error {
	return nil
}
