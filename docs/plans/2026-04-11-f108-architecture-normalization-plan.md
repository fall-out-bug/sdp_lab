# F108: Architecture Normalization — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close five architectural gaps that prevent the SDP system from assembling into a coherent whole: unified LLM client, LiveGateway wiring, ErrHarnessTerminated sentinel, Discovery→Delivery contract, and role documentation.

**Architecture:** `internal/llmclient` becomes the single HTTP layer for all LLM calls. `internal/agentloop/livegw` implements `agentloop.ModelGateway` using SSE streaming. `cmd/sdp-harness` is wired to `livegw.LiveGateway` instead of `StubGateway`. GO verdict creates a `FeatureCard` and workstream stub.

**Tech Stack:** Go 1.22+, `net/http`, SSE (text/event-stream), `bufio.Scanner`, `github.com/google/uuid`, SQLite (`modernc.org/sqlite`), OpenRouter API

**Design reference:** `docs/plans/2026-04-11-f108-architecture-normalization-design.md`

---

## Workstream dependency order

Run in this order:
1. WS-01 and WS-04 and WS-06 and WS-07 in parallel (independent)
2. WS-02 after WS-01
3. WS-03 after WS-01
4. WS-05 after WS-03 and WS-04

---

## WS-01: `internal/llmclient` — Chat + Stream + SSE parser

**Files:**
- Create: `internal/llmclient/llmclient.go`
- Create: `internal/llmclient/sse.go`
- Create: `internal/llmclient/llmclient_test.go`

### Task 1.1: Write failing tests for Chat and Stream types

**Step 1: Create the test file**

```go
// internal/llmclient/llmclient_test.go
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
```

**Step 2: Run test to verify it fails**

```bash
cd /path/to/sdp_lab
go test ./internal/llmclient/... -v 2>&1 | head -20
```
Expected: FAIL — package `sdp_dev/internal/llmclient` does not exist.

### Task 1.2: Implement `internal/llmclient/llmclient.go`

**Step 1: Create the main client file**

```go
// internal/llmclient/llmclient.go
// Package llmclient is the single HTTP client for all OpenRouter LLM calls in SDP.
// All packages (discovery, architect, strataudit, agentloop/livegw) import this package.
// Never create a separate HTTP client for LLM calls elsewhere.
package llmclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message is a chat message with a role and text content.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Tool describes a tool the model may call.
type Tool struct {
	Type     string       `json:"type"` // always "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction is the function spec within a Tool.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"` // JSON Schema object
}

// ChatRequest is the request body for both Chat and Stream.
// Set Stream: true for SSE streaming (used by LiveGateway).
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
	Tools       []Tool    `json:"tools,omitempty"`
	Stream      bool      `json:"stream,omitempty"`
}

// ChatResponse is the response from a non-streaming Chat call.
type ChatResponse struct {
	Content      string
	FinishReason string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// StreamEvent is a single event from a streaming call.
// Type is one of: "text_delta", "tool_call", "finish", "error".
type StreamEvent struct {
	Type string        // "text_delta" | "tool_call" | "finish" | "error"
	Text string        // set when Type == "text_delta"
	Tool *ToolCallChunk // set when Type == "tool_call"
	Err  error         // set when Type == "error"
}

// ToolCallChunk is a finalized tool call emitted when finish_reason == "tool_calls".
// ID is always non-empty: if the provider returned empty, llmclient generates a UUID.
type ToolCallChunk struct {
	ID        string // tool call ID (never empty)
	Name      string // function name
	Arguments string // accumulated JSON arguments string
}

// Client is the LLM HTTP client. Create with New().
type Client struct {
	apiKey  string
	baseURL string
	http    *http.Client
}

// New creates a new Client. apiKey and baseURL are required at construction;
// calls will return errors if apiKey is empty.
func New(apiKey, baseURL string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		http:    &http.Client{Timeout: 120 * time.Second},
	}
}

// Chat sends a single non-streaming request and returns the full response.
// Use for discovery, architect, strataudit — any package needing request-response semantics.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if c.apiKey == "" {
		return nil, errors.New("llmclient: API key is required")
	}

	req.Stream = false
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int     `json:"prompt_tokens"`
			CompletionTokens int     `json:"completion_tokens"`
			Cost             float64 `json:"cost"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if len(out.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}
	return &ChatResponse{
		Content:      out.Choices[0].Message.Content,
		FinishReason: out.Choices[0].FinishReason,
		InputTokens:  out.Usage.PromptTokens,
		OutputTokens: out.Usage.CompletionTokens,
		CostUSD:      out.Usage.Cost,
	}, nil
}

func (c *Client) setHeaders(r *http.Request) {
	r.Header.Set("Authorization", "Bearer "+c.apiKey)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("HTTP-Referer", "https://github.com/fall-out-bug/sdp_lab")
	r.Header.Set("X-Title", "SDP")
}
```

**Step 2: Create `internal/llmclient/sse.go` with Stream and SSE parser**

