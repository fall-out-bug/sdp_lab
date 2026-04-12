# LLM Council Raw Log: SDP Work Atomicity Normalization - Revision 3
**Date:** 2026-04-12
**Subject:** [2026-04-12-work-atomicity-normalization-r3.md](2026-04-12-work-atomicity-normalization-r3.md)
**Validity note:** This raw log contains only the final valid round. Earlier discarded attempts are not part of council evidence: `codex-rescue` architect output self-identified as a simulation rather than a distinct model runtime, and `moonshotai/kimi-k2.5` philosopher attempts returned unusable/truncated reasoning-only content.
---

## Architect - `openai/gpt-5.4`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What Rev 3 Fixes
Rev 3 materially improves the prior runtime and migration story. The main blockers from Rev 2 are mostly addressed at the architectural level:

- **Runtime authority boundary is much clearer**
  - Establishing `workgraph.lock.json` as the **only runtime input** for normalized features is the right move.
  - It removes live Markdown parsing from dispatch paths, which resolves the prior ambiguity around “authoring truth” versus “runtime truth” without making Markdown and runtime peers.
  - The `git_commit` pin and “must match HEAD” rule are a credible answer to the TOCTOU/parser drift problem.

- **State ownership is cleaner**
  - Markdown/frontmatter/`## Beads` remain the human-authored source.
  - The lock file is explicitly a generated projection.
  - `.beads-sdp-mapping.jsonl` is demoted to helper status, reducing accidental authority creep.
  - This is a healthier ownership model than Rev 2.

- **Migration coherence is substantially better**
  - The switch to **per-feature classification** (`legacy`, `normalized`, `mixed_invalid`) fixes the major migration contradiction in Rev 2.
  - Blocking `mixed_invalid` features from dispatch removes undefined mixed-parser behavior.
  - Applying new enforcement only to `normalized` features is the correct scoped cutover model.

- **Serialized execution semantics are now explicit**
  - The “one execution slot per leaf” rule is finally stated in machine terms via `active_issue_id`.
  - Explicitly declaring that `primary` and `finding` do not create separate execution lanes resolves the prior semantic ambiguity.
  - The compiler-side selection order gives deterministic dispatch intent.

- **Aggregate status is closer to derived, enforceable state**
  - Rev 3 correctly treats aggregate workstreams as non-executable and derived.
  - Requiring validator agreement between `declared_status` and `derived_status` is a reasonable compromise for human readability with machine enforcement.

- **Reshape protocol is much safer than before**
  - Freeze-before-replace is the right high-level protocol.
  - Blocking reshape when affected issues are actively claimed directly addresses the unsafe concurrent reshape problem raised in prior review.
  - The two-step cutover is structurally coherent.

Overall: Rev 3 does resolve the **conceptual** runtime-authority and migration-model failures that blocked Rev 2.

## Remaining blockers
I do not think Rev 3 fully resolves all prior **runtime and migration blockers** in an implementation-safe way yet. The remaining issues are narrower, but still material.

- **Lock file freshness rule is underspecified for real runtime safety**
  - “If the lock file does not match HEAD, normalized features are not dispatchable” is directionally correct, but not enough.
  - You need to define what counts as a match:
    - exact `git_commit == HEAD`?
    - exact tree hash of relevant source files?
    - generated artifact present in working tree but not committed?
  - In practice, dispatch safety depends on the lock file being generated from the exact source snapshot the runtime is using. “HEAD” alone is not sufficient if runtime can see dirty working trees, detached states, partial checkouts, or generated-but-uncommitted artifacts.
  - Without a stricter identity rule, authority remains slightly porous.

- **Post-claim revalidation is not a complete ownership protocol**
  - Rev 3 says to claim, then re-read HEAD/lock/live Beads state, and abort if invalidated.
  - That is useful, but it still leaves ambiguity about the ownership source for claim exclusivity:
    - Is Beads the sole source of claim truth?
    - Does the dispatcher need a compare-and-swap style claim operation?
    - What prevents two dispatchers from simultaneously passing pre-checks, claiming different issues on the same leaf, then both needing to unwind?
  - The memo correctly says the system is not globally atomic, but it still needs a stricter statement of what atomic primitive exists and which system owns it.
  - Without that, runtime safety is improved but not fully closed.

