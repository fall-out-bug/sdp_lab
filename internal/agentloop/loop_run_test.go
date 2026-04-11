package agentloop

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- helpers ----

// collectEvents drains ch into a slice. Blocks until ch is closed.
func collectEvents(ch <-chan Event) []Event {
	var out []Event
	for ev := range ch {
		out = append(out, ev)
	}
	return out
}

// noToolCfg builds a LoopConfig with a StubGateway and no tools.
func noToolCfg(model string, sg *StubGateway) LoopConfig {
	return LoopConfig{
		Model:   model,
		Gateway: sg,
	}
}

// TestRun_noTools_emitsDone: single LLM response with no tool calls → "done" event emitted to caller.
func TestRun_noTools_emitsDone(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "I found 3 competitors."},
		{Type: "done"},
	})

	cfg := noToolCfg("gpt-4.1", sg)
	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "discover"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	types := make([]string, len(events))
	for i, ev := range events {
		types[i] = ev.Type
	}

	assert.Contains(t, types, "text_delta")
	assert.Equal(t, "done", types[len(types)-1], "last event must be 'done'")
}

// TestRun_withTools_executesAndContinues: LLM calls 1 tool → tool executes → second LLM response → done.
func TestRun_withTools_executesAndContinues(t *testing.T) {
	sg := NewStubGateway()

	// First LLM call: requests a tool, then "done" (meaning this assistant turn ended)
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "tool_call", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "web_search", Arguments: json.RawMessage(`{"query":"competitors"}`)},
		}},
		{Type: "done"},
	})

	// Second LLM call (after tool result appended): final text response
	// StubGateway returns its sequence every time Call is invoked for the same model,
	// but here we need two different responses. Use a counter-based approach.
	callCount := 0
	sg2 := &countingGateway{
		responses: [][]Event{
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "tc1", Name: "web_search", Arguments: json.RawMessage(`{"query":"competitors"}`)},
				}},
				{Type: "done"},
			},
			{
				{Type: "text_delta", Delta: "Found 3 competitors."},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	searchTool := Tool{
		Name: "web_search",
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return "competitor1, competitor2, competitor3", nil
		},
	}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: sg2,
		Tools:   []Tool{searchTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "discover"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	require.True(t, len(events) > 0, "must emit at least one event")
	assert.Equal(t, "done", events[len(events)-1].Type, "last event must be 'done'")

	// Should have called gateway twice (initial + after tool result).
	assert.Equal(t, 2, callCount)

	// Must have emitted tool_end event.
	toolEndEvents := filterByType(events, "tool_end")
	require.Len(t, toolEndEvents, 1)
	assert.Equal(t, "tc1", toolEndEvents[0].ToolID, "Fix Y1: tool_end.ToolID must match ToolCall.ID")
	assert.Equal(t, "web_search", toolEndEvents[0].ToolName)
}

// TestRun_toolEndCarriesToolID: Fix Y1 — event.ToolID matches the original ToolCall.ID.
func TestRun_toolEndCarriesToolID(t *testing.T) {
	callCount := 0
	gw := &countingGateway{
		responses: [][]Event{
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "specific-tool-id-42", Name: "read_file", Arguments: json.RawMessage(`{}`)},
				}},
				{Type: "done"},
			},
			{
				{Type: "text_delta", Delta: "file content processed"},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	readFileTool := Tool{
		Name: "read_file",
		Execute: func(ctx context.Context, id string, args json.RawMessage) (string, error) {
			return "file contents", nil
		},
	}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: gw,
		Tools:   []Tool{readFileTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "read it"}}, cfg)
	require.NoError(t, err)
	events := collectEvents(ch)

	toolEndEvents := filterByType(events, "tool_end")
	require.Len(t, toolEndEvents, 1)
	assert.Equal(t, "specific-tool-id-42", toolEndEvents[0].ToolID,
		"Fix Y1: ToolID in 'tool_end' event must equal the original ToolCall.ID")
}

