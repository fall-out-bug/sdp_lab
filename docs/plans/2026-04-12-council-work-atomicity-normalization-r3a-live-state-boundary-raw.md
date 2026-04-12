# LLM Council Raw Log: SDP Work Atomicity Normalization - Revision 3a
**Date:** 2026-04-12
**Subject:** [2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md](2026-04-12-work-atomicity-normalization-r3a-live-state-boundary.md)
**Validity note:** This raw log contains only the final valid roster. Discarded attempts are excluded: the first sequential run stalled on late roles, and an earlier critic attempt on `google/gemini-3.1-pro-preview` returned truncated/incomplete output. Per user direction, `openai/gpt-5.4` was not used in this round; the architect role ran on `anthropic/claude-sonnet-4.6`.
---

## Architect - `anthropic/claude-sonnet-4.6`

## Verdict

Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

---

## What Rev 3a Fixes

**The core split-brain objection is resolved.**

The prior council's blocker was precise: compiling `active_issue_id` into a git-managed artifact created two authoritative sources for the same fact—one static and stale-able, one live and correct. Rev 3a eliminates that by drawing a hard, named boundary between topology-and-bindings (git owns) and execution selection (Beads owns at dispatch time). The fix is structurally sound, not cosmetic.

Specific improvements:

1. **`active_issue_id` removed from committed lock.** The field that caused the split-brain is gone. Issue selection is now a pure runtime derivation over the bound issue set, using the same priority algorithm as Rev 3 but executed at dispatch time rather than compile time.

2. **`source_inputs_hash` replaces `git_commit`.** The prior self-referential hash paradox (a file cannot hash itself into itself) is resolved by hashing the *inputs* to the compiler rather than the commit containing the output. This is the correct direction. The hash now describes a reproducible compiler invocation, not a circular identity claim.

3. **Post-claim revalidation barrier is explicit.** Rev 3a names the race window (between selection and claim) and closes it with a mandatory re-read-and-abort sequence. This does not achieve global atomicity—the memo correctly does not overclaim—but it creates a defined safety barrier that is implementable with today's Beads primitives.

4. **Aggregate declared/derived mismatch promoted to compile error.** This closes a nearby static/live ambiguity that Rev 3 left open. A half-valid lock file is worse than no lock file; treating the mismatch as a hard compile failure is the right call.

5. **Authority split is now legible.** The two enumerated lists (what the lock *may* contain vs. *must not* contain) give implementers and future reviewers a checkable contract rather than a prose intention. This is a meaningful governance improvement over Rev 3.

---

## Remaining Blockers

Three issues survive and require resolution before adoption. None are fatal to the architecture, but two are sharp enough to cause implementation divergence if left open.

**Blocker 1 — Leaf-wide exclusivity is underspecified (sharp).**

The claim contract states: "if Beads cannot enforce leaf-wide exclusivity directly, the dispatcher must treat any existing claim on another bound issue of the same leaf as a hard dispatch failure after live re-read." This is correct in intent but defers a load-bearing question. The dispatcher's ability to detect a cross-issue claim on the same leaf depends on whether Beads exposes a queryable claim index scoped to a bound issue set. If it does not, the dispatcher cannot reliably detect the conflict without a full scan. Rev 3a does not specify whether that query capability exists, must be added, or is approximated. An implementation team reading this memo today cannot determine which path to take. This needs a concrete answer: either confirm the Beads query surface, specify the required addition, or define the fallback scan boundary.

**Blocker 2 — `source_inputs_hash` stability contract is underspecified (sharp).**

The hash is defined as covering "workstream frontmatter, `## Beads` role blocks, any compiler config that affects graph or policy interpretation." The third term is unbounded. Compiler config can include environment variables, tool versions, flag sets, and external schema references. Without a normalization rule for what counts as "compiler config that affects interpretation," two runs of the same compiler on the same source can produce different hashes, making the freshness check non-deterministic. The memo needs to either enumerate the config surface that is included or specify a canonical serialization rule.

