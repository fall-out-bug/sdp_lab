package llm

import (
	"testing"
)

func TestNewOpenRouterClient(t *testing.T) {
	c := NewOpenRouterClient()
	if c == nil {
		t.Fatal("NewOpenRouterClient returned nil")
	}
	if c.HTTPClient == nil {
		t.Error("HTTPClient should be set")
	}
}
