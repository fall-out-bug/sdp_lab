**Verdict:** Partially aligned. The Agent Skill Operating Rules establishes strong individual-skill discipline, while the Harness Engineering Landscape demands systemic runtime, manifest, and evidence-policy changes that mostly exceed the Rules’ scope. The pair can proceed as companion research drafts, but the synthesis must close manifest-schema, evidence-taxonomy, and tool-risk gaps before operational adoption. No safety-critical contradiction, but the Rules doc under-specifies the runtime implications raised by the Landscape doc.

---

### Blocking Gaps

1. **Manifest schema lacks runtime permission semantics.**  
   Landscape Consequence #1 requires adding runtime permission semantics to the manifest. Rules doc Rule 12 lists runtime controls (sandbox, network allowlists, per-agent tool permissions) but never connects them to manifest structure or harness-specific schemas (e.g., OpenCode `permission` keys, Pi tool scopes, Codex sandbox modes). Without manifest changes, these controls remain prompt-level advice.  
   *[not_assessed: neither document reproduces the current `sdp.manifest.yaml` schema.]*

2. **Degraded evidence taxonomy is incomplete.**  
   Landscape Consequence #8 lists six degraded evidence states (`failed_provider`, `timeout`, `empty_output`, `unavailable_cli`, `unverified_benchmark`, `not_assessed_runtime`). Rules doc Rule 3 recognizes only empty/hung output as `not_assessed`. The gap means a timeout or provider failure could be silently treated as passing evidence, breaking Rule 4 (Doubt-Driven Development) and Rule 9 (Multi-Plane Review).

3. **Tool-risk classification missing from skill authoring.**  
   Landscape Section 4 proposes five tool classes (perception/read, analysis, local writes, external writes, irreversible/identity-mediated). Rules doc Rule 12 mentions “per-agent tool permissions” and “scoped credentials” but provides no taxonomy. Skills cannot declare risk class, so the runtime cannot enforce class-based gates.

---

### Major Gaps

1. **No model-routing-by-role policy.**  
   Landscape Consequence #4 proposes routing by role (scout, planner, implementer, reviewer, security, synthesis, judge) and Consequence #5 requires model diversity in review lanes. Rules doc Rule 10 prohibits recursive persona orchestration but offers no positive routing matrix or vendor-diversity constraint.

2. **No endpoint provenance requirement in evidence artifacts.**  
   Landscape Consequence #6 requires tracking endpoint provenance and model version. Rules doc Rule 5 requires detecting exact version for external APIs but does not mandate provenance metadata on every model-generated artifact, weakening auditability.

3. **Long-horizon flow semantics absent.**  
   Landscape Consequence #3 and Pi `pi-agent-flow` describe budgets, checkpoints, compaction, and review stops. Rules doc Rules 6–7 cover progressive context and small slices but do not define flow-control semantics (token/time budgets, checkpoint intervals) for multi-slice work.  
   *[not_assessed: whether SDP currently supports flow persistence.]*

4. **Adapter parity separated from runtime dispatch evidence only at headline level.**  
   Landscape Consequence #2 demands this separation. Rules doc Rule 5 verifies official docs but does not define what constitutes runtime dispatch evidence (e.g., tool-call logs, permission-denial logs) versus static config parity. Adoption backlog item #2 (“manifest/protocol lint”) is too vague to enforce this.

5. **Vendor benchmark skepticism not encoded in rules.**  
   Landscape Consequence #10 warns against assuming benchmark rankings transfer to SDP workflows. Rules doc Rule 3 anti-rationalization table lacks an entry for “high benchmark score means safe for this role.”

---

### Useful Tensions

1. **Rule 2 (Skills are workflows, not reference docs) vs Landscape’s demand for harness metadata in skills.**  
   Landscape implies skills must carry tool-risk classes, model routing hints, and harness-specific permissions. This risks bloating skills with reference material. The productive tension forces the synthesis to draw a hard boundary between executable workflow steps (skill body) and runtime configuration (manifest), preventing harness metadata from leaking into skill prose.

