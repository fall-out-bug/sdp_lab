package llmclient_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

func TestChat_success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer testkey" {
			http.Error(w, "unauthorized", 401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"choices":[{"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5,"cost":0.0001}}`)
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	resp, err := c.Chat(context.Background(), llmclient.ChatRequest{
		Model:     "openai/gpt-4o",
		Messages:  []llmclient.Message{{Role: "user", Content: "hi"}},
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Content != "hello" {
		t.Errorf("Content = %q, want %q", resp.Content, "hello")
	}
	if resp.InputTokens != 10 {
		t.Errorf("InputTokens = %d, want 10", resp.InputTokens)
	}
	if resp.OutputTokens != 5 {
		t.Errorf("OutputTokens = %d, want 5", resp.OutputTokens)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("FinishReason = %q, want %q", resp.FinishReason, "stop")
	}
}

func TestChat_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", 429)
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	_, err := c.Chat(context.Background(), llmclient.ChatRequest{Model: "openai/gpt-4o"})
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
}

func TestStream_textDelta(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"he"},"finish_reason":null}]}`)
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"llo"},"finish_reason":"stop"}]}`)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	ch, err := c.Stream(context.Background(), llmclient.ChatRequest{
		Model:    "openai/gpt-4o",
		Messages: []llmclient.Message{{Role: "user", Content: "hi"}},
		Stream:   true,
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var events []llmclient.StreamEvent
	for ev := range ch {
		events = append(events, ev)
	}

	var textEvents []string
	for _, ev := range events {
		if ev.Type == "text_delta" {
			textEvents = append(textEvents, ev.Text)
		}
	}
	if len(textEvents) == 0 {
		t.Fatal("no text_delta events received")
	}
	combined := ""
	for _, t := range textEvents {
		combined += t
	}
	if combined != "hello" {
		t.Errorf("combined text = %q, want %q", combined, "hello")
	}

	// Last event must be finish
	last := events[len(events)-1]
	if last.Type != "finish" {
		t.Errorf("last event type = %q, want %q", last.Type, "finish")
	}
}

func TestStream_toolCall(t *testing.T) {
	toolChunk1 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`
	toolChunk2 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"cmd\":\"ls\"}"}}]},"finish_reason":"tool_calls"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: "+toolChunk1)
		fmt.Fprintln(w, "data: "+toolChunk2)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	ch, err := c.Stream(context.Background(), llmclient.ChatRequest{
		Model:  "openai/gpt-4o",
		Stream: true,
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var toolEv *llmclient.StreamEvent
	for ev := range ch {
		ev := ev
		if ev.Type == "tool_call" {
			toolEv = &ev
		}
	}
	if toolEv == nil {
		t.Fatal("no tool_call event received")
	}
	if toolEv.Tool == nil {
		t.Fatal("tool_call event has nil Tool")
	}
	if toolEv.Tool.Name != "bash" {
		t.Errorf("tool name = %q, want %q", toolEv.Tool.Name, "bash")
	}
	if toolEv.Tool.ID != "call_abc" {
		t.Errorf("tool ID = %q, want %q", toolEv.Tool.ID, "call_abc")
	}
	var args map[string]string
	if err := json.Unmarshal([]byte(toolEv.Tool.Arguments), &args); err != nil {
		t.Fatalf("unmarshal arguments: %v", err)
	}
	if args["cmd"] != "ls" {
		t.Errorf("tool arg cmd = %q, want %q", args["cmd"], "ls")
	}
}

func TestStream_toolCallEmptyID_getsUUID(t *testing.T) {
	// Provider returns empty tool call ID — llmclient must generate a UUID
	toolChunk := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"","type":"function","function":{"name":"bash","arguments":"{\"cmd\":\"ls\"}"}}]},"finish_reason":"tool_calls"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: "+toolChunk)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	ch, err := c.Stream(context.Background(), llmclient.ChatRequest{Model: "openai/gpt-4o", Stream: true})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var toolEv *llmclient.StreamEvent
	for ev := range ch {
		ev := ev
		if ev.Type == "tool_call" {
			toolEv = &ev
		}
	}
	if toolEv == nil {
		t.Fatal("no tool_call event")
	}
	if toolEv.Tool.ID == "" {
		t.Error("tool ID must not be empty when provider returns empty ID — llmclient must generate UUID")
	}
}

func TestNew_rejectEmptyKey(t *testing.T) {
	// New itself doesn't error — but Chat/Stream should return error immediately
	c := llmclient.New("", "http://localhost")
	_, err := c.Chat(context.Background(), llmclient.ChatRequest{})
	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
}

func TestStream_emptyKey(t *testing.T) {
	c := llmclient.New("", "http://localhost")
	_, err := c.Stream(context.Background(), llmclient.ChatRequest{})
	if err == nil {
		t.Error("expected error for empty API key on Stream, got nil")
	}
}

func TestStream_errorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {bad json")
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	ch, err := c.Stream(context.Background(), llmclient.ChatRequest{Model: "openai/gpt-4o", Stream: true})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	var hasError bool
	for ev := range ch {
		if ev.Type == "error" {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected error event for malformed SSE data")
	}
}

func TestStream_non200Status(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "server error", 500)
	}))
	defer srv.Close()

	c := llmclient.New("testkey", srv.URL)
	_, err := c.Stream(context.Background(), llmclient.ChatRequest{Model: "openai/gpt-4o", Stream: true})
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}
