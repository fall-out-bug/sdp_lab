package strataudit

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"unicode/utf8"
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
	embs, err := client.Embed(context.Background(), []string{"hello world", "test embedding"}, "openai/text-embedding-3-small")
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

func TestExtractFinalAnswer(t *testing.T) {
	tests := []struct {
		name     string
		reasoning string
		want     string
	}{
		{
			"answer tag",
			"Let me think...\n<answer>42</answer>",
			"42",
		},
		{
			"answer tag with surrounding text",
			"I will analyze this.\n<answer>The result is clear</answer>\nDone.",
			"The result is clear",
		},
		{
			"last paragraph",
			"First paragraph.\n\nSecond paragraph.\n\nFinal conclusion here",
			"Final conclusion here",
		},
		{
			"single paragraph",
			"Just one paragraph",
			"Just one paragraph",
		},
		{
			"empty paragraphs at end",
			"Content\n\n\n\nActual answer",
			"Actual answer",
		},
		{
			"answer tag empty",
			"<answer></answer>\n\nReal answer here",
			"Real answer here",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFinalAnswer(tt.reasoning)
			if got != tt.want {
				t.Errorf("extractFinalAnswer() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReasoningFallback(t *testing.T) {
	content := "hello"
	reasoning := "Let me think step by step.\n\nThe answer is forty-two."
	shortReasoning := "ok"

	tests := []struct {
		name        string
		content     *string
		reasoning   *string
		wantContent string
		wantErr     bool
	}{
		{
			"content only",
			&content,
			nil,
			"hello",
			false,
		},
		{
			"reasoning only (long enough)",
			nil,
			&reasoning,
			"The answer is forty-two.",
			false,
		},
		{
			"reasoning too short",
			nil,
			&shortReasoning,
			"",
			true,
		},
		{
			"both present (content wins)",
			&content,
			&reasoning,
			"hello",
			false,
		},
		{
			"both nil",
			nil,
			nil,
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the fallback logic from Chat()
			result := ""
			if tt.content != nil {
				result = *tt.content
			}
			if result == "" && tt.reasoning != nil {
				r := *tt.reasoning
				if utf8.RuneCountInString(r) >= 50 {
					result = extractFinalAnswer(r)
				}
			}
			gotErr := result == ""
			if gotErr != tt.wantErr {
				t.Errorf("gotErr=%v, wantErr=%v", gotErr, tt.wantErr)
			}
			if !gotErr && result != tt.wantContent {
				t.Errorf("got %q, want %q", result, tt.wantContent)
			}
		})
	}
}
