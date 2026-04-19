package build

import (
	"context"
	"fmt"
)

// TestcontainersSandbox is a stub for the testcontainers-based sandbox.
// It will be implemented in a future iteration (F135-02 partial).
type TestcontainersSandbox struct{}

// NewTestcontainersSandbox returns an error indicating the feature is not yet implemented.
func NewTestcontainersSandbox() (*TestcontainersSandbox, error) {
	return nil, fmt.Errorf("testcontainers sandbox not yet fully implemented (F135-02 partial)")
}

// Build is unimplemented.
func (s *TestcontainersSandbox) Build(ctx context.Context, dir string) (*SandboxResult, error) {
	return nil, fmt.Errorf("testcontainers sandbox not yet fully implemented (F135-02 partial)")
}

// Test is unimplemented.
func (s *TestcontainersSandbox) Test(ctx context.Context, dir string) (*SandboxResult, error) {
	return nil, fmt.Errorf("testcontainers sandbox not yet fully implemented (F135-02 partial)")
}

// Cleanup is unimplemented.
func (s *TestcontainersSandbox) Cleanup() error {
	return nil
}
