# LLM Council Raw Log: SDP Work Atomicity Normalization - Final Spec
**Date:** 2026-04-12
**Subject:** [2026-04-12-work-atomicity-normalization-final.md](2026-04-12-work-atomicity-normalization-final.md)
**Validity note:** This raw log contains only the final valid amended pass on the consolidated final spec. Earlier council runs against pre-final drafts and one stalled sequential attempt on the amended spec are excluded from this artifact. Per user direction, `openai/gpt-5.4` was not used; the architect role ran on `anthropic/claude-sonnet-4.6`.
---

## Architect - `anthropic/claude-sonnet-4.6`

## Verdict

Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

---

## What The Amended Final Spec Fixes

**Live-state boundary is now clean.** The `active_issue_id` prohibition is explicit, `bound_primary_issue_id` is correctly defined as a compile-time static binding with a clear rationale, and the distinction between what the lock file owns versus what runtime derives is unambiguous throughout.

**Authority boundaries are properly separated.** Feature meaning lives in the Feature entity. Topology lives in the lock file. Live execution state lives in Beads. The spec no longer conflates these layers anywhere I can find.

**State machine is implementable.** Pre-claim checks, claim primitive, post-claim revalidation, abort-and-release, and the failure path for unconfirmable claim release are all specified with enough precision that an engineer can write the dispatcher without returning to the spec author for clarification.

**Reshape safety is closed.** Freeze-before-replace with the `dispatch_lifecycle` field participating in `source_inputs_hash` means a stale dispatcher will fail the freshness check during post-claim revalidation and abort before touching work. This is the correct mechanical closure.

**`mixed_invalid` is machine-checkable.** All nine conditions are enumerated, the classification is a validator error not a warning, and the feature is excluded from the lock file. This removes the ambiguity that existed in r3a about partial adoption.

**Hash canonicalization is fully specified.** Sort order, encoding, excluded fields, and the canonical JSON shape are all explicit. Two independent implementations should produce the same hash given the same inputs.

**Blocking policy is versioned and honest.** The priority 0–1 / 2–4 heuristic is labeled as a v1 heuristic with an explicit note that it will be replaced when Beads supports native blocker metadata. This is the right epistemic posture.

**Observability is enumerated.** Required counters and log codes are listed. This is the minimum needed for operational confidence.

**Aggregate status decision table is deterministic.** Row ordering is explicit, the shadow/enforced migration path is defined, and the rollout trigger is concrete enough to act on.

---

## Remaining Blockers

These are not veto-level but must be resolved before the implementation team cuts code against this spec.

**1. Claim release is underspecified for the unconfirmable case.**
The spec says emit `dispatch_claim_release_failed` and treat the claim as leaked until manually resolved. It does not say what "manually resolved" means operationally: is there an admin API call, a Beads UI action, a CLI command? Without this, leaked claims have no defined recovery path and will accumulate silently in production. This is a runtime safety gap.

**2. `dispatch_lifecycle: frozen` source-of-truth write is unowned.**
The spec says freeze requires `dispatch_lifecycle: frozen` in source and in the committed lock file. It does not say who writes the source frontmatter change: the reshape tooling, the engineer manually, a CI gate? If it is manual, the spec should say so explicitly. If it is tooling, that tooling is not mentioned anywhere in the implementation order.

**3. Advisory lock scope is underspecified.**
The interim mitigation says a dispatcher SHOULD take a local advisory lock keyed by `ws_id`. It does not specify the lock medium (file lock, in-process mutex, Redis, something else), the lock timeout, or what happens if the advisory lock cannot be acquired. SHOULD without a fallback behavior leaves implementers making inconsistent choices across runners.

**4. `aggregate_status = shadow-v1` warning sink is undefined.**
The spec says mismatch is a warning and the compiler still emits the feature. It does not say where the warning goes: stderr, a structured log, a CI annotation, a separate warning file? Without this, shadow-v1 warnings will be invisible in some CI environments and the rollout trigger ("one clean shadow cycle with no aggregate mismatch warnings in CI") cannot be mechanically verified.

