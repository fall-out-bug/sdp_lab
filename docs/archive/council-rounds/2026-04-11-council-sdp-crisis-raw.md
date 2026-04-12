# LLM Council Raw Responses — SDP Crisis

## Round 1 — Blind Review

### Critic (google/gemini-3.1-pro-preview)

**Black-Box Governance is a Pipe Dream**
Trying to enforce a strict SDLC by shelling out to opencode and parsing its stdout is fundamentally broken. You cannot inject gates, phase shifts, or circuit breakers into a proprietary runtime loop that you do not control. Your current execution path is just an elaborate wrapper script, and relying on stdout makes evidence accumulation trivially spoofable by the LLM.

**The "Build Our Own Runtime" Fallacy**
Pivoting SDP to become the runtime fixes the black-box problem but introduces a massive scope blowout. Replicating the reliability, context window management, and error-handling robustness of dedicated agents (like Claude Code) is a different product entirely. You are trading a governance problem for a distributed systems and prompt engineering nightmare.

**Delusional 3-4 Day Timeline and Tool Complexity**
Claiming you can implement reliable bash and edit_file tools in "150 lines" over a few days proves a severe lack of operational experience with LLMs. A real bash tool requires stateful session management, PTY handling, and infinite loop protection, while edit_file requires robust diff parsing to handle inevitable LLM syntax hallucinations. Your MVP tools will catastrophically fail on their first real-world mid-sized codebase refactor.

**agentloop is Architecture Astronautics**
Writing 128 tests, a PhaseRouter, and a complex FSM entirely isolated from the execution path is premature engineering. This is a governance-by-proxy problem: you built the prison before the criminal existed.

---

### Technician (deepseek/deepseek-v3.2)

**Governance Over Opaque Subprocess is Fundamentally Broken**
The current design tries to enforce a discipline over a black-box execution layer. This is impossible because SDP cannot observe, interrupt, or direct the agent's internal decision-making and tool use. The diagnosis is correct: SDP must be the runtime, not a wrapper.

**Agentloop Architecture is Sound but Currently a Library Silo**
The agentloop's core design is technically feasible and aligns with the "SDP as runtime" pivot. However, its isolation without integration into the main execution path makes it a dead architectural component. It needs real ModelGateway and tool implementations to be viable.

**Proposed Integration Path is Realistic but Timeline is Optimistic**
Building an OpenRouterGateway (~50 lines) and a basic set of core tools (~150 lines) is feasible. However, integrating these into agentloop, connecting them to a new sdp run command, and ensuring the full phase/gate/evidence flow works will involve more than wiring; it requires testing the full pipeline end-to-end. 3-4 days is aggressive for stable integration.

**Evidence Collection Must Be Based on Tool Calls, Not Self-Report**
The artifact correctly states EvidenceAccumulator only uses tool outputs. This is a critical feasibility standard: evidence must be objective, verifiable actions, not the agent's narrative.

**Phase Enforcement Requires Runtime Control of the Agent's State Transitions**
The PhaseRouter's five phases are only meaningful if the harness's FSM can prevent the agent from proceeding without completing phase-specific gates and evidence.

**First Action: Implement and Integrate a Real ModelGateway**
The immediate next step must be to replace the StubGateway with a real ModelGateway within the agentloop.

**SessionStore Append-Only Design is Correct for Auditability**
The SQLite WAL, append-only turn_records is a feasible and correct choice for an audit trail. This should be maintained.

---

### Philosopher (moonshotai/kimi-k2)

**The crisis is not technical, it is ontological**
You are debating "how to wrap the agent" when the real question is whether SDP should be a wrapper at all. Decide first: is SDP a process manager or a worldview? All else follows from that choice.

**agentloop is a cathedral in a desert: it has no land to stand on**
A state machine that never drives the real LLM is just an elaborate model railway. Either delete it or federate it immediately into the critical path; grey-zone existence burns credibility faster than bugs.

