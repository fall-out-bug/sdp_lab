package livegw_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/agentloop"
	"github.com/fall-out-bug/sdp_lab/internal/agentloop/livegw"
)

func TestNew_rejectEmptyKey(t *testing.T) {
	_, err := livegw.New("", "http://localhost")
	if err == nil {
		t.Fatal("expected error for empty API key, got nil")
	}
}

func TestNew_acceptsValidKey(t *testing.T) {
	gw, err := livegw.New("testkey", "http://localhost")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if gw == nil {
		t.Fatal("expected non-nil LiveGateway")
	}
}

func TestIsAvailable(t *testing.T) {
	gw, _ := livegw.New("testkey", "http://localhost")

	// Allowed models
	for _, m := range []string{
		"glm-5",
		"glm-4.7",
		"anthropic/claude-sonnet-4.6",
		"anthropic/claude-opus-4.6",
		"openai/gpt-5.2-codex",
		"minimax/minimax-m2.5",
		"moonshotai/kimi-k2.5",
	} {
		if !gw.IsAvailable(m) {
			t.Errorf("IsAvailable(%q) = false, want true", m)
		}
	}

	// Disallowed models
	for _, m := range []string{
		"openai/gpt-4o",
		"unknown-model",
		"",
	} {
		if gw.IsAvailable(m) {
			t.Errorf("IsAvailable(%q) = true, want false", m)
		}
	}
}

func TestCall_textResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"response text"},"finish_reason":"stop"}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	msgs := []agentloop.Message{{Role: "user", Content: "hello"}}
	cfg := agentloop.LoopConfig{Model: "glm-5"}

	ch, err := gw.Call(context.Background(), msgs, cfg)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	var events []agentloop.Event
	for ev := range ch {
		events = append(events, ev)
	}

	hasTextDelta := false
	hasTurnEnd := false
	hasDone := false
	for _, ev := range events {
		switch ev.Type {
		case "text_delta":
			hasTextDelta = true
		case "turn_end":
			hasTurnEnd = true
		case "done":
			hasDone = true
		}
	}
	if !hasTextDelta {
		t.Error("expected text_delta event")
	}
	if !hasTurnEnd {
		t.Error("expected turn_end event")
	}
	if !hasDone {
		t.Error("expected done event")
	}
}

func TestCall_toolCall(t *testing.T) {
	toolChunk1 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_xyz","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}`
	toolChunk2 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"cmd\":\"pwd\"}"}}]},"finish_reason":"tool_calls"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: "+toolChunk1)
		fmt.Fprintln(w, "data: "+toolChunk2)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	ch, err := gw.Call(context.Background(), nil, agentloop.LoopConfig{Model: "glm-5"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	var toolEvent *agentloop.Event
	for ev := range ch {
		ev := ev
		if ev.Type == "tool_call" {
			toolEvent = &ev
		}
	}
	if toolEvent == nil {
		t.Fatal("no tool_call event received")
	}
	if len(toolEvent.ToolCalls) == 0 {
		t.Fatal("tool_call event has no ToolCalls")
	}
	tc := toolEvent.ToolCalls[0]
	if tc.Name != "bash" {
		t.Errorf("tool name = %q, want %q", tc.Name, "bash")
	}
	if tc.ID != "call_xyz" {
		t.Errorf("tool ID = %q, want %q", tc.ID, "call_xyz")
	}
	var args map[string]string
	if err := json.Unmarshal(tc.Arguments, &args); err != nil {
		t.Fatalf("unmarshal Arguments: %v", err)
	}
	if args["cmd"] != "pwd" {
		t.Errorf("tool arg cmd = %q, want %q", args["cmd"], "pwd")
	}
}

func TestCall_toolsPassedToRequest(t *testing.T) {
	var capturedBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedBody)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	cfg := agentloop.LoopConfig{
		Model: "glm-5",
		Tools: []agentloop.Tool{{
			Name:        "bash",
			Description: "run bash",
			Schema:      json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}}}`),
		}},
	}
	ch, _ := gw.Call(context.Background(), nil, cfg)
	for range ch {
	}

	tools, ok := capturedBody["tools"]
	if !ok {
		t.Fatal("request body missing 'tools' field")
	}
	toolsSlice, ok := tools.([]interface{})
	if !ok || len(toolsSlice) == 0 {
		t.Fatalf("tools field is not a populated array: %v", tools)
	}
}

func TestCall_errorEvent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: {bad json")
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	ch, err := gw.Call(context.Background(), nil, agentloop.LoopConfig{Model: "glm-5"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var hasError bool
	for ev := range ch {
		if ev.Type == "error" {
			hasError = true
			if ev.Err == nil {
				t.Error("error event Err field is nil")
			}
		}
	}
	if !hasError {
		t.Error("expected error event for malformed SSE data")
	}
}

func TestCall_toolCallUUIDFallback(t *testing.T) {
	// Provider sends tool_call without an "id" field — LiveGateway must generate one.
	toolChunk1 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"type":"function","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`
	toolChunk2 := `{"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"/tmp/x\"}"}}]},"finish_reason":"tool_calls"}]}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, "data: "+toolChunk1)
		fmt.Fprintln(w, "data: "+toolChunk2)
		fmt.Fprintln(w, "data: [DONE]")
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	ch, err := gw.Call(context.Background(), nil, agentloop.LoopConfig{Model: "glm-5"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	var toolEvent *agentloop.Event
	for ev := range ch {
		ev := ev
		if ev.Type == "tool_call" {
			toolEvent = &ev
		}
	}
	if toolEvent == nil {
		t.Fatal("no tool_call event received")
	}
	if len(toolEvent.ToolCalls) == 0 {
		t.Fatal("tool_call event has no ToolCalls")
	}
	tc := toolEvent.ToolCalls[0]
	if tc.ID == "" {
		t.Error("tool call ID is empty — LiveGateway should have generated a UUID")
	}
	// Verify it looks like a UUID (36 chars with hyphens)
	if len(tc.ID) != 36 {
		t.Errorf("generated UUID length = %d, want 36", len(tc.ID))
	}
	if tc.Name != "read_file" {
		t.Errorf("tool name = %q, want %q", tc.Name, "read_file")
	}
}
