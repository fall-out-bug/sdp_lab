package harness

import (
	"context"
	"time"
)

// ZAIProvider implements the Provider interface for the z.ai platform.
type ZAIProvider struct{}

// NewZAIProvider returns a new ZAIProvider instance.
func NewZAIProvider() *ZAIProvider {
	return &ZAIProvider{}
}

// Name returns the unique identifier for the z.ai provider.
func (z *ZAIProvider) Name() string {
	return "zai"
}

// Models returns the list of models supported by z.ai.
func (z *ZAIProvider) Models() []string {
	return []string{"glm-5", "glm-4.7"}
}

// CheckLimits returns zero limits indicating the source is currently unavailable.
func (z *ZAIProvider) CheckLimits(_ context.Context) (*Limits, error) {
	return &Limits{
		Total:     0,
		Used:      0,
		Window:    "",
		Source:    "unavailable",
		CheckedAt: time.Now(),
	}, nil
}