**"Governance over opaque subprocess" is only unworkable if you insist on POSIX exit codes as the boundary**
An opaque subprocess becomes legible if you demand a structured trace (e.g., JSON-RPC over stdout + an ancillary file descriptor for telemetry). If the subprocess refuses, it is not an SDP agent. This is policy, not code.

**Making SDP the runtime is a power grab disguised as architecture**
Owning the runtime gives you enforcement today, but it also makes SDP the bottleneck for every future capability (new model, new modality, new OS). Ask whether you are ready to become the GNU/Linux of agent infrastructure instead of the IETF of agent governance: small, stubborn protocol that outlives implementations. A runtime pivot is not a three-day ticket; it is a new company mission.

**EvidenceAccumulator silently commits the category error of equating output with truth**
It logs tool results but ignores self-report, yet an agent's claim "I ran the tests" is the very thing governance must verify. Evidence without provenance is just folklore.

**The canonical pipeline document is still "proposed" because it is unimplementable fantasy**
Seven sequential phases with human gates cannot survive contact with an LLM that reasons in milliseconds and backtracks in microseconds.

**Burn the 128 passing tests and write one failing test that matters**
A test suite that green-lights an unused library is worse than no tests; it gives false confidence.

---

### Pragmatist (minimax/minimax-m2.7)

**You built a beautiful artifact nobody uses.**
agentloop has 128 tests and passes code review. It is also completely disconnected from the execution path. This is not a code quality problem; it is a scope/architecture problem.

**"Governance over opaque subprocess" is the correct diagnosis. The proposed fix underestimates the gap.**
"3-4 days" is an estimate for a prototype, not a shippable tool. A bash tool that an LLM can call freely requires input sanitization, timeout enforcement, working directory constraints, and error handling — that alone is a week of work.

**The plan omits the hard part: defining what "SDP as runtime" means concretely.**
Before writing a line of OpenRouterGateway: does SDP intercept every LLM call and inject phase context? Does it own the tool definitions? Does it control model selection mid-session? These are three different architectures with different complexity profiles.

**EvidenceAccumulator captures tool outputs, not reasoning traces. This is insufficient for governance.**
A governance layer needs to know why the agent chose to call a tool, not just what it returned.

**The pipeline document (CANONICAL_SDP_PIPELINE.md) is "Proposed" and that is the real bottleneck.**
Until the pipeline is designed and owned by the SDP team, adding a runtime layer will create more orchestration surface, not less.

**The standalone CLI (cmd/sdp-harness) uses StubGateway — this is a red flag for integration risk.**
The harness cannot be tested against a real LLM because it uses a stub. A single integration test against a real endpoint would be worth more than the entire unit test suite.

---

### Engineer (xiaomi/mimo-v2-pro)

**"Governance over opaque subprocess" is not fixable — it must be abandoned.**
An opaque process cannot expose its internal state, tool calls, or phase adherence. Real-time gate enforcement, evidence accumulation, and phase transition control are impossible.

**The agentloop library is architecturally sound but operationally incomplete.**
Its stateless loop with a stateful harness FSM and explicit phase routing is the correct abstraction. The immediate task isn't to redesign agentloop but to flesh it out with concrete components and integrate it into the main SDP execution path.

**The proposed plan (OpenRouterGateway + tools + sdp run) is necessary but insufficient.**
Building a real ModelGateway and basic tools is the minimal viable step. However, integration requires hardening tools, wiring the gateway, and full end-to-end testing.

---

### Architect (claude-sonnet-4-6 via codex:codex-rescue)

**Governance-over-Subprocess Is Structurally Broken, Not Just Inconvenient**
SDP has no visibility into intermediate tool calls, no ability to intercept or halt execution mid-phase, and no way to verify evidence beyond self-reported text. This is not a fixable gap; it is the wrong abstraction boundary.