**5. Retry jitter implementation is unspecified.**
"Jittered backoff 250ms..750ms" does not say whether this is uniform random, truncated exponential, or something else. For a single retry this is low stakes, but it should be explicit so implementations are consistent and the behavior is testable.

---

## Required Changes Before Adoption

These are the minimum changes needed. They are all additive or clarifying; none require restructuring the spec.

**Change 1 — Leaked claim recovery.**
Add a subsection under "Claim And Revalidation State Machine" titled "Leaked Claim Recovery." Specify the minimum required recovery operation: at minimum, a named CLI command or adapter method that accepts a `ws_id` and `issue_id` and forces a claim release. State whether this operation requires human confirmation. State what audit trail it must produce.

**Change 2 — Freeze write ownership.**
In the Reshape Contract, Step 1, add one sentence: either "The reshape tooling writes `dispatch_lifecycle: frozen` to source frontmatter as part of the freeze operation" or "The engineer writes `dispatch_lifecycle: frozen` manually; the freeze is not complete until the lock file is regenerated and committed." Pick one. Do not leave it implicit.

**Change 3 — Advisory lock specification.**
Replace the current SHOULD sentence with: "A dispatcher MUST attempt to acquire a local advisory lock keyed by `ws_id` using [specified medium]. If the lock cannot be acquired within [specified timeout], the dispatcher must treat this as a `leaf_conflict` and apply the retry rule." Add `advisory_lock_timeout_ms` to the minimum observability section or note it is implementation-defined with a recommended default.

**Change 4 — Shadow warning output.**
Add one sentence to the `shadow-v1` migration policy: "Aggregate status mismatch warnings must be emitted as structured log lines at WARN level with fields `feature_id`, `ws_id`, `declared_status`, `derived_status`, and `policy_version`." This makes the CI clean-cycle trigger mechanically verifiable.

**Change 5 — Jitter specification.**
Replace "jittered backoff 250ms..750ms" with "uniform random jitter in [250ms, 750ms]" or the intended distribution. One sentence.

---

## Bottom Line

This spec is substantially implementation-ready. The core architecture is sound: the live-state boundary is clean, the state machine is closed, the hash contract is deterministic, and the entity ownership model is coherent. The five remaining issues are all operational gaps in edge-case paths, not architectural problems. None of them require revisiting the fundamental design decisions.

The implementation team can begin work on items 1 and 2 of the implementation order (lock compiler and Beads adapter) immediately against this spec as written. Items 3 through 5 (dispatcher state machine onward) should wait for the five changes above to be incorporated, because three of the gaps directly affect dispatcher behavior.

This does not need another full council review. The owner should incorporate the five changes, mark the document as Final, and proceed to implementation.

---

## Critic - `mistralai/mistral-large-2512`

## Verdict
**Verdict:** CONDITIONAL_ACCEPT
**Formal veto:** NO

---

## What The Amended Final Spec Fixes
1. **State Ambiguity**: Explicitly separates static (git-committed) and live (Beads) state via `source_inputs_hash` and runtime Beads queries.
2. **Rollout Theater**: Introduces `legacy`, `normalized`, and `mixed_invalid` feature classes with clear migration paths (freeze-before-replace).
3. **Split-Brain Risk**: Bans `active_issue_id` and other live-state fields from the lock file, enforcing topology-only commits.
4. **Atomicity Leaks**: Hardens the claim/revalidation state machine with fail-closed semantics and mandatory release primitives.
5. **Schema Drift**: Canonicalizes lock file inputs via sorted JSON hashing, eliminating whitespace/comment noise.

---

## Remaining Blockers
1. **Leaf-Wide Exclusivity Gap**
   - The spec acknowledges but does not solve the race condition where two dispatchers claim different issues on the same leaf. The advisory lock mitigation is host-local and non-authoritative.
   - *Hidden State*: Beads lacks native leaf-scoped exclusivity, forcing reliance on post-claim revalidation (which may fail after work begins).

