// Package cascade implements tier-based cascade routing with heuristic short-circuit
// and pluggable confidence gates.
//
// The CascadingInvoker wraps a dispatch.Router and drives cascade escalation:
// 1. Try the cheapest tier first (TierLocal → TierFast)
// 2. Apply heuristic short-circuit: if response is too short or matches refusal pattern, escalate
// 3. Run Checker (if provided): if rejected, escalate
// 4. Continue up the tier ladder (TierBalanced → TierStrong) until accepted or limits hit
//
// This is primarily designed to minimize cost by attempting cheaper tiers first,
// while catching obvious failures early via heuristics before paying for a confidence check.
//
// Reference: [F145 design §4.3, §4.4, §4.7](../../docs/plans/2026-04-26-f145-multi-provider-dispatch-cascade-design.md)
package cascade

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"sdp_dev/internal/dispatch"
	"sdp_dev/internal/dispatch/harness"
	"sdp_dev/internal/inference/confidence"
)

// RouterInterface abstracts the dispatch.Router contract for testability.
// The concrete Router implements this implicitly.
type RouterInterface interface {
	Route(ctx context.Context, task dispatch.TaskClassification, limits map[string]*harness.Limits) (*dispatch.DispatchDecision, error)
}

// CascadingInvoker wraps a Router and drives cascade escalation logic.
// It maintains default tier ordering, budget constraints, and a pluggable Checker.
type CascadingInvoker struct {
	router    RouterInterface             // dispatch.Router (abstracted for testing)
	checker   Checker                     // confidence gate (nil = always-ok)
	maxDepth  int                         // max cascade hops (default 3)
	budget    *Budget                     // wallclock + cost cap
	tierOrder []dispatch.TierClass        // ["local", "fast", "balanced", "strong"]
	heuristic ShortCircuitConfig          // short-circuit thresholds
}

// NewInvoker creates a new CascadingInvoker with a Router, optional Checker, and defaults.
// router can be nil for testing. checker can be nil (treats all responses as acceptable).
// opts allows future extension with configuration.
func NewInvoker(router RouterInterface, checker Checker, opts ...interface{}) *CascadingInvoker {
	invoker := &CascadingInvoker{
		router:    router,
		checker:   checker,
		maxDepth:  3,
		tierOrder: defaultTierOrder(),
		heuristic: DefaultShortCircuitConfig(),
		budget: &Budget{
			MaxDuration: 30 * time.Second,
			StartTime:   time.Now(),
		},
	}
	return invoker
}

// NewWithConfidence creates a new CascadingInvoker with a F144 confidence.Checker
// automatically adapted to the cascade.Checker interface. This is a convenience
// constructor that handles the adapter wiring.
//
// Parameters:
// - router: dispatch.Router (required for production; can be nil for testing)
// - checker: F144 confidence.Checker[string] to wrap (can be nil for graceful degrade)
// - threshold: confidence score cutoff (0.0-1.0); score >= threshold → accept
// - opts: future extension hooks (currently unused)
//
// Returns a CascadingInvoker ready to run Invoke() with confidence-driven escalation.
func NewWithConfidence(router RouterInterface, checker *confidence.Checker[string], threshold float64, opts ...interface{}) *CascadingInvoker {
	adapter := NewConfidenceAdapter(checker, threshold)
	return NewInvoker(router, adapter, opts...)
}

// defaultTierOrder returns the standard cascade order: cheap to strong.
func defaultTierOrder() []dispatch.TierClass {
	return []dispatch.TierClass{
		dispatch.TierLocal,
		dispatch.TierFast,
		dispatch.TierBalanced,
		dispatch.TierStrong,
	}
}