**agentloop Has the Right Architecture but Zero Integration Surface**
A library with 128 tests and zero callers is a design artifact, not an operational component. The absence of a real ModelGateway and real tool implementations means the FSM, GateEngine, and EvidenceAccumulator are governance machinery with no process to govern.

**EvidenceAccumulator's Tool-Only Constraint Is a Critical Correctness Guarantee — Protect It**
Any path that allows an agent to write its own evidence record collapses the audit trail into self-attestation. When wiring OpenRouterGateway, enforce that evidence capture happens in the harness layer, not inside the model response handler.

**PhaseRouter Without Mandatory Gate Evaluation Is Decoration**
The five-phase map means nothing unless gate passage is a hard precondition for phase transition. The GateEngine's 5s circuit breaker suggests gates run concurrently rather than as blocking checkpoints. Clarify whether a gate failure causes rollback, halt, or escalation — and document which is enforced in code versus convention.

**The Proposed Plan Is Realistic but Sequencing Matters**
OpenRouterGateway plus standard tools in 3-4 days is credible for a single engineer who knows the codebase. Build and test gate-blocked phase transitions first; add the CLI subcommand after that invariant is verified under -race.

**CANONICAL_SDP_PIPELINE.md Must Be Retired or Promoted, Not Left Proposed**
A document in "Proposed" status that describes the canonical execution path is an authority vacuum. Either promote it to Accepted with a concrete mapping to agentloop constructs, or archive it and treat the agentloop code as the canonical spec.

---

## Round 2 — Review of v2 Artifact

### Technician

UNRESOLVED: Production Timeline is Still Non-Viable. 2-4 weeks remains optimistic. Integrating a production-grade tool layer with security, error handling, and sandboxing is a multi-month engineering project.

ADDRESSED: Unit Test Coverage is Not Integration. The artifact correctly downgrades the "128 tests."

NEW: Architectural Decision Introduces Major New Scope. Path A introduces an entire category of non-governance work (tool maintenance, sandboxing, driver compatibility). This decision requires significant feasibility analysis before implementation.

### Philosopher

UNRESOLVED: The council still equates "SDP-compliance" with "runs inside agentloop." A runtime pivot embeds one worldview; a protocol pivot keeps the space open for rival harnesses.

UNRESOLVED: EvidenceAccumulator records only tool outputs. Intent remains invisible; gates can audit but never veto in time.

NEW: "One real integration test" risks confirmation bias if written only inside Path A assumptions. If that test passes through a protocol stream with Claude Code/Codex, Path B is already viable.

### Critic

ADDRESSED: Delusion of Operational Readiness. The artifact correctly dismantles the 3-4 day timeline fantasy and recognizes that 128 unit tests against a StubGateway offer zero integration confidence. [Response truncated by model]

### Engineer

ADDRESSED: Timeline realism. Revised 2-8 week estimates are concrete and properly account for production tooling.

ADDRESSED: Agentloop isolation confirmed. The grep proof that agentloop isn't imported anywhere validates "dead component, not dead design."

UNRESOLVED: Protocol specification is still vapor. "JSON-RPC stream on stdout" remains a 2-sentence idea with no schema, versioning, or error recovery.

UNRESOLVED: Integration test description is aspirational. Which real card? Which LLM provider? What does evidence format look like?

### Pragmatist

Step 0 (pipeline fix) is correctly identified but hasn't been executed. Naming it without assigning ownership or deadline is ceremony.

Timeline revision is honest. Abandoning 3-4 days for production is correct.

Integration test priority is pragmatic but scope must be defined. For MVP: test discover + plan phases only, hard stop before build.

Path A vs B debate risks stalling before shipping. Path B requires 6-12 month adoption feedback loop. Path A is achievable in weeks. For MVP team: own the stack short-term, extract protocol once you understand the problem space.

EvidenceAccumulator intent gap is real but acceptable for v1. Tool outputs are sufficient for proving SDP can enforce phases. Intent tracking is v2.
