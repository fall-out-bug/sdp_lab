# Council Round 3 — Raw Responses (v3 fix verification)

**Date:** 2026-04-10  
**Spec:** v3 (post Round 2 fixes)  
**Quorum status:** 3/6 responded (HARD_ABORT threshold: 4/6)

---

## Architect (codex-rescue)

FIX_R2-1: INTRODUCES_NEW_BUG  
REASON: RunPhase now releases h.mu before Run(), removing the original deadlock, but there is no separate execution-state lock. While events are still draining, ApproveGate, Rollback, or a second RunPhase can acquire h.mu, call transitionTo, and h.accumulator.Reset() while AfterToolCall: acc.OnToolResult is still mutating evidence.

FIX_R2-2: INCOMPLETE  
REASON: The closure-based completionFlag fixes the shared-flag bug, but BuildLoopConfig now splits ownership of completion_signal: every PhaseConfig.Tools already lists it, and BuildLoopConfig also appends a dynamic copy. The allowlist is no longer authoritative, and duplicate-name resolution is unspecified.

FIX_R2-3: CORRECT  
REASON: The timeout path now returns Escalated: true plus a GateWarn violation instead of silently producing Blocked:false. That closes the automatic-pass hole in GateEngine.Evaluate.

FIX_R2-4: INCOMPLETE  
REASON: NextPhase() and RecoveryPhase() exist, but both hardcode AllowedNext[0] / RecoveryNext[0]. Multi-edge phases are still structurally wrong: RoleReview declares {RoleEval, RoleBuild} yet only RoleEval is reachable through this API.

FIX_R2-5: INTRODUCES_NEW_BUG  
REASON: transitionTo(..., recovery bool) validates the correct edge set, but the new public APIs ApproveGate() and Rollback() take no pending-decision ID and no target role. Any caller can move the phase machine at any time, even when no gate escalation exists, which reopens phase-skipping and makes human approvals non-auditable.

NEW_ISSUE: Missing single-flight phase execution
SEVERITY: CRITICAL
LOCATION: Harness
DESCRIPTION: The harness has no running/awaiting_human state or runID. RunPhase, ApproveGate, and Rollback can interleave on the same session. This breaks phase isolation and can mix or erase evidence mid-run.
PROPOSAL: Add a session FSM with idle|running|awaiting_human, plus a runID/phaseVersion. Reject concurrent calls or queue them. Bind accumulator, gate result, and transition to that runID.
DOMAIN_VETO: YES

NEW_ISSUE: Human gate decisions are not durable or scoped
SEVERITY: CRITICAL
LOCATION: Harness / SessionStore
DESCRIPTION: On escalation, RunPhase only emits human_gate and returns nil. There is no persisted PendingDecision object containing phase, gate result, allowed actions, and owner. ApproveGate() / Rollback() therefore operate without proof that a decision is actually pending.
PROPOSAL: Persist PendingDecision{decisionID, runID, phase, gateResultID, allowedTransitions, ownerRequired}. Require decisionID on approval/rollback and clear it atomically with the transition.
DOMAIN_VETO: YES

NEW_ISSUE: Conversation state is not persisted as control-plane source of truth
SEVERITY: HIGH
LOCATION: Harness / SessionStore boundary
DESCRIPTION: RunPhase builds msgs := append(h.session.Messages(), userPrompt) locally, but the spec never appends the new user message, model outputs, or tool outputs back into h.session or SessionStore. Persisting events alone is not a canonical replay model. Next turn context can diverge from the WAL.
PROPOSAL: Persist a canonical TurnRecord / message log atomically per turn, then derive replay context from that log. Events should be secondary telemetry, not the only durable transcript.
DOMAIN_VETO: YES