- **`active_issue_id` derivation may be unstable unless priority semantics are standardized**
  - The selection order depends on “highest-priority” blocking/non-blocking findings.
  - But the memo does not define:
    - the allowed priority domain,
    - whether absence of priority is valid,
    - whether priority comes from Beads or authoring docs,
    - whether changing priority mid-flight invalidates active execution.
  - Since `active_issue_id` is the single execution slot, its derivation must be fully specified, not left to implementation convention.

- **Aggregate derivation is improved but still has edge-case ambiguity**
  - The aggregate derivation order is much better, but rule 5 is too loose:
    - “open if any child is `open|blocked|done` but aggregate is not yet done”
  - This collapses several semantically different states and may mask compiler/model errors.
  - More importantly, the aggregate contract assumes child `derived_status` values are already valid and stable, but the memo does not specify validation ordering or cycle/error handling strongly enough.
  - If compilation encounters malformed child relationships, duplicate parentage, missing children, or orphan leaves, does the whole feature become `mixed_invalid`, or can partial aggregate derivation proceed? It should be explicit.

- **Reshape freeze semantics are not machine-defined enough**
  - “affected leaf workstreams become non-dispatchable in the next lock file” is correct, but there is no explicit runtime field for frozen/suspended/non-dispatchable state.
  - Moving current primaries to `historical` or marking them non-ready in Beads introduces dual mechanisms.
  - That weakens migration coherence because freeze safety should compile into one unambiguous runtime state in the lock file.
  - As written, reshape safety still depends partly on conventions in Beads, not solely on compiled runtime state.

- **Migration rollback is operationally coherent, but not state-safe enough**
  - “Revert normalization commit set, regenerate lock file, return feature to fully legacy or normalized” is fine at repo level.
  - But there is no explicit rule for what happens to **live claims** when a normalized feature is rolled back to legacy.
  - If state ownership changes between runtime paths, rollback must specify claim draining or blocking before mode transition.
  - Otherwise rollback can strand in-flight work across authority regimes.

- **Feature-level mode ownership needs one more hard rule**
  - Rev 3 says legacy features are excluded from the lock file and continue on the old path.
  - Good.
  - But it does not explicitly state that **cross-feature references are illegal** in normalized compilation/runtime derivation.
  - If aggregate or issue linkage crosses feature boundaries, per-feature normalization becomes incoherent.
  - This matters for migration coherence because mode is assigned per feature.

## Required changes before adoption
These are the minimum changes I would require to remove the remaining runtime/migration risk:

- **Define lock identity strictly**
  - Replace “matches HEAD” with an exact rule such as:
    - runtime dispatch is allowed only when `workgraph.lock.json.git_commit == current HEAD commit`
    - and the runtime environment must be on a clean checkout with no uncommitted changes affecting normalized feature sources
  - If dirty trees are allowed, define an alternate content-hash scheme instead. Do not leave this implicit.

- **Define the atomic authority for claims**
  - State explicitly that Beads claim acquisition is the single atomic primitive for execution ownership.
  - Require claim acquisition to fail if any issue on the same leaf is already claimed.
  - If Beads cannot enforce leaf-scoped exclusivity atomically, the dispatcher must maintain a separate atomic leaf lease; otherwise the design still has a race hole.

- **Fully specify `active_issue_id` derivation inputs**
  - Define the canonical source and legal value set for:
    - blocking/non-blocking
    - priority
    - `created_at`
  - Define compiler behavior for missing/invalid metadata.
  - Define whether any change to those inputs invalidates an already-claimed issue during post-claim revalidation.

- **Make freeze/non-dispatchable a compiled runtime state**
  - Add an explicit lock-file field, e.g. `dispatchable: true|false` or `lifecycle_state: active|frozen|archived`.
  - Do not rely on a mix of “historical” and “non-ready in Beads” as alternate encodings for freeze.
  - The dispatcher and reshape guard should consume one machine-state contract.

- **Tighten compilation failure semantics**
  - Explicitly state that normalized compilation is all-or-nothing per feature.
  - Any malformed topology, duplicate parentage, orphan child, cycle, invalid status dependency, or unresolved issue binding should classify the feature as `mixed_invalid` or otherwise non-dispatchable.
  - No partial feature emission into the lock file.

- **Define rollback/mode-transition quiescence**
  - Before a feature transitions from `normalized` to `legacy` or vice versa, all live claims on that feature’s leaf issues must be absent.
  - Otherwise migration changes authority regimes while work is in flight.