**Blocker 3 — Frozen/archived lifecycle interaction with runtime resolution (moderate).**

The pre-claim check confirms the leaf is `active`, not `frozen|archived`. But the runtime resolution algorithm (priority ordering over open issues) does not address what happens when a leaf transitions to `frozen` *after* the lock is loaded but *before* the post-claim revalidation completes. The post-claim re-read checks whether the chosen issue is still active; it does not check whether the leaf's lifecycle state has changed. This is a narrow but real window. The revalidation contract should include a leaf lifecycle re-check, or the memo should explicitly declare this out of scope with a stated rationale.

---

## Required Changes Before Adoption

The following changes are required. They are scoped to close the blockers above without reopening hierarchy, migration, or terminology.

**R1 — Specify the Beads claim query surface for leaf-wide exclusivity.**
Add a section or note that answers: does the current Beads API support a query of the form "return all active claims on issues in this set"? If yes, cite it. If no, specify the minimal addition required. If the answer is "dispatcher does a sequential scan with a defined bound," state that bound explicitly. This must be answerable before implementation begins.

**R2 — Enumerate or bound the `source_inputs_hash` input set.**
Replace "any compiler config that affects graph or policy interpretation" with either (a) an explicit enumerated list of included config fields, or (b) a canonical config snapshot format with a defined serialization. The goal is that two independent compiler invocations on identical source and config produce identical hashes.

**R3 — Add leaf lifecycle re-check to post-claim revalidation.**
Extend the post-claim revalidation step to include: "confirm leaf lifecycle state is still `active`." If the leaf has transitioned to `frozen` or `archived` during the claim window, treat this as a hard abort and release the claim. This is a one-line addition to the revalidation contract.

**R4 — Add an optional observability note on the ephemeral runtime view.**
The memo mentions an "ephemeral runtime view" for current active issue as an alternative to committed lock state. This is the right design. Add a brief note on what that artifact's lifecycle is (e.g., not committed, regenerated on each dispatch invocation, not used as input to any subsequent compiler run). This prevents a future implementer from accidentally promoting the ephemeral view back into git state, which would recreate the original split-brain at one remove.

---

## Bottom line

Rev 3a correctly identifies and resolves the architectural root cause of the split-brain objection. The authority boundary it draws—topology and bindings in git, execution selection derived live from Beads—is the right boundary, and the memo defends it clearly against the two obvious alternatives. The `source_inputs_hash` substitution for `git_commit` is a genuine improvement. The post-claim revalidation barrier is honest about what it does and does not guarantee.

What remains is not architectural disagreement but specification gaps that will cause implementation divergence if left open. The leaf-wide exclusivity query surface and the hash input normalization are the two that matter most; both are resolvable without reopening the core design. Adopt Rev 3a with R1 through R4 applied, and this patch is ready for implementation.

---

## Critic - `mistralai/mistral-large-2512`

## Verdict
Verdict: **ACCEPT**
Formal veto: **NO**

## What Rev 3a Fixes
1. **Split-brain state**: Eliminates the structural mismatch where mutable Beads state (`active_issue_id`) was baked into the immutable `workgraph.lock.json`. This resolves the critic’s strongest objection from R3.
2. **Governance theater**: Clarifies authority boundaries—Git owns topology/bindings, Beads owns live execution state—removing the illusion of atomicity where none existed.
3. **Stale lock risk**: Prevents scenarios where a lock file’s `active_issue_id` becomes invalid due to post-commit Beads changes (e.g., new blocking findings).
4. **Git-hash paradox**: Replaces the ambiguous `git_commit` field with `source_inputs_hash`, tying the lock file to its actual source inputs rather than its container.

