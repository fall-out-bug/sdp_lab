# LLM Council Dogfood Final Report

**Date:** 2026-04-10
**Rounds:** 3 of 5
**Consensus:** PARTIAL (4/5 resolved or better, 1 opposition narrowing)
**Subject:** `skills/llm-council.md` self-critique through 3 rounds

## Evolution Summary

### Round 1 → Round 2 Changes (10 fixes)
1. Dynamic quorum (80% of active models, frozen at round start)
2. Synthesis validation step with SYNTHESIS ERROR mechanism
3. Round 0 issue extraction with stable IDs + challenge step
4. Roles decoupled from models (capability-based, not model-binding)
5. Early abort (trivial input, convergence, budget cap, urgency flag)
6. Minority report preservation (verbatim, with attribution)
7. Persona anchoring in system prompts
8. Budget cap (3M tokens, 2.4M soft cap)
9. Orchestrator with bounded authority (CAN/CANNOT/MUST)
10. BLIND_REVIEW → REBUTTAL → VOTE protocol

### Round 2 → Round 3 Changes (5 fixes)
1. Decision Owner role (council is advisory, user decides)
2. Domain veto with override path (roles can block in expertise area, Decision Owner can override)
3. Deterministic artifact extraction (regex/truncation, not LLM summarization)
4. Invocation contract with full runtime flow
5. Max 2 synthesis revisions (prevents infinite loop)
6. Rebuttal token limit (500 tokens per review)
7. Quorum absolute floor (2/3 of configured models)
8. Convergence metric formula
9. Ledger mutation rules (merge, split, reopen, provenance)
10. Veto deadlock resolution (Decision Owner override)

## Round 3 Validation Results

| Model | Verdict | Key Assessment |
|-------|---------|---------------|
| **Critic** (Gemini) | OPPOSE (narrowing) | 5/8 concerns resolved; new: veto deadlock (fixed), summarizer compromise (fixed) |
| **Technician** (DeepSeek) | RESOLVED | All 7 feasibility concerns addressed, no new critical issues |
| **Philosopher** (Kimi) | RESOLVED | Philosophically viable; parallel tracks deferred to v2 |
| **Pragmatist** (MiniMax) | CONDITIONAL | Architecture sound; added invocation contract |
| **Engineer** (MiMo) | ABSENT | API failure in Round 3 |

**Convergence: 3/4 SUPPORT (Technician, Philosopher, Pragmatist), 1 OPPOSE (Critic narrowing)**

## Unresolved Items (Deferred to v2)

1. **Protocol recursion** — council cannot modify its own protocol mid-deliberation
2. **Parallel permanent tracks** — ontological forking for incommensurable views
3. **Epistemic asymmetry weighting** — domain expertise counts more than general vote
4. **Distributed consensus** — replace Orchestrator with rotating facilitation
5. **Full security audit** — comprehensive pen-testing of the pipeline

## Skill Artifacts

- **Skill file:** `skills/llm-council.md` (final v1)
- **Round 1 synthesis:** `docs/plans/2026-04-10-dogfood-r1-synthesis.md`
- **This report:** `docs/plans/2026-04-10-dogfood-final-report.md`