- **Ban cross-feature runtime topology**
  - Add a hard validator rule that parent/child workstream relationships and aggregate derivation may not cross feature boundaries.
  - If such references exist, compilation fails for that feature set.

## Bottom line
Rev 3 is a significant improvement and is the first revision that has a plausibly implementable authority and migration model.

However, it does **not yet fully close** the prior runtime and migration blockers because the design still lacks:
- a fully explicit atomic claim authority,
- a strict lock identity/freshness contract,
- machine-defined freeze state,
- and quiescent mode-transition rules.

So: no formal veto, but not unconditional acceptance. With the required changes above, this should be adoptable.

---

## Critic - `google/gemini-3.1-pro-preview`

## Verdict
Verdict: REJECT
Formal veto: YES

## What Rev 3 Fixes
Revision 3 successfully eliminates the fuzzy "warn-mode" migration by introducing strict per-feature normalization states (`legacy`, `normalized`, `mixed_invalid`). It correctly identifies that Markdown parsing must be removed from the runtime critical path, introducing `workgraph.lock.json` as a compiled artifact. The explicit definition of serialized execution (one slot per leaf) and the two-step reshape protocol are massive conceptual improvements over Revision 2. 

## Remaining blockers

**1. The Lockfile/Beads Split-Brain (Fatal)**
The spec states: *"the compiler derives `active_issue_id` [based on open findings vs primary]... The compiler must precompute adjacency and derived status so runtime callers do not scan the backlog."* 
This is a catastrophic distributed systems failure. If the compiler bakes the `active_issue_id` into a static Git lockfile, it is taking a point-in-time snapshot of live Beads state. If a critical blocking finding is filed in Beads 5 minutes after the lockfile is merged, the lockfile still points to the `primary` issue. The dispatcher will read the lockfile, ignore the live blocking finding, and execute the primary. You have recreated the exact concurrency violation you were trying to solve.

**2. The Git Hash Chicken-and-Egg**
The lockfile requires a `git_commit` field, and the runtime rules state: *"if the lock file does not match HEAD, normalized features are not dispatchable."* You cannot embed a Git commit hash inside a file that is part of that same commit. The hash changes when the file changes. This is a literal impossibility unless you are referencing the *parent* commit or a tree hash, which is underspecified.

**3. Reshape TOCTOU (Time-of-Check to Time-of-Use)**
*"Hard safety rule: If any affected issue remains actively claimed during Step 1, reshape is blocked."* 
Blocked by what? If this is a CI check on the PR, there is a race condition. CI checks the claim state at 1:00 PM and passes. A runner claims the issue at 1:10 PM. The PR merges at 1:15 PM. You have now merged a freeze state while a runner is actively executing the leaf. The spec provides no merge-time or post-merge mechanism to evict the runner.

**4. Governance Theater on Aggregate Status**
*"Validator rule: aggregate `declared_status` must match `derived_status`."*
If the machine perfectly derives the aggregate status from its children, forcing a human to manually edit the Markdown frontmatter to match the machine's derivation just to get CI to pass is pure toil. It is governance theater that will infuriate developers.

## Required changes before adoption

1. **Move `active_issue_id` derivation to the runtime:** The lockfile must only store the *topology* (leaves, aggregates, and their known issue IDs). The dispatcher must query live Beads state at runtime to determine which of those IDs is the `active_issue_id` based on the priority rules. Do not compile live state into static files.
2. **Fix the Git hash validation:** Remove the self-referential `git_commit` hash from the lockfile. Rely on Git's native tree state, or explicitly specify that the hash represents the commit that *triggered* the generation (which means it will always be `HEAD~1` relative to the generation commit).
3. **Handle the Reshape Race Condition:** Specify that if a dispatcher is executing a claim and a Step 1 Freeze commit merges, the dispatcher's post-claim or mid-execution heartbeat must detect the topology change (via the new lockfile) and immediately abort/evict the run. You cannot rely solely on PR CI to block reshapes.
4. **Eliminate Aggregate Status Toil:** Remove `declared_status` from aggregate workstream frontmatter entirely. If it is strictly derived, it belongs only in the lockfile and UI projections, not in human-authored source files.

