# SDP Agent Platform Roadmap (2026-03-31)

Status: draft
Owner: SDP core
Scope: refocus `sdp_lab` on reusable agent kernel, augmentation, adapters, and evals
Related:

- `docs/roadmap/ROADMAP.md`
- `docs/roadmap/UNIFIED_VISION_ROADMAP_2026-03-03.md`
- `docs/vision/README.md`
- `docs/plans/2026-03-31-agent-platform-kernel-boundary-spec.md`

## Why this roadmap exists

The current roadmap is strong on trust, evidence, and standards-based enforcement.
It is weak on the reusable agent substrate that products will actually build on.

The codebase already shows the gap:

- `internal/harness` is mostly contract-compliance logic, not a general agent kernel
- `internal/eval` is still transcript-pattern matching, not meaningful behavioral evaluation
- `internal/runtime` is queue-control logic, not a reusable execution runtime

That means `sdp_lab` is at risk of becoming an internal control-tower framework with many opinions and a thin core.

That is not the right platform.

## Thesis

`sdp_lab` should become the reusable agent platform layer for products like Faust.

It should own:

- kernel contracts
- augmentation mechanics
- adapter boundaries
- eval infrastructure
- trust and provenance primitives

It should not own:

- end-user product personas
- consultant-facing UX
- domain-specific workflow libraries
- product-specific approval semantics

## Hard Position

“Full refusal from vendor harnesses” is the wrong target.

The right target is **dependency inversion**:

- products should not depend on vendor-harness concepts
- `sdp_lab` may still integrate vendor or open-source runtimes as adapters
- swapping OpenCode, Claude-style, or custom substrates should not force product redesign

If we can change the substrate without changing the product contract, we have already won.

## Strategic Role of `sdp_lab`

`sdp_lab` should own five layers.

### 1. Kernel

Typed runtime contracts:

- `AgentDefinition`
- `SessionState`
- `TaskEnvelope`
- `WorkflowPack`
- `MemoryCandidate`
- `ToolPolicy`
- `TraceEvent`
- `ApprovalHook`
- `EvalCase`

### 2. Augmentation

Reusable mechanics for:

- lazy-loaded skills and packs
- planner and verifier roles
- tool gating
- memory-candidate emission
- retrieval and context-budget hooks

### 3. Adapters

Swappable execution and model layers for:

- OpenAI API
- OpenRouter API
- OpenCode
- future local or hosted runtimes

### 4. Evals

Behavioral regression infrastructure for:

- routing
- tool use
- memory writes
- workflow execution
- approval behavior
- artifact quality checks

### 5. Trust Lane

Evidence, provenance, attestation, policy, and enterprise enforcement.

Important: this is a lane, not the whole platform story.

## What `sdp_lab` must stop trying to be

- not a consultant product shell
- not a persona system
- not a giant repo of hand-written workflow content
- not a premature K8s autonomy platform before the local kernel is coherent

Kubernetes orchestration may matter later.
Right now it is a research lane, not the main path.

## Phase 0: Boundary Reset

### Goal

Rename the problem correctly and stop pretending the trust layer is the full agent platform.

### Build

- document package ownership for kernel, augmentation, adapters, evals, and trust
- separate generic runtime terms from evidence-specific terms
- treat products such as Faust as first-class consumers in the roadmap
- demote “K8s dream” work from primary execution path to research lane

### Exit Criteria

- roadmap language reflects platform layers, not only enforcement layers
- the next implementation slices map to real packages, not only documents

## Phase 1: Kernel Extraction

### Goal

Create the minimum reusable platform contracts.

### Build

- core typed contracts for agent definition, workflow pack, memory candidate, tool policy, traces, approvals, and eval cases
- session state and run identity primitives
- context-budget and compaction interfaces
- deterministic serialization rules for trace and replay

### Refactor

- narrow `internal/harness` so evidence-compliance types stop masquerading as generic agent contracts
- make generic runtime types visible and testable as their own package boundary

### Exit Criteria

- `sdp_lab` exposes a believable kernel surface
- a product can depend on these contracts without importing trust-layer baggage

## Phase 2: Adapter and Gateway Layer

### Goal

Make the execution substrate swappable.

### Build

- OpenAI API adapter
- OpenRouter adapter
- OpenCode adapter
- capability descriptors for tool calling, streaming, reasoning controls, context limits, and cost behavior
- unified gateway interface above provider specifics

### Native Runtime Position

Prefer Go for the first-party runtime helpers because the repo is already Go-heavy.

Use Rust only when there is a clear need for:

- PTY or terminal isolation
- sandboxing
- high-volume streaming
- performance-critical subprocess control

Language purity is not a strategy.

### Exit Criteria

- products can switch between at least two substrates without changing their own orchestration contracts
- provider-specific quirks are contained in adapters

## Phase 3: Augmentation Engine

### Goal

Turn packs, hooks, and policies into real reusable mechanics.

### Build

- lazy workflow-pack loader
- pack manifests and versioning
- hook surfaces for tool policy, memory candidate creation, approvals, and trace enrichment
- generic planner, verifier, and specialist-worker primitives
- replayable context-assembly pipeline

### Must Avoid

- baking consultant-domain assumptions into generic packs
- loading every skill or policy into the root context
- mixing durable memory with chat compaction summaries

### Exit Criteria

- augmentation logic is modular and lazy by default
- products can author domain packs without forking the kernel

## Phase 4: Behavioral Eval System

### Goal

Replace shallow transcript checks with regression infrastructure that actually protects behavior.

### Build

- scenario-based eval runner
- trace assertions
- routing evals
- tool-policy evals
- memory-write evals
- workflow-pack evals
- artifact-quality reference checks

### Refactor

- move beyond `required_patterns` and `forbidden_patterns` as the primary quality model
- use real traces from product consumers such as Faust to build regression suites

### Exit Criteria

- eval failures explain behavioral regressions, not just string mismatches
- platform changes require passing behavior-level gates

## Phase 5: Trust and Enterprise Hardening

### Goal

Make trust features strengthen the platform instead of substituting for it.

### Build

- provenance and trace signing
- in-toto or equivalent attestation path
- executable policy profiles
- enterprise-grade evidence and audit exports

### Position

This work remains important.
It just should not be mistaken for the whole reusable agent platform.

### Exit Criteria

- trust features sit cleanly on top of kernel, adapters, and evals
- enterprise controls do not require product teams to inherit internal complexity by accident

## Extraction Rule for New Work

New abstractions should graduate into `sdp_lab` only when at least one of these is true:

- two different products or runtimes need them
- the concern is clearly substrate-level rather than domain-level
- the abstraction improves evalability, swappability, or policy control

If it only serves one product’s UX or domain semantics, it stays out.

## Near-Term Build Order

The next platform sequence should be:

1. kernel contract map
2. adapter boundary and gateway proof
3. eval system rewrite
4. augmentation engine
5. trust-lane integration on top

That order matters.
Doing trust harder before kernel and evals are real will create more policy surface on top of a weak foundation.

## Recommendation

`sdp_lab` should spend the next cycle becoming a clean, swappable, evaluable agent platform.

Not bigger.
Cleaner.

If it does that, Faust gets a usable upstream.
If it does not, Faust will either stay trapped in vendor harness assumptions or be forced to build its own substrate in the product repo.
