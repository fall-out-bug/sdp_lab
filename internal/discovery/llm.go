package discovery

import (
	"context"

	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

// Message is re-exported from llmclient for backward compatibility within this package.
type Message = llmclient.Message

// ChatRequest is re-exported from llmclient for backward compatibility within this package.
type ChatRequest = llmclient.ChatRequest

// ChatResponse is re-exported from llmclient for backward compatibility within this package.
type ChatResponse = llmclient.ChatResponse

// LLMClient wraps llmclient.Client. Use NewLLMClient to construct.
type LLMClient struct {
	c *llmclient.Client
}

// NewLLMClient constructs an LLMClient backed by the shared llmclient package.
func NewLLMClient(apiKey, baseURL string) *LLMClient {
	return &LLMClient{c: llmclient.New(apiKey, baseURL)}
}

// Chat delegates to the underlying llmclient.Client.Chat.
func (l *LLMClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return l.c.Chat(ctx, req)
}