## Bottom line
Revision 3 has the right architecture (compiled topology, strict serialization, phased migration) but fails on the implementation details of state management. Compiling live ticket state into a static Git lockfile fundamentally breaks the atomicity model by blinding the runtime to newly filed findings. Move topology to the lockfile and live state resolution to the dispatcher, fix the Git hash paradox, and this will be ready for implementation.

---

## Technician - `deepseek/deepseek-v3.2`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What Rev 3 Fixes
Revision 3 directly addresses the four concrete failures identified in the critique of Revision 2:
1.  **Reshape safety under live claims:** Introduces a generated `workgraph.lock.json` as the sole runtime input for reshape guards, preventing TOCTOU issues by ensuring reshape decisions are based on a consistent, compiled snapshot.
2.  **Mixed old/new format contradictions:** Replaces the ambiguous Phase 2/Phase 4 model with a clear, per-feature classification (`legacy`, `normalized`, `mixed_invalid`). This eliminates runtime ambiguity by blocking dispatch for invalid mixed features.
3.  **Ambiguous `primary`/`finding` execution semantics:** Defines `active_issue_id` and the serialized execution policy. The `execution_policy` field is `serialized`.
4.  **Aggregate completion derivation:** Provides a deterministic, compiler-enforced algorithm for deriving aggregate status, making it enforceable and removing subjectivity.

It also establishes a clean **runtime authority model** by separating authoring truth (Markdown) from runtime truth (lock file), preventing parser behavior mismatches at the compiler/runtime boundary.

## Remaining blockers
1.  **Lock File State Transition Semantics:** The specification for how the lock file is updated relative to live Beads state transitions (e.g., a finding is resolved, changing the `active_issue_id`) is not defined. This creates a critical gap in the failure recovery model. The dispatcher's "post-claim revalidation" step could fail due to a state change that hasn't been compiled into the lock file, leading to undefined behavior.
2.  **Migration Edge Case: `mixed_invalid` Rollback Path:** The process for reverting a `mixed_invalid` feature to `legacy` is not specified. Does this require manual editing of all workstreams to remove `ws_kind`? Is there a tool-assisted downgrade? This is a migration blocker.
3.  **Failure Recovery for Compiler/Runtime Desync:** The procedure for when the lock file is missing or stale is defined (block dispatch), but the recovery path to *regenerate* it is not. Is it a manual `sdp compile`? Does the dispatcher trigger a rebuild? This is a runtime blocker.
4.  **Lock Semantics for Concurrent Compilation:** The spec does not address how concurrent `sdp compile` runs (e.g., from CI and a local user) should interact to avoid corrupting the lock file. This is a concurrency/TOCTOU issue at the source.

## Required changes before adoption
1.  **Define lock file update triggers and atomicity.** Add a section specifying the exact events (e.g., `git commit`, manual compile, CI run) that trigger lock file regeneration, and the lock (e.g., file system) used to prevent concurrent compilation.
2.  **Specify the rollback procedure for `mixed_invalid`.** Add a clear, tool-assisted downgrade path from `mixed_invalid` to `legacy` to unblock migration.
3.  **Clarify the recovery flow for a stale lock file.** Detail the steps (e.g., dispatcher fails, logs error, triggers a rebuild, retries) to transition from a "blocked" to a "dispatchable" state.
4.  **Add a formal state machine diagram or table** for the `active_issue_id` transitions based on Beads events (finding opened, primary closed, etc.), ensuring the runtime and compiler views can be reconciled.

## Bottom line
Revision 3 is a substantial and correct step forward. It resolves the core runtime and migration contradictions from Revision 2 by introducing a clear, generated lock file and a strict per-feature classification. The remaining issues are not flaws in the proposed model but are critical operational gaps in the specification of state transitions, concurrent updates, and recovery procedures. These must be addressed in a final revision before the changes can be safely deployed. The core design is sound and should be accepted once the operational semantics are fully defined.

---

## Philosopher - `anthropic/claude-opus-4.6`

## Verdict

Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What Rev 3 Fixes

Rev 3 resolves the deepest conceptual problems that plagued Rev 2:

1. **Single runtime oracle.** The lock file eliminates dual-authority ambiguity. Authoring truth (Markdown) and runtime truth (lock file) are cleanly separated with a one-way derivation arrow. This is ontologically sound: the lock file is a *projection*, not a competing source of meaning. The memo is explicit that the projection is generated, never hand-edited, and keyed to a specific git commit. That's a coherent contract.