2. **Rule 7 (Small revertable slices) vs vendor long-horizon autonomy claims.**  
   Landscape reports GLM “8-hour sustained autonomous work” and Kimi “agent swarm” claims, then recommends bounded workflows. Rules doc mandates thin slices. The tension challenges the synthesis to explicitly reject or conditionally cage vendor autonomy claims—e.g., permitting long-horizon flows only with explicit budgets, checkpoints, and review stops.

3. **Rule 9 (Multi-plane review) vs Landscape model-diversity requirement.**  
   Rule 9 defines independence by output plane (code, requirements, evidence, security). Landscape adds vendor-family diversity as a separate axis. A review could be multi-plane yet single-vendor (e.g., all planes use GPT-5.5). The tension forces the synthesis to decide whether plane independence is sufficient or vendor diversity is mandatory for certain planes (e.g., security).

4. **Rule 12 (Runtime beats prompt policy) vs Landscape context-engineering findings.**  
   Landscape cites a study finding instruction adherence drops as generated work accumulates, supporting runtime context management. Rule 12 says runtime enforces, prompts instruct. The tension: if runtime manages context compaction and memory, the skill author’s responsibility for context hygiene becomes unclear. The synthesis must delineate author duties (progressive loading per Rule 6) from harness duties (compaction, event-driven reminders).

---

### Concrete Changes for the Synthesis Doc

1. **Add manifest runtime permissions schema.**  
   Define a `runtime` block in `sdp.manifest.yaml` (or equivalent) with `sandbox_mode`, `network_policy`, `tool_class_allowlist`, `approval_gates`, and `harness_adapter` fields. Map OpenCode `permission` semantics, Codex sandbox modes, and Pi tool scopes to this schema.

2. **Add degraded evidence taxonomy.**  
   Enumerate canonical `degraded` states: `failed_provider`, `timeout`, `empty_output`, `unavailable_cli`, `unverified_benchmark`, `not_assessed_runtime`. Require every review, test, and model output to carry an `evidence_status` field drawn from this taxonomy.

3. **Add tool-risk classification to skill authoring.**  
   Require every skill to declare `tool_risk_classes` used (perception, analysis, local_write, external_write, irreversible). The manifest must gate `external_write` and `irreversible` behind explicit approval; skills lacking this declaration fail manifest lint.

4. **Add model routing matrix.**  
   Define SDP roles (scout, planner, implementer, reviewer, security, synthesis, judge) and require: (a) reviewer and security roles must use model families distinct from the implementer role; (b) every model result in evidence must include `model_id`, `endpoint_url`, and `harness_version`.

5. **Add long-horizon flow policy.**  
   For work exceeding one slice, require a `flow_spec` with `budget` (tokens / time / tool_calls), `checkpoint_interval`, and `review_stop_gates`. Prohibit unbounded autonomous runs; vendor long-horizon claims must be treated as `UNVERIFIED` until reproduced in SDP evals.

6. **Extend Anti-Rationalization table.**  
   Add rows:  
   - “This model scores high on SWE-Bench / benchmark X” → “Benchmarks do not transfer to SDP workflows without local eval evidence.”  
   - “The skill prompt forbids this tool, so the agent cannot use it” → “Prompt-only prohibition is not a security boundary; runtime gates required.”

7. **Add adapter parity verification checklist.**  
   Distinguish static parity (config file exists) from runtime dispatch evidence (logged tool calls, permission denials, model routing). Require runtime evidence before marking a harness adapter as `supported`.

8. **Add harness-specific source-driven requirements.**  
   When asserting behavior for Codex, OpenCode, Pi, or model APIs, cite exact doc version and URL. Mark vendor claims (e.g., GLM 8-hour autonomy, Kimi swarm) as `UNVERIFIED` in SDP planning artifacts.
