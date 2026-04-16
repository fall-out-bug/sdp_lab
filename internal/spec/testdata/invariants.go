package testdata

import (
	"context"
	"sync"
	"time"
)

// Interface compliance check
var _ Stringer = (*Wrapper)(nil)

// Stringer is a simple string interface.
type Stringer interface {
	String() string
}

// Wrapper holds a value with mutex-guarded access.
type Wrapper struct {
	mu    sync.Mutex
	value string
}

// String implements Stringer.
func (w *Wrapper) String() string {
	return w.value
}

// ProcessWithDeadline demonstrates context deadline and mutex patterns.
func (w *Wrapper) ProcessWithDeadline(ctx context.Context) error {
	childCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	w.mu.Lock()
	defer w.mu.Unlock()
	w.value = "processed"

	_ = childCtx
	return nil
}

// TypeAssertAny demonstrates a type assertion invariant.
func TypeAssertAny(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
