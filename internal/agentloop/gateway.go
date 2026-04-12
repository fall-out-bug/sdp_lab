package agentloop

import (
	"context"
	"fmt"
)

// NOTE: ModelGateway interface is defined in types.go (Task 1). gateway.go only provides
// StubGateway (the test double) and ModelCall. Fix W1: interface defined once, not duplicated.

// ModelCall records a single Call() invocation for test assertions.
type ModelCall struct {
	Model    string
	Messages []Message
	Config   LoopConfig
}

// StubGateway is an in-memory test double for ModelGateway.
// Register scripted Event sequences with AddResponse; recorded calls are in Calls.
//
// Fix R1: StubGateway uses a FIFO queue per model (map[string][][]Event).
// Calling AddResponse multiple times for the same model APPENDS to the queue.
// Call() consumes responses in order; once the queue is exhausted, Call() returns
// a safe fallback {done} sequence so multi-turn tests never block unexpectedly.
type StubGateway struct {
	responses map[string][][]Event // Fix R1: FIFO queue per model
	callIdx   map[string]int       // Fix R1: next response index per model
	Calls     []ModelCall
}

// NewStubGateway creates an initialized StubGateway.
func NewStubGateway() *StubGateway {
	return &StubGateway{
		responses: make(map[string][][]Event),
		callIdx:   make(map[string]int),
	}
}

// AddResponse appends a scripted Event sequence to the queue for model.
// Fix R1: successive Call() invocations consume responses in FIFO order.
// For a single-turn test, call AddResponse once. For multi-turn (e.g. completion_signal),
// call AddResponse once per expected gateway invocation in order.
func (sg *StubGateway) AddResponse(model string, events []Event) {
	sg.responses[model] = append(sg.responses[model], events)
}

// IsAvailable returns true if model has been registered via AddResponse (key exists in map).
// A model stays "available" even after its queue is exhausted — the fallback {done}
// ensures Call() never blocks. An unregistered model (key absent) returns false.
func (sg *StubGateway) IsAvailable(model string) bool {
	_, ok := sg.responses[model]
	return ok
}

// Call records the invocation, then returns the next scripted Event sequence from the queue.
// Fix R1: responses consumed in FIFO order. If the queue is exhausted, returns a safe
// fallback {done} sequence (covers acknowledgement turns after completion_signal).
// Returns an error only if model was never registered — callers should check IsAvailable first.
func (sg *StubGateway) Call(ctx context.Context, msgs []Message, cfg LoopConfig) (<-chan Event, error) {
	sg.Calls = append(sg.Calls, ModelCall{
		Model:    cfg.Model,
		Messages: msgs,
		Config:   cfg,
	})

	queue, ok := sg.responses[cfg.Model]
	if !ok {
		return nil, fmt.Errorf("StubGateway: model %q never registered — call IsAvailable first", cfg.Model)
	}

	idx := sg.callIdx[cfg.Model]
	var events []Event
	if idx < len(queue) {
		events = queue[idx]
		sg.callIdx[cfg.Model]++
	} else {
		// Fix R1: queue exhausted — safe fallback for extra turns (e.g. acknowledgement).
		events = []Event{{Type: "done"}}
	}

	ch := make(chan Event, len(events))
	for _, ev := range events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}
