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

	"github.com/google/uuid"
	"sdp_dev/internal/agentloop"
	"sdp_dev/internal/llmclient"
)

// LiveGateway implements agentloop.ModelGateway via SSE streaming through llmclient.
type LiveGateway struct {
	client        *llmclient.Client
	allowedModels map[string]bool
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
		allowedModels: map[string]bool{
			"glm-5":                       true,
			"glm-4.7":                     true,
			"anthropic/claude-sonnet-4.6": true,
			"anthropic/claude-opus-4.6":   true,
			"openai/gpt-5.2-codex":       true,
			"minimax/minimax-m2.5":        true,
			"moonshotai/kimi-k2.5":        true,
		},
	}, nil
}

// IsAvailable reports whether the given model can be called.
// MVP: static allowlist check, no network probe.
func (g *LiveGateway) IsAvailable(model string) bool {
	return g.allowedModels[model]
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
				// Safety: generate UUID if provider returned empty/missing tool_call.id.
				// llmclient normally handles this, but LiveGateway guarantees it at its own layer.
				tcID := ev.Tool.ID
				if tcID == "" {
					tcID = uuid.NewString()
				}
				out <- agentloop.Event{
					Type: "tool_call",
					ToolCalls: []agentloop.ToolCall{{
						ID:        tcID,
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