2. **Feature-level normalization classes.** The tripartite classification (`legacy` / `normalized` / `mixed_invalid`) eliminates the category that Rev 2 tried to hide: the partially-migrated feature that is neither fish nor fowl. Crucially, `mixed_invalid` is treated as an *error*, not a transitional state to be tolerated at runtime. This is the right philosophical move — it refuses to let an incoherent entity participate in a coherent system.

3. **Serialized execution slot.** The `active_issue_id` derivation with explicit priority ordering and deterministic tie-breaking finally gives `primary` vs `finding` a concrete operational semantics rather than a vague role distinction. One leaf, one slot, no concurrent lanes. The ontological claim ("a leaf is an atomic unit of work") now matches the runtime contract.

4. **Reshape as two-phase cutover.** The freeze-then-replace protocol acknowledges that topology mutation under live claims is a category error, not merely a race condition. This is correct: you cannot coherently redefine the boundaries of an entity while that entity is being acted upon.

5. **Migration phases no longer contradict each other.** The Phase 2/Phase 4 collision from Rev 2 is gone because enforcement scope is explicitly tied to normalization class, not to calendar phase.

## Remaining blockers

Three issues that are not fatal but represent genuine residual ambiguity:

**1. Aggregate status derivation rule 5 is a grab-bag.**

> `open` if any child is `open|blocked|done` but the aggregate is not yet done

This fires whenever the aggregate isn't captured by rules 1–4 and at least one child is non-terminal. But a state where *all* non-terminal children are `blocked` is already caught by rule 4, and "any child is done" is trivially true in most mid-flight aggregates. Rule 5 is effectively "everything else that isn't backlog," which means rule 6 (`backlog otherwise`) can only fire when the aggregate has zero non-terminal children that are open or blocked — i.e., all children are `backlog`. This is probably intended, but the rule as written obscures that logic. The derivation should be stated as an explicit decision table or the rules should be tightened so a reader can verify exhaustiveness and mutual exclusivity without mental simulation.

**2. `declared_status` must match `derived_status` — but what enforces convergence?**

The memo says the validator checks this, but it doesn't say what happens when they diverge. Is the lock file compilation blocked? Is the feature downgraded to `mixed_invalid`? Is it a CI-blocking error? The contract needs to specify the *consequence*, not just the check. Without this, the rule is an aspiration, not a guard rail.

**3. Post-claim revalidation is necessary but its failure semantics are underspecified.**

The memo says "abort execution and release the claim." But: what is the retry policy? Is the abort silent or does it surface to the user? Does a failed revalidation count as a dispatch attempt for rate-limiting or alerting purposes? The TOCTOU fix is structurally correct, but the failure path needs the same rigor as the happy path, or implementers will fill the gap with ad-hoc choices that diverge.

## Required changes before adoption

1. **Rewrite aggregate derivation rules 4–6 as an explicit, exhaustive decision table** with a proof sketch (or at minimum a worked example) showing mutual exclusivity and completeness. The current ordered-rule format invites subtle misreads.

2. **Specify the consequence of `declared_status ≠ derived_status`**: does it block lock file generation, classify the feature as `mixed_invalid`, or produce a CI error? Pick one and state it.

3. **Define failure semantics for post-claim revalidation**: at minimum, specify whether the failure is retriable, whether it is surfaced to the operator, and whether it blocks further dispatch attempts on the same leaf until the lock file advances.

These are all completions of contracts the memo already intends to make. None require reopening the ontology.

## Bottom line

Rev 3 is the first revision where the claimed ontology (features contain aggregate workstreams contain leaf workstreams; leaves are atomic execution units) is actually *enforced* by the runtime contract rather than merely *described* by it. The lock file as sole runtime oracle, the single-slot execution model, and the two-phase reshape protocol are all conceptually sound and mutually reinforcing. The three gaps above are specification incompleteness, not structural

---

## Pragmatist - `minimax/minimax-m2.7`

## Verdict

**Verdict: CONDITIONAL_ACCEPT**

**Formal veto: NO**

---

## What Rev 3 Fixes

**1. Reshape safety under concurrent claims** — The two-step freeze-then-replace protocol with the hard blocking rule ("if any affected issue remains actively claimed during Step 1, reshape is blocked") closes the race condition in Revision 2. This is operationally costly but definitively removes the unsafe path.

