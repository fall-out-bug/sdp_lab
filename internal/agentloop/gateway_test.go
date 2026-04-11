package agentloop

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModelGateway_interfaceSatisfaction verifies StubGateway satisfies ModelGateway at compile time.
func TestModelGateway_interfaceSatisfaction(t *testing.T) {
	var _ ModelGateway = (*StubGateway)(nil)
}

// TestStubGateway_isAvailable returns true for registered models, false otherwise.
func TestStubGateway_isAvailable(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{{Type: "done"}})

	assert.True(t, sg.IsAvailable("gpt-4.1"), "registered model must be available")
	assert.False(t, sg.IsAvailable("nonexistent-model"), "unknown model must not be available")
}

// TestStubGateway_call_returnsScriptedEvents verifies the event channel delivers scripted events.
func TestStubGateway_call_returnsScriptedEvents(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{
		{Type: "text_delta", Delta: "hello"},
		{Type: "done"},
	})

	cfg := LoopConfig{Model: "gpt-4.1"}
	ch, err := sg.Call(context.Background(), []Message{{Role: "user", Content: "hi"}}, cfg)
	require.NoError(t, err)

	var events []Event
	for ev := range ch {
		events = append(events, ev)
	}
	require.Len(t, events, 2)
	assert.Equal(t, "text_delta", events[0].Type)
	assert.Equal(t, "hello", events[0].Delta)
	assert.Equal(t, "done", events[1].Type)
}

// TestStubGateway_recordsCalls verifies ModelCall recording for assertion in tests.
func TestStubGateway_recordsCalls(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{{Type: "done"}})

	cfg := LoopConfig{Model: "gpt-4.1", SystemPrompt: "be helpful"}
	msgs := []Message{{Role: "user", Content: "build it"}}
	ch, err := sg.Call(context.Background(), msgs, cfg)
	require.NoError(t, err)
	// drain
	for range ch {
	}

	require.Len(t, sg.Calls, 1)
	assert.Equal(t, "gpt-4.1", sg.Calls[0].Model)
	assert.Equal(t, msgs, sg.Calls[0].Messages)
}

// TestStubGateway_call_unknownModel returns error for unregistered model.
func TestStubGateway_call_unknownModel(t *testing.T) {
	sg := NewStubGateway()
	_, err := sg.Call(context.Background(), nil, LoopConfig{Model: "unknown"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
}

// TestStubGateway_addResponse_multipleModels supports independent scripted responses per model.
func TestStubGateway_addResponse_multipleModels(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("model-a", []Event{{Type: "text_delta", Delta: "from A"}, {Type: "done"}})
	sg.AddResponse("model-b", []Event{{Type: "text_delta", Delta: "from B"}, {Type: "done"}})

	assert.True(t, sg.IsAvailable("model-a"))
	assert.True(t, sg.IsAvailable("model-b"))

	chA, err := sg.Call(context.Background(), nil, LoopConfig{Model: "model-a"})
	require.NoError(t, err)
	var evA []Event
	for ev := range chA {
		evA = append(evA, ev)
	}
	assert.Equal(t, "from A", evA[0].Delta)
}

// TestStubGateway_addResponse_queueSemantics verifies FIFO queue: multiple AddResponse
// calls for the same model are consumed in order. Fix R1: prevents response overwrite.
func TestStubGateway_addResponse_queueSemantics(t *testing.T) {
	sg := NewStubGateway()
	sg.AddResponse("gpt-4.1", []Event{{Type: "text_delta", Delta: "round 1"}, {Type: "done"}})
	sg.AddResponse("gpt-4.1", []Event{{Type: "text_delta", Delta: "round 2"}, {Type: "done"}})

	cfg := LoopConfig{Model: "gpt-4.1"}

	// First call → round 1.
	ch1, err := sg.Call(context.Background(), nil, cfg)
	require.NoError(t, err)
	var evs1 []Event
	for ev := range ch1 {
		evs1 = append(evs1, ev)
	}
	assert.Equal(t, "round 1", evs1[0].Delta, "Fix R1: first AddResponse consumed first")

	// Second call → round 2.
	ch2, err := sg.Call(context.Background(), nil, cfg)
	require.NoError(t, err)
	var evs2 []Event
	for ev := range ch2 {
		evs2 = append(evs2, ev)
	}
	assert.Equal(t, "round 2", evs2[0].Delta, "Fix R1: second AddResponse consumed second")

	// Third call → exhausted queue: safe fallback {done}.
	ch3, err := sg.Call(context.Background(), nil, cfg)
	require.NoError(t, err)
	var evs3 []Event
	for ev := range ch3 {
		evs3 = append(evs3, ev)
	}
	require.Len(t, evs3, 1)
	assert.Equal(t, "done", evs3[0].Type, "Fix R1: exhausted queue returns fallback {done}")
}