2. **Rollout Fragility**
   - Freeze-before-replace requires perfect coordination. If a dispatcher starts pre-claim checks *after* freeze but *before* lock regeneration, it may abort mid-flight, leaving orphaned claims.
   - *Race Condition*: No atomic freeze+lock operation; split-second windows exist where dispatchers see inconsistent states.

3. **Aggregate Status Ambiguity**
   - `shadow-v1` vs. `enforced-v1` creates a bifurcated state where some features are warnings and others are errors. This risks "rollout theater" where teams ignore warnings until forced to act.
   - *Hidden State*: The spec allows repositories to linger on `shadow-v1` indefinitely, delaying enforcement.

4. **Beads Query Reliability**
   - The adapter’s `beads_query_failed` contract assumes transient failures, but Beads outages could block all dispatch. No fallback to stale data is allowed.
   - *Machine-State Ambiguity*: The spec treats Beads as a single source of truth, but its availability is not guaranteed.

5. **Priority Heuristic**
   - The `priority 0..1 = blocking` heuristic is hardcoded in `dispatch_resolution = v1`. This couples policy to a fragile assumption about Beads metadata.

---

## Required Changes Before Adoption
1. **Leaf Exclusivity**
   - Add a `leaf_claim_token` field to the lock file (static, UUID per leaf). Dispatchers must include this token in Beads claim requests. Beads must reject claims with mismatched tokens.
   - *Why*: Converts leaf-wide exclusivity from advisory to authoritative.

2. **Atomic Freeze+Lock**
   - Introduce a `freeze.lock` file (committed alongside `workgraph.lock.json`) that atomically marks leaves as frozen. Dispatchers must check both files during pre-claim.
   - *Why*: Eliminates the split-second window between freeze and lock regeneration.

3. **Aggregate Enforcement Deadline**
   - Mandate that all repositories must flip to `enforced-v1` within 30 days of their first normalized feature. No indefinite `shadow-v1` allowed.
   - *Why*: Forces timely cleanup of warnings.

4. **Beads Fallback Policy**
   - Allow dispatchers to proceed with stale data (last known good Beads state) if:
     - The lock file is fresh.
     - The active issue was previously resolved and is still open.
     - A `WARN beads_stale_data_used` is emitted.
   - *Why*: Prevents total outages during Beads downtime.

5. **Priority Metadata**
   - Replace the hardcoded `priority 0..1 = blocking` heuristic with a `blocking: true|false` field in the lock file (derived from `## Beads` frontmatter).
   - *Why*: Decouples policy from Beads metadata assumptions.

6. **Claim Leak Detection**
   - Add a `claim_lease_expiry` field to Beads issues (e.g., 5 minutes). Dispatchers must refresh leases periodically. Stale claims are auto-released.
   - *Why*: Mitigates leaked claims from crashed dispatchers.

---

## Bottom Line
The spec is **technically sound** but **operationally brittle**. The fixes above address the remaining gaps without requiring a full rewrite. Key risks:
- **Leaf exclusivity races** (solved by `leaf_claim_token`).
- **Freeze/lock windows** (solved by `freeze.lock`).
- **Beads dependency** (solved by stale-data fallback).

**Conditional acceptance** hinges on implementing the required changes. The core model—static topology + live Beads state—is correct, but the edges need hardening.

---

