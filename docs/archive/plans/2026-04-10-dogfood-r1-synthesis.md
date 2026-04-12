# LLM Council Dogfood Report: Skill Design Critique

**Date:** 2026-04-10
**Rounds:** 1 of 5
**Consensus:** NOT REACHED — critical structural gaps found
**Subject:** `skills/llm-council.md` self-critique (dogfooding)

## Council Members

| Role | Model | Status | Response Size |
|------|-------|--------|---------------|
| **Critic** | Gemini 3.1 Pro | Complete | 7.1 KB |
| **Technician** | DeepSeek V3.2 | Complete | 5.8 KB |
| **Philosopher** | Kimi K2.5 | Complete | 6.0 KB |
| **Pragmatist** | MiniMax M2.7 | Complete | 6.0 KB |
| **Engineer** | MiMo V2 Pro | Complete | 6.1 KB |
| **Architect** | GPT 5.4 (Codex) | Complete | ~4.0 KB |

---

## Round 1 Synthesis

### UNANIMOUS CONSENSUS (6/6)

| # | Finding | Action |
|---|---------|--------|
| 1 | **VERDICT: CONDITIONAL** — skill is not production-ready | Fix structural gaps before use |
| 2 | **Synthesizer is SPOF** — Claude aggregates, judges convergence, controls framing with zero accountability | Add synthesis validation, minority protection |
| 3 | **Consensus math breaks on model failure** — "≥5/6" impossible with 4 active models | Dynamic quorum based on active models |
| 4 | **No early termination** — forced minimum 2 rounds even for trivial inputs, no cost ceiling | Add early abort + budget cap |
| 5 | **Roles coupled to model names** — skill breaks when models rename/discontinue | Define capabilities, not model bindings |

### STRONG MAJORITY (5/6)

| # | Finding | Dissent |
|---|---------|---------|
| 6 | **No Round 0 / issue ledger** — issues are free-form, no stable IDs, no severity taxonomy | — |
| 7 | **No defense/rebuttal phase** — protocol is "coordinated review", not deliberation | — |
| 8 | **Cost controls absent** — no budget cap, no token tracking, no cost ceiling | — |
| 9 | **Prompt injection vulnerability** — malicious artifact payload hijacks all 6 models simultaneously | Philosopher: secondary to structural issues |

### MAJORITY (4/6)

| # | Finding | Dissent |
|---|---------|---------|
| 10 | **Sycophancy/context anchoring** — feeding prior synthesis biases subsequent rounds | Philosopher: design for dissensus, not convergence |
| 11 | **Output format underspecified** — "council report" has no schema | — |
| 12 | **Missing Orchestrator role** — "who drives the loop?" is unanswered | — |

### SPLIT VOTES

| # | Issue | Position A | Position B |
|---|-------|-----------|-----------|
| 13 | **Consensus goal** | Consensus-seeking with dynamic quorum (Critic, Engineer, Pragmatist, Technician) | Dissensus preservation — clarity of disagreement over agreement (Philosopher, Architect) |
| 14 | **Role allocation** | Fixed roles with capability requirements (Pragmatist, Engineer, Architect) | Dynamic role auction per problem (Philosopher, Technician) |

---

## Per-Model Highlights

### Critic (Gemini 3.1 Pro) — Security & Exploit Focus
- **Denominator Paradox:** 5/6 consensus impossible with 4 active models — council loops to Round 5 burning tokens
- **Synthesizer Dictatorship:** Claude controls framing with no validation from council members
- **Denial of Wallet:** No early abort, forced 2-round minimum wastes tokens on garbage input
- **Persona Anchoring:** Models will abandon roles to agree with perceived majority by Round 3

### Technician (DeepSeek V3.2) — Feasibility
- **Token budgeting naive:** Doesn't account for context growth across rounds (1.4x per round)
- **No response validation:** Models may output wrong format, no schema enforcement
- **Cost control absent:** No hard budget stop
- **Missing fallback matrix:** No backup models if primary fails

### Philosopher (Kimi K2.5) — Reframing
- **Category error:** Fixed roles = "brand essentialism masquerading as cognitive architecture"
- **Consensus bias:** 5/6 threshold creates "supermajority tyranny", truth often lies with minority
- **Synthesis fallacy:** Assumes commensurability of frameworks; may produce chimera solutions
- **Proposal:** Replace consensus with "structured dissensus with crux identification"

### Pragmatist (MiniMax M2.7) — Scope Cutting
- **Skill is half-designed:** Defines content but not execution mechanism
- **Missing Orchestrator:** "Who runs this?" is unanswered
- **Output schema absent:** "Council report" is undefined
- **Role-model coupling is a design smell:** Will rot as models change
- **Bottom line:** "Ship only after adding Orchestrator, early termination, and output schema"

### Engineer (MiMo V2 Pro) — Implementation
- **Token budget calculation flawed:** No growth factor, no tokenizer variance
- **Consensus edge cases:** No abstention handling, no forced defer after 3 rounds
- **Error handling insufficient:** Different providers have different timeout limits
- **Cost control missing:** No max_tokens_per_session

### Architect (GPT 5.4) — Governance Design
- **No governance model:** Synthesis ≠ decision; need Convenor/Members/Decision Owner separation
- **No issue register:** Round 0 required with stable IDs and severity
- **Protocol contaminates discovery:** Feeding prior synthesis anchors all models to chair's framing
- **Verdict schema incomplete:** Need ABSTAIN, INSUFFICIENT_EVIDENCE states
- **Data governance absent:** Structurally unsafe for confidential data

---

## Fixes Required Before Round 2

1. **Dynamic quorum** — consensus = 80% of active non-abstaining models, minimum 3
2. **Synthesis accountability** — models must validate prior synthesis ("Did it capture your position?")
3. **Early abort** — if ≥80% models flag input as trivial/invalid, terminate after Round 1
4. **Round 0** — issue extraction with stable IDs before deliberation
5. **Decouple roles from models** — define capabilities, accept model config externally
6. **Budget cap** — hard token/cost ceiling with early termination
7. **Output schema** — structured JSON with resolved/deferred/unresolved
8. **Defense/rebuttal phase** — change from DELIBERATE→COLLECT to BLIND_REVIEW→REBUTTAL→VOTE
9. **Minority report preservation** — dissenting views must survive into final output
10. **Persona anchoring** — system prompts must instruct models to defend their role, not converge