// Invoke runs cascade invocation: try cheapest tier first, escalate on heuristic or checker rejection.
// Returns InvokeResult with tier_used, hops, and cause (ok | max_depth | budget | checker_failed).
func (ci *CascadingInvoker) Invoke(ctx context.Context, req InvokeRequest) (*InvokeResult, error) {
	if ci.budget == nil {
		ci.budget = &Budget{
			MaxDuration: 30 * time.Second,
			StartTime:   time.Now(),
		}
	}

	// Determine starting tier
	startIdx := 0
	if req.StartTier != "" {
		for i, tier := range ci.tierOrder {
			if tier == req.StartTier {
				startIdx = i
				break
			}
		}
	}

	var lastResult *harness.Result
	var lastTier dispatch.TierClass
	var lastError error
	hop := 0

	// Cascade loop: iterate through tiers from start to end
	for tierIdx := startIdx; tierIdx < len(ci.tierOrder); tierIdx++ {
		hop++

		// Check budget before each hop
		if ci.budget.Exhausted() {
			return &InvokeResult{
				Tier:      lastTier,
				Hops:      hop - 1,
				Output:    conditionalOutput(lastResult),
				Cause:     "budget",
				LastError: "budget exhausted",
			}, nil
		}

		// Check max depth
		if hop > ci.maxDepth {
			return &InvokeResult{
				Tier:      lastTier,
				Hops:      ci.maxDepth,
				Output:    conditionalOutput(lastResult),
				Cause:     "max_depth",
				LastError: fmt.Sprintf("max depth %d reached", ci.maxDepth),
			}, nil
		}

		tier := ci.tierOrder[tierIdx]
		lastTier = tier

		// Invoke harness for this tier
		// (In real impl, this would spawn harness via Router decision + Invoker)
		// For now, this is a placeholder that sets up the structure.
		// F145-11 will integrate with actual F144 confidence checker.

		slog.Debug("cascade invoking tier",
			"tier", tier,
			"hop", hop,
			"req_prompt", req.Prompt[:min(len(req.Prompt), 50)],
		)

		// Placeholder: if router is nil (testing), skip to next tier
		if ci.router == nil {
			// For now, create a dummy result to test structure
			lastResult = &harness.Result{
				Output:   fmt.Sprintf("Dummy response from tier %v", tier),
				Duration: time.Millisecond,
				ExitCode: 0,
			}
		} else {
			// In real code, we'd use Router.Route + harness.Spawn
			// This structure will be completed in F145-11
		}

		// Apply heuristic short-circuit
		if lastResult != nil {
			shortCircuit, reason := checkShortCircuit(lastResult.Output, ci.heuristic)
			if shortCircuit {
				slog.Debug("cascade heuristic short-circuit",
					"tier", tier,
					"reason", reason,
				)
				// If not on the last tier, escalate
				if tierIdx < len(ci.tierOrder)-1 {
					continue // escalate to next tier
				}
				// On last tier, return with short circuit reason
				return &InvokeResult{
					Tier:               tier,
					Hops:               hop,
					Output:             lastResult.Output,
					Cause:              "ok", // we tried all tiers, return final result
					ShortCircuitReason: reason,
				}, nil
			}
		}

		// Apply Checker gate (if provided)
		if ci.checker != nil {
			ok, checkReason := ci.checker.Check(ctx, req, lastResult)
			if !ok {
				slog.Debug("cascade checker rejected",
					"tier", tier,
					"reason", checkReason,
				)
				// If not on the last tier, escalate
				if tierIdx < len(ci.tierOrder)-1 {
					continue // escalate to next tier
				}
				// On last tier, return with checker failure
				return &InvokeResult{
					Tier:      tier,
					Hops:      hop,
					Output:    conditionalOutput(lastResult),
					Cause:     "checker_failed",
					LastError: checkReason,
				}, nil
			}
		}

		// All checks passed; return success
		return &InvokeResult{
			Tier:   tier,
			Hops:   hop,
			Output: conditionalOutput(lastResult),
			Cause:  "ok",
		}, nil
	}

	// All tiers exhausted
	msg := fmt.Sprintf("all %d tiers exhausted", len(ci.tierOrder))
	if lastError != nil {
		msg = lastError.Error()
	}
	return &InvokeResult{
		Tier:      lastTier,
		Hops:      hop,
		Output:    conditionalOutput(lastResult),
		Cause:     "max_depth",
		LastError: msg,
	}, nil
}

// checkShortCircuit returns (shouldEscalate, reason) based on heuristic rules.
func checkShortCircuit(output string, cfg ShortCircuitConfig) (bool, string) {
	// Check empty/whitespace
	if len(output) == 0 {
		if !cfg.EmptyOK {
			return true, "empty"
		}
		return false, ""
	}

	// Check minimum length
	if len(output) < cfg.MinLengthChars {
		return true, fmt.Sprintf("too_short:%d", len(output))
	}

	// Check refusal patterns (would use regex in real impl)
	// For now, simple substring checks that cover the requirements
	refusalKeywords := []string{
		"cannot", "can't", "am not able",
		"unable to", "sorry",
		"i'm unable", "i cannot",
	}
	lowerOutput := strings.ToLower(output)
	for _, kw := range refusalKeywords {
		if strings.Contains(lowerOutput, strings.ToLower(kw)) {
			return true, fmt.Sprintf("refusal:%s", kw)
		}
	}

	return false, ""
}

// conditionalOutput returns output if result is not nil, else empty string.
func conditionalOutput(result *harness.Result) string {
	if result == nil {
		return ""
	}
	return result.Output
}