## Technician - `deepseek/deepseek-v3.2`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What The Amended Final Spec Fixes
The amended spec successfully addresses several critical blockers from prior reviews:
1.  **Compiler/Runtime Boundary:** It clearly defines the `source_inputs_hash` contract, explicitly listing included and excluded inputs, and mandates canonicalization before hashing. This creates a deterministic, reproducible boundary.
2.  **Runtime Adapter Contracts:** It provides a precise, normalized JSON shape for the Beads adapter to return, defines a clear error contract, and establishes a fail-closed rule for query failures.
3.  **Race Windows & Failure Recovery:** The post-claim revalidation state machine is robust. It mandates re-checking lock freshness, lifecycle state, and live Beads data *after* claiming, closing the race window where source changes could go undetected. Failure modes (abort, claim release, observability) are well-defined.
4.  **Live State Exclusion:** The spec is unequivocal: the committed `workgraph.lock.json` must not contain `active_issue_id` or any field derived from live Beads state (claim state, openness, priority changes). `bound_primary_issue_id` is correctly defined as a static binding.
5.  **Reshape Safety:** The freeze-before-replace contract, combined with the `source_inputs_hash` and post-claim revalidation, creates a safe path for topology changes.

## Remaining Blockers
One significant conceptual/operational blocker remains:
1.  **Leaf-Wide Exclusivity is Fundamentally Broken:** The spec correctly identifies this as a "Known Limitation" but understates the severity. Relying on post-claim revalidation to detect and abort concurrent claims on the *same leaf* is a **live-lock hazard**. Dispatchers A and B can perpetually claim different issues on the same leaf, each aborting the other's claim, making the leaf undispatchable. The suggested local advisory lock is a non-solution for distributed systems. This violates the core guarantee that a leaf owns "one executable contract slice" and "one leaf-scoped execution slot."

## Required Changes Before Adoption
The spec must be amended to resolve the leaf-wide exclusivity problem **before implementation begins**. A runtime mechanism is required, not just observability of the failure mode.
*   **Proposed Amendment:** Introduce a **leaf-scoped claim token or lock** as a first-class runtime primitive. This could be:
    *   A new field in the Beads adapter's normalized response indicating if *any* issue on the leaf's bound set is claimed.
    *   A separate atomic lock primitive (e.g., in Beads or a coordination service) keyed by `ws_id`.
    The pre-claim check must include verifying the leaf is unclaimed. The claim operation must atomically acquire the leaf lock *in addition to* the issue claim. The post-claim revalidation must re-check the leaf lock. This closes the race condition at its root.

## Bottom Line
The technical architecture is sound and addresses the major concerns around boundaries, state separation, and race conditions. However, the admitted flaw in leaf-wide exclusivity is a critical deficiency that undermines the dispatch safety model. **Adoption is conditional on specifying a concrete, atomic mechanism for leaf-scoped exclusivity.** Without this, the system will be prone to live-lock scenarios and cannot be considered "safe enough to execute." Implement the proposed leaf-lock primitive, and the spec will be ready for implementation.

---

## Philosopher - `anthropic/claude-opus-4.6`



## Verdict

Verdict: **ACCEPT**
Formal veto: **NO**

---

## What The Amended Final Spec Fixes

This spec achieves something genuinely difficult: it draws a clean ontological line between **compile-time topology** and **runtime liveness**, and then holds that line consistently across every subsystem it touches.

**The core philosophical achievement is the separation of two causal orders.** The lock file captures the *structural* causal order — what depends on what, what is bound to what, what shape the work takes. The Beads query captures the *temporal* causal order — what is open now, what is claimed now, what priority obtains now. The spec never confuses these two. `bound_primary_issue_id` is a structural fact ("this leaf is about this issue"); `is_open` is a temporal fact ("this issue is currently active"). The amended spec is explicit that the former is a compile-time binding and the latter is a runtime query. This resolves the central ambiguity that plagued earlier drafts.

**Category boundaries are crisp and exhaustive.** The entity model assigns ownership cleanly: Features own *meaning*, aggregates own *decomposition*, leaves own *execution slots*, Beads issues own *live state*. No entity owns something that properly belongs to another. The "does not own" lists are as important as the "owns" lists — they function as explicit negative constraints that prevent category drift. This is good ontological hygiene.