NEW_ISSUE: Gate timeout does not cancel compliance work
SEVERITY: HIGH
LOCATION: GateEngine
DESCRIPTION: Evaluate creates evalCtx but calls harness.EvaluateCompliance(g.contract, snap.toHarness()) without passing cancellation. After timeout, the harness escalates, but the compliance goroutine may continue running or hang forever. Repeated timeouts can leak work.
PROPOSAL: Make EvaluateCompliance context-aware, pass evalCtx, and enforce bounded worker concurrency.
DOMAIN_VETO: YES

NEW_ISSUE: AfterToolCall signature mismatch
SEVERITY: HIGH
LOCATION: Loop / EvidenceAccumulator
DESCRIPTION: LoopConfig.AfterToolCall is declared as func(name, result string) error, but EvidenceAccumulator.OnToolResult expects (toolName, result string, err error). Beyond the signature mismatch, the hook model cannot reliably feed tool failures into compliance decisions.
PROPOSAL: Replace with AfterToolCall func(ToolResult) error and persist both success and failure outcomes.
DOMAIN_VETO: YES

---

## Critic (google/gemini-3.1-pro-preview) — TRUNCATED

FIX_R2-1: CORRECT  
REASON: Harness.RunPhase explicitly drops h.mu before [TRUNCATED — API response cut off]

---

## Technician (deepseek/deepseek-v3.2)

FIX_R2-1: CORRECT  
REASON: Harness.RunPhase reads state under h.mu lock, then unlocks before starting Loop.Run. Loop goroutines (via AfterToolCall) do not acquire h.mu, preventing deadlock. Mutex reacquired only after events drain for gate check and transition.

FIX_R2-2: CORRECT  
REASON: completionFlag struct with mutex is passed via BuildLoopConfig to makeCompletionSignalTool. Execute closure captures flag pointer, ensuring safe shared state. RunPhase reads flag.signaled after events drain, no race.

FIX_R2-3: CORRECT  
REASON: GateEngine.Evaluate timeout returns GateResult with Escalated=true and GateWarn violation. Forces human decision, prevents automatic unsafe pass.

FIX_R2-4: CORRECT  
REASON: PhaseRouter now has explicit NextPhase() and RecoveryPhase(). Harness calls these methods, no internal calculation.

FIX_R2-5: CORRECT  
REASON: transitionTo validates against RecoveryNext if recovery=true, else AllowedNext. ApproveGate and Rollback methods on Harness wire recovery path via SurfaceEvents.

NEW_ISSUE: EvidenceAccumulator.OnToolResult race with Snapshot  
SEVERITY: HIGH  
DESCRIPTION: OnToolResult called from Loop goroutines concurrently with Harness calling Snapshot after events drain. No guarantee all AfterToolCall callbacks finish before Snapshot() call.
PROPOSAL: Add WaitGroup synchronization point in RunPhase ensuring all AfterToolCall callbacks complete before Snapshot.

NEW_ISSUE: completionFlag.summary empty not validated  
SEVERITY: MEDIUM  
DESCRIPTION: If JSON unmarshal fails in completion_signal Execute, summary may be empty. Harness reads signaled but does not validate summary.
PROPOSAL: After reading signaled, validate summary is non-empty; log warning if empty.

NEW_ISSUE: GateEngine.Evaluate goroutine leak on timeout  
SEVERITY: MEDIUM  
DESCRIPTION: Goroutine calling harness.EvaluateCompliance may block forever after timeout, causing leak.
PROPOSAL: Pass evalCtx to EvaluateCompliance to enable cancellation.

NEW_ISSUE: completion_signal duplicate in ToolRegistry  
SEVERITY: LOW  
DESCRIPTION: PhaseConfig.Tools includes completion_signal in allowlist, and BuildLoopConfig also appends it. Creates duplicate tool.
PROPOSAL: Remove completion_signal from PhaseConfig.Tools allowlist; add only via BuildLoopConfig.

---

## Philosopher (moonshotai/kimi-k2.5) — ABSTAIN (empty choices from API)

## Pragmatist (minimax/minimax-m2.7) — ABSTAIN (empty choices from API)

## Engineer (xiaomi/mimo-v2-pro) — ABSTAIN (empty choices from API)