```go
// internal/llmclient/sse.go
package llmclient

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

// Stream opens an SSE connection and returns a channel of StreamEvents.
// The channel is closed when the server sends "data: [DONE]" or on error.
// Use for agentloop/livegw — any package needing token-by-token streaming.
func (c *Client) Stream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("llmclient: API key is required")
	}

	req.Stream = true
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	c.setHeaders(httpReq)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	ch := make(chan StreamEvent, 16)
	go func() {
		defer close(ch)
		defer func() { _ = resp.Body.Close() }()
		parseSSE(ctx, resp.Body, ch)
	}()
	return ch, nil
}

// toolCallAcc accumulates delta chunks for a single tool call by index.
type toolCallAcc struct {
	id        string
	name      string
	arguments strings.Builder
}

// parseSSE reads the SSE stream line-by-line, accumulates tool_call deltas,
// and sends StreamEvents to ch.
func parseSSE(ctx context.Context, r interface{ Read([]byte) (int, error) }, ch chan<- StreamEvent) {
	scanner := bufio.NewScanner(r)
	// tool_calls are accumulated across chunks; key = index
	acc := make(map[int]*toolCallAcc)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			ch <- StreamEvent{Type: "error", Err: ctx.Err()}
			return
		default:
		}

		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			ch <- StreamEvent{Type: "finish"}
			return
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			ch <- StreamEvent{Type: "error", Err: fmt.Errorf("unmarshal SSE chunk: %w", err)}
			return
		}
		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		finishReason := ""
		if chunk.Choices[0].FinishReason != nil {
			finishReason = *chunk.Choices[0].FinishReason
		}

		// Accumulate text delta
		if delta.Content != "" {
			ch <- StreamEvent{Type: "text_delta", Text: delta.Content}
		}

		// Accumulate tool call deltas by index
		for _, tc := range delta.ToolCalls {
			a, ok := acc[tc.Index]
			if !ok {
				a = &toolCallAcc{}
				acc[tc.Index] = a
			}
			if tc.ID != "" {
				a.id = tc.ID
			}
			if tc.Function.Name != "" {
				a.name = tc.Function.Name
			}
			a.arguments.WriteString(tc.Function.Arguments)
		}

		// On tool_calls finish: emit all accumulated tool calls
		if finishReason == "tool_calls" {
			for _, a := range acc {
				id := a.id
				if id == "" {
					id = uuid.NewString()
				}
				ch <- StreamEvent{
					Type: "tool_call",
					Tool: &ToolCallChunk{
						ID:        id,
						Name:      a.name,
						Arguments: a.arguments.String(),
					},
				}
			}
			ch <- StreamEvent{Type: "finish"}
			return
		}
	}

	if err := scanner.Err(); err != nil {
		ch <- StreamEvent{Type: "error", Err: fmt.Errorf("scanner: %w", err)}
	}
}
```

**Step 3: Run tests**

```bash
go test ./internal/llmclient/... -v -race
```
Expected: All tests PASS.

**Step 4: Verify build**

```bash
go build ./internal/llmclient/...
go vet ./internal/llmclient/...
```
Expected: No errors.

**Step 5: Commit**

```bash
git add internal/llmclient/
git commit -m "feat(llmclient): add unified LLM client with Chat and SSE Stream

Single HTTP entry point for all OpenRouter calls. Chat() for
request-response (discovery/architect/strataudit). Stream() for
SSE with tool_call chunk accumulation (livegw).

Part of F108 architecture normalization."
```

---

## WS-02: Migrate discovery/architect/strataudit to llmclient

**Files:**
- Modify: `internal/discovery/llm.go` — replace with thin wrapper over `llmclient`
- Modify: `internal/architect/llm_client.go` — replace HTTP calls with `llmclient.New()`
- Modify: `internal/strataudit/llmclient.go` — replace HTTP calls with `llmclient.New()`
- Modify: `internal/discovery/*.go` — update all call sites that reference `discovery.LLMClient`

**Depends on:** WS-01 complete.

### Task 2.1: Check all call sites in discovery

**Step 1: Find all usages of `LLMClient` in discovery**

```bash
grep -rn "LLMClient\|NewLLMClient\|\.Chat(" internal/discovery/ | grep -v "_test.go"
```

Note the list. Every `NewLLMClient` call becomes `llmclient.New()`. Every `discovery.ChatRequest` becomes `llmclient.ChatRequest`. Every `discovery.Message` becomes `llmclient.Message`.

**Step 2: Find all usages in architect and strataudit**

```bash
grep -rn "LLMClient\|LLMConfig\|NewLLM\|\.Chat\b" internal/architect/ internal/strataudit/ | grep -v "_test.go"
```

### Task 2.2: Replace `internal/discovery/llm.go`

**Step 1: Replace the file content**

The new `internal/discovery/llm.go` becomes a thin re-export adapter so callers inside `discovery/` don't need import path changes:

```go
// internal/discovery/llm.go
// LLMClient is an alias for llmclient.Client used within the discovery package.
// New code outside discovery should import sdp_dev/internal/llmclient directly.
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
```

**Step 2: Verify discovery still compiles**

```bash
go build ./internal/discovery/...
```
Expected: Success. No changes needed in callers because type aliases are used.

**Step 3: Run discovery tests**

```bash
go test ./internal/discovery/... -race
```
Expected: All PASS.

### Task 2.3: Replace `internal/architect/llm_client.go`

**Step 1: Read the current file to understand its shape**

```bash
cat internal/architect/llm_client.go
```

The architect client has `LLMConfig` with retry and `OPENROUTER_API_KEY` env var. Replace the HTTP implementation with `llmclient.New()`, preserving the retry wrapper and config struct:

```go
// internal/architect/llm_client.go
package architect

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

// LLMConfig configures the architect LLM client.
type LLMConfig struct {
	APIKey     string
	BaseURL    string
	MaxRetries int
	RetryDelay time.Duration
}

// DefaultLLMConfig returns config from environment variables.
func DefaultLLMConfig() LLMConfig {
	baseURL := os.Getenv("OPENROUTER_BASE_URL")
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return LLMConfig{
		APIKey:     os.Getenv("OPENROUTER_API_KEY"),
		BaseURL:    baseURL,
		MaxRetries: 3,
		RetryDelay: 2 * time.Second,
	}
}

// ArchitectLLMClient wraps llmclient.Client with retry logic for architect phase.
type ArchitectLLMClient struct {
	cfg    LLMConfig
	client *llmclient.Client
}

// NewArchitectLLMClient creates a client. Returns error if APIKey is empty.
func NewArchitectLLMClient(cfg LLMConfig) (*ArchitectLLMClient, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("architect: OPENROUTER_API_KEY is required")
	}
	return &ArchitectLLMClient{
		cfg:    cfg,
		client: llmclient.New(cfg.APIKey, cfg.BaseURL),
	}, nil
}

// Chat calls the LLM with retry on transient errors.
func (a *ArchitectLLMClient) Chat(ctx context.Context, req llmclient.ChatRequest) (*llmclient.ChatResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= a.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(a.cfg.RetryDelay):
			}
		}
		resp, err := a.client.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("after %d retries: %w", a.cfg.MaxRetries, lastErr)
}
```

**Step 2: Update callers in `internal/architect/` to use new types**

```bash
grep -rn "\.Chat(\|ChatRequest\|LLMClient\b" internal/architect/ | grep -v "llm_client.go"
```

For each call site: ensure `ChatRequest` now refers to `llmclient.ChatRequest`. Since `ArchitectLLMClient.Chat` takes `llmclient.ChatRequest`, update the import in caller files.

**Step 3: Compile and test**

```bash
go build ./internal/architect/...
go test ./internal/architect/... -race
```
Expected: All PASS.

### Task 2.4: Replace `internal/strataudit/llmclient.go`

The strataudit client has a rate limiter and SHA256 cache. Replace the HTTP layer with `llmclient.New()`, preserving the cache and rate limiter wrappers.

**Step 1: Read current strataudit llmclient**

```bash
cat internal/strataudit/llmclient.go
```

**Step 2: Replace HTTP calls with `llmclient.New()`**

The cache key and rate limiter remain. Only the actual HTTP call (`http.Post` or similar) is replaced by `c.inner.Chat(ctx, req)` where `c.inner` is a `*llmclient.Client`.

Pattern to follow:
```go
// Keep: SHA256-based cache, rate limiter, cache hit/miss logic
// Replace: direct http.Client usage → llmclient.Client.Chat()
import "github.com/fall-out-bug/sdp_lab/internal/llmclient"

type StratauditLLMClient struct {
    inner     *llmclient.Client
    cache     map[string]*llmclient.ChatResponse
    limiter   // existing rate limiter
}
```

**Step 3: Compile and test**

```bash
go build ./internal/strataudit/...
go test ./internal/strataudit/... -race
```
Expected: All PASS.

**Step 4: Full build check**

```bash
go build ./...
go vet ./...
```
Expected: No errors.

**Step 5: Commit**

```bash
git add internal/discovery/llm.go internal/architect/llm_client.go internal/strataudit/llmclient.go
git commit -m "refactor(llmclient): migrate discovery/architect/strataudit to shared llmclient

All three packages now delegate HTTP calls to internal/llmclient.
- discovery: type aliases preserve backward compat in-package
- architect: retry wrapper preserved, HTTP replaced
- strataudit: cache + rate limiter preserved, HTTP replaced

Part of F108 — AD-1: единый LLM-клиент."
```

---

## WS-03: `LiveGateway` in `internal/agentloop/livegw`

**Files:**
- Create: `internal/agentloop/livegw/livegw.go`
- Create: `internal/agentloop/livegw/livegw_test.go`

**Depends on:** WS-01 complete.

### Task 3.1: Write failing tests for LiveGateway

**Step 1: Create test file**

```go
// internal/agentloop/livegw/livegw_test.go
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

func TestIsAvailable(t *testing.T) {
	gw, err := livegw.New("testkey", "http://localhost")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// IsAvailable checks only for non-empty API key (key is already set)
	// It does not make network calls.
	if !gw.IsAvailable("openai/gpt-4o") {
		t.Error("IsAvailable returned false with a valid API key")
	}
}

func TestCall_textResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintln(w, `data: {"choices":[{"delta":{"content":"response text"},"finish_reason":"stop"}]}`)
		fmt.Fprintln(w, `data: [DONE]`)
	}))
	defer srv.Close()

	gw, _ := livegw.New("testkey", srv.URL)
	msgs := []agentloop.Message{{Role: "user", Content: "hello"}}
	cfg := agentloop.LoopConfig{Model: "openai/gpt-4o"}

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
	ch, err := gw.Call(context.Background(), nil, agentloop.LoopConfig{Model: "openai/gpt-4o"})
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
	if err := json.Unmarshal(tc.Input, &args); err != nil {
		t.Fatalf("unmarshal Input: %v", err)
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
		Model: "openai/gpt-4o",
		Tools: []agentloop.Tool{{
			Name:        "bash",
			Description: "run bash",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}}}`),
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
```

**Step 2: Run to verify failure**

```bash
go test ./internal/agentloop/livegw/... -v 2>&1 | head -20
```
Expected: FAIL — package does not exist.

### Task 3.2: Implement `internal/agentloop/livegw/livegw.go`

**Step 1: Check `agentloop.Event` and `agentloop.ToolCall` types**

```bash
grep -n "type Event\|type ToolCall\b\|type Tool\b\|ToolCalls\b" internal/agentloop/types.go
```

Note the exact field names. The key fields are:
- `Event.Type` (string)
- `Event.Delta` (string, for text_delta)
- `Event.ToolCalls` ([]ToolCall)
- `ToolCall.ID`, `ToolCall.Name`, `ToolCall.Input` (json.RawMessage)

**Step 2: Create `livegw.go`**

```go
// internal/agentloop/livegw/livegw.go
// Package livegw implements agentloop.ModelGateway using real LLM SSE streaming.
// It is the production ModelGateway for cmd/sdp-harness and any binary that
// runs SDP's internal agentloop.
//
// Architecture note:
// - livegw (this package) = SDP internal agent kernel connects to real LLM
// - ServeBridge (internal/executor/bridge_serve.go) = dispatches to external harness (Claude Code, Cursor, opencode)
// These two serve different roles and are not duplicates of each other.
package livegw

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fall-out-bug/sdp_lab/internal/agentloop"
	"github.com/fall-out-bug/sdp_lab/internal/llmclient"
)

