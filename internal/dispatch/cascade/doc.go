// Package cascade implements tier-based cascade routing for multi-provider LLM dispatch.
//
// See: [F145 design - Cascade Layer §4.3, §4.4, §4.7]
// (../../docs/plans/2026-04-26-f145-multi-provider-dispatch-cascade-design.md)
//
// CascadingInvoker wraps dispatch.Router and drives escalation:
//  1. Start at configured tier (or TierLocal)
//  2. Invoke harness, run heuristic short-circuit check
//  3. If heuristic triggers or Checker rejects, escalate to next tier
//  4. Repeat until response accepted, MaxDepth/Budget exhausted, or all tiers tried
//
// Heuristic short-circuit avoids expensive confidence checks on obvious failures:
//  - Empty response
//  - Response < MinLengthChars (default 50)
//  - Matches refusal pattern (e.g. "I cannot", "I'm unable")
//
// Checker is injected (F145-11 wires F144 confidence.Checker).
// For F145-10, stub implementation always accepts (nil = always-ok).
package cascade