**The normalization trichotomy (`legacy` / `normalized` / `mixed_invalid`) is genuinely exhaustive.** Every feature must be exactly one of these, and the nine machine-checkable conditions for `mixed_invalid` are sufficient to partition the space without gaps. There is no feature that could satisfy some conditions of `normalized` while failing others and yet escape classification — the validator catches it.

**The `source_inputs_hash` contract is the strongest single section.** The explicit inclusion/exclusion lists, the canonicalization rules, and the requirement that keys be lexically sorted before hashing eliminate an entire class of accidental staleness bugs. The decision to exclude `priority`, `size`, and `depends_on` from the hash is philosophically sound: these are planning-layer concerns, not topology, and changes to them should not invalidate the dispatch contract.

**The freeze-before-replace reshape contract correctly handles the temporal race.** By making `dispatch_lifecycle` part of the hash input, a freeze *necessarily* invalidates the old lock, which *necessarily* fails the freshness check in post-claim revalidation. This is causality-by-construction rather than causality-by-convention — the strongest kind.

**The Known Limitation section is honest and bounded.** Rather than pretending leaf-wide exclusivity is solved, the spec names the exact failure mode (two dispatchers claiming different issues on the same leaf), accepts it as a v1 limitation, provides an interim mitigation (advisory lock), and requires observability. This is the right philosophical posture: name what you cannot guarantee, make the failure visible, and don't let it infect the parts you *can* guarantee.

---

## Remaining Blockers

**None that rise to the level of blocking adoption.**

There are philosophical tensions that remain, but they are *managed* tensions rather than contradictions:

1. **The `children` field in aggregate lock entries is a derived redundancy.** Linkage is declared as child-to-parent only in source, but the lock file contains a `children` array on aggregates. This is a computed convenience, not a second source of truth, but the spec could be one sentence more explicit that this array is compiler-derived and never hand-authored. This is not a blocker because the lock file is entirely compiler-generated.

2. **The aggregate status decision table has a minor ordering subtlety.** Row 2 checks "all child leaves are `done|archived` **and** no open aggregate findings exist." Row 3 checks "any open blocking aggregate finding exists." If all children are done but a non-blocking finding is open, row 2 fails (because of the "no open aggregate findings" clause), row 3 fails (because the finding is non-blocking), and we fall through to row 5 which yields `open`. This is arguably correct — an aggregate with unresolved findings shouldn't be `done` — but it means a non-blocking finding can hold an aggregate in `open` even when all execution is complete. This is a policy choice, not a logical error, and it's the conservative one.

3. **The retry rule permits automatic retry only for `leaf_conflict`**, but the spec doesn't define `leaf_conflict` as a named failure type in the revalidation section — it appears only in the retry and observability sections. The meaning is inferrable from context (two dispatchers contending on the same leaf), but a one-line definition would close the loop. Minor.

---

## Required Changes Before Adoption

**None.** The items above are refinements that can be addressed in implementation or in a v1.1 clarification pass. They do not represent logical inconsistencies, category errors, or causal ambiguities.

---

## Bottom Line

This spec says **one consistent thing**: work topology is a committed structural fact; liveness is a queried temporal fact; the boundary between them is the lock file; and every subsystem — validation, compilation, dispatch, reshape — respects that boundary without exception.

The conceptual architecture is sound. The causal model is coherent. The category boundaries hold. The entity ownership is clean. The known limitations are named rather than hidden. The implementation order follows logically from the dependency structure of the concepts.

Ship it.

---

## Pragmatist - `minimax/minimax-m2.7`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What The Amended Final Spec Fixes

The spec correctly fixes the live-state boundary problem from earlier versions by:

1. **Binding vs. Live distinction** — `bound_primary_issue_id` is explicitly a static, compile-time declaration, not a live query result. This prevents false snapshotting of git state.

2. **Source inputs hash scope** — The hash contract explicitly enumerates included and excluded inputs. Fields like `priority`, `size`, and `depends_on` are correctly excluded, preventing spurious lock invalidation.

