# Agent Platform Kernel Boundary Spec

> **Status:** Draft
> **Date:** 2026-03-31
> **Goal:** Translate the platform-roadmap thesis into buildable package boundaries, first kernel contracts, and a realistic first implementation slice.

Related:

- `docs/roadmap/AGENT_PLATFORM_ROADMAP_2026-03-31.md`
- `docs/roadmap/ROADMAP.md`
- `docs/roadmap/UNIFIED_VISION_ROADMAP_2026-03-03.md`
- `docs/vision/README.md`
- `docs/architecture/REPO-BOUNDARY.md`

---

## 1. Decision

Recommendation:

- treat `AGENT_PLATFORM_ROADMAP_2026-03-31.md` as the new primary architecture direction for `sdp_lab`
- keep the current trust-roadmap content, but demote it from "the whole product thesis" to the `trust lane`
- do not rewrite every canonical doc in this pass; first lock the package boundary so the platform claim is concrete

Why this is the right call:

- the current codebase already contains early substrate pieces (`modelgateway`, `session`, `executor`, `planner`)
- products like Faust need reusable contracts and adapter boundaries, not only evidence exports
- the current evidence-first story explains compliance tooling, but it does not explain what the reusable runtime is

What this decision does **not** mean:

- no immediate repo split
- no Rust-first rewrite
- no public API promise yet
- no consultant/product workflow packs in the kernel

---

## 2. Current Reality

Today the repo has useful pieces, but the ownership is muddy.

| Current package | What it really is | Why it is not the kernel |
|----------------|-------------------|---------------------------|
| `internal/harness` | task-contract and compliance types | trust/compliance-specific semantics dominate the package |
| `internal/session` | evidence log + memory loading helpers | mixes trace, persistence, and memory presentation |
| `internal/modelgateway` | early provider abstraction | provider-specific chat API is useful, but not the whole runtime contract |
| `internal/executor` | orchestration logic with OpenCode/OmO coupling | runtime roles and vendor coupling are mixed together |
| `internal/planner` | graph and scheduler mechanics | useful augmentation primitive, not a root platform contract |
| `internal/eval` | transcript pattern checker | too shallow to protect behavior-level regressions |
| `internal/evidence`, `internal/policy`, `internal/guard` | trust-lane and enforcement components | these are platform consumers, not the platform core |

The problem is not "missing architecture".
The problem is that generic contracts and trust-specific contracts are still tangled.

---

## 3. Target Package Map

Use five internal ownership layers.
Keep the first cut boring and explicit.

| Layer | Proposed package | Owns | Must not own |
|------|------------------|------|--------------|
| Kernel | `internal/kernel` | core types, stable interfaces, serialization rules, run/session identity | provider SDK calls, evidence export formats, domain packs |
| Augmentation | `internal/augmentation` | pack loading, planner/verifier roles, context assembly, memory-candidate hooks, approval orchestration hooks | provider selection, signed provenance, product personas |
| Adapters | `internal/adapters/...` | provider adapters, runtime adapters, capability descriptors, transport quirks | product workflow logic, canonical session state |
| Evals | `internal/evals` | scenario runner, trace assertions, routing/tool/memory/workflow evals | production routing or trust export logic |
| Trust lane | `internal/trust/...` | evidence, provenance, attestation, policy profiles, audit exports | generic workflow definition, provider abstraction |

Recommended first package split:

| New package | Seed sources |
|-------------|--------------|
| `internal/kernel` | new package; absorb only generic types from `session`, `executor`, `modelgateway` |
| `internal/augmentation` | `internal/planner`, selected role logic from `internal/executor` |
| `internal/adapters/model` | `internal/modelgateway`, `internal/modelgateway/adapters` |
| `internal/adapters/runtime/opencode` | `internal/executor/omoclient` |
| `internal/evals` | replace `internal/eval` over time; keep compatibility bridge initially |
| `internal/trust/evidence` | `internal/evidence`, `internal/session` event-log persistence |
| `internal/trust/policy` | `internal/policy`, `internal/guard`, related enforcement packages |

Important rule:

- `internal/kernel` must compile without importing `internal/trust`, `internal/adapters`, or vendor-specific packages

That rule is the actual boundary test.

---

## 4. Kernel Contract Map

This is the minimum believable kernel surface.

### 4.1 `AgentDefinition`

**Owner:** `internal/kernel`

**Purpose:** static contract for one agent or role.

**Must include:**

- stable `ID`
- declared role or capability set
- allowed workflow packs
- tool-policy reference
- model capability requirements, not vendor IDs

**Must not include:**

- provider API keys
- OpenCode/Claude/OpenAI-specific fields
- trust export settings

### 4.2 `SessionState`

**Owner:** `internal/kernel`

**Purpose:** mutable state of one run/session.

**Must include:**

- `RunID`
- `SessionID`
- active workflow-pack refs
- conversation/context segments
- selected tool-policy ref
- emitted memory candidates
- artifact refs and trace refs

**Must not include:**

- signed evidence payloads
- vendor transcript wire formats
- storage-engine-specific handles

### 4.3 `WorkflowPack`

**Owner:** `internal/kernel`

**Purpose:** lazily loadable bundle of instructions, roles, tools, and hooks.

**Must include:**

- manifest metadata
- version
- declared dependencies
- role definitions
- hook registrations
- eval references

**Must not include:**

- full chat history
- provider credentials
- product-only business logic in the generic pack format

### 4.4 `MemoryCandidate`

**Owner:** `internal/kernel`

**Purpose:** proposed durable memory emitted during execution.

**Must include:**

