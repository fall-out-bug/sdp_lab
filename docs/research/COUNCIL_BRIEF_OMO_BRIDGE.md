# SDP ↔ OmO Bridge: Council Brief

## Context

SDP Lab is a Go-based spec-driven pipeline for AI-assisted software development. It currently invokes coding agents (OmO = oh-my-openagent) via raw `exec.CommandContext("opencode run")` — cold-starting a new process every time (~15s overhead per invocation).

OmO is a multi-model agent framework with 11 specialized agents:
- **Sisyphus** — main orchestrator, decomposes and delegates
- **Hephaestus** — autonomous deep worker for coding (gpt-5.3-codex)
- **Momus** — adversarial reviewer (gpt-5.4)
- **Oracle** — read-only consultant (gpt-5.4)
- **Metis** — pre-planning consultant (claude-opus-4-6)
- **Atlas** — todo orchestrator (claude-sonnet-4-6)
- **Explore** — fast codebase grep (grok-code-fast)
- **Librarian** — external docs search (gemini-3-flash)
- **Multimodal-Looker** — vision/PDF analysis
- **Prometheus** — strategic planner (internal)
- **Sisyphus-Junior** — category-spawned executor

SDP's job is NOT code generation — OmO handles that. SDP must provide:
1. Governance (constitution, policies, compliance)
2. Provenance (what was sent, what was done, evidence)
3. Lifecycle control (feature intake → build → review → QA → delivery)
4. State management (Beads as primary store)

## Transport Options

opencode exposes:
- `opencode serve` — headless HTTP server (Hono/Bun) with REST API + SSE events
- `opencode acp` — Agent Client Protocol server (JSON-RPC over stdio/stream)
- `opencode run --attach <url>` — connect to running server

Current SDP invocation (`invoke_opencode.go`):
```go
cmd := exec.CommandContext(ctx, "opencode", "run", "--agent", agent)
cmd.Stdin = strings.NewReader(prompt)
out, err := cmd.CombinedOutput()
```

## Proposed Architecture

Two-layer model: SDP (governance) wraps OmO (execution).

SDP picks entry agent per phase:
- Build → Hephaestus
- Review → Momus
- QA → Oracle
- Planning → Metis
- Exploration → Explore
- Debug → Sisyphus

SDP wraps each call with governance:
1. PRE-CALL: constitution check, dependency check, provenance injection
2. CALL: create session, send prompt, stream events
3. POST-CALL: capture evidence, validate out-of-scope changes, route findings to Beads
4. ESCALATION: retry with backoff, 3-strike block rule

## Questions for Council

1. Is the two-layer model (SDP as governance wrapper, OmO as execution engine) architecturally sound? What are the failure modes?
2. Is REST API + SSE the right transport from Go, or should we implement ACP JSON-RPC client?
3. Are the agent-per-phase mappings optimal? Should SDP ever use Sisyphus directly (letting OmO do internal delegation)?
4. What evidence/provenance must SDP capture to satisfy ADLC governance requirements?
5. How should SDP handle permission management for OmO agents running in non-TTY pipeline mode?
6. What are the risks of SDP being "just a wrapper" around OmO? How much control does SDP actually have?
7. Should SDP implement its own session management, or fully rely on OmO's SQLite-backed sessions?
8. What happens when OmO's internal agent delegation produces unexpected results (e.g., Hephaestus delegates to wrong sub-agent)?