## Remaining blockers
1. **Leaf-wide exclusivity**: The dispatcher’s post-claim revalidation is a best-effort safety net, not a true atomic primitive. If Beads cannot enforce leaf-wide exclusivity, race conditions may still occur between:
   - Dispatcher’s pre-claim check (e.g., "no other bound issue is claimed").
   - Another dispatcher claiming a different issue on the same leaf.
   *Mitigation*: Explicitly document this as a known limitation and require dispatchers to treat such races as hard failures (release claim and retry).

2. **Aggregate status ambiguity**: While R3a treats `declared_status != derived_status` as a compile error, the patch does not address how this interacts with:
   - Partial normalization (e.g., some workstreams normalized, others not).
   - Migration phases where aggregate status might temporarily diverge.
   *Risk*: Could create governance theater if teams treat the error as a "blocker" without understanding its transient nature.

3. **Runtime resolution overhead**: Deriving `active_issue_id` at dispatch time requires live Beads queries, which may introduce latency or availability dependencies. The patch assumes this is acceptable, but:
   - No performance budget is defined for the resolution algorithm.
   - No fallback behavior is specified if Beads is unavailable.

## Required changes before adoption
1. **Explicit race condition documentation**:
   - Add a "Known Limitations" section to the spec, detailing the leaf-wide exclusivity race and its mitigation (dispatcher retry).
   - Require dispatchers to log such races as `WARN`-level events for observability.

2. **Beads query contract**:
   - Define a minimal SLA for Beads’ response time during issue resolution (e.g., 99th percentile < 100ms).
   - Specify fallback behavior if Beads is unavailable (e.g., fail open with a `DEGRADED` status, or retry with exponential backoff).

3. **Aggregate status migration path**:
   - Clarify that `declared_status != derived_status` is a compile error *only for fully normalized features*.
   - For partially normalized features, allow a `WARN`-level log and proceed with `derived_status` as the source of truth.

4. **Lock file schema validation**:
   - Add a JSON Schema or equivalent to enforce that no mutable Beads-derived fields (e.g., `active_issue_id`) are present in the lock file.
   - Include the `source_inputs_hash` field as `required`.

5. **Dispatcher revalidation timeout**:
   - Add a configurable timeout for post-claim revalidation (e.g., 5 seconds) to prevent indefinite hangs if Beads is slow.

## Bottom line
Revision 3a resolves the core blocker from R3 by cleanly separating static topology (Git) from live execution state (Beads). The remaining issues are implementation details, not architectural flaws, and can be addressed with targeted changes. The patch’s clarity on authority boundaries and its rejection of governance theater make it a net improvement over R3.

**Adopt with the required changes above.** The leaf-wide exclusivity race is a known tradeoff, not a veto-worthy flaw, given the complexity of true atomicity in distributed systems.

---

## Technician - `deepseek/deepseek-v3.2`

## Verdict
Verdict: ACCEPT
Formal veto: NO

## What Rev 3a Fixes
Revision 3a resolves the core blocker from the prior council by establishing a clean and correct boundary between static, source-controlled state and live, runtime state. Specifically, it fixes:
1. **The live-state snapshotting problem:** It removes `active_issue_id` and any other fields derived from mutable Beads state (openness, blocking status, priority) from the committed `workgraph.lock.json`. This eliminates the structural mismatch where a static git artifact could become stale due to routine changes in the live issue queue.
2. **The lock identity paradox:** It replaces the ambiguous `git_commit` field with a `source_inputs_hash` computed from the authoritative source inputs (frontmatter, Beads blocks, config). This correctly ties the lock file's validity to the source state that produced it, not to the commit containing the lock file itself.
3. **The authority split:** It clearly defines that the compiler owns topology and bindings (structure), while the runtime (Beads/dispatcher) owns live execution selection. This resolves the split-brain concern.

## Remaining blockers
None. The architectural dispute over the static/live boundary is resolved. The remaining items are implementation details, not blockers.

