package agentloop

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// findTool returns the Tool with the given name from the slice, or (zero, false).
func findTool(tools []Tool, name string) (Tool, bool) {
	for _, t := range tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// executeCalls executes tool calls in parallel goroutines.
//
// Key contracts from v14 design:
//   - Fix A5: BeforeToolCall is called BEFORE Execute; error = rejection (Execute is skipped).
//   - Fix A4: AfterToolCall error is NOT ignored — it is wrapped into ToolResult.Err.
//   - AfterToolCall is called even for rejected and tool-not-found cases (evidence must know about failures).
//   - AfterToolCall is called SYNCHRONOUSLY before wg.Done() — all callbacks complete before wg.Wait() returns.
//   - Results slice is indexed by call position — order matches input calls slice.
//   - Fix T1: ToolResult.Arguments is set from ToolCall.Arguments.
func executeCalls(ctx context.Context, calls []ToolCall, tools []Tool, cfg LoopConfig) []ToolResult {
	results := make([]ToolResult, len(calls))
	var wg sync.WaitGroup

	for i, call := range calls {
		wg.Add(1)
		go func(i int, call ToolCall) {
			defer wg.Done()

			// Fix A5: BeforeToolCall — pre-hook; error = call rejected, Execute is skipped.
			if cfg.BeforeToolCall != nil {
				if err := cfg.BeforeToolCall(call.Name, call.Arguments); err != nil {
					results[i] = ToolResult{
						ID:        call.ID,
						Name:      call.Name,
						Arguments: call.Arguments,
						Err:       fmt.Errorf("before hook rejected: %w", err),
					}
					// AfterToolCall still called for rejected calls (evidence must see failures).
					if cfg.AfterToolCall != nil {
						if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
							// Fix A4: wrap both errors — neither is silently dropped.
							results[i].Err = fmt.Errorf("%w; callback: %v", results[i].Err, cbErr)
						}
					}
					return
				}
			}

			// Find tool in allowlist.
			tool, ok := findTool(tools, call.Name)
			if !ok {
				results[i] = ToolResult{
					ID:        call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
					Err:       fmt.Errorf("tool not in phase allowlist: %q", call.Name),
				}
			} else {
				// Execute with per-tool timeout (30 s default).
				tctx, cancel := context.WithTimeout(ctx, 30*time.Second)
				defer cancel()
				out, err := tool.Execute(tctx, call.ID, call.Arguments)
				results[i] = ToolResult{
					ID:        call.ID,
					Name:      call.Name,
					Arguments: call.Arguments,
					Output:    out,
					Err:       err,
				}
			}

			// Fix A4+R6: AfterToolCall error is NOT ignored.
			// If tool.Execute also failed, BOTH errors are preserved — neither is dropped.
			if cfg.AfterToolCall != nil {
				if cbErr := cfg.AfterToolCall(results[i]); cbErr != nil {
					if results[i].Err != nil {
						// Wrap both: tool error + callback error.
						results[i].Err = fmt.Errorf("%w; callback: %v", results[i].Err, cbErr)
					} else {
						results[i].Err = fmt.Errorf("callback: %w", cbErr)
					}
				}
			}
		}(i, call)
	}

	wg.Wait() // all executions AND callbacks complete before returning
	return results
}

