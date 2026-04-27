package discovery_test

import (
	"context"
	"os"
	"testing"
	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestLLMClientChat_ReturnsJSON(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}
	c := discovery.NewLLMClient(key, "https://openrouter.ai/api/v1")
	resp, err := c.Chat(context.Background(), discovery.ChatRequest{
		Model: "deepseek/deepseek-v3.2",
		Messages: []discovery.Message{
			{Role: "system", Content: "Reply with valid JSON only."},
			{Role: "user", Content: `Return {"ok":true}`},
		},
		MaxTokens:   100,
		Temperature: 0.0,
	})
	if err != nil {
		t.Fatalf("chat: %v", err)
	}
	if resp.Content == "" {
		t.Fatal("empty content")
	}
	if resp.CostUSD < 0 {
		t.Fatal("negative cost")
	}
}
