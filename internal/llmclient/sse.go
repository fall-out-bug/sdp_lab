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

		// Emit text delta
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
