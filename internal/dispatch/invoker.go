package dispatch

import (
	"context"
	"log/slog"

	"sdp_dev/internal/dispatch/harness"
)

// LLMInvoker matches orchestrate.LLMInvoker to avoid circular imports.
type LLMInvoker interface {
	Invoke(ctx context.Context, dir, agent, prompt string) (output string, exitCode int, err error)
}

// DispatchingInvoker classifies incoming invocations, routes them to the best
// harness via Router, writes a DispatchDecision, and delegates to the selected
// invoker. Falls back to Fallback on any routing error.
type DispatchingInvoker struct {
	Router       *Router
	Fallback     LLMInvoker
	Limits       map[string]*harness.Limits
	InvokerFor   func(harnessName string) LLMInvoker
	PacketLoader func(projectRoot string) (ContextPacketSummary, error)
}

// Invoke classifies the task by loading the context packet from dir, routes to
// the best harness, writes the decision, and invokes the selected invoker.
// Falls back to Fallback on any routing error.
func (d *DispatchingInvoker) Invoke(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	pkt, err := d.PacketLoader(dir)
	if err != nil {
		slog.Warn("dispatch: packet load failed, falling back", "dir", dir, "err", err)
		return d.Fallback.Invoke(ctx, dir, agent, prompt)
	}

	task := Classify(pkt)

	dec, err := d.Router.Route(ctx, task, d.Limits)
	if err != nil {
		slog.Warn("dispatch: routing failed, falling back", "err", err)
		return d.Fallback.Invoke(ctx, dir, agent, prompt)
	}

	if writeErr := WriteDecision(dir, dec); writeErr != nil {
		slog.Warn("dispatch: write decision failed (non-fatal)", "err", writeErr)
	}

	inv := d.InvokerFor(dec.Harness)
	if inv == nil {
		slog.Warn("dispatch: no invoker for harness, falling back", "harness", dec.Harness)
		return d.Fallback.Invoke(ctx, dir, agent, prompt)
	}

	slog.Info("dispatch: invoking via harness",
		"harness", dec.Harness,
		"provider", dec.Provider,
		"score", dec.Score,
	)

	return inv.Invoke(ctx, dir, agent, prompt)
}
