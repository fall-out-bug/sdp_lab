// Package livegw implements agentloop.ModelGateway using real LLM SSE streaming.
// It is the production ModelGateway for cmd/sdp-harness and any binary that
// runs SDP's internal agentloop.
//
// Architecture note:
//   - livegw (this package) = SDP internal agent kernel connects to real LLM
//   - ServeBridge (internal/executor/bridge_serve.go) = dispatches to external
//     harness (Claude Code, Cursor, opencode)
//
// These two serve different roles and are not duplicates of each other.
package livegw

import (
	"context"
	"encoding/json"
	"fmt"

	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/llmclient"
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
	req := convertRequest(msgs, cfg)

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
				out <- agentloop.Event{
					Type: "tool_call",
					ToolCalls: []agentloop.ToolCall{{
						ID:        ev.Tool.ID,
						Name:      ev.Tool.Name,
						Arguments: json.RawMessage(ev.Tool.Arguments),
					}},
				}

			case "finish":
				out <- agentloop.Event{Type: "turn_end"}
				out <- agentloop.Event{Type: "done"}
				return

			case "error":
				out <- agentloop.Event{Type: "error", Err: ev.Err}
				return
			}
		}
	}()

	return out, nil
}

// convertRequest maps agentloop types to an llmclient.ChatRequest.
func convertRequest(msgs []agentloop.Message, cfg agentloop.LoopConfig) llmclient.ChatRequest {
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
				Parameters:  t.Schema,
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

// Compile-time check that LiveGateway satisfies ModelGateway.
var _ agentloop.ModelGateway = (*LiveGateway)(nil)