## Required changes before adoption
The following concrete changes must be implemented as specified in the memo:
1. **Schema Update:** Remove `active_issue_id` from the `workgraph.lock.json` schema.
2. **Runtime Resolution Logic:** Implement the dispatcher's logic to derive the active issue at dispatch time using the defined priority order (blocking findings > primary > non-blocking findings) and tie-breakers, but only considering issue IDs bound in the lock file.
3. **Freshness Check:** Implement the `source_inputs_hash` generation and validation, replacing any `git_commit`-based check.
4. **Compilation Guard:** Implement the compile-time failure for normalized features where `declared_status != derived_status`.
5. **Claim/Revalidation:** Implement the dispatcher's pre-claim checks and post-claim revalidation as described, ensuring the chosen issue is still active per live Beads state before work begins.

## Bottom line
Revision 3a correctly draws the line. The lock file is for stable topology and bindings; live issue selection belongs to the runtime. This preserves git's role for structure, Beads' role for execution state, and provides a clear, maintainable contract for dispatch atomicity. The council's prior blocker is addressed. Implementation can proceed.

---

## Philosopher - `anthropic/claude-opus-4.6`

## Verdict

Verdict: ACCEPT
Formal veto: NO

## What Rev 3a Fixes

The prior council's blocker was precise: Revision 3 compiled a volatile, runtime-derived value (`active_issue_id`) into a git-managed artifact, creating a category error where a static document claimed authority over live state. This is not merely an engineering inconvenience; it is a conceptual incoherence. A lock file that must be regenerated whenever external mutable state changes is not functioning as a lock — it is functioning as a cache with no invalidation contract, masquerading as a source of truth.

Revision 3a resolves this cleanly by drawing a principled boundary:

1. **The ontological cut is correct.** The lock file now owns *structure* (topology, bindings, policy versions) — things that change only when the author changes them. Runtime owns *selection* (which issue is active) — things that change when the world changes. This is not an arbitrary engineering split; it reflects a genuine difference in the *causal origin* of the data. Structure is author-caused. Selection is world-caused. Mixing them in one artifact was the category error, and it is now eliminated.

2. **The `source_inputs_hash` resolves the git-hash paradox.** Revision 3's `git_commit` field tried to make a file refer to the commit containing itself — a self-referential impossibility. The replacement identifies *inputs to the compilation*, which is both computable and semantically honest. The lock file now says "I was derived from these sources" rather than "I exist in this commit," which is the correct epistemic claim.

3. **The binding-scoped runtime query preserves the lock's authority without overreach.** The dispatcher queries Beads, but *only for issue IDs already bound by the lock file*. This means the lock still constrains the space of possible actions — it defines the menu — while runtime selects from that menu. This is a genuine division of labor rather than a collapse of one authority into the other.

4. **The declared/derived mismatch rule closes a real ambiguity.** A feature where `declared_status != derived_status` is now a compile error, not a runtime surprise. This prevents a half-valid lock artifact from existing, which is the right call: the lock should be total or absent, never partial.

## Remaining blockers

None that rise to the level of architectural blocker. What remains is implementation detail within a now-coherent conceptual frame. I note two areas that deserve attention but do not constitute veto-worthy ambiguity:

**1. The "bound issue set" is doing quiet load-bearing work.** The lock file binds `primary_issue_id`, `finding_issue_ids`, `historical_issue_ids`, and `aggregate_finding_issue_ids`. The runtime resolution algorithm operates over these. But the memo does not fully specify what happens when Beads contains an issue that *should* be bound (e.g., a new finding filed against the workstream) but is not yet reflected in the lock's binding list because the lock hasn't been recompiled. This is not a category error — the lock correctly owns bindings, and a new finding requires recompilation — but it means there is a latency window where a real blocking finding is invisible to dispatch. The memo implicitly accepts this by design (the lock must be recompiled), but it should be made explicit so that operators understand the failure mode.

