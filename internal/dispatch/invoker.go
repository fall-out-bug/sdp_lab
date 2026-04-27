package dispatch

import (
	"context"
	"log/slog"

	"github.com/fall-out-bug/sdp_lab/internal/dispatch/harness"
	"github.com/fall-out-bug/sdp_lab/internal/kernel"
)

type LLMInvoker = kernel.RuntimeAdapter

// DispatchingInvoker classifies incoming invocations, routes them to the best
// harness via Router, writes a DispatchDecision, and delegates to the selected
// invoker. Falls back to Fallback on any routing error.
type DispatchingInvoker struct {
	Router       *Router
	Fallback     LLMInvoker
	Limits       map[string]*harness.Limits
	InvokerFor   func(harnessName string) LLMInvoker
	PacketLoader func(projectRoot string) (ContextPacketSummary, error)
	// ContextEnricher injects hydrated context into the prompt. If nil, the
	// prompt is passed through unchanged.
	ContextEnricher func(projectRoot, basePrompt string) string
	// VerifyHarness selects an alternative harness for review/qa agents.
	// If nil or if it returns nil, the normal routing decision is used.
	VerifyHarness func(buildDec *DispatchDecision, task TaskClassification) (*DispatchDecision, error)
}

// Invoke classifies the task by loading the context packet from dir, routes to
// the best harness, writes the decision, and invokes the selected invoker.
// Falls back to Fallback on any routing error.
//
// For review/qa agents, if VerifyHarness is set, it attempts to route to a
// different harness than the one used for build (cross-harness verification).
func (d *DispatchingInvoker) Invoke(ctx context.Context, dir, agent, prompt string) (string, int, error) {
	pkt, err := d.PacketLoader(dir)
	if err != nil {
		slog.Warn("dispatch: packet load failed, falling back", "dir", dir, "err", err)
		res, invokeErr := d.Fallback.Invoke(ctx, kernel.RuntimeInvocation{WorkDir: dir, Agent: agent, Prompt: prompt})
		return res.Output, res.ExitCode, invokeErr
	}

	task := Classify(pkt)

	dec, err := d.Router.Route(ctx, task, d.Limits)
	if err != nil {
		slog.Warn("dispatch: routing failed, falling back", "err", err)
		res, invokeErr := d.Fallback.Invoke(ctx, kernel.RuntimeInvocation{WorkDir: dir, Agent: agent, Prompt: prompt})
		return res.Output, res.ExitCode, invokeErr
	}

	// Cross-harness verification: for review/qa agents, try a different harness.
	if isVerificationAgent(agent) && d.VerifyHarness != nil {
		if altDec, altErr := d.VerifyHarness(dec, task); altErr != nil {
			slog.Warn("dispatch: verify harness selection failed, using build harness",
				"err", altErr)
		} else if altDec != nil {
			slog.Info("dispatch: cross-harness verification",
				"build_harness", dec.Harness,
				"verify_harness", altDec.Harness,
				"verify_score", altDec.Score,
			)
			dec = altDec
		}
	}

	if writeErr := WriteDecision(dir, dec); writeErr != nil {
		slog.Warn("dispatch: write decision failed (non-fatal)", "err", writeErr)
	}

	// Enrich prompt with hydrated context.
	enrichedPrompt := prompt
	if d.ContextEnricher != nil {
		enrichedPrompt = d.ContextEnricher(dir, prompt)
	}

	inv := d.InvokerFor(dec.Harness)
	if inv == nil {
		slog.Warn("dispatch: no invoker for harness, falling back", "harness", dec.Harness)
		res, invokeErr := d.Fallback.Invoke(ctx, kernel.RuntimeInvocation{WorkDir: dir, Agent: agent, Prompt: enrichedPrompt})
		return res.Output, res.ExitCode, invokeErr
	}

	slog.Info("dispatch: invoking via harness",
		"harness", dec.Harness,
		"provider", dec.Provider,
		"score", dec.Score,
	)

	res, invokeErr := inv.Invoke(ctx, kernel.RuntimeInvocation{
		WorkDir: dir,
		Agent:   agent,
		Prompt:  enrichedPrompt,
	})
	return res.Output, res.ExitCode, invokeErr
}

// isVerificationAgent returns true for agents that perform verification work
// (review, qa) and should ideally use a different harness than build.
func isVerificationAgent(agent string) bool {
	switch agent {
	case "reviewer", "qa", "review", "qa-agent":
		return true
	}
	return false
}