// LiveGateway implements agentloop.ModelGateway via SSE streaming through llmclient.
type LiveGateway struct {
	client *llmclient.Client
}

// New creates a LiveGateway. Returns error if apiKey is empty.
// baseURL defaults to OpenRouter if empty: pass "" to use the default.
func New(apiKey, baseURL string) (*LiveGateway, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("livegw: OPENROUTER_API_KEY is required")
	}
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	return &LiveGateway{
		client: llmclient.New(apiKey, baseURL),
	}, nil
}

// IsAvailable reports whether the given model can be called.
// Does not make network calls. Returns true if the gateway has an API key set
// (key presence verified at construction time).
func (g *LiveGateway) IsAvailable(_ string) bool {
	// Key was validated in New(). If we have a LiveGateway, key is non-empty.
	return true
}

// Call converts []agentloop.Message to an llmclient SSE stream and maps
// llmclient StreamEvents to agentloop Events.
//
// Event mapping:
//   llmclient text_delta  → agentloop text_delta (Event.Delta)
//   llmclient tool_call   → agentloop tool_call (Event.ToolCalls)
//   llmclient finish      → agentloop turn_end + done
//   llmclient error       → agentloop error
func (g *LiveGateway) Call(
	ctx context.Context,
	msgs []agentloop.Message,
	cfg agentloop.LoopConfig,
) (<-chan agentloop.Event, error) {
	req := agentloop2llm(msgs, cfg)

	streamCh, err := g.client.Stream(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("livegw stream: %w", err)
	}

	out := make(chan agentloop.Event, 16)
	go func() {
		defer close(out)
		for ev := range streamCh {
			switch ev.Type {
			case "text_delta":
				out <- agentloop.Event{Type: "text_delta", Delta: ev.Text}

			case "tool_call":
				if ev.Tool == nil {
					continue
				}
				input := json.RawMessage(ev.Tool.Arguments)
				out <- agentloop.Event{
					Type: "tool_call",
					ToolCalls: []agentloop.ToolCall{{
						ID:    ev.Tool.ID,
						Name:  ev.Tool.Name,
						Input: input,
					}},
				}

			case "finish":
				out <- agentloop.Event{Type: "turn_end"}
				out <- agentloop.Event{Type: "done"}
				return

			case "error":
				errMsg := "unknown stream error"
				if ev.Err != nil {
					errMsg = ev.Err.Error()
				}
				out <- agentloop.Event{Type: "error", Error: errMsg}
				return
			}
		}
	}()

	return out, nil
}

