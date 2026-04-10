package strataudit

import (
	"context"
	"encoding/json"
	"os"
	"testing"
)

func TestLLMClient_Chat_ReturnsJSON(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	client := NewLLMClient(key, "https://openrouter.ai/api/v1")
	resp, err := client.Chat(context.Background(), LLMRequest{
		Model:       "deepseek/deepseek-v3.2",
		System:      "Respond with valid JSON only.",
		User:        `Return {"status":"ok","count":42}`,
		MaxTokens:   200,
		Temperature: 0.0,
		JSONMode:    true,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Cached {
		t.Log("response was cached (OK for re-runs)")
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(resp.Content), &result); err != nil {
		t.Fatalf("parse JSON: %v\ncontent: %s", err, resp.Content)
	}
	if result["status"] != "ok" {
		t.Errorf("status = %v, want ok", result["status"])
	}
}

func TestLLMClient_Embed(t *testing.T) {
	key := os.Getenv("OPENROUTER_API_KEY")
	if key == "" {
		t.Skip("OPENROUTER_API_KEY not set")
	}

	client := NewLLMClient(key, "https://openrouter.ai/api/v1")
	embs, err := client.Embed(context.Background(), []string{"hello world", "test embedding"})
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(embs) != 2 {
		t.Fatalf("len(embs) = %d, want 2", len(embs))
	}
	if len(embs[0]) == 0 {
		t.Fatal("embedding is empty")
	}
	t.Logf("embedding dims: %d", len(embs[0]))
}

func TestParseLLMJSON(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{"clean json", `{"key":"val"}`, true},
		{"markdown wrapped", "```json\n{\"key\":\"val\"}\n```", true},
		{"with text before", `Here is the result: {"key":"val"}`, true},
		{"array", `[{"a":1},{"b":2}]`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseLLMJSON(tt.input)
			if (got != nil) != tt.wantOK {
				t.Errorf("ParseLLMJSON() = %v, wantOK=%v", got, tt.wantOK)
			}
		})
	}
}