**2. Mixed format contradiction** — Per-feature classification into exactly three buckets (`legacy`, `normalized`, `mixed_invalid`) removes the ambiguity. `mixed_invalid` as a validator error (not warning) forces complete decisions per feature, which is cleaner than the previous fuzzy warn-mode behavior.

**3. Primary versus finding execution ambiguity** — The "one execution slot" rule with an explicit priority algorithm (blocking finding → primary → non-blocking finding → null) plus tie-breakers (older created_at, then lower lexical Beads id) is deterministic and enforceable. This directly resolves Round 2's ambiguity complaint.

**4. Aggregate completion derivation** — The six-step derivation hierarchy is explicit, not hand-waved. The validator rule that `declared_status` must match `derived_status` enforces correctness without removing human readability from frontmatter.

**Runtime authority separation** — The lock file as the single runtime input addresses the TOCTOU objection by making the dispatcher read from one generated artifact rather than scanning raw Markdown live. This also solves the performance concern.

---

## Remaining Blockers

**1. Post-claim revalidation window** — The dispatch contract performs pre-claim validation, claims, then re-reads lock file and Beads state. If the re-read reveals a mismatch, the dispatcher must abort and release. The window between claim and re-read is not protected. If Beads or git state mutates between these steps, work that was validly claimed could be aborted after the fact. This is acknowledged ("does not pretend to be globally atomic"), but the abort-after-claim scenario may create operational churn. The re-read must be fast and deterministic, or the revalidation becomes a bottleneck under load.

**2. Lock file generation mechanism unspecified** — The spec defines the lock file contract but does not specify the trigger for generation. Is it a pre-commit hook, a CI step, a manual command? If it is a CI step, there is a window where the lock file lags behind `HEAD`. If it is a pre-commit hook, developer friction increases. This must be specified before Phase 3 enforcement.

**3. The two-commit reshape protocol creates operation friction** — For fast-moving teams, requiring a separate freeze commit before replacement landing is a significant workflow change. The hard safety rule is correct, but the protocol adds a minimum two-PR gate for what was previously a single atomic change. This pays for itself if the alternative is unsafe concurrent reshape, but it needs tooling support (automated freeze state detection, enforcement gate in CI) to avoid becoming a manual process that developers bypass.

**4. Legacy path elimination is a Phase 5 gate with no defined exit criteria** — Phase 5 ("only after the active backlog no longer contains legacy features") is operationally unclear. What happens if one feature resists normalization indefinitely? Does the legacy dispatch path remain in the codebase forever? The spec should define a maximum migration window or a deprecation trigger rather than an open-ended condition.

**5. No rollback trigger for benign failures** — The rollback protocol defines the mechanism but not the trigger conditions. If a normalized feature produces a derived status mismatch, does the team self-assess and decide to rollback, or is there an automated threshold? Without a defined trigger, rollback becomes a judgment call that may be delayed.

---

## Required Changes Before Adoption

1. **Specify lock file generation trigger** — The lock file must be generated predictably with a defined freshness guarantee. Recommended: CI step on every PR merge to main that updates the lock file, with the lock file committed as an artifact. Pre-claim validation must fail if the lock file `git_commit` does not match `HEAD`.

2. **Define post-claim revalidation failure mode** — When re-read after claim reveals a mismatch, the spec should clarify what happens to any partially executed work. Does the dispatched work have a chance to complete safely, or must it abort mid-stream? "Abort execution" is stated but the operational definition of "execution" (start of a hook, after initial API call, etc.) matters for teams with long-running operations.

3. **Add lock file schema version migration plan** — Schema version is `1` with no upgrade path defined. Before this becomes production data, there must be a backward-compatibility guarantee or a schema migration step in the compiler.

4. **Define Phase 5 legacy elimination policy** — Either set a maximum timeline for legacy features to migrate (e.g., "legacy path removed 90 days after Phase 4 activation") or define an escalation process for features that cannot normalize. The current language leaves the legacy path as a permanent codebase fixture.

5. **Operationalize the freeze-then-replace protocol** — The two-commit requirement needs tooling support to be enforceable. A CI check that verifies the lock file reflects a frozen topology before allowing the replacement PR to merge would prevent the protocol from becoming an honor-system.

---

## Bottom Line

