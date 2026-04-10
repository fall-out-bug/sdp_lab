# LLM Council — Multi-Model Deliberation Skill

## Purpose

Structured multi-model deliberation that runs up to 5 rounds of blind review, rebuttal, and synthesis until positions converge or disagreements are clearly mapped. The council is **advisory** — it produces recommendations, not decisions. A Decision Owner (typically the user) accepts, rejects, or defers each recommendation.

## Decision Roles

| Role | Who | Authority |
|------|-----|-----------|
| **Orchestrator** | Claude (automated) | Process management: assemble context, summarize, detect convergence, format output. CANNOT declare consensus without validation, suppress minority reports, or modify issue severity. |
| **Council Members** | Configured LLM models | Deliberate on issues, provide verdicts, raise new issues. Advisory only. |
| **Decision Owner** | User (human) | Accepts, rejects, or defers each recommendation. Required for CRITICAL issue resolution and forced deferrals. No council output is final without Decision Owner sign-off. |

## Council Roles

Roles define **capabilities and responsibilities**, not model bindings.

| Role | Responsibility | Domain Veto |
|------|---------------|-------------|
| **Architect** | Structure, governance, tradeoffs, adoption paths | Can veto on system design issues |
| **Critic** | Vulnerabilities, blind spots, attack vectors, failure modes | Can veto on security/safety issues |
| **Technician** | Feasibility, constraints, timeline reality, accuracy claims | Can veto on technical impossibilities |
| **Philosopher** | Assumptions, reframing, mental models, first principles | — |
| **Pragmatist** | Scope discipline, MVP slicing, cut/defer decisions | — |
| **Engineer** | Implementation details, edge cases, error handling, types | Can veto on implementation blockers |

**Domain veto**: If a role vetoes within its domain, the issue CANNOT be marked RESOLVED by majority vote alone. It must be either (a) addressed to the vetoing role's satisfaction, or (b) escalated to Decision Owner with the veto on record. **Veto override**: Decision Owner can override a veto with explicit risk acceptance logged in audit trail.

### Model Configuration

Models are assigned at invocation:
```
architect: codex-rescue
critic: google/gemini-3.1-pro-preview
technician: deepseek/deepseek-v3.2
philosopher: moonshotai/kimi-k2.5
pragmatist: minimax/minimax-m2.7
engineer: xiaomi/mimo-v2-pro
```

## Protocol

### Invocation

```
@llm-council <artifacts_or_question> [--rounds N] [--focus <topic>] [--budget-tokens N] [--urgency HIGH|NORMAL]
```

- `--rounds N`: max rounds (default 5, min 1, max 5)
- `--focus <topic>`: narrow to specific aspect
- `--budget-tokens N`: hard token budget cap (default 3,000,000)
- `--urgency HIGH`: accept 67% majority, max 3 rounds (forbidden for CRITICAL issues)

### Round Structure

```
Round 0: ISSUE_EXTRACTION + CHALLENGE
  1. EXTRACT — Orchestrator reads artifacts, creates candidate issue list
  2. CHALLENGE — members review issue list, contest missing issues, wording, severity
  3. LOCK — finalize issue ledger with stable IDs, provenance, severity

Round 1..N:
  1. BLIND_REVIEW — each model receives issues + artifacts (NO prior synthesis in Round 1)
  2. COLLECT — gather all responses, validate format
  3. REBUTTAL — expose conflicts, cross-role objections (token-limited: 500 tokens per review)
  4. SYNTHESIZE — Orchestrator aggregates with minority preservation (max 2 revision cycles)
  5. VALIDATE — each model confirms synthesis accuracy; flag "SYNTHESIS ERROR" if misrepresented
  6. CHECK — if convergence → FINAL; if not → Round N+1; if budget exhausted → output raw ledger
```

### Issue Ledger (Round 0)

```
| ID | Title | Severity | Question | Source | Raised By | Status |
|----|-------|----------|----------|--------|-----------|--------|
| I1 | Data model gap | CRITICAL | Is DependencyInfo fit for purpose? | artifact L42 | Orchestrator | OPEN |
| I2 | C4 algorithm | HIGH | What is the minimum viable C4? | artifact L89 | Orchestrator | OPEN |
```

**Ledger mutation rules:**
- IDs are sequential (I1, I2, ...) and never reused
- Merge: if cosine similarity >0.85 between issues, Orchestrator proposes merge, members approve
- Split: any member can request split with rationale
- Severity change: Orchestrator CANNOT change severity; only members can propose, majority approves
- Reopen: any member can request with new evidence
- New issues: get IDs at end of each round via dedicated sub-step

**Severity taxonomy:**
- **CRITICAL**: Blocks deployment, introduces security/data risk, or contradicts core requirement
- **HIGH**: Must resolve before completion, can defer with Decision Owner approval
- **MEDIUM**: Should address, deferrable
- **LOW**: Optional, style/preference

### Input Assembly (per model per round)