// makeCompletionSignalTool returns a Tool that sets flag.signaled=true and captures flag.summary.
// Fix R2-2: flag is an explicit pointer — no shared state via local vars, no race condition.
// BuildLoopConfig passes the flag; RunPhase reads it after draining the event channel.
// The Loop sees completion_signal as a normal tool call — it does NOT know about phases.
func makeCompletionSignalTool(flag *completionFlag) Tool {
	return Tool{
		Name:        "completion_signal",
		Description: "Signal that the current phase work is complete. Harness will run gate check and decide on transition.",
		Schema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"summary": {"type": "string", "description": "Brief summary of work completed in this phase"}
			},
			"required": ["summary"]
		}`),
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			var a struct {
				Summary string `json:"summary"`
			}
			// Unmarshal is best-effort; missing summary is allowed (warned by RunPhase, not here).
			_ = json.Unmarshal(args, &a)

			flag.mu.Lock()
			flag.signaled = true
			flag.summary = a.Summary
			flag.mu.Unlock()

			return "completion noted — harness will evaluate gate", nil
		},
	}
}

// Run is defined below (Task 8). See Run() for full documentation.
// Gateway field in LoopConfig (types.go) is used by Run() to call the LLM.
// StubGateway tests (Task 6) pass LoopConfig to StubGateway.Call; the Gateway field is
// set by BuildLoopConfig so Run() always has it wired correctly.

// Run executes one full phase-turn: trim context → LLM call → drain events → execute tools → loop.
// Run is stateless: it does not know about phases, gates, or session state.
// It exits when:
//
//	a) the LLM emits "done" and no tool calls were accumulated in that turn, OR
//	b) the LLM emits "done" and the completion_signal tool was called (flag set by closure), OR
//	c) ctx is cancelled (emits error event, closes channel).
//
// Design notes:
//   - Fix V3: cfg.ContextManager.Trim() called before every LLM call if non-nil.
//   - Fix Y1: "tool_end" events carry ToolID = result.ID (matches original ToolCall.ID).
//   - Fix X2: "tool_call" events carry ToolCalls slice (all parallel calls from one assistant message).
//   - Gateway field must be set in cfg; Run panics if cfg.Gateway is nil.
//   - Tool results are appended to msgs as "tool_result" Messages for the next LLM call.
//   - The completion_signal tool is expected to be included in cfg.Tools by BuildLoopConfig (Fix N6).
func Run(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	if cfg.Gateway == nil {
		return nil, fmt.Errorf("Run: cfg.Gateway must not be nil")
	}

	out := make(chan Event, 64)

	go func() {
		defer close(out)

		// Local flag for completion_signal detection.
		// Run() finds completion_signal in cfg.Tools (added by BuildLoopConfig).
		// We detect it by checking results after executeCalls.
		var completionSignalFired bool

		currentMsgs := make([]Message, len(msgs))
		copy(currentMsgs, msgs)

		for {
			// Fix V3: trim context before every LLM call.
			if cfg.ContextManager != nil {
				trimmed, err := cfg.ContextManager.Trim(currentMsgs, cfg.Model, cfg.MaxTokens)
				if err != nil {
					out <- Event{Type: "error", Err: fmt.Errorf("context trim: %w", err)}
					return
				}
				currentMsgs = trimmed
			}

			// Check context before calling gateway.
			select {
			case <-ctx.Done():
				out <- Event{Type: "error", Err: ctx.Err()}
				return
			default:
			}

			// Call the LLM via gateway.
			llmCh, err := cfg.Gateway.Call(ctx, currentMsgs, cfg)
			if err != nil {
				out <- Event{Type: "error", Err: fmt.Errorf("gateway call: %w", err)}
				return
			}

			// Drain LLM events; accumulate tool calls from this assistant turn.
			var assistantText string
			var pendingCalls []ToolCall

			for ev := range llmCh {
				switch ev.Type {
				case "text_delta":
					assistantText += ev.Delta
					out <- ev
				case "tool_call":
					// Fix X2: event carries all parallel calls from one assistant message.
					pendingCalls = append(pendingCalls, ev.ToolCalls...)
					out <- ev
				case "done":
					// "done" marks end of this assistant turn — do NOT forward yet.
					// We forward our own "done" at the very end of Run().
				case "error":
					out <- ev
					return
				default:
					// Forward unknown event types (future-proofing).
					out <- ev
				}
			}

			// Check context after draining LLM response.
			select {
			case <-ctx.Done():
				out <- Event{Type: "error", Err: ctx.Err()}
				return
			default:
			}

			// Build assistant message for conversation history.
			assistantMsg := Message{
				Role:      "assistant",
				Content:   assistantText,
				ToolCalls: pendingCalls,
			}
			currentMsgs = append(currentMsgs, assistantMsg)

			// If no tool calls, this turn is complete. Emit done and exit.
			if len(pendingCalls) == 0 {
				out <- Event{Type: "done"}
				return
			}

			// Execute all tool calls in parallel.
			results := executeCalls(ctx, pendingCalls, cfg.Tools, cfg)

			// Emit tool_end events and build tool_result messages.
			for _, result := range results {
				// Fix Y1: ToolID = result.ID (matches original ToolCall.ID).
				// Fix R3: ToolArguments carries original call args so RunPhase can
				// populate ToolResult.Arguments in TurnRecord (Fix T1).
				out <- Event{
					Type:          "tool_end",
					ToolID:        result.ID,
					ToolName:      result.Name,
					ToolResult:    result.Output,
					ToolErr:       result.Err,
					ToolArguments: result.Arguments,
				}

				// Check if this was the completion_signal tool.
				if result.Name == "completion_signal" && result.Err == nil {
					completionSignalFired = true
				}

				// Fix Z1: propagate tool error into message content so LLM can recover.
				content := result.Output
				if content == "" && result.Err != nil {
					content = fmt.Sprintf("Error: %v", result.Err)
				}
				currentMsgs = append(currentMsgs, Message{
					Role:       "tool_result",
					Content:    content,
					ToolCallID: result.ID, // Fix Y1: correlates to ToolCall.ID
				})
			}

			// If completion_signal fired, one more LLM call for the agent to
			// acknowledge, then we exit (Harness will read the flag after event drain).
			// We detect this by making the next LLM call and exiting after it.
			if completionSignalFired {
				// Make one final LLM call so the agent can acknowledge, then done.
				if cfg.ContextManager != nil {
					trimmed, err := cfg.ContextManager.Trim(currentMsgs, cfg.Model, cfg.MaxTokens)
					if err != nil {
						out <- Event{Type: "error", Err: fmt.Errorf("context trim (final): %w", err)}
						return
					}
					currentMsgs = trimmed
				}

				finalCh, err := cfg.Gateway.Call(ctx, currentMsgs, cfg)
				if err != nil {
					out <- Event{Type: "error", Err: fmt.Errorf("final gateway call: %w", err)}
					return
				}
				for ev := range finalCh {
					if ev.Type == "done" {
						break
					}
					if ev.Type == "error" {
						out <- ev
						return
					}
					out <- ev
				}
				out <- Event{Type: "done"}
				return
			}

			// Loop: another LLM call with tool results appended.
		}
	}()

	return out, nil
}