// agentloop2llm converts agentloop types to an llmclient.ChatRequest.
func agentloop2llm(msgs []agentloop.Message, cfg agentloop.LoopConfig) llmclient.ChatRequest {
	lmsgs := make([]llmclient.Message, len(msgs))
	for i, m := range msgs {
		lmsgs[i] = llmclient.Message{Role: m.Role, Content: m.Content}
	}

	var tools []llmclient.Tool
	for _, t := range cfg.Tools {
		tools = append(tools, llmclient.Tool{
			Type: "function",
			Function: llmclient.ToolFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	return llmclient.ChatRequest{
		Model:     cfg.Model,
		Messages:  lmsgs,
		MaxTokens: cfg.MaxTokens,
		Tools:     tools,
		Stream:    true,
	}
}
```

**Step 3: Check agentloop.Event for correct field names**

```bash
grep -n "Delta\|Error\b\|ToolCalls\b" internal/agentloop/types.go | head -20
```

Adjust field names if they differ from `Delta` or `Error`. The test will show which fields are wrong.

**Step 4: Run tests**

```bash
go test ./internal/agentloop/livegw/... -v -race
```
Expected: All PASS.

**Step 5: Verify interface is satisfied**

Add a compile-time check at bottom of `livegw.go`:

```go
var _ agentloop.ModelGateway = (*LiveGateway)(nil)
```

**Step 6: Full build**

```bash
go build ./...
go vet ./...
```

**Step 7: Commit**

```bash
git add internal/agentloop/livegw/
git commit -m "feat(livegw): implement ModelGateway via SSE streaming

LiveGateway wraps llmclient.Stream() to satisfy agentloop.ModelGateway.
Maps llmclient events to agentloop events with tool_call accumulation.
IsAvailable: non-blocking, no network calls.

Part of F108 — AD-2: LiveGateway."
```

---

## WS-04: `ErrHarnessTerminated` sentinel

**Files:**
- Modify: `internal/agentloop/harness.go` — line ~363

**No dependencies. Run in parallel with WS-01.**

### Task 4.1: Write failing test for sentinel behavior

**Step 1: Find existing tests for harness termination**

```bash
grep -n "terminated\|ErrHarness\|errors.Is" internal/agentloop/harness_test.go 2>/dev/null || echo "no test file yet"
grep -rn "terminated\|ErrHarness" internal/agentloop/ | grep "_test.go"
```

**Step 2: Add test to `internal/agentloop/harness_test.go`**

Add this test to the existing test file:

```go
func TestErrHarnessTerminated_isSentinel(t *testing.T) {
	// ErrHarnessTerminated must be an exported sentinel variable, not a string-formatted error.
	// This ensures errors.Is() works correctly for callers.
	if agentloop.ErrHarnessTerminated == nil {
		t.Fatal("ErrHarnessTerminated is nil — must be errors.New(...)")
	}
	// Simulate what RestoreHarness returns for a terminated session
	wrapped := fmt.Errorf("restore: %w", agentloop.ErrHarnessTerminated)
	if !errors.Is(wrapped, agentloop.ErrHarnessTerminated) {
		t.Error("errors.Is failed — ErrHarnessTerminated must be wrappable with %%w")
	}
}
```

**Step 3: Run to verify failure**

```bash
go test ./internal/agentloop/... -run TestErrHarnessTerminated -v
```
Expected: FAIL — `agentloop.ErrHarnessTerminated undefined`.

### Task 4.2: Add sentinel variable to harness.go

**Step 1: Open `internal/agentloop/harness.go` and find the var block near the top**

```bash
grep -n "^var\|errInject\|ErrHarness" internal/agentloop/harness.go | head -10
```

**Step 2: Add the sentinel after the existing var block**

Find the line with `var errInjectFailure` (unexported sentinel pattern already exists). Add above or alongside it:

```go
// ErrHarnessTerminated is returned by RestoreHarness when the session was
// stopped via Stop() and cannot be resumed. Callers must use errors.Is() to
// detect this condition — never compare error strings directly.
var ErrHarnessTerminated = errors.New("harness: session was terminated")
```

**Step 3: Replace the string-formatted error at line ~363**

Find:
```go
return nil, fmt.Errorf("session %s was terminated by Stop() — cannot restore", sessionID)
```

Replace with:
```go
return nil, fmt.Errorf("session %s: %w", sessionID, ErrHarnessTerminated)
```

**Step 4: Check all callers that check for termination**

```bash
grep -rn "terminated\|was terminated" cmd/ internal/ | grep -v "_test.go" | grep -v "harness.go"
```

For each caller that does string comparison (`strings.Contains(err.Error(), "terminated")`), replace with `errors.Is(err, agentloop.ErrHarnessTerminated)`.

**Step 5: Run tests**

```bash
go test ./internal/agentloop/... -v -race
```
Expected: All PASS including `TestErrHarnessTerminated_isSentinel`.

**Step 6: Commit**

```bash
git add internal/agentloop/harness.go
git commit -m "fix(agentloop): make ErrHarnessTerminated an exported sentinel

Replaces fmt.Errorf string with var ErrHarnessTerminated = errors.New(...).
RestoreHarness wraps it with %%w. Callers use errors.Is().
Fixes F106 spec violation and brittle string comparison.

Part of F108 — AD-3: ErrHarnessTerminated sentinel."
```

---

## WS-05: Wire `sdp-harness` with LiveGateway + integration test

**Files:**
- Modify: `cmd/sdp-harness/main.go` — replace `NewStubGateway()` with `livegw.New()`
- Create: `cmd/sdp-harness/main_integration_test.go`

**Depends on:** WS-03 (LiveGateway) and WS-04 (ErrHarnessTerminated sentinel).

### Task 5.1: Write failing integration test

**Step 1: Create test file**

```go
// cmd/sdp-harness/main_integration_test.go
package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/agentloop"
)

// TestCmdNew_createSession verifies that `sdp-harness new` creates a session DB.
func TestCmdNew_createSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)

	err := cmdNew([]string{"--session=test-session-123"})
	if err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	dbFile := filepath.Join(dir, "test-session-123.db")
	if _, err := os.Stat(dbFile); err != nil {
		t.Fatalf("DB file not created: %v", err)
	}
}

// TestCmdRun_noAPIKey verifies that run fails with a clear error when OPENROUTER_API_KEY is absent.
// This is the key behavior change from F108: empty key must fail loudly.
func TestCmdRun_noAPIKey(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "") // explicitly empty

	// Create session first
	if err := cmdNew([]string{"--session=nokey-test"}); err != nil {
		t.Fatalf("cmdNew: %v", err)
	}

	err := cmdRun([]string{"--session=nokey-test", "--prompt=hello"})
	if err == nil {
		t.Fatal("expected error when OPENROUTER_API_KEY is empty, got nil")
	}
	// The error must mention API key, not a silent stub fallback.
	if !containsAny(err.Error(), "API key", "OPENROUTER_API_KEY", "required") {
		t.Errorf("error message %q does not mention API key requirement", err.Error())
	}
}