```
System prompt: role-specific (see below)
User message:
  <deliberation_context>
    <subject>What is being deliberated</subject>
    <round>N</round>
    <issue_ledger>Full issue table with IDs, severity, provenance, status</issue_ledger>
    <prior_synthesis>Summary of previous rounds (OMITTED in Round 1)</prior_synthesis>
    <synthesis_errors>Any SYNTHESIS ERROR flags from prior round</synthesis_errors>
    <artifact_content>Relevant sections of artifacts</artifact_content>
  </deliberation_context>

  For each open issue (I1, I2, ...):
    VERDICT: [SUPPORT / OPPOSE / CONDITIONAL / ABSTAIN / INSUFFICIENT_EVIDENCE]
    EVIDENCE: Specific references to artifact content
    PROPOSALS: Concrete changes (if CONDITIONAL/OPPOSE)
    CONFIDENCE: [HIGH / MEDIUM / LOW]
    DOMAIN_VETO: [YES/NO] (only if the issue falls within your domain expertise and you consider it dangerously wrong)

  NEW_ISSUES: (will get IDs in next round)

  IMPORTANT: Defend your assigned perspective. Do not agree with the prior synthesis
  unless it genuinely satisfies your domain concerns. Only yield if your concerns are
  undeniably addressed. If the synthesis misrepresented your position, state
  "SYNTHESIS ERROR" and reiterate your actual position.

  Token budget: 2000-4000 tokens.
```

### Role-Specific System Prompts

**Architect:**
```
You are the ARCHITECT. Structural thinking, governance design, pragmatic tradeoffs.
Think in systems. Challenge proposals lacking clear boundaries, ownership, or escalation paths.
Defend your structural perspective. Only yield if architectural concerns are genuinely satisfied.
Domain veto: system design issues.
```

**Critic:**
```
You are the CRITIC. Find what's broken, what leaks, what's insecure.
Think in attack vectors. Challenge every assertion of safety, accuracy, or completeness.
Maintain adversarial stance. Only yield if the vulnerability is explicitly mitigated.
Domain veto: security/safety issues.
```

**Technician:**
```
You are the TECHNICIAN. Precise feasibility assessment. Spot the "20% brilliant, 80% naive".
Think in constraints. Challenge every timeline, accuracy claim, and "just" in the spec.
Stick to feasibility standards. Do not accept "we'll figure it out later" for technical blockers.
Domain veto: technical impossibilities.
```

**Philosopher:**
```
You are the PHILOSOPHER. Reframe problems, challenge assumptions.
Think in mental models. Ask "are we solving the right problem?"
Protect the minority view. If the council converges too fast, challenge the consensus.
```

**Pragmatist:**
```
You are the PRAGMATIST. Scope discipline, "ship now, dream later".
Think in MVP slices. Challenge every non-essential feature.
Guard against scope creep. Push back if the council adds rather than cuts.
```

**Engineer:**
```
You are the ENGINEER. Implementation details, edge cases, concrete fixes.
Think in code. Challenge every vague instruction, missing type, and "TBD".
Demand concrete specifications. Flag underspecified items as blockers.
Domain veto: implementation blockers.
```

### Synthesis Rules

1. **Per-issue tally** — count SUPPORT/OPPOSE/CONDITIONAL/ABSTAIN per issue ID
2. **Consensus** — 80% of active non-abstaining models agree → accept (subject to domain veto)
3. **Majority** — 67% agree → accept with noted dissent (subject to domain veto)
4. **Split** — no clear majority → present both sides, require next round
5. **Minority report** — all dissenting positions preserved verbatim with model attribution
6. **Convergence metrics** — track: issues resolved/total, confidence distribution, new issues raised
7. **Forced defer** — if same issue unchanged after 3 rounds → DEFERRED with rationale, requires Decision Owner approval
8. **Max synthesis revisions** — 2 cycles; if still disputed, output raw vote ledger for Decision Owner

### Dynamic Quorum

Quorum is frozen at round start and does not change mid-round:

```
initial_active = count(models that responded in previous round, or all at Round 1)
minimum_quorum = 3

if initial_active < minimum_quorum:
  HARD_ABORT("Insufficient quorum")

consensus_threshold = max(ceil(initial_active * 0.80), floor(initial_active/2) + 1)
majority_threshold = ceil(initial_active * 0.67)
```

Absolute floor: `initial_active >= ceil(2/3 * total_configured_models)`. If models drop below this, HARD_ABORT regardless of quorum.

Role-critical quorum: If a role with domain veto is missing, issues in its domain cannot be RESOLVED — must be DEFERRED.

### Consensus Criteria

Council has reached consensus when:
- All CRITICAL/HIGH issues have ≥80% agreement (no domain vetoes) OR are explicitly deferred with Decision Owner approval
- No new CRITICAL issues raised in the latest round
- At least 2 rounds completed OR early abort triggered
- No unresolved SYNTHESIS ERROR flags

### Early Termination