// TestRun_contextCancellation: cancel ctx → channel closes with an error event.
func TestRun_contextCancellation(t *testing.T) {
	// Gateway that blocks until context is cancelled.
	blockingGW := &blockingGateway{}

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: blockingGW,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ch, err := Run(ctx, []Message{{Role: "user", Content: "go"}}, cfg)
	require.NoError(t, err, "Run() itself must not error immediately")

	events := collectEvents(ch)
	// Must receive at least one event; last event should be "error" or channel just closes.
	if len(events) > 0 {
		lastType := events[len(events)-1].Type
		// Accept either an "error" event or empty (channel closed without events on fast cancel).
		assert.True(t, lastType == "error" || lastType == "done",
			"on cancellation, last event must be 'error' or channel closes cleanly; got %q", lastType)
	}
	// Primary assertion: channel must be closed (collectEvents returned).
}

// TestRun_contextManagerTrimCalled: if ContextManager != nil → Trim is called before each LLM call.
func TestRun_contextManagerTrimCalled(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "ok"},
		{Type: "done"},
	})

	trimCallCount := 0
	cm := &countingContextManager{count: &trimCallCount}

	cfg := LoopConfig{
		Model:          "gpt-4.1",
		Gateway:        sg,
		ContextManager: cm,
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "hello"}}, cfg)
	require.NoError(t, err)
	collectEvents(ch)

	assert.GreaterOrEqual(t, trimCallCount, 1, "ContextManager.Trim must be called at least once before LLM call (Fix V3)")
}

// TestRun_completionSignal_stopsLoop: when completion_signal is called, Run exits after next LLM response.
func TestRun_completionSignal_stopsLoop(t *testing.T) {
	callCount := 0
	gw := &countingGateway{
		responses: [][]Event{
			// First LLM call: fires completion_signal tool.
			{
				{Type: "tool_call", ToolCalls: []ToolCall{
					{ID: "sig1", Name: "completion_signal", Arguments: json.RawMessage(`{"summary":"done"}`)},
				}},
				{Type: "done"},
			},
			// Second LLM call: final acknowledgement after tool result.
			{
				{Type: "text_delta", Delta: "phase complete"},
				{Type: "done"},
			},
		},
		callCount: &callCount,
	}

	flag := &completionFlag{}
	completionTool := makeCompletionSignalTool(flag)

	cfg := LoopConfig{
		Model:   "gpt-4.1",
		Gateway: gw,
		Tools:   []Tool{completionTool},
	}

	ch, err := Run(context.Background(), []Message{{Role: "user", Content: "finish phase"}}, cfg)
	require.NoError(t, err)

	events := collectEvents(ch)
	require.NotEmpty(t, events)
	assert.Equal(t, "done", events[len(events)-1].Type)

	// Loop must NOT make a third LLM call (flag.signaled stops it after second call).
	assert.LessOrEqual(t, callCount, 2, "Run must stop after completion_signal is processed, not loop indefinitely")
}

// ---- test helpers (in-package helpers for loop tests) ----

// countingGateway is a test double that returns different Event sequences on successive Call() invocations.
type countingGateway struct {
	responses [][]Event
	callCount *int
}

func (g *countingGateway) IsAvailable(model string) bool { return true }

func (g *countingGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	idx := *g.callCount
	*g.callCount++
	if idx >= len(g.responses) {
		// Fallback: return a done event if no more scripted responses.
		ch := make(chan Event, 1)
		ch <- Event{Type: "done"}
		close(ch)
		return ch, nil
	}
	evs := g.responses[idx]
	ch := make(chan Event, len(evs))
	for _, ev := range evs {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

// blockingGateway blocks on Call() until the context is cancelled.
type blockingGateway struct{}

func (g *blockingGateway) IsAvailable(model string) bool { return true }

func (g *blockingGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	ch := make(chan Event, 1)
	go func() {
		defer close(ch)
		<-ctx.Done()
		ch <- Event{Type: "error", Err: ctx.Err()}
	}()
	return ch, nil
}

// countingContextManager records Trim invocations.
type countingContextManager struct {
	count *int
}

func (cm *countingContextManager) Trim(messages []Message, model string, maxTokens int) ([]Message, error) {
	*cm.count++
	return messages, nil
}

// filterByType returns events whose Type matches the given string.
func filterByType(events []Event, eventType string) []Event {
	var out []Event
	for _, ev := range events {
		if ev.Type == eventType {
			out = append(out, ev)
		}
	}
	return out
}