// TestCmdRun_terminatedSession verifies errors.Is(err, ErrHarnessTerminated) works.
func TestCmdRun_terminatedSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SDP_DATA_DIR", dir)
	t.Setenv("OPENROUTER_API_KEY", "testkey")

	// Manually create a terminated session via the store
	path := filepath.Join(dir, "term-test.db")
	store, err := agentloop.NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	_, err = agentloop.NewSession("term-test", store)
	if err != nil {
		t.Fatalf("new session: %v", err)
	}
	// Stop the session to mark it terminated
	h, err := agentloop.RestoreHarness("term-test", "", store, nil, nil, nil)
	if err != nil {
		t.Fatalf("restore for stop: %v", err)
	}
	h.Stop()
	_ = store.Close()

	// Now attempt to restore — should return ErrHarnessTerminated
	store2, _ := agentloop.NewSQLiteStore(path)
	defer store2.Close()
	_, err = agentloop.RestoreHarness("term-test", "", store2, nil, nil, nil)
	if !errors.Is(err, agentloop.ErrHarnessTerminated) {
		t.Errorf("expected ErrHarnessTerminated, got: %v", err)
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
```

**Step 2: Run to verify failures**

```bash
go test ./cmd/sdp-harness/... -v -run TestCmdRun_noAPIKey 2>&1
```
Expected: FAIL — `cmdRun` does not check for API key; `NewStubGateway` works with empty key.

### Task 5.2: Wire LiveGateway in `cmd/sdp-harness/main.go`

**Step 1: Open `main.go` and find `cmdRun`**

The critical section is around line 161:
```go
gateway := agentloop.NewStubGateway()
```

**Step 2: Replace with LiveGateway construction**

```go
// cmd/sdp-harness/main.go — in cmdRun(), replace the gateway construction block:

apiKey := os.Getenv("OPENROUTER_API_KEY")
gw, err := livegw.New(apiKey, "") // "" → uses default OpenRouter baseURL
if err != nil {
    return fmt.Errorf("create LiveGateway: %w\n(hint: set OPENROUTER_API_KEY env var)", err)
}
registry := agentloop.NewToolRegistry(nil)
router := agentloop.NewPhaseRouter(agentloop.DefaultPhaseMap, registry, gw, nil)
```

**Step 3: Add import for livegw**

```go
import (
    // existing imports...
    "github.com/fall-out-bug/sdp_lab/internal/agentloop/livegw"
)
```

**Step 4: Run tests**

```bash
go test ./cmd/sdp-harness/... -v -race
```
Expected: All PASS.

**Step 5: Full build**

```bash
go build ./cmd/sdp-harness/
```
Expected: Binary compiles. Test with:
```bash
./sdp-harness --help
```

**Step 6: Commit**

```bash
git add cmd/sdp-harness/main.go cmd/sdp-harness/main_integration_test.go
git commit -m "feat(sdp-harness): wire LiveGateway — replace StubGateway in production

- cmdRun reads OPENROUTER_API_KEY from env
- Empty key → explicit error (no silent stub fallback)
- Integration tests verify key-missing error and terminated-session sentinel

Part of F108 — AD-4: sdp-harness wire with LiveGateway."
```

---

## WS-06: Discovery → Delivery contract

**Files:**
- Modify: `internal/control/control.go` — add `DiscoveryDir` field to `FeatureCard`
- Modify: `cmd/sdp/cmd_discover.go` — add `createFeatureCard()` and `createWorkstreamStub()` called on GO verdict
- Create: `cmd/sdp/cmd_discover_test.go` (if it doesn't exist) or add to existing

**No dependencies. Run in parallel with WS-01.**

### Task 6.1: Write failing tests for GO verdict actions

**Step 1: Check for existing discover tests**

```bash
ls cmd/sdp/*discover*test* 2>/dev/null || echo "no test file"
```

**Step 2: Create/add test**

```go
// cmd/sdp/cmd_discover_test.go
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fall-out-bug/sdp_lab/internal/control"
	"github.com/fall-out-bug/sdp_lab/internal/discovery"
)

func TestCreateFeatureCard_onGO(t *testing.T) {
	dir := t.TempDir()
	store := control.NewStore(dir)

	frame := &discovery.FrameResult{
		ProblemStatement: "Users cannot track their goals",
		Scope:            "Goal tracking only; no social features",
	}
	hyp := &discovery.HypothesisResult{
		Requirements: []string{"Track goals", "Set reminders"},
	}
	discoveryDir := filepath.Join(dir, "docs/discovery/test-feature")

	card, err := createFeatureCard(store, "test-feature", frame, hyp, discoveryDir)
	if err != nil {
		t.Fatalf("createFeatureCard: %v", err)
	}
	if card.NormalizedIntent != frame.ProblemStatement {
		t.Errorf("NormalizedIntent = %q, want %q", card.NormalizedIntent, frame.ProblemStatement)
	}
	if card.DiscoveryDir != discoveryDir {
		t.Errorf("DiscoveryDir = %q, want %q", card.DiscoveryDir, discoveryDir)
	}
	if card.Status != "shaping" {
		t.Errorf("Status = %q, want %q", card.Status, "shaping")
	}
}

func TestCreateWorkstreamStub_onGO(t *testing.T) {
	dir := t.TempDir()
	wsDir := filepath.Join(dir, "docs/workstreams/backlog")
	_ = os.MkdirAll(wsDir, 0o755)

	frame := &discovery.FrameResult{
		ProblemStatement: "Users cannot track their goals",
		Scope:            "Goal tracking only",
	}
	hyp := &discovery.HypothesisResult{
		Requirements: []string{"Track goals", "Set reminders"},
	}
	discoveryDir := "docs/discovery/test-feature"
	featureID := "F999"
	wsID := "00-999-01"

	path, err := createWorkstreamStub(wsDir, wsID, featureID, frame, hyp, discoveryDir)
	if err != nil {
		t.Fatalf("createWorkstreamStub: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workstream file: %v", err)
	}
	s := string(content)

	if !strings.Contains(s, "ws_id: "+wsID) {
		t.Error("missing ws_id in frontmatter")
	}
	if !strings.Contains(s, "feature_id: "+featureID) {
		t.Error("missing feature_id in frontmatter")
	}
	if !strings.Contains(s, frame.ProblemStatement) {
		t.Error("missing ProblemStatement in content")
	}
	if !strings.Contains(s, discoveryDir) {
		t.Error("missing discoveryDir reference")
	}
	if !strings.Contains(s, "Track goals") {
		t.Error("missing acceptance criteria from requirements")
	}
}
```

**Step 3: Run to verify failure**

```bash
go test ./cmd/sdp/... -run TestCreateFeatureCard -v 2>&1 | head -20
```
Expected: FAIL — `createFeatureCard` undefined, `DiscoveryDir` undefined.

### Task 6.2: Add `DiscoveryDir` field to `FeatureCard`

**Step 1: Open `internal/control/control.go` at the FeatureCard struct (line ~23)**

Find the block with `NormalizedIntent`. Add `DiscoveryDir` alongside it:

```go
NormalizedIntent        string                 `yaml:"normalized_intent,omitempty" json:"normalized_intent,omitempty"`
DiscoveryDir            string                 `yaml:"discovery_dir,omitempty" json:"discovery_dir,omitempty"`
```

**Step 2: Verify control package still builds**

```bash
go build ./internal/control/...
```

### Task 6.3: Implement `createFeatureCard` in `cmd/sdp/cmd_discover.go`

**Step 1: Add the function at end of file**

```go
// createFeatureCard creates a control.FeatureCard from Discovery phase outputs.
// Called only on GO verdict. Returns the saved card with its assigned ID.
func createFeatureCard(
	store *control.Store,
	slug string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	discoveryDir string,
) (*control.FeatureCard, error) {
	card := &control.FeatureCard{
		Title:            slug,
		NormalizedIntent: frame.ProblemStatement,
		DiscoveryDir:     discoveryDir,
		Status:           "shaping",
		ScopeIn:          []string{frame.Scope},
	}
	if hyp != nil {
		card.AcceptanceShape = hyp.Requirements
	}
	if err := store.SaveCard(card); err != nil {
		return nil, fmt.Errorf("save feature card: %w", err)
	}
	return card, nil
}
```

**Step 2: Add the workstream stub function**

```go
// createWorkstreamStub writes a backlog workstream file for the first delivery step.
// Returns the path of the created file.
func createWorkstreamStub(
	wsDir, wsID, featureID string,
	frame *discovery.FrameResult,
	hyp *discovery.HypothesisResult,
	discoveryDir string,
) (string, error) {
	if err := os.MkdirAll(wsDir, 0o755); err != nil {
		return "", fmt.Errorf("create ws dir: %w", err)
	}

	var acLines strings.Builder
	if hyp != nil {
		for _, r := range hyp.Requirements {
			fmt.Fprintf(&acLines, "- [ ] %s\n", r)
		}
	}
	if acLines.Len() == 0 {
		acLines.WriteString("- [ ] (to be defined in planning phase)\n")
	}

	content := fmt.Sprintf(`---
ws_id: %s
feature_id: %s
status: backlog
priority: P2
size: M
depends_on: []
---

# %s: Delivery — Phase 1

Feature: %s

## Goal

%s

## Beads

_(issue will be created when workstream is claimed)_

## Acceptance Criteria

%s
## Out of Scope

%s

## Discovery Reference

Artifacts: %s/

## Implementation Notes

Read Discovery artifacts before planning. Start with the frame and hypothesis docs.
`, wsID, featureID, wsID, featureID,
		frame.ProblemStatement,
		acLines.String(),
		frame.Scope,
		discoveryDir,
	)

	path := filepath.Join(wsDir, wsID+".md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", fmt.Errorf("write workstream stub: %w", err)
	}
	return path, nil
}
```

### Task 6.4: Wire createFeatureCard and createWorkstreamStub into the GO verdict handler

**Step 1: Find the GO verdict section in `runDiscover`**

```bash
grep -n "VerdictGO\|GO\|createDiscoveryIssue\|beads issue" cmd/sdp/cmd_discover.go | head -20
```

The section is around line 208-215 (after `createDiscoveryIssue` call).

**Step 2: Add calls for FeatureCard and workstream stub**

In `runDiscover()`, after the beads issue is created, add:

```go
// ── On GO verdict: create FeatureCard and workstream stub ──────────
if validation != nil && validation.FinalVerdict == discovery.VerdictGO {
    fmt.Printf("\n🃏 Creating FeatureCard...\n")
    // Derive slug from idea (lowercase, spaces to dashes)
    slug := strings.ToLower(strings.ReplaceAll(idea, " ", "-"))
    
    // control.Store uses project root — find it relative to output dir
    projectRoot := filepath.Dir(filepath.Dir(absOut)) // absOut = docs/discovery/slug/
    store := control.NewStore(projectRoot)
    
    card, err := createFeatureCard(store, slug, frame, hypothesis, absOut)
    if err != nil {
        fmt.Fprintf(os.Stderr, "   warning: could not create feature card: %v\n", err)
    } else {
        fmt.Printf("   feature card: %s\n", card.ID)
    }

    fmt.Printf("\n📝 Creating workstream stub...\n")
    wsDir := filepath.Join(projectRoot, "docs/workstreams/backlog")
    // Feature number from card ID or beads issue ID — use issueID for now
    // Use a placeholder wsID derived from the discovery dir name
    wsID := "00-NEW-01" // Operator sets real ID when picking up work
    featureID := "FNEW"
    if card != nil && card.ID != "" {
        featureID = card.ID
    }
    wsPath, err := createWorkstreamStub(wsDir, wsID, featureID, frame, hypothesis, absOut)
    if err != nil {
        fmt.Fprintf(os.Stderr, "   warning: could not create workstream stub: %v\n", err)
    } else {
        fmt.Printf("   workstream stub: %s\n", wsPath)
    }
}
```

**Note:** `wsID` and `featureID` are placeholders until the operator assigns a real feature number. The FeatureCard and workstream stub provide the context. The operator renames the file when assigning the feature number.

**Step 3: Add imports**

```go
import (
    // existing...
    "github.com/fall-out-bug/sdp_lab/internal/control"
)
```

**Step 4: Run tests**

```bash
go test ./cmd/sdp/... -v -race
```
Expected: All PASS.

**Step 5: Build**

```bash
go build ./cmd/sdp/
```

**Step 6: Commit**

```bash
git add internal/control/control.go cmd/sdp/cmd_discover.go
git commit -m "feat(discovery): create FeatureCard + workstream stub on GO verdict

On VerdictGO, cmd_discover now:
1. Creates control.FeatureCard with NormalizedIntent, DiscoveryDir, Status=shaping
2. Writes docs/workstreams/backlog/00-NEW-01.md with goal, AC, discovery ref

DiscoveryDir is the contract between Discovery and Delivery agents.
Delivery reads DiscoveryDir from FeatureCard — does not search by slug.

Part of F108 — AD-5: Discovery→Delivery contract."
```

---

## WS-07: Role documentation — modelgateway, ServeBridge, agentloop

**Files:**
- Create: `internal/modelgateway/README.md`
- Modify: `internal/executor/bridge_serve.go` — add package/type doc comment
- Modify: `internal/agentloop/harness.go` — add package doc clarifying role

**No dependencies. Run in parallel with WS-01.**

### Task 7.1: Create `internal/modelgateway/README.md`

```markdown
# internal/modelgateway

**Status:** Intentionally not wired (0 production callers as of 2026-04-11).

**Future role:** Multi-tenant LLM credential management for enterprise deployments.

## What this package is

`modelgateway` provides credential-aware LLM routing:
- `CredentialStore` — per-tenant API key storage
- `PolicyRouter` — route requests to providers based on policy
- `AuditLog` — log all LLM calls with provenance

This is an enterprise abstraction layer on top of `internal/llmclient`.

## What this package is NOT

- **Not the current LLM client.** Production code uses `internal/llmclient` directly.
- **Not wired to agentloop.** `agentloop.ModelGateway` is implemented by `internal/agentloop/livegw`.
- **Not deprecated.** It will be wired when multi-tenant credential management is needed.

## When to wire this

When SDP needs:
- Multiple tenants with separate API keys
- Per-request provider routing based on policy
- Full audit log of all LLM calls with provenance chain

Until then, use `internal/llmclient` directly.
```

### Task 7.2: Add role comments to ServeBridge and agentloop

**Step 1: Find the ServeBridge package comment**

```bash
head -20 internal/executor/bridge_serve.go
```

**Step 2: Add/update the package comment**

At the top of `bridge_serve.go`, ensure there is a comment like:

```go
// ServeBridge dispatches agent work to an external harness (opencode serve, Claude Code, Cursor).
// It is NOT the same as agentloop.Harness — they serve different roles:
//
//   - ServeBridge: SDP delegates implementation to an EXTERNAL agent (opencode/Claude/Cursor).
//     The external harness runs the code; SDP supervises and collects evidence.
//
//   - agentloop.Harness: SDP itself IS the agent, running its own internal LLM loop.
//     Used for SDP's autonomous phases (discovery analysis, planning, review).
//
// Both components are in production after F108. They are not duplicates.
```

**Step 3: Find the agentloop package comment**

```bash
head -10 internal/agentloop/harness.go
```

**Step 4: Add/update the package comment in harness.go**

```go
// Package agentloop implements SDP's internal LLM agent loop.
// SDP uses this when it is itself the agent — running its own phases
// (discovery shaping, architectural analysis, evidence review) through
// a structured FSM with gates.
//
// This is distinct from internal/executor.ServeBridge, which dispatches
// work to an EXTERNAL harness (Claude Code, Cursor, opencode serve).
// See internal/executor/bridge_serve.go for that path.
```

**Step 5: Commit**

```bash
git add internal/modelgateway/README.md internal/executor/bridge_serve.go internal/agentloop/harness.go
git commit -m "docs(arch): clarify roles of modelgateway, ServeBridge, agentloop

- modelgateway/README.md: documents intent, 0 callers status, future use
- bridge_serve.go: explains ServeBridge vs agentloop.Harness distinction
- harness.go: package comment clarifies SDP-as-agent role

Part of F108 — AD-6: explicit role documentation."
```

---

## Final: Full verification pass

After all 7 workstreams are complete:

**Step 1: Full build**

```bash
go build ./...
```
Expected: Success.

**Step 2: Full test suite**

```bash
go test ./... -race -count=1 2>&1 | tail -30
```
Expected: All PASS, no data races.

**Step 3: Vet**

```bash
go vet ./...
```
Expected: Clean.

**Step 4: Verify llmclient is the only HTTP LLM caller**

```bash
grep -rn "openrouter.ai\|chat/completions\|http.NewRequest.*chat" internal/ cmd/ | grep -v "_test.go" | grep -v "internal/llmclient/"
```
Expected: No results (all HTTP LLM calls go through `internal/llmclient`).

**Step 5: Verify ErrHarnessTerminated is sentinel**

```bash
grep -rn "errors.New.*terminated\|ErrHarnessTerminated" internal/agentloop/
grep -rn "terminated.*by Stop\|was terminated" internal/ cmd/ | grep -v "_test.go"
```
Expected: Only `ErrHarnessTerminated = errors.New(...)` in harness.go; no string-formatted "terminated" errors.

**Step 6: Verify StubGateway not used in production**

```bash
grep -rn "NewStubGateway\|StubGateway" cmd/ | grep -v "_test.go"
```
Expected: No results (StubGateway only in tests).

**Step 7: Final push**

```bash
git push
```