**2. The claim/revalidation sequence is not fully atomic, and the memo knows it.** The post-claim revalidation is described as "an explicit safety barrier" rather than true atomicity. This is honest and appropriate. The remaining race window (between claim and re-read) is narrow and bounded. I flag it only because the memo's own language ("This does not create global atomicity") deserves to be preserved in the final specification rather than softened during implementation.

## Required changes before adoption

These are minor specification clarifications, not architectural changes:

1. **Make the new-binding latency window explicit.** Add a short paragraph acknowledging that findings filed in Beads but not yet compiled into the lock's `finding_issue_ids` are invisible to dispatch until recompilation. State whether this is acceptable by design or whether a runtime warning should be emitted when Beads contains unbound issues referencing the workstream.

2. **Specify the `source_inputs_hash` computation deterministically.** The memo says it covers "workstream frontmatter, `## Beads` role blocks, any compiler config." The word "any" hides ambiguity. Enumerate the exact inputs or define a canonical ordering and serialization so that two independent implementations produce the same hash.

3. **Preserve the non-atomicity acknowledgment in the normative spec.** The sentence "This does not create global atomicity" is the most important sentence in the claim/revalidation section. It should survive into whatever becomes the binding specification, not just this memo.

## Bottom line

Revision 3a resolves the prior council's blocker by making a principled ontological cut: structure is compiled, selection is derived. The boundary is not arbitrary — it tracks the genuine causal difference between author-determined topology and world-determined issue state. The `source_inputs_hash` eliminates the self-reference problem. The binding-scoped runtime query preserves lock authority without overreach. The remaining work is specification hygiene, not architectural dispute. This is ready for adoption with the minor clarifications noted above.

---

## Pragmatist - `minimax/minimax-m2.7`

## Verdict
Verdict: CONDITIONAL_ACCEPT
Formal veto: NO

## What Rev 3a Fixes

**Primary blocker resolved:** The structural mismatch between the committed lock file and mutable Beads state. By removing `active_issue_id` from the lock schema and deferring issue selection to dispatch-time runtime resolution, Revision 3a restores the correct authority boundary—git owns topology and bindings, Beads owns live execution state.

**Secondary fixes:**
1. **Lock identity paradox resolved:** Replacing the loose `git_commit` field with `source_inputs_hash` removes the circular reference and ties runtime to actual source state rather than commit metadata.
2. **Aggregate half-validity closed:** Treating `declared_status != derived_status` as a compile error prevents a feature from emitting a partially-invalid static artifact into the lock file.
3. **Claim contract clarified:** Post-claim revalidation against both lock identity and live Beads state provides an explicit safety barrier without baking runtime state into git.

**Minimality confirmation:** Revision 3a performs exactly one focused change—it answers the hard question about what belongs in the lock file without reopening hierarchy, migration phases, or terminology. This is the correct scope for a patch.

## Remaining blockers

1. **`source_inputs_hash` scope not formally specified:** The memo states the hash is computed over "normalized workstream sources," but does not enumerate the exact inputs. Different implementations could produce divergent hashes, breaking cross-check guarantees. Without a normative list of included/excluded inputs, this field risks becoming another source of drift.

2. **Post-claim revalidation edge cases not covered:** The revalidation contract mentions "if the chosen issue is no longer active, execution aborts," but does not specify behavior when: (a) the lock file itself has changed between dispatch and claim, or (b) the Beads state query returns an error rather than a definitive answer. The safety barrier assumes happy-path availability.

3. **Aggregate declared/derived mismatch rule lacks migration path:** For existing normalized features, this rule would cause compile failures. The memo does not specify whether existing violations should block the patch itself or be treated as pre-existing technical debt with a grace period.

## Required changes before adoption

1. **Specify `source_inputs_hash` contract precisely:** Add an explicit enumeration of included inputs (workstream frontmatter, `## Beads` blocks, compiler config, policy versions) and explicitly excluded inputs (derived fields, runtime state, ephemeral metadata). This must be implementable by any compliant compiler without ambiguity.