Revision 3 resolves the four runtime and migration blockers from Round 2. The strictness is justified by the problems it solves: reshape safety under live claims requires a hard gate, and mixed-format ambiguity requires a hard classification error rather than a soft warning. The one-execution-slot rule and aggregate derivation algorithm are deterministic and enforceable, which is a significant improvement over the prior ambiguity.

The conditional acceptance is not a veto. It reflects that three operational details (lock file generation, post-claim failure modes, Phase 5 exit criteria) are underspecified in the current text and would create confusion in implementation. These are implementation details, not design revisions. If the owner clarifies the generation trigger and failure modes, the spec is ready to adopt.

The pragmatic test is: does this spec tell a team what to build in a way that produces consistent behavior across different implementers? For the runtime contract, yes. For the migration lifecycle and rollback triggers, partially. The gaps are fillable without reopening the design.

Adopt conditionally. Specify the three operational details before Phase 3 enforcement goes live.

---

## Engineer - `xiaomi/mimo-v2-pro`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What Rev 3 Fixes
Revision 3 successfully addresses the four concrete failures from Round 2 with clear, machine-enforceable contracts:

1.  **Reshape Safety:** The two-step "freeze-before-replace" protocol with a hard safety rule (no active claims during freeze) directly resolves the unsafe concurrent reshape issue.
2.  **Parser Contradiction:** The introduction of the `mixed_invalid` feature class and the rule that only `normalized` features enter the lock file eliminates ambiguous mixed-mode runtime behavior.
3.  **Execution Ambiguity:** The serialized execution contract, defining a single `active_issue_id` per leaf with a deterministic priority/tie-breaker rule, makes `primary`/`finding` semantics unambiguous.
4.  **Aggregate Status:** The explicit, ordered derivation rules for aggregate workstream status and the validator rule that `declared_status` must match `derived_status` make completion fully enforceable.

The core architectural fix—making `workgraph.lock.json` the sole runtime input—resolves the TOCTOU and dual-authority concerns from Round 2 by design.

## Remaining blockers
1.  **Lock File Generation & Validation Tooling:** The specification assumes a robust compiler exists to generate the lock file and validate feature classification. Implementation risk is high if this tooling is not built first and is not itself rigorously tested.
2.  **Dispatcher Revalidation Barrier Complexity:** The post-claim revalidation step (re-reading `HEAD`, lock file, and Beads state) introduces a critical but complex synchronization point. Failure modes (e.g., partial state reads, network issues with Beads) need explicit error-handling rules.
3.  **Migration Phase Coordination:** The phased migration is logical but requires strict orchestration. A feature incorrectly classified as `normalized` before all its workstreams are valid could cause runtime failures. The audit phase (Phase 1) must be infallible.
4.  **Lock File Schema Completeness:** The provided schema examples are minimal. The full contract needs explicit definitions for all fields (e.g., `execution_policy` enum values, `historical_issue_ids` purpose) and validation rules (e.g., format of `ws_id`, referential integrity of `parent_ws_id`/`children`).

## Required changes before adoption
1.  **Publish the Full Lock File JSON Schema:** Provide a complete, machine-readable JSON Schema (or equivalent) for `workgraph.lock.json` that defines all allowed values, required fields, and referential constraints.
2.  **Define the Validator's Error Catalog:** Specify the exact error codes and messages for all validator checks (e.g., `MIXED_INVALID_FEATURE`, `AGGATE_STATUS_MISMATCH`, `MULTIPLE_ACTIVE_ISSUES`). This is critical for the dry-run and migration phases.
3.  **Specify Dispatcher State Machine:** Document the exact states and transitions for the dispatcher's claim process, including the revalidation barrier and all rollback/abort procedures, to mitigate implementation risk.
4.  **Clarify `historical_issue_ids` Role:** Define the runtime significance of this field. Is it purely informational, or does it affect dispatch or status derivation? If it's only for audit, state that explicitly.

## Bottom line
Revision 3 is a substantial and necessary improvement. It replaces ambiguous prose with a concrete, machine-readable runtime contract centered on the lock file, directly attacking the prior blockers. The architecture is sound.

Acceptance is conditional on the **Required changes** being incorporated into the adoption plan. The primary risk has shifted from *conceptual ambiguity* to *implementation fidelity*. The council must ensure the tooling (compiler, validator, dispatcher) is built to the strict letter of this clarified contract, particularly around the lock file schema and revalidation barrier. With those details solidified, this revision provides a viable path to safe, normalized work atomicity.

---
