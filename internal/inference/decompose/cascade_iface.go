package decompose

import "context"

// CascadeInvoker is a narrow local interface for F145 provider-escalation
// (cascade-per-request). Defined here so F146 does not depend on the not-yet-merged
// F145 package (sdplab-5ii8). After F145 merges, alias this to the real
// internal/dispatch/cascade.Invoker and delete this file.
type CascadeInvoker interface {
	// Invoke wraps fn (the stage's raw LLM call thunk) with provider-escalation logic.
	// Returns the selected output, the combined stage trace, the cascade trace, and any error.
	Invoke(ctx context.Context, fn func() (any, StageTrace, error)) (any, StageTrace, CascadeTrace, error)
}

// CascadeTrace records which provider handled the stage and how many attempts cascade made.
type CascadeTrace struct {
	Provider string
	Attempts int
}