3. **Shadow-to-enforced migration path** — The `aggregate_status = shadow-v1 | enforced-v1` dual-mode with CI-triggered flip is the right shape. It lets teams adopt without immediate enforcement while providing a clear exit criterion.

4. **Reshape safety** — Freeze-before-replace with post-claim revalidation abort is sound. The freshness check during revalidation handles the freeze-mid-flight case.

5. **Known limitations documented** — Leaf-wide exclusivity is flagged as best-effort with an advisory lock mitigation. This prevents scope creep while being honest about the gap.

## Remaining blockers

### 1. Unclear what "blocking finding" means in aggregate status table

The decision table for `derived_status` references "open blocking aggregate finding" in row 3, but the only explicit blocking policy defined is in the Active Issue Resolution section for **leaf** dispatch:

> `priority 0..1` => blocking finding

The aggregate status table does not explain whether this same priority rule applies to aggregate findings, or whether aggregate findings have a different blocking semantics. **This creates a real divergence risk**: a dispatcher might interpret "blocking finding" for leaves differently than the aggregate status engine interprets it for aggregates.

**Impact**: Silent mismatches between what blocks leaf dispatch and what blocks aggregate status. This is subtle and will surface only in edge cases.

### 2. No recovery procedure for leaked claims

The spec acknowledges:

> If claim release cannot be confirmed, the dispatcher must emit `dispatch_claim_release_failed` and treat the claim as leaked until manually resolved.

"Manually resolved" is not defined. In a multi-runner CI environment, manual resolution requires someone to:

- Detect the leak (counter is incremented, log is written)
- Identify which runner leaked it (not stored in Beads today based on this spec)
- Know which claim was held
- Release it

If Beads does not expose "who holds this claim" to operators, manual resolution is **impossible without Beads schema changes**. The spec should either require Beads to expose ownership visibility, or define an explicit TTL-based auto-release fallback.

**Impact**: In any runner failure scenario, you can permanently block a leaf from dispatching until someone patches the Beads database directly.

### 3. `mixed_invalid` validation is undefined

The spec lists machine-checkable conditions for `mixed_invalid` classification but does not define:

- Who runs the validator (pre-commit hook? CI? on-demand?)
- At what point in the workflow validation triggers
- Whether validation failure blocks `workgraph.lock.json` regeneration or blocks dispatch
- Whether a `mixed_invalid` feature can be reclassified by fixing violations, or requires manual re-flagging

In practice, if a team member makes a typo in a workstream header, does their next commit fail to generate a lock file? Does CI fail? Does dispatch silently skip the feature? The spec says "validator error, not warning" for `mixed_invalid`, which implies hard failure, but it doesn't say where or when.

**Impact**: Teams will have inconsistent validation behavior until this is standardized. Someone will ship a pre-commit hook, it will be wrong, and CI will break.

### 4. No explicit schema for the Beads adapter response

The runtime Beads query contract says the adapter "must return this normalized shape":

```json
{
  "id": "sdplab-123",
  "is_open": true,
  "is_claimed": false,
  "priority": 1,
  "created_at": "2026-04-12T15:22:25Z",
  "status": "open"
}
```

But there is no JSON Schema, Protobuf definition, or TypeScript interface in the spec. Every implementation will derive this from prose, and the interpretation of "non-terminal issues only" for `is_open` will vary.

**Impact**: Adapter implementations will drift. The `status: open` field conflicts conceptually with `is_open` (is it redundant? must they agree? what if they disagree?). This will cause subtle dispatch failures.

## Required changes before adoption

| # | Change | Priority | Rationale |
|---|--------|----------|-----------|
| 1 | Define "blocking finding" semantics for aggregate status.

---

## Engineer - `xiaomi/mimo-v2-pro`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What The Amended Final Spec Fixes
The amended spec successfully addresses the core architectural flaws identified in previous critiques:

1.  **Clear Separation of Static vs. Live State:** The explicit prohibition of `active_issue_id` and other mutable Beads-derived fields in the committed `workgraph.lock.json` is the most critical fix. This prevents the lock file from becoming a stale snapshot of live state, which was a fundamental design flaw.
2.  **Machine-Readable Contracts:** The specification now provides complete, machine-parseable schemas for:
    *   Workstream frontmatter (with strict `ws_kind` and `parent_ws_id` rules).
    *   The `## Beads` section (with defined roles and cardinality rules).
    *   The lock file structure (with a clear `source_inputs_hash` contract).
    *   The Beads adapter's query and error response formats.
3.  **Explicit Adapter Boundaries:** The "Runtime Beads Query Contract" and "Binding visibility rule" cleanly define the adapter's responsibility: it is a stateless query interface for bound issue IDs only. The spec forbids the adapter from influencing topology or injecting unbound issues into dispatch logic.
4.  **Normalized Feature Lifecycle:** The `legacy/normalized/mixed_invalid` classification and the `dispatch_lifecycle` field provide a safe, incremental migration path. The machine-checkable conditions for `mixed_invalid` are particularly well-specified.
5.  **Deterministic Compilation:** The `source_inputs_hash` canonicalization rules are exhaustive, explicitly excluding volatile data (Markdown prose, comments, live state, timestamps). This makes lock file freshness a verifiable, deterministic property.

## Remaining blockers
1.  **Ambiguous "Active Issue" in Revalidation:** The "Post-claim revalidation" step requires re-checking the "resolved active issue." However, the "Active Issue Resolution" algorithm is priority-based and can change if a new finding is created or an issue's priority changes between pre-claim and post-claim. The spec states "resolved active issue changed" as an abort condition but does not specify if this is a strict equality check on the issue ID or a re-execution of the full resolution algorithm. This ambiguity could lead to inconsistent dispatcher implementations.
2.  **Adapter Contract Lacks Bulk Query Semantics:** The contract shows a single-issue response shape. For a leaf with a `primary` and multiple `finding` issues, the dispatcher must query all bound IDs. The spec does not define if the adapter should support a bulk query endpoint (`POST /query` with `["sdplab-123", "sdplab-456"]`) or if the dispatcher must make N sequential calls. This has performance and atomicity implications.
3.  **No Schema Versioning for Source Inputs:** The `schema_version` in the lock file refers to the lock file format itself. There is no version identifier for the *source input schema* (the frontmatter and `## Beads` rules). A future, breaking change to the allowed frontmatter fields or Beads roles would require a coordinated update of the compiler, all validators, and potentially the dispatcher, with no machine-checkable way to detect incompatibility.

## Required changes before adoption
1.  **Clarify Revalidation Logic:** In the "Post-claim revalidation" section, explicitly state: "The dispatcher must re-run the 'Active Issue Resolution' algorithm using the fresh Beads query results. If the resulting `active_issue_id` differs from the one just claimed, revalidation fails."
2.  **Define Adapter Query Interface:** Add a subsection under "Runtime Beads Query Contract" specifying:
    *   The adapter MUST support a bulk query method accepting an array of issue IDs.
    *   The response MUST be a JSON array of the defined issue objects, in the same order as the request, or a top-level error.
    *   The query for a single leaf's bound issues MUST be atomic (all issues reflect the same point in time).
3.  **Introduce Source Schema Version:** Add a `source_schema_version` field to the canonical JSON object used for `source_inputs_hash` calculation and to the lock file's `policy_versions` map. Initialize it to `"v1"`.

## Bottom line
This amended spec is a **substantial and necessary improvement**. It correctly isolates static topology from live execution state, defines rigorous machine-readable contracts, and establishes a safe migration framework. The core architecture is sound.

Acceptance is **conditional** on resolving the three blockers above, which are primarily about removing ambiguity and hardening the contracts between components. The "Known Limitation: Leaf-Wide Exclusivity" is honestly documented and acceptable for a v1 resolution. With the required clarifications, this spec provides a viable target for implementation.

---