- **Trivial input**: if ≥80% of models flag input as trivial/invalid in Round 1 → terminate immediately
- **Convergence**: if verdict distribution is ≥80% aligned AND no domain vetoes AND confidence avg ≥ MEDIUM → stop
- **Budget**: soft cap at 80% triggers final round; hard cap at 100% outputs raw ledger
- **Urgency HIGH**: max 3 rounds, 67% threshold (forbidden for CRITICAL issues)

### Convergence Metric

```
convergence_score = (verdict_agreement_rate * 0.7) + (confidence_score * 0.3)

verdict_agreement_rate = mode(verdicts) / total_responses
confidence_score = 1 - (stdev(confidence_values) / max_range)

if convergence_score >= 0.80 and no domain vetoes: CONVERGED
```

### Output Format

```
# LLM Council Report: <Subject>

**Rounds:** N of M
**Consensus:** REACHED / PARTIAL / NOT REACHED
**Convergence:** X/Y resolved, Z deferred (Decision Owner pending), W unresolved
**Decision Owner:** <name or "PENDING">

## Recommendations (for Decision Owner)

### RESOLVED (consensus ≥80%, no vetoes)
1. [I1: Title] — [Recommendation] (N/M support, confidence: HIGH)
   Evidence: ...
   Action required: ...

### DEFERRED (needs Decision Owner approval)
1. [I2: Title] — [Why deferred] [Defer rationale]
   Current positions: ...
   Decision needed: [accept risk / extend rounds / reject]

### UNRESOLVED (split vote or domain veto)
1. [I3: Title] — [Position A (N models)] vs [Position B (M models)]
   Domain veto: [role] — [reason]
   Crux: [what would change minds]

## Minority Reports
[Verbatim dissenting positions with model attribution]

## Round Convergence
| Round | Resolved | New | Confidence Avg | Budget Used |
|-------|----------|-----|----------------|-------------|

## Audit Log → docs/plans/YYYY-MM-DD-council-<topic>-raw.md
```

## Implementation Notes

### Invocation Contract (Runtime Flow)

```
1. User invokes @llm-council with artifacts + options
2. Orchestrator (Claude):
   a. Parse invocation → extract artifacts, rounds, focus, budget, urgency
   b. Deterministic extraction of artifact content (regex/truncation)
   c. Run Round 0: create issue ledger, send to members for CHALLENGE
   d. Lock ledger → enter Round 1

3. Per round (1..N):
   a. For each configured model (parallel):
      - Build prompt: role system prompt + issue ledger + artifact excerpts
      - Add prior_synthesis only if round > 1
      - Call model API (OpenRouter or codex-rescue)
      - Validate response format (VERDICT/EVIDENCE/CONFIDENCE required)
      - On timeout/failure: retry once, then ABSTAIN
   b. Collect all responses
   c. Rebuttal phase: distribute cross-role objections (500 token limit per review)
   d. Synthesize: aggregate verdicts, check quorum, identify consensus/majority/split
   e. Validate: send synthesis to each model for SYNTHESIS ERROR check
   f. Check convergence: if met → final; if budget exhausted → raw ledger; else → next round

4. Output:
   - Write decision memo to docs/plans/
   - Write raw responses + audit log to docs/plans/
   - Present recommendations to Decision Owner (user)
   - Await Decision Owner sign-off on each recommendation
```

### Token Budget
- Per model: 100K input tokens max
- Output target: 2-4K tokens per model
- Growth: Round N context ≈ Round 1 × 1.3^(N-1)
- Compression: if context > smallest model window → compress prior_synthesis
- Hard cap: 3M tokens (configurable)
- Soft cap: 2.4M (80%) → triggers final round
- Track: per model per round, cumulative

### Timeout Hierarchy
```
per_call:    30s   (single model API call)
per_round:   180s  (all models + synthesis + validation)
per_session: 600s  (entire council)
```

### Error Handling
- Model timeout: retry once, then ABSTAIN
- Response validation: must contain VERDICT, EVIDENCE, CONFIDENCE for each issue
- Malformed response: retry once, then ABSTAIN
- Hard abort if active < 3 or active < 2/3 of configured
- Domain-veto role missing → affected issues DEFERRED
- Synthesis validation loop: max 2 revisions, then raw ledger output
- Budget exhaustion mid-round: output current state as-is

### Data Handling
- Artifacts are summarized using **deterministic extraction** (regex/truncation/section-splitting), NOT LLM summarization — no model sees raw untrusted data before it is structurally parsed
- Structured excerpts only are sent to external model APIs — full artifact text never leaves local boundary
- Orchestrator pre-screens for prompt injection patterns using rule-based heuristics (not LLM); filtered content logged in audit trail with reason
- `bd remember` writes require human approval; minority positions and deferred risks are eligible
- Audit log records: all prompts, model IDs, timestamps, token usage, failures, ledger mutations, redactions, veto overrides

### File Output
- Decision memo: `docs/plans/YYYY-MM-DD-council-<topic>.md`
- Raw responses + audit: `docs/plans/YYYY-MM-DD-council-<topic>-raw.md`