2. **Document revalidation failure modes:** Explicitly define dispatcher behavior when revalidation fails or returns indeterminate state. Recommend fail-closed (abort and release claim) as the default, with explicit acknowledgment that this is the intended behavior.

3. **Add migration guidance for aggregate violations:** Either (a) allow existing mismatches to compile with a warning until corrected, or (b) define a cutoff date after which the compile error applies. Shipping this rule without a migration path creates an immediate adoption blocker for any team with pre-existing violations.

4. **Update test matrix for runtime derivation paths:** Ensure the test suite covers the priority/tie-breaker logic at dispatch time, including race conditions where Beads state changes between the dispatcher's initial selection and the post-claim revalidation read.

## Bottom line

Revision 3a correctly resolves the core architectural dispute: the lock file should not snapshot mutable runtime state. The separation of topology/bindings (static) from issue selection (runtime) is clean, operationally sound, and minimally invasive. The patch is shippable provided the four required changes above are addressed.

The changes are low-cost (single field removal, contract clarification, no new infrastructure) and high-value (eliminates the split-brain at the lock/Beads boundary). If the council accepts these conditions, the remaining work becomes normal implementation detail rather than architectural dispute—which is exactly what a well-scoped patch should achieve.

---

## Engineer - `xiaomi/mimo-v2-pro`

## Verdict
Verdict: ACCEPT
Formal veto: NO

## What Rev 3a Fixes
Revision 3a directly resolves the primary architectural blocker from the prior council: the **structural mismatch** between a static, git-managed lock file and volatile, runtime-derived issue state. By removing `active_issue_id` and any field derived from mutable Beads state (openness, blocking, priority) from the committed `workgraph.lock.json`, it eliminates the risk of the lock file going stale due to live finding changes without a corresponding topology commit. This cleanly separates the two authorities:
1.  **Git owns structure:** Topology, issue bindings, and policy versions.
2.  **Beads owns live execution state:** The current active issue selection, resolved at dispatch time.

The patch also fixes the ambiguous `git_commit` identity field by replacing it with a deterministic `source_inputs_hash`, which correctly identifies the lock's inputs rather than creating a self-referential paradox.

## Remaining blockers
1.  **Dispatcher Complexity & Race Conditions:** The two-source evaluation (lock file + live Beads query) and the post-claim revalidation step introduce implementation complexity. The dispatcher must now handle potential race conditions where the active issue changes between pre-claim check and post-claim revalidation. The memo defines the abort behavior, but the implementation must ensure this is robust and performant.
2.  **Runtime Resolution Algorithm Specification:** While the algorithm is stated, its exact machine-readable contract (e.g., how "priority" is defined and fetched from Beads, the precise schema of the Beads query) needs to be formally specified for interoperability between the compiler and dispatcher.
3.  **`source_inputs_hash` Computation:** The exact set of "authoritative source inputs" and their canonical normalization process for hashing must be rigorously defined to prevent drift between implementations.

## Required changes before adoption
1.  **Formalize the Beads Query Contract:** The SDP specification must include a machine-readable schema for the live Beads query that the dispatcher executes, including the exact fields used for priority, blocking status, and tie-breaking (`created_at`, issue id).
2.  **Define `source_inputs_hash` Canonicalization:** Publish a strict, versioned algorithm for computing the `source_inputs_hash`, detailing the exact files/frontmatter included, their serialization order, and the hashing method.
3.  **Specify Dispatcher State Machine:** Provide a clear state diagram or contract for the dispatcher's claim sequence, explicitly handling the pre-claim check, claim primitive, and post-claim revalidation, including error states and claim release semantics.

## Bottom line
Revision 3a is a necessary and correct architectural refinement. It resolves the critical split-brain problem by establishing a clear, defensible boundary between static and live state. The remaining blockers are implementation details that can be addressed in the subsequent specification and implementation phases. The core design is sound and ready for adoption.

---