- candidate ID
- content or structured payload
- confidence
- reason
- source trace refs
- scope (`user`, `project`, `task`)

**Must not include:**

- direct database writes
- product-specific retention logic

### 4.5 `ToolPolicy`

**Owner:** `internal/kernel`

**Purpose:** declarative or executable contract for allow/deny/ask behavior over tool use.

**Must include:**

- policy ID
- evaluation input shape
- decision enum: `allow | ask | deny`
- optional explanation and remediation

**Must not include:**

- GitHub-specific, Gmail-specific, or product-specific policy hard-coding
- attestation export details

### 4.6 `TraceEvent`

**Owner:** `internal/kernel`

**Purpose:** normalized event emitted by the runtime regardless of substrate.

**Must include:**

- event ID
- run/session IDs
- event kind
- actor or agent ID
- timestamp
- payload envelope
- parent or correlation refs

**Must not include:**

- hash-chain persistence details
- one specific transcript schema

### 4.7 `ApprovalHook`

**Owner:** `internal/kernel`

**Purpose:** interception point for human or automated approval decisions.

**Must include:**

- request type
- approval context
- response contract: `approve | reject | escalate`

**Must not include:**

- UI transport assumptions
- one fixed human-review workflow

### 4.8 `EvalCase`

**Owner:** `internal/kernel`

**Purpose:** behavior-level scenario definition portable across adapters.

**Must include:**

- scenario ID
- inputs
- expected trace assertions
- expected tool-policy outcomes
- expected memory or artifact outputs

**Must not include:**

- raw transcript substring checks as the primary model

### 4.9 Supporting primitives

The first slice also needs small supporting types:

- `RunID`
- `SessionID`
- `ArtifactRef`
- `ContextSegment`
- `CapabilitySet`
- `ApprovalDecision`
- `ToolCallRequest`
- `ToolCallDecision`

Do not make these fancy.
Boring types are good here.

---

## 5. First Implementation Slices

Build this in narrow slices.

### Slice A: Kernel skeleton

Deliver:

- `internal/kernel` package
- contract types for `AgentDefinition`, `SessionState`, `WorkflowPack`, `MemoryCandidate`, `ToolPolicy`, `TraceEvent`, `ApprovalHook`, `EvalCase`
- JSON serialization round-trip tests

Rules:

- no behavior migration yet
- no package renames yet
- only type extraction and compile-safe adapters

Exit criteria:

- at least one consumer can import `internal/kernel` without importing trust-lane code

### Slice B: Adapter split

Deliver:

- move `internal/modelgateway` under `internal/adapters/model`
- define capability descriptors above provider specifics
- define runtime-adapter interface for OpenCode/OmO-style execution

Rules:

- existing routing behavior can stay simple
- adapter boundaries matter more than smarter routing

Exit criteria:

- at least two model/runtime backends can satisfy the same kernel-facing contract

### Slice C: Session/trace split

Deliver:

- keep generic `TraceEvent` in `internal/kernel`
- move hash-chained persistence and evidence-specific export into `internal/trust/evidence`
- reduce `internal/session` to consumer-specific helpers or delete it after migration

Exit criteria:

- trace production is substrate-agnostic
- trust export is optional and layered on top

### Slice D: Eval rebuild

Deliver:

- new `internal/evals` runner using `EvalCase` + `TraceEvent`
- keep current transcript-pattern runner only as compatibility fallback

Exit criteria:

- at least one routing or tool-policy regression is expressed as a trace assertion, not a string-match check

---

## 6. Faust-Oriented Minimum Viable Kernel

If the near-term consumer is Faust, the minimum useful slice is smaller than the full roadmap.

Faust Q2 should be able to consume:

1. `AgentDefinition`
2. `WorkflowPack`
3. `SessionState`
4. `TraceEvent`
5. `ToolPolicy`
6. model/runtime adapter capability descriptors

Faust does **not** need in the first slice:

- signed provenance
- enterprise audit export
- full memory persistence
- K8s orchestration

If the first slice cannot support one real product consumer, it is not a kernel.

---

## 7. Public vs Private Boundary

Recommendation:

- keep kernel, augmentation, adapters, and eval engine private inside `sdp_lab` for now
- publish only wire-level artifacts to the public `sdp` surface once they stabilize

Publish candidates later:

- trace-event JSON schema
- workflow-pack manifest schema
- attestation predicate schema
- policy profile schema

Do **not** publish yet:

- Go package APIs for kernel contracts
- runtime adapter internals
- product-facing pack authoring APIs

Reason:

- the architecture is still moving
- there is not yet proof of two stable external consumers
- publishing unstable package APIs will freeze the wrong seams

---

## 8. Current Contradictions To Clean Up

These are real and should be handled explicitly in follow-up work:

1. `docs/vision/README.md` still frames SDP as evidence-layer-only.
2. `docs/roadmap/ROADMAP.md` still presents trust/enforcement as the main story.
3. `docs/architecture/REPO-BOUNDARY.md` still assumes the public `sdp/` path mechanics from the old submodule model, while the current repo state no longer has a git submodule.
4. `docs/MULTI-REPO-WORKFLOW.md` is referenced by repo instructions but is missing.
5. local `bd` state is broken because the Dolt server is running but the `sdplab` database is unavailable.

Do not ignore these contradictions.
But also do not use them as an excuse to postpone the kernel boundary.

---

## 9. Immediate Next Step After This Spec

The next implementation pass should be:

1. create `internal/kernel`
2. add the first contract types with tests
3. move no more than one existing consumer onto the new contracts
4. create follow-up doc cleanup for canonical vision and repo-boundary drift

Anything larger than that will sprawl.
